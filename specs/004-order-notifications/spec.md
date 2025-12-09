# Feature Specification: Order Notifications (Order Paid + Real-time Order List)

**Feature Branch**: `004-order-notifications`  
**Created**: 2025-12-09  
**Status**: Draft  
**Input**: User description: "as a tenant admin (Owner/Manager/Cashier) i want to receive notification via email and in-app when order has been paid, also i want real-time order list update when order submitted or have updated status"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Receive immediate notification when an order is paid (Priority: P1)

As a Tenant Admin (Owner / Manager / Cashier), when an order is paid, I want to receive an in-app notification and an email so I can acknowledge payment and prepare the order.

**Why this priority**: Payment confirmation is critical to business flow — staff need to act when payment completes.

**Independent Test**: Simulate a paid order event; verify (a) the tenant user receives an in-app notification, (b) an email is produced and queued, and (c) the order status in the order list shows the updated status.

**Acceptance Scenarios**:

1. **Given** an order in "pending payment" state, **When** payment is confirmed, **Then** in-app notification appears for tenant admins within acceptable time and an email is generated and handed to the delivery pipeline.
2. **Given** the tenant user has multiple devices/browsers open, **When** payment is confirmed, **Then** a notification appears in each active in-app session.

---

### User Story 2 - Real-time order list updates for submitted or status-changed orders (Priority: P1)

As a Tenant Admin, I want the order list (in-app) to update in real time when new orders are submitted or existing orders change status so my staff don't need to refresh the page.

**Why this priority**: Real-time updates reduce missed orders and manual refresh, improving throughput.

**Independent Test**: Submit a new order and update an order status; verify the order list updates without manual refresh for active sessions.

**Acceptance Scenarios**:

1. **Given** the order list view is open, **When** a new order is submitted, **Then** the new order appears in the list within the acceptable time window.
2. **Given** an order's status changes (e.g., Paid → Preparing → Completed), **When** the change occurs, **Then** the order row updates to reflect the new status and any status-specific badges/indicators update.

---

### User Story 3 - Notification preferences and role scoping (Priority: P2)

As a Tenant Admin, I want notifications scoped to tenant roles (Owner, Manager, Cashier) and to respect any per-user notification preferences so that notifications are relevant.

**Why this priority**: Avoids spamming irrelevant users and enables control over notification volume.

**Independent Test**: Toggle notification preference for a user and confirm they stop receiving in-app notifications and emails while others continue to receive them.

**Acceptance Scenarios**:

1. **Given** a user has disabled email notifications, **When** an order is paid, **Then** no email is sent to that user but in-app notifications can still be delivered (if enabled).

---

### Edge Cases

- Order payment events received twice (duplicate events): system must deduplicate notifications and idempotently update order status.
- Offline device or lost SSE connection: when a session reconnects, the client must receive the latest order list snapshot or missing events.
- Large bursts of events (high-order volume): system must avoid overwhelming in-app clients and should ensure ordering or sensible de-duping.
- User without email address configured: email step must be skipped for that user and logged for observability.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST deliver an in-app notification to all active sessions of tenant users (roles: Owner, Manager, Cashier) when an order transitions to a "paid" status.
- **FR-002**: System MUST enqueue an email notification to each tenant user who has email notifications enabled when an order transitions to "paid".
- **FR-003**: System MUST update the order list in active in-app sessions in real time when an order is submitted or its status changes.
- **FR-004**: System MUST provide per-user notification preference controls for in-app and email notifications.
- **FR-005**: System MUST deduplicate duplicate events so users do not receive duplicate notifications for the same order payment.
- **FR-006**: System MUST support re-sync/recovery for clients that reconnect after network loss to ensure no critical order updates are missed.
- **FR-007**: System MUST scope notifications to tenant context — users only receive notifications relevant to their tenant.

### Key Entities *(include if feature involves data)*

- **Order**: identifier, tenant_id, status, total_amount, payment_info, timestamps
- **TenantUser**: user_id, tenant_id, roles (Owner/Manager/Cashier), email, notification_preferences
- **Notification**: notification_id, user_id(s), type (in-app/email), payload reference, created_at, delivered_at
- **Event**: event_id, source (order-service), event_type (order_paid, order_created, order_status_updated), payload, dedupe_key

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 95% of paid-order events result in in-app notification visible in active tenant sessions within 5 seconds of the originating event under normal operating load.
- **SC-002**: 95% of paid-order events result in an email being produced and handed to the email/event pipeline within 30 seconds of the originating event (where the recipient has email enabled and a valid address).
- **SC-003**: Real-time order list reflects submitted orders or status changes in active sessions within 5 seconds for 95% of events under normal load.
- **SC-004**: Notification duplication rate is ≤ 1% for paid-order notifications (measured after dedupe logic applied).
- **SC-005**: Manual verification: Tenant admin can complete verification steps in the independent tests outlined in User Scenarios.

## Constraints (context provided by requester)

- The stakeholder has requested a real-time approach for in-app updates and a decoupled pipeline for email notifications. Detailed implementation notes provided in `implementation-notes.md` in this feature directory for architecture review.

Note: the main specification avoids implementation details; see the separate implementation notes file for stakeholder-requested technologies and considerations.

## Assumptions

- Tenant roles that should receive notifications are Owner, Manager, and Cashier.
- All tenant users have a `tenant_id` and role metadata available to the notification routing layer.
- Email addresses, where present, are validated and reachable by the email pipeline; if absent, the user will not receive emails.
- The order-service (or a payment processor integration) will emit reliable events indicating payment success.

## Acceptance Criteria Summary

- When an order is marked as paid, tenant admin users see an in-app notification and receive an email (if enabled).
- Active order-list views update in real time for new orders and status changes.
- Notification preferences and tenant scoping are respected.
- Dedupe and reconnection/resync scenarios handled to avoid persistent message loss or duplication.

---

**Next Steps / Implementation Handoff**

- Review this spec with product and backend architects to confirm SSE and Kafka constraints and to validate performance targets and operational considerations (e.g., topic partitioning, consumer lag monitoring, SSE connection scaling).
- Create implementation tasks for the order-service event emission, notification-service subscription, and frontend SSE client integration.