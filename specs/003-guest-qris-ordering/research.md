# Phase 0: Research & Investigation

**Feature**: QRIS Guest Ordering System  
**Date**: 2025-12-03  
**Status**: Complete

## Research Tasks

### 1. Midtrans QRIS Integration

**Question**: How does Midtrans QRIS payment flow work and what are the webhook requirements?

**Decision**: Use Midtrans Snap API with QRIS payment method

**Rationale**:
- Midtrans provides official Go SDK (github.com/midtrans/midtrans-go)
- Snap API simplifies payment UI by handling QR code display
- Webhook notifications sent to configurable endpoint with signature verification
- Supports payment status: pending, settlement, cancel, deny, expire
- Server-side integration reduces client-side complexity and improves security

**Implementation Details**:
- Create Snap transaction on backend when order is created
- Redirect guest to Snap payment URL (or show QR in iframe)
- Implement `/payments/midtrans/notification` endpoint with POST handler
- Verify notification signature using SHA512 hash: `order_id + status_code + gross_amount + server_key`
- Parse notification JSON for: `order_id`, `transaction_status`, `fraud_status`, `gross_amount`
- Map transaction statuses:
  - `settlement` → Order status PAID
  - `pending` → Keep order pending
  - `cancel`, `deny`, `expire` → Release inventory reservation
- Store Midtrans transaction_id for reconciliation
- Handle idempotency by checking if order already processed

**Alternatives Considered**:
- Core API (direct charge): Rejected - requires more frontend logic and PCI compliance concerns
- Xendit or other gateway: Rejected - Midtrans specified in requirements

**References**:
- Midtrans Go SDK: https://github.com/midtrans/midtrans-go
- Notification handling: https://docs.midtrans.com/en/after-payment/http-notification
- Snap API: https://docs.midtrans.com/en/snap/overview

---

### 2. Google Maps Geocoding & Service Area Validation

**Question**: How to geocode addresses and validate service area boundaries?

**Decision**: Use Google Maps Geocoding API with polygon-based service area validation

**Rationale**:
- Google Maps Geocoding API provides reliable address-to-coordinate conversion
- Supports Indonesian addresses with high accuracy
- Returns structured address components for validation
- No need for custom geocoding implementation
- Service area validation via point-in-polygon algorithm for flexibility

**Implementation Details**:
- Integrate Google Maps Geocoding API (googlemaps package for Go)
- When guest enters delivery address, call Geocoding API
- Extract coordinates (lat, lng) from response
- Store geocoded address in `delivery_addresses` table
- Tenant configuration stores service area as:
  - **Option 1**: Radius from center point (simple circular area)
  - **Option 2**: Polygon coordinates (complex service areas)
- Validation algorithm:
  - Radius: Calculate distance using Haversine formula
  - Polygon: Use ray-casting algorithm for point-in-polygon test
- Cache geocoding results in Redis (key: address hash, TTL: 7 days)
- Handle API errors gracefully with user-friendly messages

**Distance Calculation (Haversine)**:
```
a = sin²(Δφ/2) + cos φ1 ⋅ cos φ2 ⋅ sin²(Δλ/2)
c = 2 ⋅ atan2(√a, √(1−a))
d = R ⋅ c
where R = 6371 km (Earth radius)
```

**Alternatives Considered**:
- Manual address entry without geocoding: Rejected - cannot validate service area
- OpenStreetMap Nominatim: Rejected - lower accuracy for Indonesian addresses
- Store address as text only: Rejected - cannot calculate delivery fees

**References**:
- Google Maps Geocoding API: https://developers.google.com/maps/documentation/geocoding
- Go googlemaps package: https://github.com/googlemaps/google-maps-services-go
- Haversine formula: https://en.wikipedia.org/wiki/Haversine_formula

---

### 3. Delivery Fee Calculation Strategy

**Question**: What pricing models should be supported for delivery fees?

**Decision**: Support both distance-based tiers and zone-based flat fees

**Rationale**:
- Different tenants have different business models
- Urban areas: Zone-based (flat fee per district)
- Suburban/rural: Distance-based (graduated pricing)
- Flexibility allows tenant configuration without code changes

**Implementation Details**:

**Distance-Based Pricing**:
```
Tenant configures tiers in JSON:
{
  "pricing_type": "distance",
  "tiers": [
    {"max_km": 3, "fee": 10000},
    {"max_km": 5, "fee": 15000},
    {"max_km": 10, "fee": 25000}
  ],
  "base_fee": 5000
}
```

**Zone-Based Pricing**:
```
Tenant configures zones with polygons:
{
  "pricing_type": "zone",
  "zones": [
    {
      "name": "Downtown",
      "fee": 12000,
      "polygon": [[lat1,lng1], [lat2,lng2], ...]
    },
    {
      "name": "Suburbs",
      "fee": 20000,
      "polygon": [[...]]
    }
  ]
}
```

**Calculation Flow**:
1. Geocode delivery address → get coordinates
2. Load tenant delivery fee config from database
3. If distance-based:
   - Calculate distance from tenant location to delivery address
   - Find matching tier, return fee
4. If zone-based:
   - Check which zone polygon contains the address
   - Return zone fee
5. Store calculated fee in order record

**Alternatives Considered**:
- Fixed flat fee: Rejected - not flexible enough
- Third-party logistics API: Rejected - manual courier handling specified
- Dynamic pricing (surge): Rejected - complexity not justified for MVP

---

### 4. Session & Cart Management Strategy

**Question**: How to handle guest sessions and cart persistence without accounts?

**Decision**: Use Redis for session storage with browser fingerprinting

**Rationale**:
- Guests don't have accounts, need alternative to user ID
- Redis provides fast key-value storage with TTL support
- Browser sessionStorage + backend session ID for cart persistence
- Allows cart recovery within session lifetime

**Implementation Details**:

**Session Flow**:
1. Guest visits public menu → Frontend generates UUID session_id
2. Store session_id in browser sessionStorage
3. Send session_id with all cart API requests in header: `X-Session-ID`
4. Backend creates Redis key: `cart:{tenant_id}:{session_id}`
5. Store cart items as JSON: `{"items": [{"product_id": "...", "quantity": 2}]}`
6. Set TTL: 24 hours (configurable)
7. On cart update, extend TTL

**Cart Operations**:
- Add item: HSET cart:{tenant}:{session} items {json}
- Get cart: HGET cart:{tenant}:{session} items
- Remove item: Update JSON, HSET
- Clear cart: DEL cart:{tenant}:{session}

**Session Cleanup**:
- Background job runs every hour
- Scans for expired cart sessions (TTL = 0)
- Removes associated inventory reservations

**Alternatives Considered**:
- Browser localStorage only: Rejected - cannot validate inventory server-side
- Cookies: Rejected - size limits for large carts
- Database sessions: Rejected - slower than Redis, unnecessary persistence

---

### 5. Inventory Reservation with TTL

**Question**: How to prevent overselling with temporary reservations?

**Decision**: Use PostgreSQL with background job for cleanup + Redis for fast checks

**Rationale**:
- PostgreSQL ensures ACID compliance for inventory
- Redis cache for fast "available inventory" checks
- Background job releases expired reservations
- Prevents race conditions with database-level locks

**Implementation Details**:

**Reservation Flow**:
1. Guest clicks "Proceed to Payment"
2. Backend creates pending order
3. For each order item:
   - Check product inventory: `SELECT quantity FROM products WHERE id = ? FOR UPDATE`
   - Calculate available: `quantity - (active_reservations + permanent_allocations)`
   - If insufficient: Rollback, return error
   - Create reservation: `INSERT INTO inventory_reservations (order_id, product_id, quantity, expires_at)`
   - Set expires_at = NOW() + 15 minutes
4. Commit transaction
5. Update Redis cache: DECR available:{product_id} by quantity

**Reservation Release**:
- Background job runs every 1 minute
- Query: `SELECT * FROM inventory_reservations WHERE status = 'active' AND expires_at < NOW()`
- For each expired reservation:
  - Update status to 'expired'
  - Update Redis: INCR available:{product_id} by quantity
  - Log expiration event

**On Payment Success**:
- Update reservation status to 'converted'
- Decrement product quantity permanently
- No Redis change (already decremented)

**On Payment Failure**:
- Update reservation status to 'released'
- Update Redis: INCR available:{product_id} by quantity

**Race Condition Prevention**:
- Use `SELECT FOR UPDATE` for row-level locks
- Redis operations as cache, PostgreSQL as source of truth
- If Redis fails, fall back to PostgreSQL query

**Alternatives Considered**:
- Redis-only reservations: Rejected - not ACID compliant, data loss risk
- Optimistic locking: Rejected - high contention could cause failures
- No reservations: Rejected - overselling risk unacceptable

---

### 6. Order Status State Machine

**Question**: What are the valid order states and transitions?

**Decision**: Four-state model with restricted transitions

**States**:
1. **PENDING**: Order created, payment not completed
2. **PAID**: Payment confirmed by Midtrans
3. **COMPLETE**: Delivery finished, marked by staff
4. **CANCELLED**: Order cancelled (by system or staff)

**Valid Transitions**:
```
PENDING → PAID (payment notification)
PENDING → CANCELLED (payment timeout/failure, staff action)
PAID → COMPLETE (staff action in admin)
PAID → CANCELLED (staff action for refunds)
```

**Invalid Transitions** (should error):
- COMPLETE → any other state
- CANCELLED → any other state
- PENDING → COMPLETE (must go through PAID)

**Implementation**:
- Store current status in orders table
- Validate transitions in order_service.go
- Log all status changes with timestamp and actor (system/staff)
- Emit events for status changes (future: notify customer)

**Rationale**:
- Simple model matches business flow
- Prevents invalid states (e.g., completing unpaid order)
- Clear audit trail for financial reconciliation

---

### 7. Multi-Tenant Data Isolation

**Question**: How to ensure tenant data isolation in public endpoints?

**Decision**: Extract tenant from URL path + middleware validation

**Rationale**:
- Public menu URL format: `/menu/{tenant_id}/products`
- Middleware validates tenant exists and is active
- All queries filtered by tenant_id
- Prevents cross-tenant data leaks

**Implementation Details**:

**URL Structure**:
- Public menu: `GET /public/menu/:tenant_id/products`
- Cart: `POST /public/cart/:tenant_id/items`
- Checkout: `POST /public/orders/:tenant_id/checkout`
- Payment webhook: `POST /payments/midtrans/notification` (tenant from order lookup)

**Middleware**:
```go
func TenantScopeMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        tenantID := c.Param("tenant_id")
        // Validate tenant exists and is active
        tenant := tenantRepo.FindByID(tenantID)
        if tenant == nil || !tenant.Active {
            return echo.ErrNotFound
        }
        c.Set("tenant_id", tenantID)
        return next(c)
    }
}
```

**Query Pattern**:
```sql
SELECT * FROM products 
WHERE tenant_id = $1 AND active = true
```

**Alternatives Considered**:
- Subdomain per tenant: Rejected - DNS/SSL complexity
- Header-based tenant: Rejected - harder for public URLs
- Database per tenant: Rejected - operational complexity

---

## Technology Stack Summary

| Component | Technology | Justification |
|-----------|-----------|---------------|
| Payment Gateway | Midtrans Snap API | Business requirement, official Go SDK |
| Geocoding | Google Maps API | Best accuracy for Indonesian addresses |
| Session Store | Redis | Fast TTL-based expiration, existing infrastructure |
| Database | PostgreSQL | ACID for inventory, existing in project |
| Backend Framework | Echo v4 | Existing project standard |
| Frontend Framework | Next.js 16 | Existing project standard |
| Delivery Fee Logic | Custom calculation | Flexible tenant configuration |
| Inventory Locks | PostgreSQL SELECT FOR UPDATE | Prevents race conditions |
| Background Jobs | Go goroutines + time.Ticker | Simple for TTL cleanup |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Midtrans webhook failures | Medium | High | Implement retry logic, manual reconciliation UI |
| Geocoding API quota exceeded | Low | Medium | Cache results, implement fallback to manual validation |
| Inventory race conditions | Low | High | Database locks + comprehensive testing |
| Session hijacking | Low | Medium | Secure session IDs, rate limiting |
| Payment notification replay attacks | Medium | High | Signature verification + idempotency checks |
| TTL cleanup job failures | Low | Medium | Monitoring + alerts, manual cleanup procedure |

---

## Open Questions for Phase 1

1. **Tenant Configuration UI**: Where/how do tenants configure delivery types, service areas, and pricing? (Admin dashboard scope)
2. **Failed Payment Recovery**: Should system automatically retry failed payments or require guest to start over?
3. **Partial Inventory**: If guest orders 5 but only 3 available, allow partial order or reject entirely?
4. **Geocoding Fallback**: If Google Maps fails, should system allow manual lat/lng entry or reject delivery order?
5. **Multi-Language**: Should public menu support i18n? (Existing frontend has i18n infrastructure)

**Resolution Strategy**: Document assumptions in data-model.md, flag for stakeholder review before implementation.
