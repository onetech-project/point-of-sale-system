# Implementation Notes — Landing Page, Subscription Management & Email Notifications

This document summarises all changes made in the
`copilot/start-implementation-multiple-agents` branch.

---

## Verified Status

Implemented and verified:
- Public landing page at `/` with authenticated redirect to `/dashboard`.
- Runtime pricing fetch through `GET /api/v1/public/plans`.
- Tenant trial/subscription fields, 7-day trial registration flow, and 2 GB quota constant.
- Billing service for subscription status, upgrade invoices, Midtrans payment initiation, webhooks, and renewal/trial jobs.
- Notification handlers and templates for trial and invoice lifecycle events.

Completed follow-up fixes:
- Fresh setup now includes billing env templates and verification.
- `BILLING_SERVICE_URL` is defined for API Gateway.
- Root Compose is documented as infrastructure-only for local development.
- `invoice.generated`, `invoice.paid`, and `invoice.payment_failed` are the canonical billing notification events.
- `subscription.payment_received` remains a legacy alias only.
- Trial notification payloads now use `tenant_name`, with `business_name` accepted as a fallback.
- Storage quota migration safety is handled by `000067_normalize_trial_storage_quota`.
- `000065` rollback no longer drops storage columns owned by the product-photo migration.
- Expired tenants are redirected to a focused subscription recovery page after login and on `402 Payment Required`.
- Grace-period tenants keep normal app access with a red warning banner.
- Tenant operational data cleanup is eligible 30 days after grace period starts.
- Terms of Service acceptance is required and recorded during signup.
- Frontend subscription status now uses a shared cached provider with in-flight request deduplication.
- Billing service invalidates the API Gateway subscription cache after successful payment and job-driven status changes.

Remaining operational notes:
- Production Midtrans credentials must be configured before testing real subscription payments.
- Application containers in root Compose are intentionally not enabled; use `./scripts/start-all.sh all` for local services.

---

## 1. Public Landing Page

### What was done
The root route (`/`) previously redirected all visitors to `/login`.
It now renders a full marketing landing page for unauthenticated users while
still redirecting authenticated users to `/dashboard`.

### Files changed
| File | Change |
|------|--------|
| `frontend/app/page.tsx` | Renders `<LandingPage />` for guests; redirects authenticated users to `/dashboard` |
| `frontend/src/components/landing/LandingPage.tsx` | Composer — wraps all sections inside `PublicLayout` |
| `frontend/src/components/landing/HeroSection.tsx` | Headline, dual CTAs, mock POS dashboard visual |
| `frontend/src/components/landing/FeaturesSection.tsx` | Six feature cards with inline SVG icons |
| `frontend/src/components/landing/PricingSection.tsx` | Pricing card with monthly / annual billing toggle |
| `frontend/src/components/landing/CtaSection.tsx` | Gradient call-to-action section |
| `frontend/src/utils/general.ts` | Added `/` to `PUBLIC_PAGES` so the route is not gated |

### Landing page sections

#### Hero
- Headline: *"Manage Your Business Smarter with Posku"*
- Sub-copy highlighting the 7-day free trial with no credit card required
- Two CTAs: **Start Free Trial** (→ `/signup`) and **Learn More** (smooth-scrolls to features)
- Mock POS dashboard card showing sample sales stats

#### Features
Six feature highlights rendered as a grid of cards with inline SVG icons:
1. Multi-Tenant workspaces
2. Inventory management & low-stock alerts
3. Online ordering via shareable QR menu
4. QRIS & credit card payments (Midtrans)
5. Analytics dashboard & revenue reports
6. Team management with role-based access

#### Pricing
- **Single plan** card that is data-driven: pricing values are fetched at runtime
  from `GET /api/v1/public/plans` (see §3) and fall back to hard-coded defaults
  if the endpoint is unavailable.
- Monthly / Annual billing toggle.
- Annual view shows per-month equivalent price, yearly total, and savings amount.
- "Save X%" badge dynamically reflects the `annual_discount_pct` returned by the
  API (default **20 %**).
- Lists all included features in a two-column grid.

#### CTA
- Gradient section: *"Try Posku free for 7 days — no credit card required."*
- **Start Free Trial Now** button (→ `/signup`)

---

## 2. Subscription Model

### What was done
Added first-class subscription management to the tenant data model and
registration flow so that every new tenant automatically starts a 7-day free
trial with a 2 GB storage quota.

### Files changed
| File | Change |
|------|--------|
| `backend/tenant-service/src/models/tenant.go` | New fields, typed constants, updated `TenantResponse` / `ToResponse()` |
| `backend/tenant-service/src/repository/tenant_repository.go` | `Create`, `FindBySlug`, `FindByID` updated |
| `backend/tenant-service/src/services/tenant_service.go` | Sets quota; fires `subscription.trial_started` event |
| `backend/tenant-service/src/queue/event_publisher.go` | `PublishTrialStarted` method added |
| `backend/migrations/000065_add_subscription_fields.up.sql` | Schema migration (up) |
| `backend/migrations/000065_add_subscription_fields.down.sql` | Schema migration (down) |
| `backend/migrations/000067_normalize_trial_storage_quota.up.sql` | Normalizes old 5 GB default quota rows to 2 GB |
| `backend/migrations/000067_normalize_trial_storage_quota.down.sql` | Restores earlier default without rewriting tenant-specific quotas |
| `backend/migrations/000068_subscription_retention_terms.up.sql` | Adds retention timestamps, Terms acceptance records, and nullable order item product references |
| `backend/migrations/000068_subscription_retention_terms.down.sql` | Rolls back retention and Terms schema changes |

### New `Tenant` fields
```go
SubscriptionPlan    string     // "trial" | "starter" | "professional" | "enterprise"
BillingCycle        string     // "monthly" | "annual"
TrialStartedAt      *time.Time
TrialEndsAt         *time.Time
SubscribedAt        *time.Time
SubscriptionEndsAt  *time.Time
RetentionStartedAt *time.Time
DataAnonymizedAt   *time.Time
```

### New constants
```go
DefaultStorageQuotaBytes    int64 = 2 * 1024 * 1024 * 1024  // 2 GB
TrialDurationDays           int   = 7
DefaultAnnualDiscountPercent int  = 20
```

### Registration flow
1. `RegisterTenant` sets `StorageQuotaBytes = 2 GB` on the tenant before it is
   persisted.
2. The repository `Create` method automatically fills `subscription_plan = trial`,
   `billing_cycle = monthly`, `trial_started_at = NOW()`,
   `trial_ends_at = NOW() + 7 days`, and the storage fields.
3. After the transaction commits a `subscription.trial_started` Kafka event is
   published asynchronously.
4. Signup requires `terms_accepted = true` and `terms_version = "1.0.0"`.
5. Terms acceptance is stored with tenant ID, owner user ID, version, timestamp,
   IP address, and user agent.

### Expiry and retention rules
- Free trial length: **7 days**.
- Access grace period after trial/subscription expiry: **7 days**.
- Operational-data retention window: **30 days from grace-period start**.
- When billing jobs move a tenant into `grace_period`,
  `subscription_retention_started_at` is set if empty.
- Existing `grace_period` or `expired` tenants are backfilled with
  `subscription_retention_started_at = NOW()` by migration `000068`.
- Payment before cleanup reactivates the tenant and clears
  `subscription_retention_started_at`.
- Payment after cleanup can reactivate the account, but cleaned operational data
  remains gone.
- Billing invoices, payment attempts, Terms acceptance records, consent records,
  audit events, and financial order rows are preserved as historical/compliance
  records.
- Customer PII in preserved financial orders is anonymized during retention
  cleanup.

### Database migration (`000065`)
```sql
ALTER TABLE tenants
  ADD COLUMN subscription_plan     VARCHAR(20) NOT NULL DEFAULT 'trial' CHECK (...),
  ADD COLUMN billing_cycle         VARCHAR(10) NOT NULL DEFAULT 'monthly' CHECK (...),
  ADD COLUMN trial_started_at      TIMESTAMPTZ,
  ADD COLUMN trial_ends_at         TIMESTAMPTZ,
  ADD COLUMN subscribed_at         TIMESTAMPTZ,
  ADD COLUMN subscription_ends_at  TIMESTAMPTZ,
  ADD COLUMN storage_quota_bytes   BIGINT NOT NULL DEFAULT 2147483648,
  ADD COLUMN storage_used_bytes    BIGINT NOT NULL DEFAULT 0;
```
Existing rows are back-filled: `trial_started_at = created_at`,
`trial_ends_at = created_at + 7 days`.

Indexes added: `idx_tenants_subscription_plan`, `idx_tenants_trial_ends_at`,
`idx_tenants_subscription_ends_at`.

---

## 3. Public Plans API Endpoint

### What was done
Added `GET /api/v1/public/plans` (no auth required) so the frontend can read
current pricing at runtime rather than having it baked into the build.

### Files changed
| File | Change |
|------|--------|
| `backend/billing-service/api/billing_handler.go` | Public plan handler — reads billing env vars with fallbacks |
| `backend/billing-service/main.go` | Route registered as `/public/plans` |
| `api-gateway/main.go` | Public URL proxies `/api/v1/public/plans` to billing-service |

### Handler behaviour
Values are read from environment variables so they can be updated without a code
deploy:

| Env var | Default |
|---------|---------|
| `PLAN_MONTHLY_PRICE_IDR` | `299000` |
| `PLAN_ANNUAL_DISCOUNT_PCT` | `20` |
| `PLAN_TRIAL_DAYS` | `7` |

Example response:
```json
{
  "monthly_price_idr":  299000,
  "annual_discount_pct": 20,
  "annual_price_idr":   2870400,
  "trial_days":         7
}
```

---

## 4. Email Notifications

### What was done
Added email notification support for the subscription lifecycle:
trial started, trial about to expire, invoice generated, invoice paid, and payment failed.

### Files changed
| File | Change |
|------|--------|
| `backend/notification-service/src/models/events.go` | Subscription and billing event type constants |
| `backend/notification-service/src/services/notification_service.go` | Trial and invoice lifecycle handlers + templates loaded |
| `backend/notification-service/templates/trial_started.html` | New trial started template |
| `backend/notification-service/templates/trial_ending.html` | New email template |
| `backend/notification-service/templates/payment_received.html` | New email template |
| `backend/notification-service/templates/invoice_generated.html` | New invoice generated template |
| `backend/notification-service/templates/invoice_paid.html` | New invoice paid template |
| `backend/notification-service/templates/invoice_payment_failed.html` | New invoice payment failed template |

### Billing event types
| Kafka event type | Trigger |
|-----------------|---------|
| `subscription.trial_started` | Fired immediately after a new tenant registers |
| `subscription.trial_ending` | Fired by billing-service when trial has <= N days left |
| `invoice.generated` | Fired when billing-service generates a renewal invoice |
| `invoice.paid` | Fired after a successful billing invoice payment |
| `invoice.payment_failed` | Fired after a failed billing invoice payment |
| `subscription.payment_received` | Legacy alias handled by notification-service only |

### `trial_started.html`
- Confirms the 7-day trial has started
- Shows trial end time when available
- **Go to Dashboard** button
- **View subscription options** link

### `trial_ending.html`
- Warns the tenant that the trial expires in `{{.DaysRemaining}}` day(s)
- Lists the features they have been using during the trial
- **Upgrade Now** button (-> `/subscription`)
- **View Pricing** secondary link

### Invoice templates
- `invoice_generated.html` sends pending invoice details and payment link
- `invoice_paid.html` confirms a successful invoice payment
- `invoice_payment_failed.html` asks the owner to retry payment

### `payment_received.html`
- Legacy template for `subscription.payment_received`
- Retained for backward compatibility with older event producers

These templates match the existing Posku email style (indigo header `#4F46E5`,
light-grey content area, auto-copyright footer).

---

## Environment Variables Reference

| Service | Variable | Purpose | Default |
|---------|----------|---------|---------|
| `billing-service` | `BILLING_SERVICE_URL` | API Gateway target for billing endpoints | `http://localhost:8090` |
| `billing-service` | `PLAN_MONTHLY_PRICE_IDR` | Base monthly subscription price (IDR) | `299000` |
| `billing-service` | `PLAN_ANNUAL_DISCOUNT_PCT` | Annual billing discount percentage | `20` |
| `billing-service` | `PLAN_TRIAL_DAYS` | Free trial length in days shown in public pricing | `7` |
| `billing-service` | `PLAN_GRACE_PERIOD_DAYS` | Grace period after trial/subscription expiry | `7` |
| `billing-service` | `PLAN_RETENTION_DAYS` | Operational-data retention window from grace-period start | `30` |
| `billing-service` | `KAFKA_AUDIT_TOPIC` | Audit topic for retention cleanup events | `audit-events` |
| `billing-service` | `REDIS_HOST` | Redis address for API Gateway subscription-cache invalidation | `localhost:6379` |
| `billing-service` | `REDIS_PASSWORD` | Redis password for cache invalidation | `pos_password` |
| `billing-service` | `MIDTRANS_SERVER_KEY` | Midtrans server key for subscription payments | `YOUR_SERVER_KEY_HERE` |
| `billing-service` | `MIDTRANS_ENV` | Midtrans environment | `sandbox` |

---

## Architecture Overview

```
Browser (Landing Page)
  └─ GET /api/v1/public/plans ──► api-gateway ──► billing-service /public/plans
                                                       └─ reads env vars

User clicks "Start Free Trial"
  └─ POST /api/v1/tenants/register ──► tenant-service
        ├─ Creates tenant with trial subscription & 2 GB quota (DB)
        ├─ Publishes user.registered ──► notification-service ──► Welcome email
        └─ Publishes subscription.trial_started ──► notification-service ──► Trial started email

Billing service background jobs
  ├─ Publishes subscription.trial_ending ──► notification-service ──► Trial expiry warning
  ├─ Moves expired trial/subscription tenants to grace_period and starts retention timer
  ├─ Moves grace-expired tenants to expired after 7 days
  ├─ Runs 30-day retention cleanup for grace/expired tenants
  ├─ Publishes invoice.generated ──► notification-service ──► Invoice email
  ├─ Publishes invoice.paid ──► notification-service ──► Payment confirmation
  ├─ Publishes invoice.payment_failed ──► notification-service ──► Retry payment email
  └─ Publishes tenant subscription-data anonymization events ──► audit topic
```

## 5. Expired Subscription Recovery and Terms

### User experience
- Login checks `GET /api/v1/billing/subscription` after authentication.
- `expired` tenants are redirected to `/subscription?reason=expired`.
- Protected API calls returning `402 Payment Required` redirect to the same
  subscription recovery route unless the user is already under `/subscription`.
- Frontend subscription state is cached for 60 seconds, deduplicates concurrent
  requests, refreshes on focus/visibility, and can be explicitly invalidated
  after payment initiation or a `402` response.
- The expired recovery page does not render dashboard navigation or unrelated
  dashboard content. It shows suspended status, payment CTA, invoice link,
  retention warning, and logout.
- `grace_period` tenants stay in the normal app and see a red warning banner.

### Gateway cache behavior
- API Gateway caches subscription status in Redis under `sub:{tenant_id}`.
- Normal active/trial status uses a 5-minute TTL.
- `grace_period` status uses a 1-minute TTL.
- `expired` and `cancelled` status use a 30-second TTL.
- Billing service deletes the same cache key after successful payment and after
  background jobs move a tenant into `grace_period` or `expired`.

### Public interfaces
`POST /api/tenants/register` now requires:

```json
{
  "terms_accepted": true,
  "terms_version": "1.0.0"
}
```

`GET /api/v1/billing/subscription` now includes:

```json
{
  "retention_started_at": "2026-05-01T00:00:00Z",
  "retention_cleanup_at": "2026-05-31T00:00:00Z",
  "data_anonymized_at": null
}
```

The public Terms page is available at `/terms-of-service`.
