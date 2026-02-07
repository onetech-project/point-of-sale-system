# Phase 1: Data Model & Entities

**Feature**: Offline Order Management  
**Date**: February 7, 2026  
**Status**: Complete

## Entity-Relationship Overview

```
┌──────────────┐
│    Tenant    │
│  (existing)  │
└──────┬───────┘
       │
       │ 1:N
       │
┌──────▼───────────┐         ┌──────────────┐
│  Guest Orders    │         │   Product    │
│  (EXTEND)        │         │  (existing)  │
│  + order_type    │         └──────┬───────┘
│  + consent_*     │                │
└──────┬───────────┘                │
       │                            │
       │ 1:N                        │ N:1
       │                            │
┌──────▼───────────┐         ┌──────▼─────────┐
│   Order Items    │─────────│  Order Items   │
│   (existing)     │  N:1    │  (existing)    │
└──────────────────┘         └────────────────┘
       │
       │ N:1
       │
┌──────▼──────────────┐
│  Payment Terms      │
│  (NEW - 1:1 order)  │
└──────┬──────────────┘
       │
       │ 1:N
       │
┌──────▼──────────────┐
│  Payment Records    │
│  (NEW - N:1 order)  │
└─────────────────────┘

┌──────────────────────────────┐
│     Audit Trail Entry        │
│     (audit-service)          │
│  - Captures all offline      │
│    order operations          │
└──────────────────────────────┘
```

**Relationships**:

- **Tenant → GuestOrders**: 1:N (tenant isolation via RLS)
- **GuestOrders → OrderItems**: 1:N (order line items)
- **Product → OrderItems**: 1:N (product reference for inventory)
- **GuestOrders → PaymentTerms**: 1:1 (optional, only for installment orders)
- **GuestOrders → PaymentRecords**: 1:N (payment transaction log)
- **User → AuditTrailEntry**: Implicit via user_id in audit events

---

## Database Schema

### 1. Guest Orders (EXTEND EXISTING TABLE)

**Table**: `guest_orders` (add new columns to existing table)

```sql
-- Migration: 000018_add_offline_orders.up.sql
ALTER TABLE guest_orders
ADD COLUMN IF NOT EXISTS order_type VARCHAR(20) NOT NULL DEFAULT 'online',
ADD COLUMN IF NOT EXISTS data_consent_given BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS consent_method VARCHAR(20),
ADD COLUMN IF NOT EXISTS recorded_by_user_id UUID REFERENCES users(id),
ADD COLUMN IF NOT EXISTS last_modified_by_user_id UUID REFERENCES users(id),
ADD COLUMN IF NOT EXISTS last_modified_at TIMESTAMP;

-- Add CHECK constraint for order_type
ALTER TABLE guest_orders
ADD CONSTRAINT check_order_type
CHECK (order_type IN ('online', 'offline'));

-- Add CHECK constraint for consent_method
ALTER TABLE guest_orders
ADD CONSTRAINT check_consent_method
CHECK (consent_method IS NULL OR consent_method IN ('verbal', 'written', 'digital'));

-- Create index for offline order queries
CREATE INDEX IF NOT EXISTS idx_guest_orders_type_status
ON guest_orders(order_type, status, tenant_id);

-- Create index for user tracking
CREATE INDEX IF NOT EXISTS idx_guest_orders_recorded_by
ON guest_orders(recorded_by_user_id)
WHERE order_type = 'offline';

-- Create partial index for pending payment offline orders
CREATE INDEX IF NOT EXISTS idx_offline_orders_pending_payment
ON guest_orders(tenant_id, created_at DESC)
WHERE order_type = 'offline' AND status = 'PENDING';

COMMENT ON COLUMN guest_orders.order_type IS 'Distinguishes online (public self-service) vs offline (staff-recorded) orders';
COMMENT ON COLUMN guest_orders.data_consent_given IS 'UU PDP/GDPR: Whether customer explicitly consented to data collection';
COMMENT ON COLUMN guest_orders.consent_method IS 'How consent was obtained: verbal, written form, digital signature';
COMMENT ON COLUMN guest_orders.recorded_by_user_id IS 'Staff user who created the offline order';
COMMENT ON COLUMN guest_orders.last_modified_by_user_id IS 'Staff user who last edited the order';
```

**New Columns**:

- `order_type VARCHAR(20)`: 'online' or 'offline'
- `data_consent_given BOOLEAN`: UU PDP/GDPR compliance
- `consent_method VARCHAR(20)`: 'verbal', 'written', 'digital', or NULL
- `recorded_by_user_id UUID`: Staff who created offline order
- `last_modified_by_user_id UUID`: Staff who last edited order
- `last_modified_at TIMESTAMP`: Edit timestamp

**Existing Columns** (reused for offline orders):

- `id`, `order_reference`, `tenant_id`, `status`, `subtotal_amount`, `delivery_fee`, `total_amount`
- `customer_name`, `customer_phone`, `customer_email` (all encrypted)
- `delivery_type`, `table_number`, `notes`
- `created_at`, `paid_at`, `completed_at`, `cancelled_at`

**Validation Rules**:

- `order_type` must be 'online' or 'offline'
- `recorded_by_user_id` required when `order_type = 'offline'`
- `data_consent_given` should be TRUE for offline orders (staff responsibility)
- Cannot modify `order_reference`, `tenant_id`, `recorded_by_user_id` after creation
- `last_modified_at` auto-updated on UPDATE operations

---

### 2. Payment Terms (NEW TABLE)

**Table**: `payment_terms`

```sql
-- Migration: 000019_add_payment_terms.up.sql
CREATE TABLE IF NOT EXISTS payment_terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL UNIQUE REFERENCES guest_orders(id) ON DELETE CASCADE,

    -- Payment structure
    total_amount INTEGER NOT NULL CHECK (total_amount > 0),
    down_payment_amount INTEGER CHECK (down_payment_amount >= 0 AND down_payment_amount < total_amount),
    installment_count INTEGER CHECK (installment_count >= 0),
    installment_amount INTEGER CHECK (installment_amount >= 0),

    -- Schedule
    payment_schedule JSONB, -- Array of {installment_number, due_date, amount}

    -- Status tracking
    total_paid INTEGER NOT NULL DEFAULT 0 CHECK (total_paid >= 0 AND total_paid <= total_amount),
    remaining_balance INTEGER NOT NULL CHECK (remaining_balance >= 0),

    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by_user_id UUID NOT NULL REFERENCES users(id),

    -- Constraints
    CONSTRAINT check_payment_structure
    CHECK (
        (down_payment_amount IS NULL AND installment_count = 0) OR
        (down_payment_amount >= 0 AND installment_count > 0)
    ),

    CONSTRAINT check_remaining_balance
    CHECK (remaining_balance = total_amount - total_paid)
);

-- Indexes
CREATE INDEX idx_payment_terms_order_id ON payment_terms(order_id);
CREATE INDEX idx_payment_terms_balance ON payment_terms(remaining_balance, order_id)
WHERE remaining_balance > 0;

-- Comments
COMMENT ON TABLE payment_terms IS 'Payment schedule definition for offline orders with installments';
COMMENT ON COLUMN payment_terms.payment_schedule IS 'JSONB array: [{"installment_number": 1, "due_date": "2026-03-07", "amount": 50000}, ...]';
```

**Example `payment_schedule` JSONB**:

```json
[
  { "installment_number": 1, "due_date": "2026-03-07", "amount": 50000, "status": "pending" },
  { "installment_number": 2, "due_date": "2026-04-07", "amount": 50000, "status": "pending" },
  { "installment_number": 3, "due_date": "2026-05-07", "amount": 50000, "status": "pending" }
]
```

**Validation Rules**:

- Optional table (only created for orders with payment terms)
- `total_amount` must match `guest_orders.total_amount`
- `down_payment_amount + (installment_count × installment_amount)` must equal `total_amount`
- `remaining_balance` computed field: `total_amount - total_paid`
- `payment_schedule` validated: each installment has number, due_date, amount

---

### 3. Payment Records (NEW TABLE)

**Table**: `payment_records`

```sql
-- Migration: 000020_add_payment_records.up.sql
CREATE TABLE IF NOT EXISTS payment_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    payment_terms_id UUID REFERENCES payment_terms(id) ON DELETE SET NULL,

    -- Payment details
    payment_number INTEGER NOT NULL, -- 0 for down payment, 1+ for installments
    amount_paid INTEGER NOT NULL CHECK (amount_paid > 0),
    payment_date TIMESTAMP NOT NULL DEFAULT NOW(),
    payment_method VARCHAR(50) NOT NULL,

    -- Balance tracking
    remaining_balance_after INTEGER NOT NULL CHECK (remaining_balance_after >= 0),

    -- Metadata
    recorded_by_user_id UUID NOT NULL REFERENCES users(id),
    notes TEXT,
    receipt_number VARCHAR(100),

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT check_payment_method
    CHECK (payment_method IN ('cash', 'card', 'bank_transfer', 'check', 'other'))
);

-- Indexes
CREATE INDEX idx_payment_records_order_id ON payment_records(order_id, payment_date DESC);
CREATE INDEX idx_payment_records_date ON payment_records(payment_date DESC);
CREATE INDEX idx_payment_records_recorded_by ON payment_records(recorded_by_user_id);

-- Comments
COMMENT ON TABLE payment_records IS 'Transaction log of payments received for offline orders';
COMMENT ON COLUMN payment_records.payment_number IS '0 = down payment, 1+ = installment number';
COMMENT ON COLUMN payment_records.remaining_balance_after IS 'Outstanding balance after this payment applied';
```

**Validation Rules**:

- `payment_number = 0` for down payment (if exists)
- `payment_number >= 1` for installments (sequential)
- `amount_paid` must be > 0
- `remaining_balance_after` computed: previous remaining balance - `amount_paid`
- `payment_method` must be one of enum values
- Cannot delete payment records (append-only for audit trail)

---

## Entity Lifecycle Diagrams

### Offline Order Status Flow

```
┌─────────────┐
│   CREATE    │ recordedby_user creates offline order
│   PENDING   │ with customer data + items
└──────┬──────┘
       │
       ├─────────────────────────────────┐
       │                                 │
       │ Full payment                    │ Partial payment/
       │ recorded                        │ installment plan
       │                                 │
┌──────▼──────┐                   ┌──────▼──────────┐
│    PAID     │                   │ PENDING_PAYMENT │
│             │                   │ (with terms)    │
└──────┬──────┘                   └──────┬──────────┘
       │                                 │
       │                                 │ Payments recorded
       │                                 │ until total_paid =
       │                                 │ total_amount
       │                                 │
       │                          ┌──────▼──────┐
       │                          │    PAID     │
       │                          └──────┬──────┘
       │                                 │
       └─────────────┬───────────────────┘
                     │
                     │ Order fulfilled
                     │ (items delivered/picked up)
                     │
              ┌──────▼─────────┐
              │   COMPLETE     │
              └────────────────┘

       ┌─────────────────────────┐
       │  CANCELLED (any time)   │
       │  Soft delete or status  │
       └─────────────────────────┘
```

### Payment Recording Flow

```
[Staff creates order]
→ [Select payment option: Full / Down Payment + Installments]
→ IF full payment:
    → Record payment_record (payment_number=0, full amount)
    → Update order.status = PAID
    → No payment_terms created

→ IF down payment + installments:
    → Create payment_terms record
    → Record payment_record (payment_number=0, down payment amount)
    → Update payment_terms.total_paid
    → Calculate remaining_balance
    → Order remains PENDING_PAYMENT

[Staff records subsequent payment]
→ Record new payment_record (payment_number=N)
→ Update payment_terms.total_paid
→ IF total_paid >= total_amount:
    → Update order.status = PAID
    → Update order.paid_at = NOW()
```

---

## Data Integrity Constraints

### Referential Integrity

1. **Tenant Isolation** (existing RLS):

   ```sql
   -- RLS policy ensures users only see their tenant's orders
   CREATE POLICY orders_tenant_isolation ON guest_orders
   USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
   ```

2. **Cascade Deletes**:
   - Delete order → cascade delete payment_terms, payment_records, order_items
   - Soft delete via `cancelled_at` timestamp preferred over hard delete

3. **User References**:
   - `recorded_by_user_id` → users(id) (cannot be NULL for offline orders)
   - `last_modified_by_user_id` → users(id) (NULL allowed)

### Business Logic Constraints

1. **Payment Balanceconsistency**:

   ```sql
   -- Trigger to update payment_terms.total_paid on new payment_record
   CREATE TRIGGER update_payment_totals
   AFTER INSERT ON payment_records
   FOR EACH ROW
   EXECUTE FUNCTION update_payment_terms_totals();
   ```

2. **Status Transition Rules**:
   - Cannot set status = COMPLETE unless status = PAID
   - Cannot modify order items after status = PAID
   - Cannot delete order with status = COMPLETE (must cancel first)

3. **Encryption Requirements**:
   - `customer_name`: AES-GCM encrypted (non-searchable)
   - `customer_phone`: Deterministic encryption (searchable)
   - `customer_email`: Deterministic encryption (searchable)
   - Encryption handled by application layer (Vault integration)

---

## Migration Strategy

### Migration Order

1. **000060_add_offline_orders.up.sql**: Extend `guest_orders` table
   - Add columns: order_type, consent fields, user tracking
   - Add indexes for offline order queries
   - Backfill existing rows: `UPDATE guest_orders SET order_type = 'online' WHERE order_type IS NULL`

2. **000061_add_payment_terms.up.sql**: Create `payment_terms` table
   - Define schema with validation constraints
   - Add indexes

3. **000062_add_payment_records.up.sql**: Create `payment_records` table
   - Define schema with payment method enum
   - Add indexes
   - Create trigger for auto-updating payment totals

### Rollback Plan

Each migration has corresponding `.down.sql`:

- `000062_add_payment_records.down.sql`: DROP TABLE payment_records CASCADE
- `000061_add_payment_terms.down.sql`: DROP TABLE payment_terms CASCADE
- `000060_add_offline_orders.down.sql`:

  ```sql
  ALTER TABLE guest_orders
  DROP COLUMN order_type,
  DROP COLUMN data_consent_given,
  DROP COLUMN consent_method,
  DROP COLUMN recorded_by_user_id,
  DROP COLUMN last_modified_by_user_id,
  DROP COLUMN last_modified_at;

  DROP INDEX IF EXISTS idx_guest_orders_type_status;
  DROP INDEX IF EXISTS idx_guest_orders_recorded_by;
  DROP INDEX IF EXISTS idx_offline_orders_pending_payment;
  ```

### Data Backfill

- Existing online orders automatically tagged with `order_type = 'online'` (default value)
- No manual data migration needed

---

## Query Patterns & Indexes

### Common Queries

1. **List all offline orders for tenant**:

   ```sql
   SELECT * FROM guest_orders
   WHERE tenant_id = $1 AND order_type = 'offline'
   ORDER BY created_at DESC
   LIMIT 50;
   ```

   **Index**: `idx_guest_orders_type_status (order_type, status, tenant_id)`

2. **Get orders with outstanding balance**:

   ```sql
   SELECT o.*, pt.remaining_balance
   FROM guest_orders o
   JOIN payment_terms pt ON pt.order_id = o.id
   WHERE o.tenant_id = $1
     AND o.order_type = 'offline'
     AND pt.remaining_balance > 0
   ORDER BY pt.remaining_balance DESC;
   ```

   **Index**: `idx_payment_terms_balance (remaining_balance, order_id) WHERE remaining_balance > 0`

3. **Payment history for order**:

   ```sql
   SELECT * FROM payment_records
   WHERE order_id = $1
   ORDER BY payment_date ASC;
   ```

   **Index**: `idx_payment_records_order_id (order_id, payment_date DESC)`

4. **Audit trail for order**:
   ```sql
   SELECT * FROM audit_events
   WHERE resource_type = 'offline_order'
     AND resource_id = $1
   ORDER BY timestamp DESC;
   ```
   **(Handled by audit-service, separate database)**

---

## Performance Considerations

### Database Optimization

1. **Partitioning** (future optimization if needed):
   - Partition `guest_orders` by `order_type` if online orders vastly outnumber offline orders
   - Current scale (10-50 tenants, 1000 orders/month) doesn't require partitioning yet

2. **Connection Pooling**:
   - Reuse existing connection pool (already configured in order-service)
   - No additional connections needed

3. **Query Performance**:
   - All indexes support tenant-scoped queries (tenant_id in WHERE clause)
   - Partial indexes reduce index size for filtered queries
   - JSONB payment_schedule indexed with GIN index if frequent searches needed

### Write Performance

- Payment record inserts: O(1) append-only, no updates
- Order updates: Row-level locking, typically <10ms
- Concurrent offline order creation: No contention with online orders (different endpoints)

---

## Security Considerations

### Encryption At Rest

All PII columns encrypted before storage:

```go
// Example: Creating offline order
encryptedName := encryptor.Encrypt(customerName)  //AES-GCM
encryptedPhone := encryptor.EncryptSearchable(customerPhone)  // Deterministic
encryptedEmail := encryptor.EncryptSearchable(customerEmail)  // Deterministic

order := &models.GuestOrder{
    CustomerName: encryptedName,
    CustomerPhone: encryptedPhone,
    CustomerEmail: encryptedEmail,
    OrderType: "offline",
    // ...
}
```

### Row-Level Security (RLS)

Tenant isolation enforced at database via existing RLS policies:

```sql
-- Already exists, applies to offline orders too
CREATE POLICY orders_tenant_isolation ON guest_orders
FOR ALL
USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
```

### API Authorization

- Middleware validates JWT token + user role
- RBAC enforced: Delete operations require `owner` or `manager` role
- Audit log captures all access attempts

---

## Open Questions: None

All data model decisions finalized based on research.md findings.
