# Phase 0 — Research: Order Notifications (SSE + Kafka)

This document resolves outstanding technical clarifications from the implementation plan and recommends concrete decisions for Phase 1 design.

## Decision 1 — SSE Endpoint Ownership

- Decision: Host SSE endpoints in `notification-service` initially.
- Rationale: `notification-service` already consumes Kafka events and owns notification semantics, templates, and retry logic. Co-locating SSE reduces cross-service coupling, avoids proxying event payloads between services, and keeps tenant-scoped routing in one place.
- Alternatives considered:
  - `api-gateway` proxying SSE: centralizes auth and routing but introduces an extra hop and requires reliable proxying of streaming connections. Recommended if organization policy mandates centralized auth/rate limiting.
  - Dedicated SSE service: isolates SSE scaling and lifecycle but increases operational complexity and duplicate state (needs subscription/replication of event streams).
- Mitigation / Follow-up: If SSE connection scale or topology constraints arise, we will introduce a lightweight SSE proxy layer (could be part of `api-gateway`) and use Redis as a fan-out layer so `notification-service` publishes to Redis per-tenant channels; SSE instances (gateway or notification-service replicas) subscribe to Redis channels.

## Decision 2 — Fan-out and Scaling Strategy

- Decision: Use Redis Pub/Sub (or Redis Streams if persistence needed) as the low-latency fan-out mechanism between Kafka consumer logic and SSE connection handlers.
- Rationale: Kafka is optimized for durable storage and consumer groups, but using Kafka directly to fan-out to many concurrent SSE clients is inefficient (lots of consumers or custom logic). Redis Pub/Sub provides low-latency, ephemeral fan-out; Redis Streams can provide durable replay if required.
- Alternatives considered:
  - Use Kafka with a consumer per SSE instance and partitioning by tenant: works for moderate scale but increases consumer management complexity and may add latency.
  - Use an in-process broadcast to all connected clients (simple, but fails for multi-replica scaling).
- Implementation note: notification-service will remain the canonical Kafka consumer and publish lightweight notification messages to Redis tenant channels. SSE handlers (in the same service or in the gateway) will subscribe to Redis channels.

## Decision 3 — Kafka Topic & Event Shape

- Decision: Consume order-related events from a topic such as `orders.events` where events include `event_type` (order_created, order_paid, order_status_updated), `order_id`, `tenant_id`, `event_id` (UUID), `timestamp`, and payload.
- Rationale: A single topic with typed events simplifies topic management and allows the notification-service to filter by event_type.
- Dedupe Key: Use `event_id`/UUID and `order_id` for deduplication. Persist processed `event_id` entries (TTL or bounded retention) in a dedupe store (Postgres table or Redis) to ensure idempotent processing.

## Decision 4 — Client Reconnect & Resync Strategy

- Decision: Support SSE `Last-Event-ID` replay semantics and provide a lightweight `GET /api/orders/snapshot?since=<timestamp|event_id>` endpoint to obtain missed order states after reconnection.
- Rationale: SSE Last-Event-ID covers ordered replay for short-term reconnection; snapshots provide recovery for long disconnects and help bring UI into consistent state.
- Implementation:
  - When publishing SSE messages, include `id` (the event_id) and `event` (event_type). Keep a short-lived ring buffer per tenant in Redis (e.g., last N events or last T minutes) to support Last-Event-ID replays.
  - If Last-Event-ID is older than buffer retention, client should call snapshot endpoint which returns current order list or incremental changes since the provided marker.

## Decision 5 — Monitoring & Observability

- Metrics to emit:
  - Kafka consumer lag and errors (per topic/partition)
  - Notification events processed / failed (per event_type, per-tenant aggregated)
  - Email send attempts / failures / retries
  - SSE connection count (per instance), SSE event latency (p50/p95)
  - Redis pub/sub channel depth / stream length (if using Streams)
- Tracing: Add spans around Kafka consumption → processing → publish-to-Redis → SSE delivery / email send for end-to-end latency measurement.

## Decision 6 — Performance Targets

- Decision: Align targets with spec success criteria. For capacity planning, propose an initial soft target of 5k concurrent SSE connections per cluster and a peak write throughput of 200 orders/sec for typical tenants. BOTH values are proposals and flagged NEEDS CLARIFICATION for production capacity planning with stakeholders/ops.

## Action Items for Phase 1

1. Finalize capacity numbers with DevOps/Operations (confirm expected concurrent SSE connections and peak order rates).  
2. Design Kafka topic schema and event contract (OpenAPI/AsyncAPI or JSON schema).  
3. Add Redis subscription channels and retention policy design (Pub/Sub vs Streams).  
4. Implement SSE contract (event names, payload shape, Last-Event-ID usage).  
5. Create monitoring dashboards and alerting playbooks for consumer lag, email failure spikes, and SSE connection anomalies.

## Resolved Unknowns Summary

- SSE hosting: notification-service (initial) — OK to change if scaling/ops require.  
- Fan-out mechanism: Redis Pub/Sub or Redis Streams (recommended) rather than direct Kafka-to-client fan-out.  
- Topic/event contract: `orders.events` with `event_id`, `order_id`, `tenant_id`, `event_type`.  
- Deduplication: Use `event_id` persisted in dedupe store (Postgres or Redis).  
- Reconnect strategy: SSE Last-Event-ID + snapshot endpoint; Redis ring buffer for short replay.  
- Performance: initial proposal 5k SSEs per cluster and 200 orders/sec — NEEDS CLARIFICATION before capacity provisioning.
