# Research: Business Insights Dashboard

**Feature**: Business Insights Dashboard  
**Date**: 2026-01-31  
**Purpose**: Research technical decisions for analytics service, chart visualization, and time-series data aggregation

---

## 1. Chart Library Selection for React/Next.js

### Decision: **Recharts**

**Rationale**:
- Built specifically for React with declarative API matching React paradigm
- Responsive by default - charts adapt to container size
- Supports all required chart types (line, bar, area) with mixed chart capability
- MIT licensed, actively maintained (23k+ GitHub stars)
- Bundle size reasonable (~160KB gzipped) - smaller than Chart.js + react-chartjs-2
- TypeScript support built-in
- Good documentation and examples for time-series data
- Easy customization of tooltips, axes, legends

**Alternatives Considered**:

| Library | Why Rejected |
|---------|-------------|
| Chart.js + react-chartjs-2 | Requires wrapper library; imperative API less idiomatic for React; larger bundle size (~180KB+ gzipped) |
| Apache ECharts | Overkill for our needs (3D charts, maps, graphs); much larger bundle (>400KB); Chinese-focused documentation; steeper learning curve |
| Victory | More opinionated styling; less flexible; smaller community; similar bundle size but fewer features |
| Nivo | Beautiful defaults but heavier bundle; more suited for static dashboards than interactive filtering |

**Installation**:
```bash
npm install recharts
npm install --save-dev @types/recharts
```

**Usage Pattern**:
```tsx
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

<ResponsiveContainer width="100%" height={300}>
  <LineChart data={salesData}>
    <CartesianGrid strokeDasharray="3 3" />
    <XAxis dataKey="date" />
    <YAxis />
    <Tooltip />
    <Line type="monotone" dataKey="sales" stroke="#8884d8" />
  </LineChart>
</ResponsiveContainer>
```

---

## 2. Time-Series Data Aggregation Strategy

### Decision: **PostgreSQL Window Functions + Materialized CTEs**

**Approach**: Use PostgreSQL's native date/time functions and window functions for efficient aggregation. Generate time series using `generate_series()` for complete date ranges including zero-value periods.

**Example Query Pattern (Daily Sales)**:
```sql
WITH date_series AS (
  SELECT generate_series(
    $1::date,  -- start_date
    $2::date,  -- end_date
    '1 day'::interval
  )::date AS date
),
daily_sales AS (
  SELECT 
    DATE(created_at) AS sale_date,
    COUNT(*) AS order_count,
    SUM(total_amount) AS total_sales,
    SUM(total_amount - total_cost) AS profit
  FROM orders
  WHERE tenant_id = $3
    AND status = 'completed'
    AND created_at >= $1
    AND created_at < $2 + interval '1 day'
  GROUP BY DATE(created_at)
)
SELECT 
  ds.date,
  COALESCE(dss.order_count, 0) AS order_count,
  COALESCE(dss.total_sales, 0) AS total_sales,
  COALESCE(dss.profit, 0) AS profit
FROM date_series ds
LEFT JOIN daily_sales dss ON ds.date = dss.sale_date
ORDER BY ds.date;
```

**Rationale**:
- `generate_series()` ensures complete date range even with zero sales days
- Single query with CTE for performance (vs multiple round trips)
- PostgreSQL date functions (DATE_TRUNC, EXTRACT) efficient for grouping
- Window functions can calculate running totals, moving averages if needed later
- Parameterized queries prevent SQL injection
- Supports all granularities: daily, weekly, monthly, quarterly, yearly

**Performance**:
- Index on `(tenant_id, created_at, status)` essential for fast filtering
- Typical query time <100ms for 30-90 day ranges with proper indexes
- Consider partial index on completed orders only

**Indexing Strategy**:
```sql
-- Composite index for order analytics
CREATE INDEX idx_orders_analytics 
ON orders(tenant_id, status, created_at) 
WHERE status = 'completed';

-- Partial index for tenant+date filtering
CREATE INDEX idx_orders_tenant_date 
ON orders(tenant_id, created_at DESC) 
INCLUDE (total_amount, status);
```

---

## 3. Caching Strategy for Analytics

### Decision: **Redis with TTL-based Invalidation**

**Pattern**: Cache aggregated metrics in Redis with short TTL (60-300 seconds). Cache key includes tenant_id, metric type, time range, and granularity.

**Cache Key Structure**:
```
analytics:{tenant_id}:{metric}:{granularity}:{start_date}:{end_date}
```

Examples:
```
analytics:uuid-1234:sales-overview:daily:2026-01-01:2026-01-31
analytics:uuid-1234:top-products:monthly:2026-01:2026-01
analytics:uuid-1234:top-customers:current-month
```

**TTL Strategy**:
- Current month metrics: 5 minutes (300s) - data changes frequently
- Historical data (older months): 1 hour (3600s) - immutable
- Operational tasks (delayed orders, low stock): 1 minute (60s) - needs freshness

**Rationale**:
- Analytics queries are read-heavy, expensive aggregations
- Dashboard accessed multiple times by same tenant
- TTL auto-expires cache, simple invalidation
- Redis fast enough for dashboard load time requirement (<2s)
- No cache stampede - background refresh pattern if needed

**Implementation Pattern**:
```go
func (s *AnalyticsService) GetSalesOverview(ctx context.Context, tenantID uuid.UUID, timeRange TimeRange) (*SalesMetrics, error) {
    cacheKey := fmt.Sprintf("analytics:%s:sales-overview:%s:%s:%s", 
        tenantID, timeRange.Granularity, timeRange.Start, timeRange.End)
    
    // Try cache first
    if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
        var metrics SalesMetrics
        json.Unmarshal([]byte(cached), &metrics)
        return &metrics, nil
    }
    
    // Cache miss - query database
    metrics, err := s.repo.GetSalesMetrics(ctx, tenantID, timeRange)
    if err != nil {
        return nil, err
    }
    
    // Cache result
    data, _ := json.Marshal(metrics)
    ttl := s.calculateTTL(timeRange)
    s.cache.Set(ctx, cacheKey, data, ttl)
    
    return metrics, nil
}
```

---

## 4. PostgreSQL Performance Optimization

### Decision: **Composite Indexes + Query Optimization**

**Key Optimizations**:

1. **Covering Indexes**: Include frequently accessed columns in index to avoid table lookups
   ```sql
   CREATE INDEX idx_orders_analytics_covering
   ON orders(tenant_id, status, created_at)
   INCLUDE (total_amount, customer_phone);
   ```

2. **Partial Indexes**: Index only relevant rows (completed orders)
   ```sql
   CREATE INDEX idx_completed_orders
   ON orders(tenant_id, created_at)
   WHERE status = 'completed';
   ```

3. **Materialized Views**: For complex frequently-accessed aggregations (consider for MVP+)
   ```sql
   -- Future optimization if needed
   CREATE MATERIALIZED VIEW daily_sales_summary AS
   SELECT 
     tenant_id,
     DATE(created_at) AS sale_date,
     COUNT(*) AS order_count,
     SUM(total_amount) AS total_sales
   FROM orders
   WHERE status = 'completed'
   GROUP BY tenant_id, DATE(created_at);
   
   CREATE UNIQUE INDEX ON daily_sales_summary(tenant_id, sale_date);
   REFRESH MATERIALIZED VIEW CONCURRENTLY daily_sales_summary;
   ```

4. **Query Optimization**:
   - Use `EXPLAIN ANALYZE` to verify index usage
   - Avoid SELECT * - only fetch needed columns
   - Use prepared statements for parameter binding
   - Batch queries where possible (single request for overview)

5. **Connection Pooling**:
   - Max 25 connections per service (existing pattern)
   - Analytics service read-only - can use read replica if available

**Performance Targets**:
- Simple aggregations (current month totals): <50ms
- Time-series queries (30-90 days daily): <100ms
- Top N queries (top 5 products/customers): <75ms
- Complex multi-table joins: <150ms

---

## 5. Customer Identification Strategy

### Decision: **Phone Number as Primary Identifier**

**Approach**: Use `customer_phone` from orders table as unique customer identifier. Phone numbers are encrypted at rest (existing security pattern).

**Query Pattern**:
```sql
SELECT 
  customer_phone,
  customer_name,
  COUNT(*) AS order_count,
  SUM(total_amount) AS total_spent
FROM orders
WHERE tenant_id = $1
  AND status = 'completed'
  AND created_at >= $2
  AND created_at < $3
GROUP BY customer_phone, customer_name
ORDER BY total_spent DESC
LIMIT 5;
```

**Rationale**:
- Phone number mandatory for guest orders (per clarification)
- Encrypted in database - analytics service decrypts for display
- Simple, direct query - no complex user account joining
- Handles guest checkouts (no user account required)

**Considerations**:
- Phone number displayed masked in UI (123-456-7890 → (123) ***-7890)
- Full number accessible only to tenant owner
- Multiple orders same phone = same customer (correct behavior)

---

## 6. Operational Task Alerts

### Decision: **Query-Based Real-Time Alerts**

**Pattern**: Query database on dashboard load for current operational tasks. No background jobs or scheduled tasks for MVP.

**Delayed Orders Query**:
```sql
SELECT 
  id,
  order_number,
  customer_phone,
  created_at,
  EXTRACT(EPOCH FROM (NOW() - created_at))/60 AS minutes_elapsed
FROM orders
WHERE tenant_id = $1
  AND status IN ('pending', 'processing')
  AND created_at < NOW() - INTERVAL '15 minutes'
ORDER BY created_at ASC
LIMIT 10;
```

**Low Stock Query**:
```sql
SELECT 
  id,
  name,
  quantity,
  low_stock_threshold,
  (low_stock_threshold - quantity) AS units_needed
FROM products
WHERE tenant_id = $1
  AND (quantity = 0 OR quantity < low_stock_threshold)
  AND status = 'active'
ORDER BY quantity ASC, units_needed DESC
LIMIT 10;
```

**Rationale**:
- Simple queries - no additional infrastructure
- Dashboard refresh shows latest status
- Low latency (<50ms per query)
- Alerts don't require push notifications (future enhancement)

---

## 7. API Design Pattern

### Decision: **RESTful JSON APIs with Pagination**

**Endpoint Structure**:
```
GET /api/v1/analytics/overview?granularity=daily&start=2026-01-01&end=2026-01-31
GET /api/v1/analytics/sales-trend?granularity=monthly&start=2025-10&end=2026-01
GET /api/v1/analytics/top-products?metric=quantity&limit=5&period=current-month
GET /api/v1/analytics/top-customers?limit=5&period=current-month
GET /api/v1/analytics/tasks?types=delayed-orders,low-stock
```

**Response Format**:
```json
{
  "status": "success",
  "data": {
    "granularity": "daily",
    "period": {
      "start": "2026-01-01",
      "end": "2026-01-31"
    },
    "metrics": {
      "total_sales": 125000,
      "total_orders": 450,
      "net_profit": 35000
    },
    "time_series": [
      {"date": "2026-01-01", "sales": 4500, "orders": 15, "profit": 1200},
      {"date": "2026-01-02", "sales": 3800, "orders": 12, "profit": 950}
    ]
  },
  "cache_hit": true,
  "query_time_ms": 45
}
```

**Error Format**:
```json
{
  "status": "error",
  "error": {
    "code": "INVALID_DATE_RANGE",
    "message": "Start date must be before end date",
    "details": {}
  }
}
```

**Rationale**:
- Follows existing service patterns
- Query parameters for filtering (RESTful)
- Includes metadata (cache hit, query time) for debugging
- Consistent error structure across all endpoints

---

## 8. Frontend Architecture

### Decision: **Server-Side Rendering (SSR) with Client-Side Interactivity**

**Pattern**: Next.js App Router with SSR for initial load, client-side updates on filter changes.

**Page Structure**:
```tsx
// app/dashboard/page.tsx (Server Component)
export default async function DashboardPage() {
  const session = await getServerSession();
  const tenantId = session.user.tenantId;
  
  // Fetch initial current month data server-side
  const initialData = await analyticsService.getOverview(tenantId, 'current-month');
  
  return <DashboardLayout initialData={initialData} />;
}

// components/dashboard/DashboardLayout.tsx (Client Component)
'use client';
export function DashboardLayout({ initialData }) {
  const [timeRange, setTimeRange] = useState('current-month');
  const [data, setData] = useState(initialData);
  
  // Client-side updates on filter change
  const handleTimeRangeChange = async (newRange) => {
    setTimeRange(newRange);
    const updated = await fetchAnalytics(newRange);
    setData(updated);
  };
  
  return (
    <div>
      <TimeSeriesFilter value={timeRange} onChange={handleTimeRangeChange} />
      <MetricsGrid data={data} />
      <Charts data={data.time_series} />
    </div>
  );
}
```

**Rationale**:
- SSR for fast initial load (<2s requirement)
- Client-side rendering for interactive filters
- Reduces API calls - server fetches initial data
- Progressive enhancement - works without JS for basic view

---

## 9. Testing Strategy

### Decision: **3-Layer Testing: Unit → Integration → Contract**

**Test Pyramid**:

1. **Unit Tests** (Go backend):
   ```go
   func TestCalculateSalesMetrics(t *testing.T) {
       // Test business logic in isolation
       // Mock repository layer
   }
   ```

2. **Integration Tests** (Database queries):
   ```go
   func TestGetSalesTrend_Integration(t *testing.T) {
       // Real database with test data
       // Verify query results
       // Check performance (<100ms)
   }
   ```

3. **Contract Tests** (API schemas):
   ```go
   func TestAnalyticsEndpoints_ContractCompliance(t *testing.T) {
       // Validate against OpenAPI spec
       // Ensure response schemas match
   }
   ```

4. **Frontend Tests** (React Testing Library):
   ```tsx
   test('dashboard renders metrics correctly', () => {
       render(<DashboardLayout initialData={mockData} />);
       expect(screen.getByText('$125,000')).toBeInTheDocument();
   });
   ```

5. **E2E Tests** (Optional - Playwright):
   - User loads dashboard
   - Changes time range filter
   - Charts update correctly

**Test Data**:
- Seed script for test orders (various dates, amounts, customers)
- Factories for generating test data
- Cleanup between tests

---

## 10. Deployment & Scalability

### Decision: **Docker Container + Horizontal Scaling**

**Deployment**:
```yaml
# docker-compose.yml
analytics-service:
  build: ./backend/analytics-service
  environment:
    - PORT=8086
    - DATABASE_URL=postgresql://pos_user:pos_password@postgres:5432/pos_db
    - REDIS_URL=redis://redis:6379/0
    - JWT_SECRET=${JWT_SECRET}
  ports:
    - "8086:8086"
  depends_on:
    - postgres
    - redis
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8086/health"]
    interval: 30s
    timeout: 10s
    retries: 3
```

**Scalability**:
- Analytics service is stateless - can run multiple instances
- Redis cache shared across instances
- Read-only queries - can use PostgreSQL read replicas
- API Gateway load balances requests

**Monitoring**:
- Prometheus metrics: query latency, cache hit rate, error rate
- Grafana dashboards: queries per second, response times
- Alert on p95 latency >200ms or error rate >1%

---

## Summary

| Decision Area | Choice | Key Benefit |
|---------------|--------|-------------|
| Chart Library | Recharts | React-native, lightweight, TypeScript support |
| Time-Series Aggregation | PostgreSQL window functions + generate_series | Single efficient query, handles gaps |
| Caching | Redis with TTL | Reduces load, simple invalidation |
| Database Optimization | Composite indexes + partial indexes | Query time <100ms |
| Customer ID | Phone number (encrypted) | Simple, works for guest orders |
| Task Alerts | Query on load | No background jobs needed |
| API Design | RESTful JSON with metadata | Consistent with existing services |
| Frontend | Next.js SSR + client hydration | Fast initial load, interactive filters |
| Testing | Unit → Integration → Contract | Comprehensive coverage |
| Deployment | Docker + horizontal scaling | Standard pattern, scalable |

All decisions align with Constitution principles: simple, testable, secure, observable, and following established patterns.
