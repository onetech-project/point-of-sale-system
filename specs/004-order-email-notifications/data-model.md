# Data Model: Order Email Notifications

**Feature**: 004-order-email-notifications  
**Date**: 2025-12-11  
**Phase**: 1 - Design

## Entity Terminology Mapping

This section clarifies how conceptual entities in spec.md map to database tables:

- **Order Notification** (spec.md) → `notifications` table with `event_type='order.paid.staff'`
- **Receipt Email** (spec.md) → `notifications` table with `event_type='order.paid.customer'`
- **Notification Configuration** (spec.md) → `notification_configs` table (tenant-level) + `users.receive_order_notifications` (per-user)
- **Notification History Entry** (spec.md) → Query result from `notifications` table filtered by order-related event types

## Entity Overview

This feature extends existing entities and adds new ones to support email notifications for paid orders.

---

## 1. Extended Entities

### 1.1 User (Extended)

**Existing Table**: `users`  
**Extension**: Add notification preference flag

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| receive_order_notifications | BOOLEAN | NOT NULL, DEFAULT false | Whether this user receives email notifications for paid orders |

**Indexes**:
- `idx_users_order_notifications` on `(tenant_id, receive_order_notifications)` WHERE `receive_order_notifications = true`

**Relationships**:
- User belongs to Tenant (existing: tenant_id FK)
- User has many Notifications (existing relationship)

**Validation Rules**:
- Only applies to users with role in ('owner', 'manager', 'cashier')
- Requires valid email address when enabled

**State Transitions**:
- Can be enabled/disabled by tenant administrator
- Persists across user sessions

---

### 1.2 Notification (Extended Usage)

**Existing Table**: `notifications`  
**Extension**: New event_type and metadata structure for order notifications

**New Event Types**:
- `order.paid.staff` - Notification to tenant staff about paid order
- `order.paid.customer` - Receipt to customer about paid order

**Metadata Schema for Order Notifications**:
```json
{
  "order_id": "uuid",
  "order_reference": "ORD-001",
  "transaction_id": "midtrans-txn-123",
  "customer_name": "John Doe",
  "customer_phone": "+62123456789",
  "customer_email": "customer@example.com",
  "delivery_type": "delivery|pickup|dine_in",
  "delivery_address": "Full address string",
  "table_number": "Optional table number",
  "items": [
    {
      "product_name": "Product A",
      "quantity": 2,
      "unit_price": 50000,
      "total_price": 100000
    }
  ],
  "subtotal_amount": 150000,
  "delivery_fee": 10000,
  "total_amount": 160000,
  "payment_method": "QRIS",
  "paid_at": "2025-12-11T10:30:00Z"
}
```

**Validation Rules**:
- `event_type` must match pattern: `order.paid.(staff|customer)`
- `metadata->>'transaction_id'` required for duplicate detection
- `status` must be one of: pending, sent, failed, cancelled
- `retry_count` must be <= `max_retries` (default 3)

**State Transitions**:
```
pending → sent (on successful email delivery)
pending → failed (on delivery error, retry_count++)
failed → sent (on successful retry)
failed → failed (on retry failure, retry_count++)
pending/failed → cancelled (manual cancellation)
```

**Duplicate Prevention Query**:
```sql
SELECT EXISTS(
    SELECT 1 FROM notifications 
    WHERE tenant_id = $1 
      AND event_type LIKE 'order.paid.%'
      AND metadata->>'transaction_id' = $2
      AND status IN ('sent', 'pending')
)
```

---

## 2. New Entities

### 2.1 NotificationConfig

**Purpose**: Tenant-level configuration for notification behavior

**Table**: `notification_configs`

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | UUID | PRIMARY KEY, DEFAULT gen_random_uuid() | Unique identifier |
| tenant_id | UUID | NOT NULL, UNIQUE, FK(tenants.id) ON DELETE CASCADE | Tenant owner |
| order_notifications_enabled | BOOLEAN | NOT NULL, DEFAULT true | Global enable/disable for order notifications |
| test_mode | BOOLEAN | NOT NULL, DEFAULT false | If true, emails go to test address only |
| test_email | VARCHAR(255) | NULL | Email address for test mode |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | Last update timestamp |

**Indexes**:
- `idx_notification_configs_tenant` on `(tenant_id)` (unique constraint serves as index)

**Relationships**:
- NotificationConfig belongs to Tenant (tenant_id FK)

**Validation Rules**:
- `tenant_id` must be unique (one config per tenant)
- If `test_mode` is true, `test_email` must be valid email address
- Created automatically when tenant is created (default values)

**Business Rules**:
- If `order_notifications_enabled` is false, no order notifications sent
- `test_mode` is for development/staging environments only
- Admin can toggle settings without service restart

---

## 3. Data Relationships Diagram

```
┌─────────────┐
│   Tenants   │
└──────┬──────┘
       │
       │ 1:N
       ├──────────────────────────────┐
       │                              │
       ▼                              ▼
┌─────────────┐              ┌──────────────────┐
│    Users    │              │ NotificationConfigs│
│  (Extended) │              │     (New)        │
└──────┬──────┘              └──────────────────┘
       │                              
       │ receive_order_notifications  
       │ (new field)                  
       │                              
       │ 1:N                          
       ▼                              
┌─────────────────┐           
│  Notifications  │           
│   (Extended)    │◄──────────┐
└─────────────────┘           │
       ▲                      │
       │                      │
       │ references           │
       │                      │
┌──────┴──────┐         ┌─────┴──────┐
│ GuestOrders │         │OrderItems  │
│  (Existing) │         │ (Existing) │
└─────────────┘         └────────────┘
```

**Key Relationships**:
1. Tenant has many Users (existing)
2. Tenant has one NotificationConfig (new, 1:1)
3. User has many Notifications (existing, as sender/recipient)
4. GuestOrder triggers Notifications (event-driven, no direct FK)
5. Notification metadata contains denormalized order data (for audit trail)

---

## 4. Domain Events

### 4.1 OrderPaidEvent

**Event Type**: `order.paid`  
**Published By**: order-service  
**Consumed By**: notification-service  
**Kafka Topic**: `notification-events`

**Event Schema**:
```json
{
  "event_id": "uuid",
  "event_type": "order.paid",
  "tenant_id": "uuid",
  "timestamp": "2025-12-11T10:30:00Z",
  "metadata": {
    "order_id": "uuid",
    "order_reference": "ORD-001",
    "customer_email": "customer@example.com",
    "customer_name": "John Doe",
    "customer_phone": "+62123456789",
    "delivery_type": "delivery",
    "delivery_address": "...",
    "table_number": null,
    "items": [
      {
        "product_id": "uuid",
        "product_name": "Product A",
        "quantity": 2,
        "unit_price": 50000,
        "total_price": 100000
      }
    ],
    "subtotal_amount": 150000,
    "delivery_fee": 10000,
    "total_amount": 160000,
    "payment_method": "QRIS",
    "transaction_id": "midtrans-txn-123",
    "paid_at": "2025-12-11T10:30:00Z"
  }
}
```

**Event Guarantees**:
- At-least-once delivery (Kafka default)
- Ordered per partition (keyed by tenant_id)
- Idempotent processing (duplicate detection in notification-service)

**Event Processing Flow**:
1. order-service updates order status to PAID
2. order-service publishes OrderPaidEvent to Kafka
3. notification-service consumes event
4. notification-service checks for duplicate (transaction_id)
5. If not duplicate:
   - Query users with receive_order_notifications=true for tenant
   - Send staff notification emails
   - If customer_email provided, send customer receipt
   - Log all notifications to database
6. If duplicate: skip processing, log warning

---

## 5. Query Patterns

### 5.1 Get Staff Recipients for Tenant

**Use Case**: Determine which staff members to notify for an order

```sql
SELECT 
    id,
    email,
    name,
    role
FROM users
WHERE tenant_id = $1
  AND receive_order_notifications = true
  AND email IS NOT NULL
  AND email != ''
ORDER BY role, name;
```

**Performance**: Uses `idx_users_order_notifications` index

---

### 5.2 Check for Duplicate Notification

**Use Case**: Prevent duplicate notifications for same payment event

```sql
SELECT EXISTS(
    SELECT 1 
    FROM notifications
    WHERE tenant_id = $1
      AND event_type LIKE 'order.paid.%'
      AND metadata->>'transaction_id' = $2
      AND status IN ('sent', 'pending')
) AS already_sent;
```

**Performance**: Uses `idx_notifications_tenant_id` and `idx_notifications_event_type` indexes. JSONB query on metadata is acceptable for low volume.

---

### 5.3 Get Notification History for Order

**Use Case**: Admin views all notifications sent for specific order

```sql
SELECT 
    id,
    event_type,
    type,
    recipient,
    subject,
    status,
    sent_at,
    failed_at,
    error_msg,
    retry_count,
    created_at
FROM notifications
WHERE tenant_id = $1
  AND metadata->>'order_reference' = $2
ORDER BY created_at ASC;
```

**Performance**: Requires new index: `CREATE INDEX idx_notifications_order_ref ON notifications USING GIN (metadata jsonb_path_ops);`

---

### 5.4 Get Failed Notifications Ready for Retry

**Use Case**: Background worker finds notifications to retry

```sql
SELECT 
    id,
    tenant_id,
    type,
    recipient,
    subject,
    body,
    metadata,
    retry_count,
    created_at,
    failed_at
FROM notifications
WHERE status = 'failed'
  AND retry_count < max_retries
  AND (
      (retry_count = 0 AND failed_at < NOW() - INTERVAL '1 minute') OR
      (retry_count = 1 AND failed_at < NOW() - INTERVAL '5 minutes') OR
      (retry_count = 2 AND failed_at < NOW() - INTERVAL '15 minutes')
  )
ORDER BY created_at ASC
LIMIT 100;
```

**Performance**: Uses `idx_notifications_status` index. Interval calculation acceptable for retry worker frequency (1 minute poll).

---

### 5.5 Get Notification History for Admin Dashboard

**Use Case**: Admin views recent notification history with pagination

```sql
SELECT 
    id,
    event_type,
    type,
    recipient,
    subject,
    status,
    sent_at,
    failed_at,
    error_msg,
    retry_count,
    created_at,
    metadata->>'order_reference' AS order_reference
FROM notifications
WHERE tenant_id = $1
  AND event_type LIKE 'order.paid.%'
  AND created_at >= $2  -- filter by date range
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;  -- pagination
```

**Performance**: Uses `idx_notifications_tenant_id` and `idx_notifications_created_at` indexes.

---

## 6. Data Validation Rules

### 6.1 Email Address Validation

**Rule**: Must match RFC 5322 simplified pattern  
**Regex**: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`  
**Applied To**:
- `users.email` when saving
- `guest_orders.customer_email` when provided
- `notifications.recipient` before sending

**Go Implementation**:
```go
import "regexp"

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
    return emailRegex.MatchString(email)
}
```

---

### 6.2 Transaction ID Validation

**Rule**: Must be non-empty string from payment provider  
**Length**: 1-255 characters  
**Applied To**: `metadata->>'transaction_id'` in order.paid events  
**Purpose**: Duplicate detection

---

### 6.3 Retry Count Validation

**Rule**: `retry_count` must be <= `max_retries`  
**Default**: max_retries = 3  
**Applied To**: notifications table  
**Business Logic**: Worker skips notifications where retry_count >= max_retries

---

## 7. Data Integrity Constraints

### 7.1 Referential Integrity

- `notification_configs.tenant_id` → `tenants.id` (CASCADE DELETE)
- `notifications.tenant_id` → `tenants.id` (CASCADE DELETE)
- `notifications.user_id` → `users.id` (SET NULL on delete - preserve audit trail)

### 7.2 Unique Constraints

- `notification_configs.tenant_id` UNIQUE (one config per tenant)
- `notifications.id` PRIMARY KEY (UUID, system-generated)

### 7.3 Check Constraints

- `notifications.status` IN ('pending', 'sent', 'failed', 'cancelled')
- `notifications.type` IN ('email', 'sms', 'push')
- `notifications.retry_count` >= 0

---

## 8. Data Archival Strategy

### 8.1 Retention Policy

**Rule**: Keep notification history for 90 days (per FR-012)  
**After 90 days**: Archive or delete old notifications

**Implementation Options**:

**Option A: Time-based Partitioning (Recommended)**
```sql
-- Create partitioned table
CREATE TABLE notifications_partitioned (
    LIKE notifications INCLUDING ALL
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE notifications_2025_12 PARTITION OF notifications_partitioned
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');

-- Cron job drops old partitions
DROP TABLE IF EXISTS notifications_2025_09;  -- >90 days old
```

**Option B: Scheduled Deletion**
```sql
-- Weekly cron job
DELETE FROM notifications
WHERE created_at < NOW() - INTERVAL '90 days'
  AND status IN ('sent', 'cancelled');  -- Keep failed for investigation
```

**Decision**: Use Option B initially (simpler). Migrate to Option A if table size exceeds 10M rows.

---

## 9. Data Privacy & Security

### 9.1 PII Fields

**Personal Identifiable Information in notifications table**:
- `recipient` (email address)
- `metadata->>'customer_email'`
- `metadata->>'customer_name'`
- `metadata->>'customer_phone'`
- `metadata->>'delivery_address'`

**Protection Measures**:
- Row Level Security (RLS) on notifications table
- Encryption at rest (PostgreSQL transparent encryption)
- Masked in application logs: `customer@example.com` → `c***@e***.com`
- GDPR compliance: Cascade delete on tenant deletion

### 9.2 RLS Policy

```sql
-- Enable RLS on notifications table
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see their tenant's notifications
CREATE POLICY tenant_isolation ON notifications
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

**Application Code**: Must set RLS context before queries
```go
setContextSQL := fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)
_, err := tx.Exec(setContextSQL)
```

---

## 10. Performance Considerations

### 10.1 Indexes Summary

**Existing Indexes** (on notifications table):
- `idx_notifications_tenant_id` on (tenant_id)
- `idx_notifications_user_id` on (user_id)
- `idx_notifications_status` on (status)
- `idx_notifications_event_type` on (event_type)
- `idx_notifications_created_at` on (created_at DESC)
- `idx_notifications_recipient` on (recipient)

**New Indexes Needed**:
```sql
-- For duplicate detection on order notifications
CREATE INDEX idx_notifications_order_metadata 
ON notifications USING GIN (metadata jsonb_path_ops)
WHERE event_type LIKE 'order.paid.%';

-- For user notification preferences
CREATE INDEX idx_users_order_notifications 
ON users(tenant_id, receive_order_notifications)
WHERE receive_order_notifications = true;
```

### 10.2 Query Performance Targets

- Get staff recipients: <10ms (indexed query)
- Duplicate check: <20ms (JSONB GIN index)
- Notification history page: <100ms (paginated, indexed)
- Retry worker query: <500ms (low frequency, batch processing)

### 10.3 Storage Estimates

**Assumptions**:
- 1000 orders/day per tenant
- 10 tenants
- Each order generates 2-5 notifications (1 customer + 1-4 staff)
- Average notification row size: 2KB

**Calculation**:
- Daily: 10K orders × 3 notifications avg = 30K rows/day
- Monthly: 30K × 30 = 900K rows/month
- 90-day retention: 2.7M rows = ~5.4GB

**Mitigation**: Partitioning + archival keeps table size manageable.

---

## Data Model Summary

**Extended Entities**: 2 (User, Notification)  
**New Entities**: 1 (NotificationConfig)  
**New Indexes**: 2  
**Domain Events**: 1 (OrderPaidEvent)  
**Query Patterns**: 5 optimized queries  
**Validation Rules**: 3 categories  
**Retention Policy**: 90 days with archival strategy  
**Privacy**: RLS + encryption + masking  
**Storage**: 5.4GB for 90-day retention at 10K orders/day

**Ready for**: Contract definition and quickstart documentation.
