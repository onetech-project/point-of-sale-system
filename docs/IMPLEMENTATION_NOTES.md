# Implementation Notes — Landing Page, Subscription Management & Email Notifications

This document summarises all changes made in the
`copilot/start-implementation-multiple-agents` branch.

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

### New `Tenant` fields
```go
SubscriptionPlan    string     // "trial" | "starter" | "professional" | "enterprise"
BillingCycle        string     // "monthly" | "annual"
TrialStartedAt      *time.Time
TrialEndsAt         *time.Time
SubscribedAt        *time.Time
SubscriptionEndsAt  *time.Time
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
| `backend/tenant-service/api/plan_handler.go` | New handler — reads env vars with fallbacks |
| `backend/tenant-service/main.go` | Route registered |
| `api-gateway/main.go` | Proxy rule added for `/api/v1/public/plans` |

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
trial started, trial about to expire, and payment received.

### Files changed
| File | Change |
|------|--------|
| `backend/notification-service/src/models/events.go` | Three new event type constants |
| `backend/notification-service/src/services/notification_service.go` | Three new handlers + templates loaded |
| `backend/notification-service/templates/trial_ending.html` | New email template |
| `backend/notification-service/templates/payment_received.html` | New email template |

### New event types
| Kafka event type | Trigger |
|-----------------|---------|
| `subscription.trial_started` | Fired immediately after a new tenant registers |
| `subscription.trial_ending` | Fired by a scheduler when trial has ≤ N days left |
| `subscription.payment_received` | Fired after a successful subscription payment |

### `trial_ending.html`
- Warns the tenant that the trial expires in `{{.DaysRemaining}}` day(s)
- Lists the features they have been using during the trial
- **Upgrade Now** button (→ `/settings/subscription`)
- **View Pricing** secondary link

### `payment_received.html`
- Confirms a successful payment
- Subscription details table: Plan, Billing Cycle, Amount, Next Billing Date
- **View Invoice** and **Go to Dashboard** buttons

Both templates match the existing Posku email style (indigo header `#4F46E5`,
light-grey content area, auto-copyright footer).

---

## Environment Variables Reference

| Service | Variable | Purpose | Default |
|---------|----------|---------|---------|
| `tenant-service` | `PLAN_MONTHLY_PRICE_IDR` | Base monthly subscription price (IDR) | `299000` |
| `tenant-service` | `PLAN_ANNUAL_DISCOUNT_PCT` | Annual billing discount percentage | `20` |
| `tenant-service` | `PLAN_TRIAL_DAYS` | Free trial length in days | `7` |

---

## Architecture Overview

```
Browser (Landing Page)
  └─ GET /api/v1/public/plans ──► api-gateway ──► tenant-service /public/plans
                                                       └─ reads env vars

User clicks "Start Free Trial"
  └─ POST /api/v1/tenants/register ──► tenant-service
        ├─ Creates tenant with trial subscription & 2 GB quota (DB)
        ├─ Publishes user.registered ──► notification-service ──► Welcome email
        └─ Publishes subscription.trial_started ──► notification-service ──► Trial started email

[Scheduler — future]
  └─ Publishes subscription.trial_ending ──► notification-service ──► Trial expiry warning

[Payment service — future]
  └─ Publishes subscription.payment_received ──► notification-service ──► Payment confirmation
```
