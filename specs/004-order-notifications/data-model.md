# Data Model: Order Snapshot & Replay

This file defines the minimal snapshot schema and replay message shape used by SSE clients and the snapshot endpoint (`GET /api/orders/snapshot`). The snapshot contract is intentionally small and designed for efficient client resync.

## Order Snapshot (response from `GET /api/orders/snapshot`)

- Content-Type: `application/json`
- Example payload:

```json
{
  "tenant_id": "ten_123",
  "snapshot_as_of": "2025-12-09T12:34:56Z",
  "orders": [
    {
      "order_id": "ord_001",
      "reference": "#1001",
      "status": "paid",
      "total_amount_cents": 12500,
      "currency": "IDR",
      "created_at": "2025-12-09T12:30:00Z",
      "updated_at": "2025-12-09T12:33:10Z",
      "metadata": { }
    }
  ],
  "pagination": {
    "limit": 100,
    "offset": 0,
    "more": false
  }
}
```

Notes:
- `snapshot_as_of` is the canonical timestamp for the snapshot. Clients should use this when reconciling events streamed after the snapshot.
- Orders list should be ordered by `updated_at` descending for efficient UI rendering.
- `pagination` is optional for large result sets; clients may request pages.

## SSE Replay Message Shape (lightweight event published to Redis Streams)

Each Redis Stream entry (per-tenant stream `tenant:<tenant_id>:stream`) should contain a JSON payload with the following fields:

```json
{
  "id": "event_uuid",
  "event": "order_paid|order_created|order_status_updated",
  "data": {
    "order_id": "ord_001",
    "tenant_id": "ten_123",
    "status": "paid",
    "reference": "#1001",
    "total_amount_cents": 12500,
    "timestamp": "2025-12-09T12:33:10Z"
  }
}
```

Clients consuming SSE should handle the above event names and use `GET /api/orders/snapshot` when they detect gaps (e.g., missing Last-Event-ID or reconnect after a long disconnect). The server MUST include the Redis Stream entry ID (or a generated `event_uuid`) as the SSE `id:` field to enable Last-Event-ID semantics.
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
