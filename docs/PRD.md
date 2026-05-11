# Product Requirements Document: Multi-Tenant Point of Sale System

Last updated: 2026-05-11  
Document status: Draft, generated from repository scan  
Primary sources: `README.md`, `docker-compose.yml`, `api-gateway/main.go`, service route files, `frontend/app`, `docs/API.md`, `docs/db_structure.md`, and `specs/001` through `specs/008`.

## 1. Executive Summary

The product is a multi-tenant point of sale platform for small to medium merchants that need a single system to manage staff, products, inventory, online customer ordering, offline order recording, notifications, compliance, and operational insight.

The current implementation is organized as Go + Echo microservices behind an API gateway, a Next.js frontend, PostgreSQL, Redis, Kafka in KRaft mode, Vault Transit encryption, MinIO object storage, and a Grafana observability stack. The product supports tenant-isolated authentication, role-based team management, product and inventory operations, QRIS guest ordering through Midtrans, staff-recorded offline orders with installments, email notifications, audit/compliance flows, and analytics dashboards.

## 2. Product Goals

1. Allow multiple independent businesses to use the same POS platform without cross-tenant data exposure.
2. Let merchants manage their product catalog, stock, categories, and product photos in one place.
3. Let customers browse a merchant-specific public menu and place QRIS-paid orders without creating an account.
4. Let staff record orders that happen outside the system, including walk-in, phone, WhatsApp, and installment orders.
5. Give owners and managers timely visibility into orders, sales performance, inventory health, and urgent operational tasks.
6. Provide UU PDP/GDPR-oriented privacy controls, encryption, consent, audit trails, retention, and data rights.
7. Provide reliable email notifications and receipts so paid orders are not missed.

## 3. Non-Goals

1. Native mobile applications.
2. In-system courier dispatch or logistics marketplace integration.
3. Full accounting, tax filing, payroll, or procurement workflows.
4. Multi-location enterprise inventory transfers.
5. Loyalty, coupons, promotions, refunds, or returns beyond the documented future scope.
6. SSO/OAuth login. Current auth is email/password plus tenant context.

## 4. Users and Personas

| Persona | Needs | Primary Interfaces |
| --- | --- | --- |
| Platform owner | Operate the platform, verify compliance, monitor services, manage system risk | Observability, audit APIs, deployment docs |
| Tenant owner | Register business, configure tenant, manage team, products, orders, billing, compliance | Dashboard, settings, products, users, billing |
| Manager | Manage products, stock, orders, notification preferences, analytics | Dashboard, products, orders, settings |
| Cashier or staff | Handle daily orders and offline order entry | Orders, offline order form, order details |
| Guest customer | Browse public menu, checkout, pay by QRIS, track order, manage guest data rights | Public menu, checkout, order lookup, guest data pages |

## 5. Current System Overview

### 5.1 Service Ownership

| Area | Service or module | Evidence from scan |
| --- | --- | --- |
| API gateway | `api-gateway` | Public/protected route proxying, JWT, tenant scope, RBAC, subscription middleware |
| Authentication | `backend/auth-service` | Login, session, refresh, logout, account verification, password reset |
| Tenant management | `backend/tenant-service` | Registration, tenant config, Midtrans config, plans, tenant data export |
| Team management | `backend/user-service` | Invitations, invitation accept/resend/list, notification preferences, user deletion |
| Product and inventory | `backend/product-service` | Product/category CRUD, stock adjustments, public menu, photo APIs |
| Order management | `backend/order-service` | Public cart/checkout/orders, admin orders, offline orders, payments, guest data rights |
| Notifications | `backend/notification-service` | Notification config, history, resend, test emails, templates |
| Audit and consent | `backend/audit-service` | Audit events, consent purposes/status/history/revoke, privacy policy, compliance reports |
| Analytics | `backend/analytics-service` | Overview, sales trend, top products, top customers, operational tasks |
| Billing | `backend/billing-service` | Subscription, invoices, payment initiation, billing webhook |
| Frontend | `frontend` | Next.js app routes for dashboard, auth, products, orders, settings, billing, guest ordering |

### 5.2 Core Frontend Surfaces

Key app routes found in `frontend/app`:

- Public: `/`, `/signup`, `/login`, `/forgot-password`, `/reset-password`, `/verify-email`, `/privacy-policy`
- Guest ordering: `/menu/[tenantSlug]`, `/checkout/[tenantSlug]`, `/payment/return`, `/orders/[orderReference]`, `/guest/order-lookup`
- Admin workspace: `/dashboard`, `/products`, `/products/new`, `/products/[id]`, `/products/categories`, `/orders`, `/orders/offline-orders`
- Settings: `/settings`, `/settings/orders`, `/settings/payment`, `/settings/notifications`, `/settings/privacy`, `/settings/tenant`, `/settings/tenant-data`, `/settings/audit-log`
- Team and billing: `/users/invite`, `/subscription`, `/subscription/invoices`

## 6. Product Scope

### 6.1 Authentication and Multi-Tenancy

The system must allow business owners to register a tenant, create an owner user, verify account/email flows, log in, maintain sessions, refresh sessions, and log out. All authenticated access must be scoped to the user's tenant.

Core requirements:

- Tenant registration creates a tenant, owner user, tenant slug, and default configuration.
- Users authenticate with email/password in tenant context.
- Sessions expire and can be refreshed or terminated.
- Password reset and account verification are supported.
- Same email may exist across different tenants.
- API gateway injects tenant and user context into downstream services.
- All tenant-owned records are isolated through tenant IDs and repository-level filters, with RLS where applicable.

### 6.2 Team Management and RBAC

Tenant owners and managers need to invite staff, list invitations, resend invitations, and allow invite acceptance. Roles include owner, manager, cashier, and staff-like operational users depending on service context.

Core requirements:

- Owners and managers can invite users into their tenant.
- Invitees can accept invitations without an existing session.
- Invitations expire and can be resent.
- RBAC must protect admin, product, analytics, notification, tenant data, and delete operations.
- Owner-only areas include tenant configuration, tenant data export, compliance/audit access, billing, and destructive user operations.
- Owner/manager areas include product management, order settings, analytics, and notification configuration.
- Order operations may allow cashier access where operationally needed.

### 6.3 Product, Inventory, Categories, and Photos

Merchants need to create and maintain product data for online and offline ordering.

Core requirements:

- Product CRUD: create, list, view, update, archive, restore, and delete where safe.
- Product fields include SKU/barcode, name, description, category, selling price, cost price, tax rate, stock quantity, and photo metadata.
- Category CRUD supports product grouping and display ordering.
- Inventory dashboard exposes stock summaries and stock adjustment history.
- Stock adjustments require reason, notes, previous quantity, new quantity, delta, user ID, and timestamp.
- Public menu exposes available products for guest ordering.
- Product photos are stored in MinIO/S3-compatible object storage.
- Photo management supports upload, list, get, replace, metadata update, delete, reorder, primary photo selection, quota tracking, and placeholder fallbacks.
- Photo object keys must be tenant-prefixed to prevent cross-tenant access.

### 6.4 Online Guest Ordering

Guest customers can order without logging in through a tenant-specific menu.

Core requirements:

- Guests access a public tenant menu URL and see only that tenant's products and configuration.
- Guests can add, update, remove, and clear cart items.
- Cart state is tenant-isolated and session-based, with Redis-backed persistence and client-side recovery.
- Checkout collects name, phone, optional email, notes, and delivery-type-specific fields.
- Supported delivery types are pickup, delivery, and dine-in, controlled by tenant configuration.
- Delivery checkout can geocode addresses, validate service area, and calculate delivery fees if enabled.
- Checkout creates a pending order, order items, payment transaction, and inventory reservations.
- Inventory reservations prevent overselling and expire if payment is not completed.
- Midtrans QRIS payment is initiated server-side.
- Payment webhook verifies signature, validates amount, handles idempotency, and updates payment/order status.
- Paid online orders become visible in admin order management.
- Guests can view public order status by order reference.

### 6.5 Admin Order Management

Tenant staff need to manage online order fulfillment after payment.

Core requirements:

- Staff can list admin orders with filters.
- Staff can view full order details including items, delivery type, customer contact, address, payment status, and timestamps.
- Staff can update order status through the supported lifecycle.
- Staff can add internal notes such as courier or fulfillment notes.
- Order settings allow owners/managers to configure delivery behavior and order defaults.

### 6.6 Offline Order Management

Staff can record orders taken outside the public ordering flow while preserving auditability and analytics compatibility.

Core requirements:

- Authenticated staff can create offline orders from a form similar to guest checkout.
- Offline orders are marked with `order_type='offline'` and must not disturb online order flow.
- Offline order items snapshot product name, SKU, quantity, unit price, and total.
- Payment options include full payment and installment/down-payment flows.
- Installments track payment terms, payment schedule, payment records, amount paid, and outstanding balance.
- Orders with outstanding balance remain pending payment and become completed only when fully paid.
- Staff can view, edit, and record payments for offline orders.
- Owner and manager roles can soft-delete offline orders with a reason.
- Create, read, update, delete, access-denied, and payment operations must be audit-trailed.
- Offline order PII must follow the same encryption, masking, consent, and data-rights rules as online order PII.
- Offline order data must feed analytics and dashboard metrics.

### 6.7 Notifications

The system sends email notifications for account, order, billing, and compliance events.

Core requirements:

- Paid orders trigger staff notification emails.
- Guest receipt emails are sent when a guest email is provided.
- Email templates include registration, password reset, login alert, team invitation, order staff notification, order invoice/receipt, payment received, billing invoice, trial/grace/expiry, and user/guest data deletion messages.
- Notifications have status tracking, retries, failure reason, and history.
- Owners/managers can configure tenant notification settings and staff notification preferences.
- Test notification and resend endpoints are available.
- Email failures must not block order processing.

### 6.8 Audit, Consent, Privacy, and Data Rights

The platform must support UU PDP/GDPR-oriented privacy controls.

Core requirements:

- PII is encrypted at rest using Vault Transit-backed application encryption patterns.
- Sensitive log values are masked before writing logs.
- Consent purposes are defined and exposed publicly.
- Required consent is collected during registration and guest checkout.
- Optional consent can be revoked where business rules allow.
- Privacy policy is publicly accessible.
- Tenant owners can view/export tenant data and manage user deletion.
- Guest customers can verify identity using order reference plus email or phone and request personal data deletion/anonymization.
- Audit events are append-oriented and include tenant ID, actor, action, resource, timestamp, IP/session context where available, and metadata.
- Audit logs and consent records support filtering for compliance investigation.
- Retention and cleanup jobs remove expired sessions, tokens, invitations, guest order PII, and deleted users according to policies.

### 6.9 Analytics Dashboard

Owners and managers need dashboard insights for daily decision making.

Core requirements:

- Overview metrics include sales, order count, average order value, net profit, and inventory value.
- Product rankings show top and bottom performers by quantity and revenue.
- Customer rankings identify top spenders using privacy-preserving customer identifiers.
- Sales trend supports daily, weekly, monthly, quarterly, and yearly granularity.
- Operational tasks show delayed orders and low/out-of-stock products.
- Offline order metrics distinguish online vs offline revenue, order count, installment revenue, and pending installment exposure.
- Analytics must be tenant-scoped and cacheable with Redis.

### 6.10 Billing and Subscription

The repository includes billing and subscription functionality even though it is adjacent to the POS feature list.

Core requirements:

- Owners can view subscription status and invoices.
- Owners can update billing cycle and upgrade subscription.
- Owners can initiate invoice payment.
- Billing webhooks update invoice/subscription payment status.
- API gateway subscription enforcement must allow expired tenants to reach billing recovery endpoints.

## 7. Key Data Entities

| Entity | Purpose |
| --- | --- |
| Tenant | Business account, slug, status, storage quota, configuration owner |
| User | Tenant staff account, role, status, encrypted PII, notification preferences |
| Session | Authenticated session/token tracking |
| Invitation | Pending team onboarding |
| Product | Sellable item with pricing, tax, inventory, category, archive state |
| Category | Product grouping |
| ProductPhoto | Object storage metadata and primary/display order state |
| StockAdjustment | Manual inventory changes and audit context |
| GuestOrder | Online and offline order header, customer PII, status, totals, order type |
| OrderItem | Snapshot line items for order history |
| InventoryReservation | Temporary stock hold during online checkout/payment |
| PaymentTransaction | Midtrans QRIS transaction data and webhook audit |
| PaymentTerms | Offline installment/down-payment schedule |
| PaymentRecord | Offline payment receipt history |
| DeliveryAddress | Geocoded delivery information and serviceability result |
| Notification | Email event record with delivery/retry status |
| ConsentRecord | Consent grant/revoke state and evidence |
| AuditEvent | Compliance and operational audit trail |
| Subscription/Invoice | Billing plan, billing cycle, invoice, payment status |

## 8. Functional Requirements

### Authentication and Tenant Isolation

- FR-AUTH-001: The system shall register a new tenant and owner account from a public endpoint.
- FR-AUTH-002: The system shall authenticate users and issue session/JWT credentials.
- FR-AUTH-003: The system shall expose session, refresh, logout, password reset, and account verification flows.
- FR-AUTH-004: The system shall reject protected requests without valid authentication.
- FR-AUTH-005: The system shall enforce tenant scope on all protected tenant data.
- FR-AUTH-006: The system shall rate-limit sensitive public auth and registration endpoints.

### Team and Roles

- FR-TEAM-001: The system shall allow owners/managers to invite team members.
- FR-TEAM-002: The system shall allow invite acceptance using invitation token.
- FR-TEAM-003: The system shall allow listing and resending invitations.
- FR-TEAM-004: The system shall enforce owner, manager, cashier, and staff permissions by route and action.
- FR-TEAM-005: The system shall allow owner-managed user deletion workflows with audit trail.

### Products and Inventory

- FR-PROD-001: The system shall support product CRUD, archive, restore, and safe delete.
- FR-PROD-002: The system shall enforce tenant-specific unique SKUs where required.
- FR-PROD-003: The system shall support category CRUD and product/category assignment.
- FR-PROD-004: The system shall track stock quantity per product.
- FR-PROD-005: The system shall support stock adjustment with reason and audit context.
- FR-PROD-006: The system shall expose inventory summary and adjustment history.
- FR-PROD-007: The system shall expose tenant public catalog data for guest menus.
- FR-PROD-008: The system shall store product photos in S3-compatible storage with tenant isolation.
- FR-PROD-009: The system shall support up to five photos per product, display ordering, and primary photo.

### Online Orders

- FR-ONL-001: The system shall expose public tenant menu, cart, checkout, and order-status flows without requiring customer login.
- FR-ONL-002: The system shall persist guest cart state by tenant and session.
- FR-ONL-003: The system shall validate cart quantities against available inventory.
- FR-ONL-004: The system shall create pending orders with item price snapshots.
- FR-ONL-005: The system shall create temporary inventory reservations during checkout.
- FR-ONL-006: The system shall support pickup, delivery, and dine-in checkout based on tenant settings.
- FR-ONL-007: The system shall integrate with Midtrans QRIS for payment.
- FR-ONL-008: The system shall verify payment webhooks and process them idempotently.
- FR-ONL-009: The system shall convert successful payment into paid order state and permanent inventory reduction.
- FR-ONL-010: The system shall let staff manage paid orders through admin order APIs and UI.

### Offline Orders

- FR-OFF-001: The system shall let authenticated staff create offline orders.
- FR-OFF-002: The system shall distinguish online and offline orders in database, APIs, and UI.
- FR-OFF-003: The system shall support full payment and installment payment terms.
- FR-OFF-004: The system shall track payment records and outstanding balance.
- FR-OFF-005: The system shall only mark installment orders completed when total paid equals total due.
- FR-OFF-006: The system shall allow offline order editing with audit trail.
- FR-OFF-007: The system shall restrict offline order deletion to owner/manager roles.
- FR-OFF-008: The system shall include offline orders in analytics.
- FR-OFF-009: The system shall preserve compliance controls for offline order PII.

### Notifications

- FR-NOTIF-001: The system shall send staff email notifications for paid orders.
- FR-NOTIF-002: The system shall send customer receipt emails when customer email is available.
- FR-NOTIF-003: The system shall store notification history and delivery status.
- FR-NOTIF-004: The system shall retry failed email delivery.
- FR-NOTIF-005: The system shall allow owners/managers to configure notification settings.
- FR-NOTIF-006: The system shall allow test notifications and manual resend.

### Compliance and Audit

- FR-COMP-001: The system shall encrypt PII and sensitive credentials at rest.
- FR-COMP-002: The system shall mask PII in logs.
- FR-COMP-003: The system shall collect and store consent with purpose, actor, timestamp, IP, user agent, and version where available.
- FR-COMP-004: The system shall provide consent status, history, grant, and revoke flows.
- FR-COMP-005: The system shall expose privacy policy content publicly.
- FR-COMP-006: The system shall support tenant data access/export.
- FR-COMP-007: The system shall support guest data access and anonymization/deletion.
- FR-COMP-008: The system shall write audit events for PII access and sensitive operations.
- FR-COMP-009: The system shall retain audit records for compliance review.
- FR-COMP-010: The system shall run cleanup jobs for expired or retained data.

### Analytics

- FR-ANL-001: The system shall show sales overview metrics for selected time ranges.
- FR-ANL-002: The system shall show product rankings and customer rankings.
- FR-ANL-003: The system shall show time-series sales trends.
- FR-ANL-004: The system shall show operational tasks for delayed orders and low stock.
- FR-ANL-005: The system shall include offline order and installment metrics.
- FR-ANL-006: The system shall cache analytics results while preserving tenant isolation.

### Billing

- FR-BILL-001: The system shall expose subscription status to owners.
- FR-BILL-002: The system shall list and retrieve invoices.
- FR-BILL-003: The system shall initiate invoice payments.
- FR-BILL-004: The system shall process billing webhooks with payment-provider verification.
- FR-BILL-005: The system shall enforce subscription access while allowing billing remediation.

## 9. Non-Functional Requirements

### Security

- Passwords must be hashed using strong one-way hashing.
- JWT/session secrets must be configurable and production-grade.
- Public webhooks must verify provider signatures.
- Admin endpoints must use RBAC and tenant scope.
- Cross-tenant access must be prevented in API gateway, service repositories, cache keys, object storage keys, and database policies.
- Rate limiting must apply to public auth, registration, cart/checkout, test notification, and high-risk admin operations.

### Privacy and Compliance

- PII must be encrypted at rest through Vault-backed application encryption.
- Searchable encrypted fields must use deterministic search hashes where lookup is required.
- Logs must never expose raw passwords, tokens, payment keys, phone numbers, emails, addresses, or customer names.
- Consent records and audit records must be queryable for compliance.
- Deletion/anonymization must preserve lawful merchant transaction records where required.

### Performance

- Public menu should load in under 2 seconds for typical tenant catalogs.
- Dashboard overview should load in under 2 seconds for normal data volumes.
- Analytics query p95 target is under 200 ms after cache warmup.
- Photo retrieval should return displayable URLs fast enough for catalog browsing.
- Offline order entry for a typical 5-item order should complete in under 3 minutes.

### Reliability

- Email delivery failures must not block order state transitions.
- Payment webhook processing must be idempotent.
- Inventory reservation cleanup must release abandoned stock holds.
- Kafka event publishing should tolerate retryable failures or use outbox/retry patterns.
- MinIO/S3 unavailability should show placeholders or user-friendly errors without corrupting product data.

### Observability

- Services expose health endpoints and Prometheus metrics.
- Logs should be structured and trace-aware.
- OpenTelemetry tracing should connect API gateway and downstream service spans.
- Grafana dashboards should cover service health, offline orders, audit trails, cleanup jobs, and infrastructure.

### Accessibility and Internationalization

- Frontend supports English and Indonesian locales.
- Core forms, errors, consent copy, privacy copy, and user-facing flows should be translated.
- Admin and guest flows should be responsive and usable on mobile devices.

## 10. Key User Journeys

### 10.1 Tenant Onboarding

1. Owner opens signup.
2. Owner accepts required consent.
3. Owner enters business and account details.
4. System creates tenant, owner user, default config, and verification/notification records.
5. Owner verifies account and logs in.
6. Owner lands on dashboard and can invite team members.

Acceptance:

- Registration completes in under 3 minutes.
- Owner can log in and only access their tenant data.
- Required consent is stored and auditable.

### 10.2 Product Setup

1. Owner or manager creates categories.
2. Owner or manager creates products with SKU, pricing, tax, cost, stock, and photo.
3. Product photos are uploaded to object storage.
4. Staff can adjust stock with reason.
5. Products appear in internal catalog and public menu if active and available.

Acceptance:

- Product can be created, edited, archived, restored, and displayed.
- Stock changes are recorded with user and reason.
- Tenant A cannot access Tenant B photos or products.

### 10.3 Online QRIS Order

1. Guest opens `/menu/[tenantSlug]`.
2. Guest adds products to cart.
3. Guest checks out with pickup, delivery, or dine-in.
4. System validates inventory, creates reservation, creates order, and starts Midtrans QRIS payment.
5. Midtrans webhook marks payment as paid.
6. Staff receive notification and manage fulfillment.
7. Guest tracks order by reference.

Acceptance:

- Guest never needs an account.
- Payment success updates order exactly once.
- Inventory is not oversold.
- Staff can see and complete paid orders.

### 10.4 Offline Order with Installments

1. Staff opens offline order form.
2. Staff enters customer data and items.
3. Staff chooses full payment or installment terms.
4. System records order and payment terms.
5. Staff records payments over time.
6. Order becomes completed after full settlement.
7. Owner/manager can delete only with reason; all changes are audited.

Acceptance:

- All roles can create/edit, but only owner/manager can delete.
- Outstanding balance is accurate.
- Offline orders appear in analytics.

### 10.5 Compliance Data Rights

1. Tenant owner opens data/privacy settings.
2. Owner views/export tenant data and manages optional consent.
3. Guest verifies order ownership with reference plus email/phone.
4. Guest views personal data and requests deletion.
5. System anonymizes PII while preserving transaction record.
6. Audit event records the request and outcome.

Acceptance:

- Required verification prevents unauthorized guest data access.
- PII is anonymized as documented.
- Audit trail is immutable from product interfaces.

## 11. Metrics and Success Criteria

| Area | Success metric |
| --- | --- |
| Tenant isolation | 0 known cross-tenant data leaks |
| Registration | Owner registration under 3 minutes |
| Login | Successful login under 10 seconds |
| Guest menu | 95 percent of public menu loads under 2 seconds |
| Checkout | Payment initiation succeeds for 99 percent of valid checkout attempts |
| Payment webhook | Duplicate payment processing under 0.1 percent |
| Notifications | Staff order emails delivered within 1 minute for 99 percent of successful paid orders |
| Inventory | No oversell in concurrent last-stock checkout tests |
| Offline orders | Typical 5-item offline order recorded under 3 minutes |
| Analytics | Dashboard overview loads under 2 seconds for expected tenant data volumes |
| Compliance | 100 percent of designated PII fields encrypted and masked in logs |
| Audit | 100 percent audit coverage for configured sensitive operations |

## 12. Dependencies and Integrations

| Dependency | Purpose |
| --- | --- |
| PostgreSQL | Primary relational storage, migrations, tenant data |
| Redis | Sessions, cart persistence, caching, rate limiting, distributed locks |
| Kafka KRaft | Event streaming for audit, notifications, analytics, consent, billing |
| Vault Transit | Encryption key management and field encryption support |
| MinIO/S3 | Product photo object storage |
| Midtrans | QRIS and billing payment processing |
| SMTP/Mailhog | Email delivery and local email testing |
| Grafana, Prometheus, Loki, Tempo, Promtail, OTel Collector | Metrics, logs, tracing, dashboards |
| Google Maps API | Optional delivery geocoding and service-area validation |

## 13. Release Readiness Notes

The repository contains implementation-complete documentation for QRIS guest ordering and offline orders, completed task lists for product photo work, live route registrations for major backend services, and frontend pages/components for the described product surface.

Known caveats from scan:

- Several integration tests are documented but still contain `TODO` or `t.Skip` markers, especially in offline order and analytics integration suites.
- `docker-compose.yml` includes infrastructure services active by default, while many application services are currently commented out for local development.
- Some implementation status documents are stale relative to task files and code. Example: product photo status says frontend is pending, but frontend photo components and API clients now exist.
- Analytics has implemented service routes and frontend dashboard usage, but some offline analytics integration tests include placeholder helpers.
- Public guest data rights are implemented in order-service routes, while API gateway comments still show older disabled proxy paths. The current public proxy group may still route `/api/v1/public/:tenantId/*`, but guest data routes without tenant ID should be verified through the gateway.
- Production hardening should verify real Vault, Kafka topics, webhook secrets, SMTP provider, MinIO bucket policy, and service health checks end to end.

## 14. Open Questions

1. Should tenant public URLs use tenant slug everywhere, tenant UUID everywhere, or support both consistently?
2. Should cashier role have access to all admin order management, or only fulfillment actions?
3. What is the canonical order status model across online and offline orders?
4. Should offline orders decrement inventory immediately on creation, on full payment, or per business setting?
5. Should refunds/returns become first-class product scope before production launch?
6. What is the expected subscription plan model beyond the current billing service endpoints?
7. What compliance retention periods are final for each data category in production?
8. Should order email receipts be required for all guests, or remain optional when email is provided?
9. What delivery fee rules are required for production tenants: flat, distance, zone, polygon, external courier, or all?

## 15. Suggested Next Product Priorities

1. Verify end-to-end local environment with all active services, migrations, Vault, Kafka, MinIO, and frontend.
2. Convert skipped/TODO integration tests into executable tests for checkout, payment webhook, offline order lifecycle, analytics, and compliance flows.
3. Normalize gateway routes for guest order lookup and guest data rights.
4. Confirm canonical tenant slug vs tenant ID behavior in frontend, gateway, product service, and order service.
5. Add production readiness checks for secrets, webhook verification, Kafka topics, MinIO policies, SMTP, and observability dashboards.
6. Define refund/return behavior and inventory reversal rules.
7. Tighten API documentation into a single OpenAPI bundle for frontend and QA.

