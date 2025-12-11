# SSE Contract: /api/v1/sse

Endpoint: `GET /api/v1/sse`

Auth: OAuth/JWT session cookie required (tenant-scoped). Clients must present valid credentials; SSE connection will be scoped to the tenant of the authenticated user.

Connection Query Params:
- `channels` (optional): comma-separated channel names, e.g., `orders,notifications`

Headers:
- `Accept: text/event-stream`
- `Last-Event-ID` (optional): event id for replay

Server-Sent Events (SSE) messages format:

- `id`: event_id (UUID)
- `event`: event_type (e.g., `order_paid`, `order_created`, `order_status_updated`, `in_app_notification`)
- `data`: JSON string of event payload. Example payload for `order_paid`:

```json
{
  "order_id": "<uuid>",
  "tenant_id": "<uuid>",
  "status": "paid",
  "total_amount": 123000,
  "reference": "INV-00123",
  "timestamp": "2025-12-09T12:34:56Z"
}
```

Behavior:
- On connect, server sends a `connected` event with `id` set to a generated connection id.
- Events delivered in near-real-time as they are published by the notification-service.
- If `Last-Event-ID` is provided and message is available in replay buffer, server replays missing events up to current.
- If replay buffer does not contain requested `Last-Event-ID`, client should call `GET /api/orders/snapshot?since=<event_id|timestamp>` to resync.

Recommended HTTP response headers:
- `Cache-Control: no-cache`
- `Connection: keep-alive`
- `Content-Type: text/event-stream`
