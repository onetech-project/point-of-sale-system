# Quickstart Guide: QRIS Guest Ordering System

**Feature**: Guest ordering with Midtrans QRIS payment  
**Target Audience**: Developers implementing this feature  
**Last Updated**: 2025-12-03

## Overview

This feature enables guests to:
1. Browse a tenant's public menu without authentication
2. Add items to a session-based cart
3. Select delivery type (pickup/delivery/dine-in)
4. Complete payment via Midtrans QRIS
5. Track order status

Tenant staff can:
1. View all orders in admin dashboard
2. Update order status to COMPLETE when fulfilled
3. Add notes for courier tracking

## Prerequisites

### Development Environment
- Go 1.23.0 or higher
- Node.js 18+ (for frontend)
- PostgreSQL 14+
- Redis 6+
- Docker & Docker Compose

### External Services
1. **Midtrans Account**
   - Sign up at https://midtrans.com
   - Get Server Key and Client Key from dashboard
   - Configure webhook URL: `https://your-domain.com/payments/midtrans/notification`

2. **Google Maps API**
   - Create project in Google Cloud Console
   - Enable Geocoding API
   - Create API key with IP restrictions

### Environment Variables

Create `.env` files for services:

**backend/order-service/.env**:
```bash
# Database
DATABASE_URL=postgresql://user:password@localhost:5432/pos_db?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379/0

# Midtrans
MIDTRANS_SERVER_KEY=your_server_key_here
MIDTRANS_CLIENT_KEY=your_client_key_here
MIDTRANS_IS_PRODUCTION=false

# Google Maps
GOOGLE_MAPS_API_KEY=your_google_maps_api_key

# Service
PORT=8084
CORS_ALLOWED_ORIGINS=http://localhost:3000

# Sessions
SESSION_TTL_HOURS=24
CART_TTL_HOURS=24
INVENTORY_RESERVATION_TTL_MINUTES=15

# Logging
LOG_LEVEL=debug
```

## Quick Setup (5 minutes)

### 1. Database Migration

```bash
cd backend
# Run migrations
psql $DATABASE_URL -f migrations/000020_create_orders.up.sql
psql $DATABASE_URL -f migrations/000021_create_order_items.up.sql
psql $DATABASE_URL -f migrations/000022_create_inventory_reservations.up.sql
psql $DATABASE_URL -f migrations/000023_create_payment_transactions.up.sql
psql $DATABASE_URL -f migrations/000024_create_delivery_addresses.up.sql
```

### 2. Start Order Service

```bash
cd backend/order-service
go mod download
go run main.go
```

Service should start on port 8084.

### 3. Start Frontend (Development)

```bash
cd frontend
npm install
npm run dev
```

Frontend should start on port 3000.

### 4. Test Public Menu

Visit: `http://localhost:3000/menu/{tenant_id}`

Replace `{tenant_id}` with a valid tenant UUID from your database.

## Core Workflows

### A. Guest Ordering Flow

```mermaid
sequenceDiagram
    Guest->>Frontend: Visit /menu/{tenant_id}
    Frontend->>Order Service: GET /public/menu/{tenant_id}/products
    Order Service->>Frontend: Return products
    
    Guest->>Frontend: Add item to cart
    Frontend->>Order Service: POST /public/cart/{tenant_id}/items
    Order Service->>Redis: Store cart
    Order Service->>Frontend: Return cart
    
    Guest->>Frontend: Proceed to checkout
    Frontend->>Order Service: POST /public/orders/{tenant_id}/checkout
    Order Service->>DB: Create order + reservations
    Order Service->>Midtrans: Create Snap transaction
    Midtrans->>Order Service: Return payment URL
    Order Service->>Frontend: Return checkout response
    
    Frontend->>Guest: Redirect to Midtrans
    Guest->>Midtrans: Scan QR & pay
    Midtrans->>Order Service: POST /payments/midtrans/notification
    Order Service->>DB: Update order status to PAID
```

### B. Payment Webhook Flow

```bash
# Midtrans sends notification to your webhook
POST /payments/midtrans/notification
{
  "transaction_status": "settlement",
  "order_id": "GO-ABC123",
  "gross_amount": "150000.00",
  "signature_key": "..."
}

# Your service:
1. Verifies signature: SHA512(order_id + status_code + amount + server_key)
2. Checks idempotency (duplicate notifications)
3. Validates amount matches order
4. Updates order status to PAID
5. Converts inventory reservations to permanent
6. Returns 200 OK
```

### C. Admin Order Management

```bash
# List orders
GET /admin/orders?status=PAID
Authorization: Bearer {jwt_token}

# Update order status
PATCH /admin/orders/{order_id}/status
{
  "status": "COMPLETE",
  "notes": "Delivered by GrabExpress"
}
```

## API Quick Reference

### Public Endpoints (No Auth)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/public/menu/:tenant_id/products` | Get product catalog |
| GET | `/public/cart/:tenant_id` | Get cart contents |
| POST | `/public/cart/:tenant_id/items` | Add item to cart |
| PATCH | `/public/cart/:tenant_id/items/:product_id` | Update cart item |
| DELETE | `/public/cart/:tenant_id/items/:product_id` | Remove from cart |
| POST | `/public/orders/:tenant_id/checkout` | Create order & payment |
| GET | `/public/orders/:tenant_id/:order_ref` | Get order status |

### Admin Endpoints (Auth Required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/orders` | List orders with filters |
| GET | `/admin/orders/:order_id` | Get order details |
| PATCH | `/admin/orders/:order_id/status` | Update order status |
| POST | `/admin/orders/:order_id/notes` | Add staff note |

### Webhook Endpoint

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/payments/midtrans/notification` | Midtrans payment webhook |

## Testing Guide

### 1. Unit Tests

```bash
cd backend/order-service
go test ./src/services/... -v
go test ./src/repository/... -v
```

### 2. Integration Tests

```bash
# Start test dependencies
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test ./tests/integration/... -v

# Cleanup
docker-compose -f docker-compose.test.yml down
```

### 3. Contract Tests (Midtrans Webhook)

```bash
# Uses recorded webhook payloads
go test ./tests/contract/midtrans_webhook_test.go -v
```

### 4. Manual Testing with Midtrans Sandbox

1. Set `MIDTRANS_IS_PRODUCTION=false` in .env
2. Use Midtrans sandbox credentials
3. Complete checkout flow
4. Use test QRIS payments: https://docs.midtrans.com/en/technical-reference/sandbox-test
5. Check webhook logs for notifications

## Common Tasks

### Add New Delivery Fee Zone

```sql
-- Update tenant config
UPDATE tenant_configs
SET delivery_fee_config = '{
  "pricing_type": "zone",
  "zones": [
    {
      "name": "Downtown",
      "fee": 12000,
      "polygon": [[lat1,lng1], [lat2,lng2], ...]
    }
  ]
}'::jsonb
WHERE tenant_id = 'your-tenant-id';
```

### Configure Service Area

```sql
-- Radius-based
UPDATE tenant_configs
SET service_area_type = 'radius',
    service_area_data = '{
      "center": {"lat": -6.200000, "lng": 106.816666},
      "radius_km": 10
    }'::jsonb
WHERE tenant_id = 'your-tenant-id';
```

### Manual Order Status Update

```sql
-- Mark order as complete
UPDATE guest_orders
SET status = 'COMPLETE',
    completed_at = NOW()
WHERE order_reference = 'GO-ABC123';
```

### Release Expired Reservations (Manual)

```sql
-- Find expired reservations
SELECT * FROM inventory_reservations
WHERE status = 'active' AND expires_at < NOW();

-- Release them
UPDATE inventory_reservations
SET status = 'expired', released_at = NOW()
WHERE status = 'active' AND expires_at < NOW();
```

## Troubleshooting

### Problem: Cart items disappear

**Cause**: Redis eviction or TTL expired

**Solution**:
1. Check Redis memory: `redis-cli INFO memory`
2. Verify TTL: `redis-cli TTL cart:{tenant}:{session}`
3. Increase cart TTL in config
4. Configure Redis eviction policy: `maxmemory-policy allkeys-lru`

### Problem: Payment notification not received

**Cause**: Webhook URL not accessible or signature verification failing

**Solution**:
1. Check Midtrans dashboard webhook settings
2. Use ngrok for local testing: `ngrok http 8084`
3. Verify signature calculation matches Midtrans spec
4. Check logs for signature verification errors
5. Test with Midtrans notification tester

### Problem: Inventory overselling

**Cause**: Race condition or reservation not created

**Solution**:
1. Verify `SELECT FOR UPDATE` is used
2. Check reservation creation logs
3. Verify Redis inventory cache is in sync
4. Run inventory reconciliation:
   ```sql
   SELECT p.id, p.name, p.quantity,
          COUNT(r.id) as active_reservations
   FROM products p
   LEFT JOIN inventory_reservations r 
     ON p.id = r.product_id AND r.status = 'active'
   GROUP BY p.id
   HAVING p.quantity < COUNT(r.id);
   ```

### Problem: Geocoding fails

**Cause**: Invalid API key or quota exceeded

**Solution**:
1. Verify API key in Google Cloud Console
2. Check Geocoding API is enabled
3. Monitor quota usage
4. Implement caching (already done in Redis)
5. Add fallback for manual lat/lng entry

### Problem: Order stuck in PENDING

**Cause**: Payment notification lost or webhook failure

**Solution**:
1. Check Midtrans transaction status manually
2. Query Midtrans API: 
   ```bash
   curl -u "SERVER_KEY:" \
     https://api.sandbox.midtrans.com/v2/{order_id}/status
   ```
3. Manual reconciliation if needed:
   ```sql
   UPDATE guest_orders
   SET status = 'PAID', paid_at = NOW()
   WHERE order_reference = 'GO-ABC123';
   ```

## Performance Optimization

### Database Indexes

Already included in migrations:
- `idx_tenant_status` on orders table
- `idx_order_reference` for quick lookups
- `idx_expires_at` for reservation cleanup

### Redis Caching

- Cart data: 24 hour TTL
- Inventory availability: No TTL, invalidate on changes
- Geocoding results: 7 day TTL

### Rate Limiting

Apply to public endpoints:
```go
// In middleware
limiter := rate.NewLimiter(rate.Limit(10), 20) // 10 req/s, burst 20
```

## Monitoring & Alerts

### Key Metrics to Track

1. **Order funnel**:
   - Cart creation rate
   - Checkout initiation rate
   - Payment completion rate

2. **Payment processing**:
   - Webhook processing time (target: <3s)
   - Signature verification failures
   - Duplicate notification rate

3. **Inventory**:
   - Reservation expiration rate
   - Overselling incidents (should be 0)

4. **Geocoding**:
   - API call success rate
   - Cache hit rate
   - Average response time

### Logging

Structure logs with:
- `order_reference`: For tracking specific orders
- `tenant_id`: For multi-tenant filtering
- `session_id`: For debugging cart issues
- `transaction_id`: For payment reconciliation

Example:
```go
log.WithFields(log.Fields{
    "order_reference": "GO-ABC123",
    "tenant_id": "uuid",
    "event": "payment_received",
    "amount": 150000,
}).Info("Payment notification processed")
```

## Next Steps

1. **Implement background jobs**:
   - Reservation cleanup (every 1 minute)
   - Cart cleanup (every 1 hour)
   - Payment reconciliation (every 15 minutes)

2. **Add monitoring**:
   - Prometheus metrics
   - Grafana dashboards
   - Alert rules for critical events

3. **Enhance admin UI**:
   - Order search and filtering
   - Export to CSV
   - Bulk status updates

4. **Customer notifications**:
   - Email/SMS on order status changes
   - WhatsApp integration

5. **Analytics**:
   - Order completion rates
   - Popular products
   - Peak ordering times
   - Average order value by delivery type

## Resources

- [Midtrans Documentation](https://docs.midtrans.com/)
- [Google Maps Geocoding API](https://developers.google.com/maps/documentation/geocoding)
- [API Contracts](./contracts/)
- [Data Model](./data-model.md)
- [Research & Decisions](./research.md)
