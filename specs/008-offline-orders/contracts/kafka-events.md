# Kafka Event Contracts: Offline Orders

**Feature**: Offline Order Management  
**Date**: February 7, 2026  
**Status**: Complete

## Overview

Offline order operations publish events to Kafka topics for audit trail and analytics integration. This document defines event schemas and publishing patterns.

## Topics

### 1. `audit-events` (Existing Topic)

**Purpose**: Capture all offline order operations for compliance and audit trail.  
**Consumers**: `audit-service`  
**Retention**: 90 days (configurable)  
**Partitioning**: By `tenant_id` (ensures ordered events per tenant)

---

### 2. `order-events` (Existing Topic)

**Purpose**: Notify analytics service of order lifecycle events.  
**Consumers**: `analytics-service`  
**Retention**: 7 days  
**Partitioning**: By `tenant_id`

---

## Event Schemas

### Audit Event: Offline Order Created

**Topic**: `audit-events`  
**Event Key**: `{tenant_id}:{order_id}`  
**Event Type**: `offline_order.created`

```json
{
  "event_type": "offline_order.created",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2026-02-07T10:30:00.123Z",
  "tenant_id": "tenant-uuid",
  "user_id": "user-uuid",
  "user_name": "John Staff",
  "user_role": "staff",
  "resource_type": "offline_order",
  "resource_id": "order-uuid",
  "action": "CREATE",
  "metadata": {
    "order_reference": "GO-ABC123",
    "customer_name_encrypted": true,
    "customer_phone_hash": "sha256:...",
    "total_amount": 200000,
    "payment_type": "full",
    "status": "PAID"
  },
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0 ..."
}
```

**Fields**:

- `event_type`: Identifies the event (pattern: `{resource}.{action}`)
- `event_id`: Unique event identifier (UUID v4)
- `timestamp`: ISO 8601 timestamp with milliseconds
- `tenant_id`: Tenant performing the action
- `user_id`: User who performed the action
- `resource_type`: Type of resource modified
- `resource_id`: Unique identifier of the resource
- `action`: Operation performed (CREATE, UPDATE, DELETE, ACCESS)
- `metadata`: Event-specific contextual data (no PII in plaintext)

---

### Audit Event: Offline Order Updated

**Topic**: `audit-events`  
**Event Key**: `{tenant_id}:{order_id}`  
**Event Type**: `offline_order.updated`

```json
{
  "event_type": "offline_order.updated",
  "event_id": "660e8400-e29b-41d4-a716-446655440001",
  "timestamp": "2026-02-07T11:00:00.456Z",
  "tenant_id": "tenant-uuid",
  "user_id": "user-uuid",
  "user_name": "Jane Manager",
  "user_role": "manager",
  "resource_type": "offline_order",
  "resource_id": "order-uuid",
  "action": "UPDATE",
  "changes": {
    "customer_phone": {
      "from_hash": "sha256:old_hash",
      "to_hash": "sha256:new_hash",
      "changed": true
    },
    "notes": {
      "from": "Will call tomorrow",
      "to": "Called - will pick up today",
      "changed": true
    },
    "items": {
      "added": [{ "product_id": "prod-uuid", "quantity": 2, "unit_price": 50000 }],
      "removed": [],
      "modified": []
    }
  },
  "metadata": {
    "order_reference": "GO-ABC123",
    "previous_total": 200000,
    "new_total": 300000
  },
  "ip_address": "192.168.1.101",
  "user_agent": "Mozilla/5.0 ..."
}
```

**Changes Object**:

- For PII fields: Log hashes only (never plaintext)
- For non-PII: Log `from` and `to` values
- For complex objects (items): Log specific operations (added, removed, modified)

---

### Audit Event: Offline Order Deleted

**Topic**: `audit-events`  
**Event Key**: `{tenant_id}:{order_id}`  
**Event Type**: `offline_order.deleted`

```json
{
  "event_type": "offline_order.deleted",
  "event_id": "770e8400-e29b-41d4-a716-446655440002",
  "timestamp": "2026-02-07T15:00:00.789Z",
  "tenant_id": "tenant-uuid",
  "user_id": "user-uuid",
  "user_name": "Admin Owner",
  "user_role": "owner",
  "resource_type": "offline_order",
  "resource_id": "order-uuid",
  "action": "DELETE",
  "metadata": {
    "order_reference": "GO-ABC123",
    "deletion_reason": "Duplicate order entry",
    "order_status_at_deletion": "PENDING",
    "total_amount": 200000,
    "soft_delete": true
  },
  "ip_address": "192.168.1.102",
  "user_agent": "Mozilla/5.0 ..."
}
```

---

### Audit Event: Offline Order Accessed

**Topic**: `audit-events`  
**Event Key**: `{tenant_id}:{order_id}`  
**Event Type**: `offline_order.accessed`

```json
{
  "event_type": "offline_order.accessed",
  "event_id": "880e8400-e29b-41d4-a716-446655440003",
  "timestamp": "2026-02-07T12:30:00.321Z",
  "tenant_id": "tenant-uuid",
  "user_id": "user-uuid",
  "user_name": "Staff Member",
  "user_role": "staff",
  "resource_type": "offline_order",
  "resource_id": "order-uuid",
  "action": "ACCESS",
  "metadata": {
    "order_reference": "GO-ABC123",
    "access_type": "view_details",
    "pii_fields_accessed": ["customer_name", "customer_phone", "customer_email"]
  },
  "ip_address": "192.168.1.103",
  "user_agent": "Mozilla/5.0 ..."
}
```

**Note**: ACCESS events optional, enabled via configuration flag for high-security tenants.

---

### Audit Event: Payment Recorded

**Topic**: `audit-events`  
**Event Key**: `{tenant_id}:{order_id}`  
**Event Type**: `offline_order.payment_recorded`

```json
{
  "event_type": "offline_order.payment_recorded",
  "event_id": "990e8400-e29b-41d4-a716-446655440004",
  "timestamp": "2026-02-07T14:00:00.654Z",
  "tenant_id": "tenant-uuid",
  "user_id": "user-uuid",
  "user_name": "Cashier User",
  "user_role": "cashier",
  "resource_type": "payment_record",
  "resource_id": "payment-record-uuid",
  "action": "CREATE",
  "metadata": {
    "order_id": "order-uuid",
    "order_reference": "GO-ABC123",
    "payment_number": 1,
    "amount_paid": 116667,
    "payment_method": "cash",
    "remaining_balance_after": 233333,
    "order_status_changed": false,
    "new_order_status": "PENDING"
  },
  "ip_address": "192.168.1.104",
  "user_agent": "Mozilla/5.0 ..."
}
```

---

### Analytics Event: Offline Order Created

**Topic**: `order-events`  
**Event Key**: `{tenant_id}`  
**Event Type**: `order.created`

```json
{
  "event_type": "order.created",
  "event_id": "aa0e8400-e29b-41d4-a716-446655440005",
  "timestamp": "2026-02-07T10:30:00.123Z",
  "tenant_id": "tenant-uuid",
  "order_id": "order-uuid",
  "order_reference": "GO-ABC123",
  "order_type": "offline",
  "status": "PAID",
  "subtotal_amount": 200000,
  "delivery_fee": 0,
  "total_amount": 200000,
  "delivery_type": "pickup",
  "item_count": 3,
  "payment_type": "full",
  "created_at": "2026-02-07T10:30:00Z",
  "recorded_by_user_id": "user-uuid"
}
```

**Purpose**: Enable analytics dashboard to include offline orders in revenue metrics, order counts, etc.

---

### Analytics Event: Offline Order Completed

**Topic**: `order-events`  
**Event Key**: `{tenant_id}`  
**Event Type**: `order.completed`

```json
{
  "event_type": "order.completed",
  "event_id": "bb0e8400-e29b-41d4-a716-446655440006",
  "timestamp": "2026-02-07T16:00:00.789Z",
  "tenant_id": "tenant-uuid",
  "order_id": "order-uuid",
  "order_reference": "GO-ABC123",
  "order_type": "offline",
  "status": "COMPLETE",
  "total_amount": 200000,
  "completion_time_hours": 5.5,
  "completed_at": "2026-02-07T16:00:00Z"
}
```

---

### Analytics Event: Payment Received

**Topic**: `order-events`  
**Event Key**: `{tenant_id}`  
**Event Type**: `payment.received`

```json
{
  "event_type": "payment.received",
  "event_id": "cc0e8400-e29b-41d4-a716-446655440007",
  "timestamp": "2026-02-07T14:00:00.654Z",
  "tenant_id": "tenant-uuid",
  "order_id": "order-uuid",
  "order_reference": "GO-ABC123",
  "order_type": "offline",
  "payment_id": "payment-record-uuid",
  "payment_number": 1,
  "amount_paid": 116667,
  "payment_method": "cash",
  "remaining_balance": 233333,
  "order_status": "PENDING",
  "payment_date": "2026-02-07T14:00:00Z"
}
```

---

## Publishing Patterns

### Transactional Outbox Pattern

**Problem**: Ensure database writes and Kafka publishes are atomic (no lost events or orphaned publishes).

**Solution**: Use transactional outbox pattern:

1. Write order data + event to `outbox` table in same transaction
2. Background worker polls `outbox` table
3. Publish event to Kafka
4. Mark event as published in `outbox`

**Outbox Table Schema**:

```sql
CREATE TABLE event_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    event_key VARCHAR(255) NOT NULL,
    event_payload JSONB NOT NULL,
    topic VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP,
    retry_count INTEGER DEFAULT 0,
    last_error TEXT
);

CREATE INDEX idx_outbox_pending ON event_outbox(created_at)
WHERE published_at IS NULL;
```

**Example Transaction**:

```sql
BEGIN;
  -- Insert offline order
  INSERT INTO guest_orders (...) VALUES (...);
  -- Insert to outbox
  INSERT INTO event_outbox (event_type, event_key, event_payload, topic)
  VALUES (
    'offline_order.created',
    'tenant-uuid:order-uuid',
    '{"event_type": "offline_order.created", ...}'::jsonb,
    'audit-events'
  );
COMMIT;
```

---

### Error Handling & Retries

**Retry Strategy**:

- Exponential backoff: 1s, 2s, 4s, 8s, 16s, 32s (max 6 retries)
- Dead letter queue: Events failing > 6 retries moved to `audit-events-dlq` topic
- Alerting: Monitor DLQ size, alert if > 10 events

**Idempotency**:

- Each event has unique `event_id` (UUID v4)
- Consumers should deduplicate based on `event_id` (use cache or database)
- Safe to replay events (no side effects)

---

## Consumer Implementations

### Audit Service

**Role**: Persist audit events to audit database for compliance reporting.

**Schema**:

```sql
CREATE TABLE audit_trail (
    id UUID PRIMARY KEY,
    event_type VARCHAR(100),
    tenant_id UUID,
    user_id UUID,
    resource_type VARCHAR(50),
    resource_id UUID,
    action VARCHAR(20),
    changes JSONB,
    metadata JSONB,
    timestamp TIMESTAMP,
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_audit_trail_tenant ON audit_trail(tenant_id, timestamp DESC);
CREATE INDEX idx_audit_trail_resource ON audit_trail(resource_type, resource_id);
CREATE INDEX idx_audit_trail_user ON audit_trail(user_id, timestamp DESC);
```

**Processing**: Read from `audit-events`, deserialize, insert to database (idempotent based on event_id).

---

### Analytics Service

**Role**: Aggregate order and payment events for dashboard metrics.

**Aggregations**:

- Daily revenue by order_type (online vs. offline)
- Order count by status and type
- Average order value by type
- Payment method distribution
- Installment completion rate

**Processing**: Read from `order-events`, update aggregation tables (time-series or materialized views).

---

## Monitoring & Observability

### Metrics to Track

1. **Publishing Metrics**:
   - `offline_orders_events_published_total{topic, event_type}`: Counter
   - `offline_orders_event_publish_duration_seconds{topic}`: Histogram
   - `offline_orders_outbox_pending_count`: Gauge

2. **Consumer Lag**:
   - `kafka_consumer_lag{topic, consumer_group}`: Gauge
   - Alert if lag > 1000 messages

3. **Error Rates**:
   - `offline_orders_event_publish_errors_total{topic, error_type}`: Counter
   - `audit_events_dlq_size`: Gauge

### Logging

- Log event publish success/failure with event_id and order_id
- Structured logs (JSON format) for easy parsing
- Include trace_id for distributed tracing

---

## Security Considerations

### PII Protection

- **Never log plaintext PII** in events (names, phones, emails, addresses)
- Use hashes (SHA-256) for change tracking: `"customer_phone_hash": "sha256:..."`
- Encrypt `event_payload` in outbox table if it contains sensitive metadata

### Access Control

- Kafka topics use ACLs (order-service can publish, audit/analytics services can consume)
- No direct tenant access to Kafka (all via services)

---

## Testing Strategies

### Unit Tests

```go
func TestPublishOfflineOrderCreatedEvent(t *testing.T) {
    // Mock Kafka producer
    // Create offline order
    // Assert event published with correct schema
    // Assert event_outbox record created
}
```

### Integration Tests

```go
func TestEventOutboxWorker(t *testing.T) {
    // Insert event to outbox
    // Run worker
    // Assert event published to Kafka
    // Assert published_at timestamp set
}
```

### Contract Tests

- Validate event schemas match documented contracts
- Use JSON Schema validation
- Run as part of CI/CD pipeline

---

## Migration & Rollout

### Phase 1: Enable Publishing

- Deploy order-service with event publishing code (feature flag OFF)
- Enable feature flag for staging environment
- Monitor for errors, tune retry logic

### Phase 2: Enable Consumers

- Deploy audit-service consumer (idempotency enabled)
- Deploy analytics-service consumer
- Verify events processed correctly

### Phase 3: Production Rollout

- Enable feature flag in production
- Monitor event lag and error rates
- Validate audit trail completeness

---

## Open Questions: None

All event contracts defined. No blocking issues for implementation.
