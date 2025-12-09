# Data Model: Order Notifications

This document captures entities and fields required to implement the feature. Focus is on domain fields, validation, and relationships — not storage implementation details.

## Entities

- **Order**
  - id: string (UUID) — primary identifier
  - tenant_id: string (UUID) — identifies tenant
  - reference: string — human-friendly order reference
  - status: enum {pending, paid, preparing, completed, cancelled}
  - total_amount: integer (cents)
  - payment_info: object (payment_method, provider_reference, paid_at)
  - created_at: timestamp
  - updated_at: timestamp

- **TenantUser**
  - id: string (UUID)
  - tenant_id: string (UUID)
  - email: string (nullable)
  - roles: array of enum {owner, manager, cashier, staff}
  - notification_preferences: object {
      email_enabled: boolean,
      in_app_enabled: boolean
    }

- **Notification**
  - id: string (UUID)
  - tenant_id: string (UUID)
  - user_id: string|null (UUID)
  - type: enum {email, in_app}
  - subject: string|null
  - body: string|null
  - payload: json (original event payload reference)
  - status: enum {pending, sent, failed}
  - error_msg: string|null
  - retry_count: integer
  - created_at: timestamp
  - sent_at: timestamp|null
  - failed_at: timestamp|null

- **EventRecord (for dedupe & idempotency)**
  - id: string (UUID) — event_id from Kafka
  - order_id: string (UUID)
  - tenant_id: string (UUID)
  - event_type: string
  - processed_at: timestamp
  - metadata: json

## Validation Rules

- `email` must conform to a standard email regex when present.
- `status` transitions should follow allowed paths; e.g., `pending` -> `paid` -> `preparing` -> `completed` or `cancelled`.
- `notification_preferences.email_enabled` implies `email` must exist; otherwise skip email delivery.

## State Transitions (Order)

- `pending` -> `paid` (on payment confirmation)
- `paid` -> `preparing` (on staff acceptance)
- `preparing` -> `completed` (on order completion)
- `*` -> `cancelled` (on cancellation)

## Notes

- The `Notification` store supports observability and retries for email delivery. The notification-service currently owns templates and retry logic.
- `EventRecord` is a bounded dedupe store (could be Postgres table with TTL / cleanup job or Redis with expiry) used to ensure idempotent processing of Kafka events.
