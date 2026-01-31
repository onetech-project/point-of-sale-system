# Quick Start Guide: Business Insights Dashboard

**Feature**: Business Insights Dashboard  
**Date**: 2026-01-31  
**Purpose**: Setup, testing, and operational guide for the analytics dashboard

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Environment Setup](#environment-setup)
3. [Database Setup](#database-setup)
4. [Service Startup](#service-startup)
5. [Testing the Dashboard](#testing-the-dashboard)
6. [Common Workflows](#common-workflows)
7. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Software

- Go 1.24.0+
- Node.js 18+ and npm
- PostgreSQL 14+
- Redis 7+
- Docker and Docker Compose (optional but recommended)

### Existing Services Running

- `auth-service` (port 8082)
- `tenant-service` (port 8084)
- `order-service` (port 8085)
- `product-service` (port 8086)
- `api-gateway` (port 8080)
- `frontend` (port 3000)

---

## Environment Setup

### Analytics Service Configuration

Create `.env` file in `backend/analytics-service/`:

```env
# Service Configuration
PORT=8087
SERVICE_NAME=analytics-service

# Database
DATABASE_URL=postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=300s

# Redis Cache
REDIS_URL=redis://localhost:6379/0
REDIS_PASSWORD=
REDIS_DB=0
REDIS_MAX_RETRIES=3
REDIS_POOL_SIZE=10

# JWT Authentication
JWT_SECRET=your-jwt-secret-must-match-auth-service

# Cache TTLs (seconds)
CACHE_TTL_CURRENT_MONTH=300      # 5 minutes
CACHE_TTL_HISTORICAL=3600        # 1 hour
CACHE_TTL_TASKS=60               # 1 minute

# Query Timeouts
QUERY_TIMEOUT_SECONDS=5

# Observability
LOG_LEVEL=info
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
```

### Frontend Dependencies

Add Recharts to frontend:

```bash
cd frontend
npm install recharts
npm install --save-dev @types/recharts
```

---

## Database Setup

### 1. Create Indexes for Analytics Queries

Run these SQL statements on your PostgreSQL database:

```sql
-- Connect to database
psql "postgresql://pos_user:pos_password@localhost:5432/pos_db"

-- Index for order analytics (sales, trends)
CREATE INDEX IF NOT EXISTS idx_orders_analytics 
ON orders(tenant_id, status, created_at DESC) 
WHERE status IN ('completed', 'paid');

-- Index for customer analytics
CREATE INDEX IF NOT EXISTS idx_orders_customer_analytics
ON orders(tenant_id, customer_phone, created_at DESC)
WHERE status IN ('completed', 'paid');

-- Covering index with included columns
CREATE INDEX IF NOT EXISTS idx_orders_analytics_covering
ON orders(tenant_id, status, created_at DESC)
INCLUDE (total_amount, customer_phone, customer_name);

-- Index for delayed orders
CREATE INDEX IF NOT EXISTS idx_orders_delayed
ON orders(tenant_id, status, created_at)
WHERE status IN ('pending', 'processing');

-- Index for product stock alerts
CREATE INDEX IF NOT EXISTS idx_products_stock_alerts
ON products(tenant_id, status, current_quantity, low_stock_threshold)
WHERE status = 'active';

-- Index for product analytics
CREATE INDEX IF NOT EXISTS idx_order_items_analytics
ON order_items(order_id, product_id, quantity, subtotal);

-- Verify indexes created
\d+ orders
\d+ products
\d+ order_items
```

### 2. Create Dashboard Settings Table (Optional)

```sql
-- Tenant-specific dashboard configuration
CREATE TABLE IF NOT EXISTS tenant_dashboard_settings (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    low_stock_default_threshold INTEGER DEFAULT 10 CHECK (low_stock_default_threshold >= 0),
    delayed_order_threshold_minutes INTEGER DEFAULT 15 CHECK (delayed_order_threshold_minutes > 0),
    default_date_range VARCHAR(20) DEFAULT 'current-month',
    default_granularity VARCHAR(20) DEFAULT 'daily',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dashboard_settings_tenant ON tenant_dashboard_settings(tenant_id);
```

### 3. Seed Test Data (Development Only)

```sql
-- Insert test orders for analytics
INSERT INTO orders (id, tenant_id, order_number, customer_phone, customer_name, status, total_amount, created_at)
SELECT 
    gen_random_uuid(),
    '< your-tenant-id>'::uuid,
    'ORD-' || generate_series,
    '+1234567' || LPAD(generate_series::text, 4, '0'),
    'Customer ' || generate_series,
    CASE WHEN random() < 0.9 THEN 'completed' ELSE 'pending' END,
    (random() * 500 + 50)::decimal(10,2),
    NOW() - (random() * interval '30 days')
FROM generate_series(1, 100);

-- Insert test products
INSERT INTO products (id, tenant_id, name, current_quantity, low_stock_threshold, cost, price, status)
SELECT 
    gen_random_uuid(),
    '<your-tenant-id>'::uuid,
    'Product ' || generate_series,
    (random() * 50)::integer,
    10,
    (random() * 20 + 5)::decimal(10,2),
    (random() * 40 + 10)::decimal(10,2),
    'active'
FROM generate_series(1, 50);
```

---

## Service Startup

### Option 1: Docker Compose (Recommended)

Update `docker-compose.yml`:

```yaml
services:
  # ... existing services ...
  
  analytics-service:
    build: ./backend/analytics-service
    container_name: analytics-service
    environment:
      PORT: 8087
      DATABASE_URL: postgresql://pos_user:pos_password@postgres:5432/pos_db?sslmode=disable
      REDIS_URL: redis://redis:6379/0
      JWT_SECRET: ${JWT_SECRET}
      LOG_LEVEL: info
    ports:
      - "8087:8087"
    depends_on:
      - postgres
      - redis
    networks:
      - pos-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8087/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

Start services:

```bash
docker-compose up -d analytics-service
docker-compose logs -f analytics-service
```

### Option 2: Local Development

#### Start Analytics Service

```bash
cd backend/analytics-service

# Install dependencies
go mod download

# Build service
go build -o analytics-service

# Run service
./analytics-service

# Or run directly
go run main.go
```

#### Start Frontend with Dashboard

```bash
cd frontend

# Install dependencies (first time only)
npm install

# Start development server
npm run dev
```

---

## Testing the Dashboard

### 1. Verify Service Health

```bash
# Check analytics service health
curl http://localhost:8087/health

# Expected response:
# {"status":"ok","service":"analytics-service"}
```

### 2. Get Authentication Token

```bash
# Login to get JWT token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "owner@tenant.com",
    "password": "password123"
  }'

# Save the token from response
export TOKEN="your-jwt-token-here"
```

### 3. Test Analytics Endpoints

#### Get Overview Metrics

```bash
curl http://localhost:8087/api/v1/analytics/overview?period=current-month \
  -H "Authorization: Bearer $TOKEN"

# Expected response:
# {
#   "status": "success",
#   "data": {
#     "period": {"start": "2026-01-01", "end": "2026-01-31"},
#     "metrics": {
#       "total_sales": 125000.50,
#       "order_count": 450,
#       "net_profit": 35000.25,
#       "inventory_value": 48500.00,
#       "avg_order_value": 277.78
#     }
#   },
#   "cache_hit": false,
#   "query_time_ms": 45
# }
```

#### Get Sales Trend

```bash
curl "http://localhost:8087/api/v1/analytics/sales-trend?granularity=daily&start=2026-01-01&end=2026-01-31" \
  -H "Authorization: Bearer $TOKEN"
```

#### Get Top Products

```bash
curl "http://localhost:8087/api/v1/analytics/top-products?metric=sales&type=top&limit=5&period=current-month" \
  -H "Authorization: Bearer $TOKEN"
```

#### Get Top Customers

```bash
curl "http://localhost:8087/api/v1/analytics/top-customers?limit=5&period=current-month" \
  -H "Authorization: Bearer $TOKEN"
```

#### Get Task Alerts

```bash
curl "http://localhost:8087/api/v1/analytics/tasks?types=delayed-orders,low-stock" \
  -H "Authorization: Bearer $TOKEN"
```

### 4. Test Frontend Dashboard

1. Open browser to `http://localhost:3000`
2. Login as tenant owner
3. Navigate to `/dashboard`
4. Verify:
   - Metric cards display correctly
   - Charts render with data
   - Time series filter works
   - Top products/customers tables populated
   - Task alerts show if applicable
   - Quick actions are clickable

---

## Common Workflows

### Workflow 1: View Current Month Performance

1. **Load Dashboard**: Navigate to `/dashboard`
2. **Default View**: Current month metrics displayed automatically
3. **Review Metrics**: Check total sales, profit, orders
4. **Analyze Charts**: Review sales trend line chart
5. **Check Rankings**: View top 5 products and customers

### Workflow 2: Compare Monthly Performance

1. **Select Time Series**: Choose "monthly" from dropdown
2. **Set Date Range**: Select last 3 months
3. **View Comparison**: Chart shows month-over-month trends
4. **Identify Patterns**: Spot growth or decline trends
5. **Export Insights**: (Future: export to CSV/PDF)

### Workflow 3: Address Delayed Orders

1. **View Task Alerts**: Check delayed orders count badge
2. **Click Alert**: Opens list of delayed orders
3. **Review Details**: See order number, customer, elapsed time
4. **Take Action**: Click order to view details
5. **Process Order**: Complete order from order management

### Workflow 4: Restock Low Inventory

1. **View Task Alerts**: Check low stock count badge
2. **Click Alert**: Opens list of products needing restock
3. **Review List**: Sorted by priority (out of stock first)
4. **Create Purchase Order**: (Integration with inventory system)
5. **Update Stock**: Mark as restocked when received

---

## Troubleshooting

### Issue: Service Won't Start

**Symptoms**: Analytics service fails to start, error in logs

**Solutions**:

```bash
# Check if port 8087 is already in use
lsof -i :8087

# Kill process on port if needed
kill -9 $(lsof -ti:8087)

# Check database connection
psql "postgresql://pos_user:pos_password@localhost:5432/pos_db" -c "SELECT 1"

# Check Redis connection
redis-cli -h localhost -p 6379 ping
# Should return: PONG

# Verify environment variables
cd backend/analytics-service
env | grep -E "DATABASE_URL|REDIS_URL|JWT_SECRET"
```

### Issue: Slow Query Performance

**Symptoms**: Dashboard takes > 2 seconds to load, timeout errors

**Solutions**:

```sql
-- Check if indexes exist
SELECT schemaname, tablename, indexname 
FROM pg_indexes 
WHERE tablename IN ('orders', 'products', 'order_items')
AND indexname LIKE 'idx%analytics%';

-- Analyze query performance
EXPLAIN ANALYZE
SELECT COUNT(*), SUM(total_amount)
FROM orders
WHERE tenant_id = '<tenant-id>'
AND status = 'completed'
AND created_at >= '2026-01-01'
AND created_at < '2026-02-01';

-- Check for missing indexes
-- Should use idx_orders_analytics index
-- If using Seq Scan, indexes might be missing

-- Rebuild statistics
ANALYZE orders;
ANALYZE products;
ANALYZE order_items;
```

### Issue: Cache Not Working

**Symptoms**: Every request shows `cache_hit: false`

**Solutions**:

```bash
# Check Redis connection
redis-cli -h localhost -p 6379

# Test Redis operations
SET test "value"
GET test
DEL test

# Check Redis keys
KEYS analytics:*

# Monitor Redis operations in real-time
redis-cli MONITOR

# Check TTL of cached keys
TTL analytics:<tenant-id>:sales-overview:current-month

# Clear analytics cache for debugging
redis-cli KEYS "analytics:*" | xargs redis-cli DEL
```

### Issue: Authentication Errors

**Symptoms**: 401 Unauthorized responses

**Solutions**:

```bash
# Verify JWT secret matches across services
grep JWT_SECRET backend/auth-service/.env
grep JWT_SECRET backend/analytics-service/.env
# Must be identical

# Test JWT token validity
curl http://localhost:8082/api/v1/auth/verify \
  -H "Authorization: Bearer $TOKEN"

# Check JWT expiration
# Decode JWT at jwt.io
echo $TOKEN | cut -d. -f2 | base64 -d | jq .

# Get fresh token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"owner@test.com","password":"password"}'
```

### Issue: No Data Displayed

**Symptoms**: Dashboard loads but shows zero for all metrics

**Solutions**:

```sql
-- Check if orders exist for tenant
SELECT COUNT(*) 
FROM orders 
WHERE tenant_id = '<tenant-id>'
AND status = 'completed';

-- Check date range
SELECT MIN(created_at), MAX(created_at)
FROM orders
WHERE tenant_id = '<tenant-id>'
AND status = 'completed';

-- Verify tenant_id in JWT matches database
-- Check tenant isolation is working correctly
SELECT tenant_id, COUNT(*)
FROM orders
GROUP BY tenant_id;
```

### Issue: Charts Not Rendering

**Symptoms**: Metrics show but charts are blank

**Solutions**:

```bash
# Check browser console for errors
# Open DevTools (F12) → Console tab

# Verify Recharts installed
cd frontend
npm list recharts

# Reinstall if missing
npm install recharts

# Check API response format
curl http://localhost:8087/api/v1/analytics/sales-trend?granularity=daily&start=2026-01-01&end=2026-01-31 \
  -H "Authorization: Bearer $TOKEN" | jq .

# Verify time_series array exists and has data

# Clear browser cache
# Hard refresh: Ctrl+Shift+R (Windows/Linux) or Cmd+Shift+R (Mac)
```

---

## Performance Optimization

### Database Query Optimization

```sql
-- Check slow queries (requires pg_stat_statements extension)
SELECT 
    calls,
    mean_exec_time,
    max_exec_time,
    query
FROM pg_stat_statements
WHERE query LIKE '%orders%'
ORDER BY mean_exec_time DESC
LIMIT 10;

-- Add additional covering index if needed
CREATE INDEX idx_orders_analytics_extended
ON orders(tenant_id, status, created_at DESC)
INCLUDE (total_amount, customer_phone, customer_name, total_cost);
```

### Redis Cache Monitoring

```bash
# Monitor cache hit rate
redis-cli INFO stats | grep keyspace_hits
redis-cli INFO stats | grep keyspace_misses

# Calculate hit rate = hits / (hits + misses)

# Check memory usage
redis-cli INFO memory | grep used_memory_human

# View most accessed keys
redis-cli --bigkeys
```

### Application Monitoring

```bash
# Check service metrics (Prometheus)
curl http://localhost:8087/metrics

# Key metrics to monitor:
# - analytics_query_duration_seconds (histogram)
# - analytics_cache_hit_total (counter)
# - analytics_cache_miss_total (counter)
# - analytics_errors_total (counter)
```

---

## Testing Checklist

### Unit Tests

```bash
cd backend/analytics-service
go test ./src/services/... -v
go test ./src/repository/... -v
```

### Integration Tests

```bash
# Requires test database
export DATABASE_URL="postgresql://pos_user:pos_password@localhost:5432/pos_test_db"
go test ./tests/integration/... -v
```

### Contract Tests

```bash
# Validate OpenAPI spec
cd specs/007-business-insights-dashboard/contracts
npx @apidevtools/swagger-cli validate analytics-api.yaml

# Run contract tests
cd backend/analytics-service
go test ./tests/contract/... -v
```

### Frontend Tests

```bash
cd frontend
npm test -- dashboard
```

---

## Production Deployment

### Pre-Deployment Checklist

- [ ] All tests passing (unit, integration, contract)
- [ ] Database indexes created on production
- [ ] Environment variables configured
- [ ] SSL/TLS certificates installed
- [ ] Rate limiting configured
- [ ] Monitoring dashboards setup
- [ ] Backup strategy verified
- [ ] Rollback plan documented

### Health Check Endpoints

```bash
# Service health
GET /health

# Readiness check (includes DB and Redis)
GET /ready
```

### Monitoring Alerts

Configure alerts for:
- Query latency p95 > 200ms
- Error rate > 1%
- Cache hit rate < 80%
- Database connection pool exhaustion
- Redis connection failures

---

## Support & Documentation

- **API Reference**: [analytics-api.yaml](contracts/analytics-api.yaml)
- **Data Model**: [data-model.md](data-model.md)
- **Research**: [research.md](research.md)
- **Implementation Plan**: [plan.md](plan.md)

For issues or questions, contact the development team or file a ticket in the issue tracker.
