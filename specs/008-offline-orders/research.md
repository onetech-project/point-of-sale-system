# Phase 0: Research & Decisions

**Feature**: Offline Order Management  
**Date**: February 7, 2026  
**Status**: Complete

## Research Questions

Based on Technical Context analysis, all technology choices are clear (Go 1.24, Echo, PostgreSQL, etc.). Research focuses on **implementation patterns** for feature requirements.

### 1. Database Schema Design: Extend vs. New Table?

**Question**: Should offline orders use the existing `guest_orders` table or create a separate `offline_orders` table?

**Research**:

- ✅ **Extend existing table**: Add `order_type` ENUM ('online', 'offline') column
  - **Pros**: Unified order history per customer, simpler queries for analytics, reuses existing order repository code, follows DRY principle
  - **Cons**: Table grows larger (but manageable at projected scale), need to ensure RLS policies cover both types
- ❌ **Separate table**: Create new `offline_orders` table
  - **Pros**: Clear separation of concerns, independent schema evolution
  - **Cons**: Duplicate logic for order operations, complex joins for unified analytics, violates DRY, unnecessarily complex (YAGNI)

**Decision**: **Extend `guest_orders` table** with `order_type` column  
**Rationale**: Follows Constitution Principle VI (Simplicity First + DRY). Both order types share 90% of columns (customer data, items, status, amounts). Separate tables would require duplicate repositories, services, and analytics queries.

**Alternatives Considered**: Separatetable with view unifying both - rejected as over-engineering.

---

### 2. Payment Terms & Installments Data Model

**Question**: How to model payment schedules with down payments and installments?

**Research**:

- **Pattern 1**: Single `payment_terms` JSON column in orders table
  - **Pros**: Flexible schema, no joins
  - **Cons**: Can't query by payment status, difficult validation
- **Pattern 2**: Separate `payment_records` table + `payment_terms` summary
  - **Pros**: Queryable, auditable, supports partial payments, follows normalization
  - **Cons**: Additional table and joins

**Decision**: **Two-table approach** - `payment_terms` (1:1 with order) + `payment_records` (1:N with order)  
**Rationale**:

- `payment_terms`: Stores schedule definition (down payment %, installment count, due dates)
- `payment_records`: Tracks actual payments received with timestamps
- Enables queries like "show all overdue payments" or "total outstanding balance by tenant"
- Audit trail via payment_records timestamps

**Schema**:

```sql
-- Summary of payment agreement
payment_terms: {
  order_id, total_amount, down_payment_amount,
  installment_count, installment_amount, due_dates[]
}

-- Individual payment transactions
payment_records: {
  order_id, payment_number, amount_paid, payment_date,
  payment_method, remaining_balance, recorded_by_user_id
}
```

**Alternatives Considered**: Store everything in JSONB in orders table - rejected as unqueryable.

---

### 3. Role-Based Deletion Implementation

**Question**: How to enforce owner/manager-only deletion at service boundary?

**Research**:

- **Pattern 1**: Database-level CHECK constraint
  - **Pros**: Enforced at data layer
  - **Cons**: Can't access user role from DB, would need triggers
- **Pattern 2**: Middleware role check before handler execution
  - **Pros**: Clean separation, reusable middleware, fail-fast
  - **Cons**: Must ensure all delete routes use middleware
- **Pattern 3**: Service-layer role validation
  - **Pros**: Business logic in service layer
  - **Cons**: Easy to forget, not enforced architecturally

**Decision**: **Middleware + Service-layer double-check**  
**Rationale**:

- API Gateway already extracts JWT claims including `user_role`
- Echo middleware checks `user_role ∈ {owner, manager}` before routing to handler
- Service layer validates again (defense in depth)
- Returns 403 Forbidden if unauthorized

**Implementation**:

```go
// middleware/role_check.go
func RequireRole(allowedRoles ...string) echo.MiddlewareFunc {
  return func(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
      userRole := c.Get("user_role").(string)
      if !contains(allowedRoles, userRole) {
        return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
      }
      return next(c)
    }
  }
}

// Route definition
e.DELETE("/offline-orders/:id", handler.DeleteOfflineOrder,
  middleware.RequireRole("owner", "manager"))
```

**Alternatives Considered**: Database RLS policy - rejected because RLS is tenant-based, not role-based in current architecture.

---

### 4. Audit Trail Integration

**Question**: How to log all offline order operations (CREATE, READ, UPDATE, DELETE, ACCESS)?

**Research**:

- Existing `audit-service` logs events via Kafka topic: `audit-events`
- Event schema: `{user_id, tenant_id, action, resource_type, resource_id, changes, timestamp}`

**Decision**: **Reuse existing audit-service via Kafka events**  
**Rationale**:

- Already handles CREATE, UPDATE, DELETE actions for other entities
- READ/ACCESS actions: Enable optional access logging flag for sensitive operations (e.g., viewing order with PII)
- Change tracking: Log `before` and `after` state for UPDATE operations

**Event Examples**:

```json
// CREATE
{
  "action": "CREATE",
  "resource_type": "offline_order",
  "resource_id": "order-uuid",
  "user_id": "user-uuid",
  "tenant_id": "tenant-uuid",
  "changes": {"status": "pending_payment", "total": 150000},
  "timestamp": "2026-02-07T10:30:00Z"
}

// UPDATE
{
  "action": "UPDATE",
  "resource_type": "offline_order",
  "resource_id": "order-uuid",
  "user_id": "user-uuid",
  "changes": {
    "customer_phone": {"from": "+6281234567890", "to": "+6281234567891"},
    "items": {"added": [{"product": "SKU123", "qty": 2}]}
  },
  "timestamp": "2026-02-07T11:00:00Z"
}

// DELETE
{
  "action": "DELETE",
  "resource_type": "offline_order",
  "resource_id": "order-uuid",
  "user_id": "user-uuid",
  "metadata": {"reason": "Fraudulent transaction"},
  "timestamp": "2026-02-07T15:00:00Z"
}
```

**Implementation**: Service layer publishes Kafka event after each DB operation commits.

**Alternatives Considered**: Database triggers - rejected because audit-service already exists.

---

### 5. Analytics Integration

**Question**: How to include offline orders in existing analytics dashboards?

**Research**:

- `analytics-service` subscribes to Kafka topics: `order-events`, `payment-events`
- Current metrics: daily revenue, order count, avg order value, top products

**Decision**: **Publish offline order events to existing Kafka topics with `order_type` field**  
**Rationale**:

- Analytics service already aggregates `guest_orders` table data
- Adding `order_type='offline'` filter enables segmented reporting
- No changes needed to analytics service queries (they aggregate all orders)
- Dashboard filters can show "online only", "offline only", or "all orders"

**New Metrics** (to add in analytics service):

- Offline order conversion rate (orders with pending payments vs. completed)
- Outstanding offline order balance by tenant
- Average payment term length
- Offline vs. online revenue split

**Implementation**: Order service publishes same event schema, analytics service automatically includes offline orders in aggregations.

**Alternatives Considered**: Separate analytics pipeline - rejected as over-engineering.

---

### 6. Avoiding Online Order Flow Disruption (FR-008)

**Question**: How to ensure offline order feature doesn't degrade online order performance?

**Research**:

- Current online order flow: session cart (Redis) → checkout → Midtrans payment → order record (PostgreSQL)
- Performance bottlenecks: Database writes (order + order_items inserts), Midtrans API calls

**Decision**: **Architectural Isolation**  
**Strategies**:

1. **Separate endpoints**: `/api/v1/offline-orders/*` vs. `/api/v1/public/*` (online)
2. **Separate transactions**: Offline orders use distinct DB transactions (no locking conflicts)
3. **No shared locks**: Offline orders don't reserve inventory (staff manually verify stock)
4. **Index optimization**: Add composite index `(order_type, status, tenant_id)` for filtering queries
5. **Query isolation**: Online order queries add `WHERE order_type = 'online'` clause
6. **Monitoring**: Track p95 latency separately for online vs. offline endpoints

**Performance Testing Plan**:

- Load test: 1000 concurrent online orders should maintain <200ms p95
- Concurrent offline order writes must not increase online order latency >5%

**Alternatives Considered**: Separate database - rejected as premature optimization.

---

### 7. Compliance (UU PDP & GDPR)

**Question**: What additional compliance measures are needed for offline orders?

**Research**:

- Existing compliance: PII encrypted (customer name, phone, email, address), data retention policies, right to erasure
- Offline orders introduce: Walk-in customer data (may not have digital consent record)

**Decision**: **Reuse existing compliance framework + consent field**  
**Measures**:

1. **Encryption**: Use same Vault-backed deterministic encryption for searchable fields (phone, email), AES-GCM for names/addresses
2. **Consent tracking**: Add `data_consent_given` boolean + `consent_method` enum ('verbal', 'written', 'digital') to offline orders
3. **Data retention**: Apply same 7-year retention policy as online orders
4. **Right to erasure**: Support deletion requests via existing data subject request process
5. **Audit**: Log all PII access (who viewed customer details and when)

**Compliance Checklist**:

- ✅ PII encrypted at rest
- ✅ PII encrypted in transit (TLS)
- ✅ Audit trail for PII access
- ✅ Data retention policy
- ✅ Right to erasure implementation
- ✅ Minimal data collection (only what's needed for order fulfillment)

**Alternatives Considered**: Separate consent management service - rejected as over-engineering.

---

### 8. Payment Method Handling

**Question**: How to record payment methods for offline orders (cash, card, bank transfer)?

**Research**:

- Online orders: Midtrans handles payment method tracking
- Offline orders: Staff manually records payment type

**Decision**: **Simple enum in `payment_records` table**  
**Payment Methods**:

- `cash`: Physical cash payment
- `card`: Credit/debit card (manual terminal)
- `bank_transfer`: Manual bank transfer
- `check`: Paper check
- `other`: Other methods

**Schema**:

```sql
payment_records.payment_method ENUM('cash', 'card', 'bank_transfer', 'check', 'other')
```

**Rationale**: Simple, extensible, no external API dependencies. Staff selects method when recording payment.

**Alternatives Considered**: Integrate with POS terminal API - rejected as out of scope (YAGNI).

---

### 9. Order Status Transitions

**Question**: What status lifecycle should offline orders follow?

**Research**:

- Online orders: PENDING → PAID → COMPLETE → (CANCELLED)
- Offline orders: May skip PENDING if full payment upfront, or stay PENDING with installments

**Decision**: **Extend existing status enum**  
**Status Flow**:

1. **PENDING_PAYMENT**: Created with down payment or zero payment
2. **PAID**: All payments received (total paid = order total)
3. **COMPLETE**: Order fulfilled (items delivered/picked up)
4. **CANCELLED**: Order cancelled (soft delete alternative)

**Validation Rules**:

- Can only mark COMPLETE if status = PAID
- Can cancel at any time (requires reason)
- Cannot edit order items after status = PAID (only payments)

**Alternatives Considered**: Add new statuses like PARTIAL_PAYMENT - rejected to keep simple.

---

### 10. Technology Best Practices

#### Go Echo Framework (REST APIs)

- ✅ Use Echo v4 middleware pattern for auth/RBAC
- ✅ Structured JSON responses with error codes
- ✅ Request validation using validator.v10
- ✅ Context propagation for tracing

#### PostgreSQL (Data Storage)

- ✅ Use transactions for multi-table writes (order + items + payments)
- ✅ Add partial indexes for common query patterns: `WHERE order_type = 'offline' AND status = 'PENDING_PAYMENT'`
- ✅ Use `JSONB` for payment schedule flexibility (due dates array)
- ✅ Enable RLS policies for tenant isolation

#### Encryption (Vault)

- ✅ Deterministic encryption for searchable PII (phone, email) using existing `EncryptSearchable()`
- ✅ AES-GCM for non-searchable PII (names, addresses) using `Encrypt()`
- ✅ Batch encryption operations to reduce Vault API calls

#### Testing Strategy

- ✅ **Unit tests**: Service layer business logic (payment calculations, status transitions)
- ✅ **Integration tests**: Repository layer with test database
- ✅ **Contract tests**: API endpoint schemas match OpenAPI spec
- ✅ **E2E tests**: Full user journey (create → pay installments → complete)

---

## Summary of Decisions

| Topic                     | Decision                                              | Key Rationale                               |
| ------------------------- | ----------------------------------------------------- | ------------------------------------------- |
| **Schema**                | Extend `guest_orders` table with `order_type` column  | DRY, simpler analytics, unified history     |
| **Payment Modeling**      | `payment_terms` + `payment_records` tables            | Queryable, auditablepartial payments        |
| **Role Enforcement**      | Middleware + service-layer double-check               | Defense in depth, architectural enforcement |
| **Audit Trail**           | Reuse audit-service via Kafka events                  | Existing infrastructure, no duplication     |
| **Analytics**             | Publish to existing Kafka topics with `order_type`    | No service changes needed, auto-included    |
| **Performance Isolation** | Separate endpoints, no inventory locks, query filters | Zero impact to online orders                |
| **Compliance**            | Reuse encryption + add consent tracking               | Consistent with existing framework          |
| **Payment Methods**       | Simple enum in `payment_records`                      | No external dependencies, extensible        |
| **Status Lifecycle**      | Extend existing enum (PENDING_PAYMENT variant)        | Consistent with online orders               |

---

## Open Questions: None

All NEEDS CLARIFICATION items from Technical Context resolved. No blocking unknowns remain.
