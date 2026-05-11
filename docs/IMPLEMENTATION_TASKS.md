# Implementation Tasks - Subscription, Landing, and Billing

## P0 Runtime Wiring

- [x] Add tracked billing service environment template.
- [x] Add billing port, URL, and plan defaults to root environment example.
- [x] Add `BILLING_SERVICE_URL` to API Gateway environment example.
- [x] Include billing service in `setup-env.sh` and `verify-env.sh`.
- [x] Fix billing output formatting in `start-all.sh`.
- [x] Include billing service in `stop-all.sh`.

## P1 Data and Event Contracts

- [x] Keep `invoice.generated`, `invoice.paid`, and `invoice.payment_failed` as canonical billing events.
- [x] Keep `subscription.payment_received` as a legacy notification alias.
- [x] Standardize billing notification payloads on `tenant_name`.
- [x] Add a dedicated `trial_started.html` template.
- [x] Add migration `000067` to normalize default tenant storage quota to 2 GB.
- [x] Fix `000065` rollback so it does not drop product-photo storage columns.

## P2 Documentation

- [x] Update implementation notes with verified status and remaining operational notes.
- [x] Document root Compose as infrastructure-only for local development.
- [x] Add Billing Service to quick start and script references.
- [x] Add this implementation task checklist.

## P3 Tests

- [x] Add tenant public plans API tests.
- [x] Add billing subscription helper tests.
- [x] Add API Gateway subscription middleware tests.
- [x] Add notification billing event tests.
- [x] Add frontend pricing API fallback/success tests.

## Verification

- [x] `bash -n` for updated scripts.
- [x] `go test ./...` for touched Go modules.
- [x] `npm run test:ci` targeted pricing tests.
- [x] `npm run build`.

---

# Implementation Tasks - Expired Subscription, Retention, and Terms

## P0 Subscription UX and Routing

- [x] Redirect expired tenants to `/subscription?reason=expired` after login.
- [x] Redirect protected API `402 Payment Required` responses to subscription recovery.
- [x] Replace expired modal behavior with focused subscription recovery page.
- [x] Keep grace-period tenants inside the normal app with warning banners.

## P1 Retention and Audit

- [x] Add retention timestamps to tenants.
- [x] Start the 30-day retention timer when tenants enter `grace_period`.
- [x] Clear the retention timer when payment reactivates the tenant before cleanup.
- [x] Add daily retention cleanup for eligible grace/expired tenants.
- [x] Preserve tenant, owner login, billing, payment, consent, audit, and financial order history.
- [x] Anonymize customer PII in preserved financial order rows.
- [x] Publish subscription-data anonymization audit events.

## P2 Terms Acceptance

- [x] Add public `/terms-of-service` page.
- [x] Add required signup Terms checkbox with trial, grace, retention, and historical-record notes.
- [x] Reject tenant registration without supported Terms acceptance.
- [x] Persist Terms acceptance in a dedicated table.

## P3 Tests and Docs

- [x] Add billing retention helper test.
- [x] Add frontend registration payload test for Terms fields.
- [x] Update implementation notes with current behavior and public interfaces.

---

# Implementation Tasks - Subscription Performance and UX

## P0 Shared Frontend Subscription State

- [x] Replace duplicate per-component subscription fetches with a shared cached provider.
- [x] Deduplicate concurrent subscription requests.
- [x] Refresh subscription status on focus/visibility when stale.
- [x] Invalidate frontend subscription cache after payment initiation and `402` responses.

## P1 Gateway Cache Freshness

- [x] Add billing-service Redis invalidator for API Gateway subscription status cache.
- [x] Clear `sub:{tenant_id}` after successful subscription payment.
- [x] Clear `sub:{tenant_id}` when billing jobs move tenants to `grace_period` or `expired`.
- [x] Reduce gateway cache TTL for `grace_period`, `expired`, and `cancelled` statuses.
