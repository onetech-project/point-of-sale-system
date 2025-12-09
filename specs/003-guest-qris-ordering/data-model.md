# Phase 1: Data Model & Entities

**Feature**: QRIS Guest Ordering System  
**Date**: 2025-12-03  
**Status**: Complete

## Entity-Relationship Overview

```
┌─────────────┐
│   Tenant    │
│ (existing)  │
└──────┬──────┘
       │
       │ 1:N
       │
┌──────▼──────────┐         ┌──────────────────┐
│ TenantConfig    │         │    Product       │
│   (extend)      │         │   (existing)     │
└─────────────────┘         └────────┬─────────┘
                                     │
                                     │ 1:N
        ┌────────────────────────────┼──────────────┐
        │                            │              │
┌───────▼──────┐            ┌────────▼─────┐  ┌────▼─────────────┐
│  GuestCart   │            │ GuestOrder   │  │InventoryReserv.  │
│  (Redis)     │            │              │  │                  │
└──────────────┘            └──────┬───────┘  └──────────────────┘
                                   │
                     ┌─────────────┼─────────────┐
                     │             │             │
            ┌────────▼──────┐  ┌──▼──────────┐  ┌▼───────────────┐
            │  OrderItem    │  │PaymentTrans.│  │DeliveryAddress │
            │               │  │             │  │                │
            └───────────────┘  └─────────────┘  └────────────────┘
```

## Database Schema

### 1. Guest Orders

**Table**: `guest_orders`

```sql
CREATE TABLE guest_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_reference VARCHAR(20) UNIQUE NOT NULL, -- Human-readable: GO-XXXXXX
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    
    -- Order details
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING', 
        -- PENDING, PAID, COMPLETE, CANCELLED
    subtotal_amount INTEGER NOT NULL, -- In smallest currency unit (IDR cents)
    delivery_fee INTEGER NOT NULL DEFAULT 0,
    total_amount INTEGER NOT NULL,
    
    -- Customer contact
    customer_name VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(20) NOT NULL,
    
    -- Delivery type
    delivery_type VARCHAR(20) NOT NULL, -- pickup, delivery, dine_in
    table_number VARCHAR(50), -- For dine_in
    notes TEXT, -- Staff notes or customer preferences
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMP,
    completed_at TIMESTAMP,
    cancelled_at TIMESTAMP,
    
    -- Metadata
    session_id VARCHAR(255), -- For tracking
    ip_address INET,
    user_agent TEXT,
    
    -- Indexes
    INDEX idx_tenant_status (tenant_id, status),
    INDEX idx_order_reference (order_reference),
    INDEX idx_created_at (created_at DESC)
);
```

**Validation Rules**:
- `status` must be one of: PENDING, PAID, COMPLETE, CANCELLED
- `delivery_type` must be one of: pickup, delivery, dine_in
- `subtotal_amount`, `delivery_fee`, `total_amount` must be >= 0
- `total_amount` must equal `subtotal_amount + delivery_fee`
- `customer_phone` must match pattern: `^\+?[0-9]{10,15}$`
- `paid_at` required when status = PAID
- `completed_at` required when status = COMPLETE

**Business Rules**:
- Order reference generated on creation: `GO-{random_6_chars}`
- Status transitions validated (see research.md)
- Cannot modify order after status = PAID (except status changes)
- Soft delete: use cancelled_at instead of DELETE

---

### 2. Order Items

**Table**: `order_items`

```sql
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    
    -- Item details (snapshot at order time)
    product_name VARCHAR(255) NOT NULL,
    product_sku VARCHAR(100),
    quantity INTEGER NOT NULL,
    unit_price INTEGER NOT NULL, -- Price at time of order
    total_price INTEGER NOT NULL, -- quantity * unit_price
    
    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Indexes
    INDEX idx_order_id (order_id),
    INDEX idx_product_id (product_id)
);
```

**Validation Rules**:
- `quantity` must be > 0
- `unit_price` must be >= 0
- `total_price` must equal `quantity * unit_price`
- `product_name` copied from products table (snapshot)

**Business Rules**:
- Immutable after creation (order history preservation)
- Product details snapshotted to preserve pricing history
- If product deleted, order items retain product_name

---

### 3. Inventory Reservations

**Table**: `inventory_reservations`

```sql
CREATE TABLE inventory_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    
    -- Reservation details
    quantity INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
        -- active, expired, converted, released
    
    -- Timing
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL, -- created_at + 15 minutes
    released_at TIMESTAMP,
    
    -- Indexes
    INDEX idx_order_product (order_id, product_id),
    INDEX idx_expires_at (expires_at),
    INDEX idx_status_expires (status, expires_at)
);
```

**Validation Rules**:
- `status` must be one of: active, expired, converted, released
- `quantity` must be > 0
- `expires_at` must be > `created_at`
- `released_at` required when status != active

**Business Rules**:
- Created when order enters PENDING status
- Expires after configurable TTL (default: 15 minutes)
- Background job marks expired reservations
- Status = 'converted' when order PAID
- Status = 'released' when order CANCELLED or payment fails
- Multiple reservations for same product allowed (different orders)

---

### 4. Payment Transactions

**Table**: `payment_transactions`

```sql
CREATE TABLE payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id),
    
    -- Midtrans details
    midtrans_transaction_id VARCHAR(255) UNIQUE,
    midtrans_order_id VARCHAR(255) NOT NULL, -- Our order_reference
    
    -- Transaction details
    amount INTEGER NOT NULL,
    payment_type VARCHAR(50), -- qris, gopay, etc.
    transaction_status VARCHAR(50), -- pending, settlement, cancel, etc.
    fraud_status VARCHAR(50), -- accept, challenge, deny
    
    -- Notification
    notification_payload JSONB, -- Full webhook payload
    signature_key VARCHAR(512), -- Webhook signature
    signature_verified BOOLEAN NOT NULL DEFAULT false,
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    notification_received_at TIMESTAMP,
    settled_at TIMESTAMP,
    
    -- Idempotency
    idempotency_key VARCHAR(255) UNIQUE, -- midtrans_transaction_id + status
    
    -- Indexes
    INDEX idx_order_id (order_id),
    INDEX idx_midtrans_transaction_id (midtrans_transaction_id),
    INDEX idx_created_at (created_at DESC)
);
```

**Validation Rules**:
- `amount` must match order total_amount
- `signature_verified` must be true before processing
- `notification_payload` must be valid JSON

**Business Rules**:
- One transaction per order (main payment)
- Idempotency key prevents duplicate processing: `{midtrans_id}:{status}`
- If webhook received multiple times with same key, skip processing
- Store full notification payload for audit
- Transaction created on payment initiation
- Updated on webhook notification

---

### 5. Delivery Addresses

**Table**: `delivery_addresses`

```sql
CREATE TABLE delivery_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    
    -- Address details
    address_text TEXT NOT NULL, -- Full address as entered
    
    -- Geocoding results
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    geocoded_address TEXT, -- Formatted address from Google
    place_id VARCHAR(255), -- Google Place ID
    
    -- Service area validation
    is_serviceable BOOLEAN NOT NULL DEFAULT false,
    service_area_zone VARCHAR(100), -- Zone name if applicable
    
    -- Delivery fee
    calculated_delivery_fee INTEGER,
    distance_km DECIMAL(6, 2), -- Distance from tenant
    
    -- Metadata
    geocoded_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Indexes
    INDEX idx_order_id (order_id),
    INDEX idx_lat_lng (latitude, longitude)
);
```

**Validation Rules**:
- `address_text` required for delivery_type = 'delivery'
- If `latitude` present, `longitude` must also be present
- `calculated_delivery_fee` must be >= 0

**Business Rules**:
- Only created for delivery orders
- Geocoding attempted on creation
- If geocoding fails, order blocked
- Address cached in Redis by hash for reuse
- Service area validation before accepting order

---

### 6. Tenant Configuration (Extension)

**Table**: `tenant_configs` (new) or extend `tenants` table

```sql
-- Option 1: Separate config table
CREATE TABLE tenant_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) UNIQUE,
    
    -- Delivery types
    enabled_delivery_types TEXT[] NOT NULL DEFAULT '{pickup}', 
        -- Array of: pickup, delivery, dine_in
    
    -- Service area (for delivery)
    service_area_type VARCHAR(20), -- radius, polygon
    service_area_data JSONB, -- Flexible config
        -- radius: {"center": {"lat": X, "lng": Y}, "radius_km": 5}
        -- polygon: {"coordinates": [[lat,lng], [lat,lng], ...]}
    
    -- Delivery fee pricing
    enable_delivery_fee_calculation BOOLEAN DEFAULT true, -- Tenant can disable automatic fee calculation
    delivery_fee_type VARCHAR(20), -- distance, zone, flat (null when disabled)
    delivery_fee_config JSONB, -- Flexible pricing rules (null when disabled)
        -- See research.md for examples
    
    -- Operational settings
    inventory_reservation_ttl_minutes INTEGER DEFAULT 15,
    min_order_amount INTEGER DEFAULT 0,
    
    -- Tenant location (for distance calculation)
    location_lat DECIMAL(10, 8),
    location_lng DECIMAL(11, 8),
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Option 2: Extend tenants table (if schema allows)
ALTER TABLE tenants ADD COLUMN config JSONB;
```

**Validation Rules**:
- `enabled_delivery_types` must not be empty
- If 'delivery' enabled, `service_area_type` and `service_area_data` required
- `delivery_fee_type` required if 'delivery' enabled
- `inventory_reservation_ttl_minutes` must be >= 5

**Business Rules**:
- Default config created when tenant registered
- Config changes require staff permissions
- Pricing changes don't affect existing orders
- Service area changes require validation

---

## Redis Data Structures

### Guest Cart

**Key Pattern**: `cart:{tenant_id}:{session_id}`

**Value**: JSON string
```json
{
  "tenant_id": "uuid",
  "session_id": "uuid",
  "items": [
    {
      "product_id": "uuid",
      "quantity": 2,
      "added_at": "2025-12-03T10:00:00Z"
    }
  ],
  "created_at": "2025-12-03T10:00:00Z",
  "updated_at": "2025-12-03T10:05:00Z"
}
```

**TTL**: 24 hours (configurable)

**Operations**:
- `SET cart:{tenant}:{session} {json} EX 86400`
- `GET cart:{tenant}:{session}`
- `DEL cart:{tenant}:{session}`
- `EXPIRE cart:{tenant}:{session} 86400` (on update)

---

### Session Tracking

**Key Pattern**: `session:{session_id}`

**Value**: JSON string
```json
{
  "session_id": "uuid",
  "tenant_id": "uuid",
  "created_at": "2025-12-03T10:00:00Z",
  "last_activity": "2025-12-03T10:30:00Z",
  "ip_address": "192.168.1.1",
  "user_agent": "Mozilla/5.0..."
}
```

**TTL**: 24 hours

---

### Inventory Cache

**Key Pattern**: `inventory:available:{product_id}`

**Value**: Integer (available quantity)

**TTL**: None (invalidated on changes)

**Operations**:
- `GET inventory:available:{product_id}`
- `DECR inventory:available:{product_id}` (on reservation)
- `INCR inventory:available:{product_id}` (on release)
- `DEL inventory:available:{product_id}` (force refresh)

---

### Geocoding Cache

**Key Pattern**: `geocode:{address_hash}`

**Value**: JSON string
```json
{
  "latitude": 1.234567,
  "longitude": 103.123456,
  "formatted_address": "123 Main St, City",
  "place_id": "ChIJ...",
  "geocoded_at": "2025-12-03T10:00:00Z"
}
```

**TTL**: 7 days

---

## Entity Lifecycle Diagrams

### Order Lifecycle

```
┌─────────┐
│ PENDING │ ← Order created, inventory reserved
└────┬────┘
     │
     │ Payment notification (success)
     ▼
┌─────────┐
│  PAID   │ ← Inventory converted to permanent
└────┬────┘
     │
     │ Staff marks complete
     ▼
┌─────────┐
│COMPLETE │ ← Order fulfilled
└─────────┘

Alternate paths:
PENDING → CANCELLED (payment fails/expires)
PAID → CANCELLED (staff refund)
```

---

### Inventory Reservation Lifecycle

```
┌────────┐
│ active │ ← Created on order PENDING
└───┬────┘
    │
    ├─ expires_at reached → expired (inventory released)
    ├─ order PAID → converted (inventory deducted)
    └─ order CANCELLED → released (inventory restored)
```

---

## Data Integrity Constraints

### Database Level
- Foreign keys with CASCADE on parent delete
- UNIQUE constraints on order_reference, idempotency_key
- CHECK constraints on status enums
- NOT NULL on critical fields

### Application Level
- Validate status transitions in service layer
- Verify amount calculations before persist
- Confirm inventory availability with locks
- Validate geocoding results before acceptance

### Audit Trail
- Log all status changes with timestamp
- Store payment notification payload
- Track inventory reservation state changes
- Record delivery address validation results

---

## Migration Strategy

### Phase 1: Schema Creation
1. Run migrations 000020-000024 (tables above)
2. Create indexes for performance
3. Set up foreign key constraints

### Phase 2: Seed Data
1. Default tenant configs for existing tenants
2. Test data for development environment

### Phase 3: Redis Setup
1. Configure Redis keyspace
2. Set up eviction policy (allkeys-lru)
3. Configure persistence (AOF)

### Phase 4: Validation
1. Run integration tests against schema
2. Verify constraint enforcement
3. Test cascade deletes
4. Validate query performance

---

## Open Questions Resolved

1. **Tenant Configuration UI**: Will be created in admin dashboard (separate feature)
2. **Failed Payment Recovery**: Guest must start new order (simple UX)
3. **Partial Inventory**: Reject order entirely, show clear error (prevents complexity)
4. **Geocoding Fallback**: Require valid geocoding, no manual override (data quality)
5. **Multi-Language**: Public menu uses existing i18n, product names in tenant's language

---

## Performance Considerations

### Indexes
- Composite indexes on (tenant_id, status) for order filtering
- Index on expires_at for reservation cleanup job
- Index on created_at DESC for recent orders

### Caching
- Redis for cart (avoid DB roundtrips)
- Redis for inventory availability (fast checks)
- Redis for geocoding results (reduce API calls)

### Query Optimization
- Use SELECT FOR UPDATE sparingly (only inventory checks)
- Paginate order lists in admin
- Denormalize order items (avoid joins)

### Background Jobs
- Reservation cleanup every 1 minute
- Cart cleanup every 1 hour
- Payment reconciliation every 15 minutes
