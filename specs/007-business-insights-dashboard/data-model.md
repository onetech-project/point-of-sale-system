# Data Model: Business Insights Dashboard

**Feature**: Business Insights Dashboard  
**Date**: 2026-01-31  
**Purpose**: Define entities, relationships, and validation rules for analytics and dashboard features

---

## Entity Diagram

```
┌─────────────────┐         ┌──────────────────┐
│   TimeRange     │────────>│  SalesMetrics    │
│                 │         │                  │
│ - granularity   │         │ - total_sales    │
│ - start_date    │         │ - order_count    │
│ - end_date      │         │ - net_profit     │
│ - tenant_id     │         │ - inventory_value│
└─────────────────┘         └──────────────────┘
         │
         │                  ┌──────────────────┐
         └─────────────────>│ TimeSeriesData   │
                            │                  │
                            │ - date           │
                            │ - sales          │
                            │ - orders         │
                            │ - profit         │
                            └──────────────────┘
         
┌─────────────────┐         ┌──────────────────┐
│ ProductRanking  │         │ CustomerRanking  │
│                 │         │                  │
│ - product_id    │         │ - customer_phone │
│ - product_name  │         │ - customer_name  │
│ - quantity_sold │         │ - total_spent    │
│ - total_sales   │         │ - order_count    │
│ - rank          │         │ - rank           │
└─────────────────┘         └──────────────────┘

┌─────────────────┐         ┌──────────────────┐
│ DelayedOrder    │         │ RestockAlert     │
│                 │         │                  │
│ - order_id      │         │ - product_id     │
│ - order_number  │         │ - product_name   │
│ - customer_phone│         │ - current_qty    │
│ - elapsed_mins  │         │ - threshold      │
│ - created_at    │         │ - units_needed   │
└─────────────────┘         └──────────────────┘
```

---

## 1. TimeRange

**Purpose**: Represents the time period and granularity for analytics queries.

### Attributes

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `tenant_id` | UUID | Yes | Valid UUID from JWT | Tenant requesting analytics |
| `granularity` | enum | Yes | One of: daily, weekly, monthly, quarterly, yearly | Time series bucket size |
| `start_date` | date | Yes | Valid ISO 8601 date, not in future | Beginning of analysis period |
| `end_date` | date | Yes | Valid ISO 8601 date, >= start_date, not in future | End of analysis period |
| `current_month` | boolean | No | - | Flag indicating if this is the current month preset |

### Validation Rules

- `end_date` must be >= `start_date`
- Date range constraints by granularity:
  - Daily: max 90 days
  - Weekly: max 52 weeks (~365 days)
  - Monthly: max 12 months
  - Quarterly: max 8 quarters (24 months)
  - Yearly: max 5 years
- Both dates must not be in the future
- Dates must align with granularity (e.g., monthly starts on 1st of month)

### Relationships

- Used as input parameter for all analytics queries
- Determines aggregation strategy and cache key

---

## 2. SalesMetrics

**Purpose**: Aggregated financial and operational metrics for a specific time period.

### Attributes

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `tenant_id` | UUID | Yes | Valid UUID | Tenant these metrics belong to |
| `period_start` | date | Yes | ISO 8601 date | Start of metric period |
| `period_end` | date | Yes | ISO 8601 date | End of metric period |
| `total_sales` | decimal | Yes | >= 0 | Sum of all completed order totals |
| `order_count` | integer | Yes | >= 0 | Count of completed orders |
| `net_profit` | decimal | Yes | Can be negative | total_sales - total_cost - operational_costs |
| `inventory_value` | decimal | Yes | >= 0 | Sum of (product.cost × product.quantity) for all products |
| `avg_order_value` | decimal | Yes | >= 0 | total_sales / order_count (or 0 if no orders) |
| `calculated_at` | timestamp | Yes | ISO 8601 timestamp | When these metrics were calculated |

### Validation Rules

- `total_sales` must be non-negative
- `order_count` must be non-negative integer
- `net_profit` can be negative (losses) but must be <= `total_sales`
- `inventory_value` must be non-negative
- `avg_order_value` = `total_sales` / `order_count` (0 if no orders)
- All monetary values in tenant's configured currency

### Relationships

- Derived from `orders` table (completed orders only)
- Derived from `products` table (for inventory value)
- Associated with `TimeRange` for period specification

### Business Rules

- Only count orders with `status = 'completed'` or `status = 'paid'`
- Exclude cancelled, pending, abandoned orders
- Refunds subtract from total_sales and adjust profit
- Inventory value uses current stock levels (not historical)

---

## 3. TimeSeriesData

**Purpose**: Single data point in a time-series chart representing metrics for one time bucket.

### Attributes

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `date` | date/datetime | Yes | ISO 8601 format | Date or datetime for this data point |
| `sales` | decimal | Yes | >= 0 | Total sales for this period |
| `orders` | integer | Yes | >= 0 | Order count for this period |
| `profit` | decimal | Yes | Can be negative | Net profit for this period |
| `customers` | integer | No | >= 0 | Unique customer count (optional) |

### Validation Rules

- `date` format varies by granularity:
  - Daily: YYYY-MM-DD
  - Weekly: YYYY-Www (e.g., 2026-W05)
  - Monthly: YYYY-MM
  - Quarterly: YYYY-Qq (e.g., 2026-Q1)
  - Yearly: YYYY
- Array of TimeSeriesData must be ordered by date ascending
- Zero values allowed (days with no sales)

### Relationships

- Collection belongs to `SalesMetrics` for a given `TimeRange`
- Each point represents one bucket in the time series

### Business Rules

- Include zero-value points for periods with no sales (complete time series)
- Use LEFT JOIN with generated date series to ensure completeness
- Aggregation aligns with TimeRange.granularity

---

## 4. ProductRanking

**Purpose**: Represents a product's performance ranking (top performers or bottom performers).

### Attributes

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `product_id` | UUID | Yes | Valid UUID | Product identifier |
| `product_name` | string | Yes | 1-255 chars | Product name |
| `category_name` | string | No | 1-100 chars | Product category (if exists) |
| `quantity_sold` | integer | Yes | >= 0 | Total units sold in period |
| `total_sales` | decimal | Yes | >= 0 | Total revenue from this product |
| `order_count` | integer | Yes | >= 0 | Number of orders containing this product |
| `rank` | integer | Yes | 1-N | Position in ranking (1 = best) |
| `ranking_metric` | enum | Yes | quantity or sales | What this ranking is based on |

### Validation Rules

- `product_name` must not be empty
- `quantity_sold` must be non-negative
- `total_sales` must be non-negative
- `rank` must be positive integer
- `ranking_metric` must be either "quantity" or "sales"

### Relationships

- Links to `products` table via `product_id`
- Filtered by `tenant_id` (tenant isolation)
- Associated with `TimeRange` for period

### Business Rules

- Top performers: rank 1-5 by quantity_sold or total_sales (DESC)
- Bottom performers: rank based on lowest quantity_sold or total_sales (ASC)
- Only include products with at least 1 sale in period for top performers
- For bottom performers, can include products with 0 sales if < 5 products have sales
- Ties in metric value get same rank (e.g., two products ranked #3)

---

## 5. CustomerRanking

**Purpose**: Represents a customer's spending ranking (top spenders).

### Attributes

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `customer_phone` | string | Yes | Valid phone format | Customer's phone number (encrypted at rest) |
| `customer_name` | string | No | 1-255 chars | Customer name from most recent order |
| `total_spent` | decimal | Yes | > 0 | Sum of all completed orders for this customer |
| `order_count` | integer | Yes | >= 1 | Number of completed orders |
| `avg_order_value` | decimal | Yes | > 0 | total_spent / order_count |
| `first_order_date` | date | Yes | ISO 8601 date | Date of customer's first order |
| `last_order_date` | date | Yes | ISO 8601 date | Date of customer's most recent order |
| `rank` | integer | Yes | 1-5 | Position in top spenders (1 = highest) |

### Validation Rules

- `customer_phone` must be valid phone format (E.164 recommended)
- `total_spent` must be positive (customers with $0 excluded)
- `order_count` must be >= 1
- `avg_order_value` = `total_spent` / `order_count`
- `last_order_date` >= `first_order_date`
- `rank` between 1 and 5

### Relationships

- Aggregated from `orders` table grouped by `customer_phone`
- Filtered by `tenant_id` and `TimeRange`
- Phone number is primary identifier (no user account required)

### Business Rules

- Customer uniqueness determined by phone number only
- Same phone = same customer, even if name differs
- Only completed/paid orders count toward spending
- Top 5 by total_spent in descending order
- Phone number masked in UI: (123) ***-7890

---

## 6. DelayedOrder

**Purpose**: Represents an order that has exceeded the processing time threshold (operational task alert).

### Attributes

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `order_id` | UUID | Yes | Valid UUID | Order identifier |
| `order_number` | string | Yes | 1-50 chars | Human-readable order number |
| `customer_phone` | string | Yes | Valid phone format | Customer's contact |
| `customer_name` | string | No | 1-255 chars | Customer name |
| `status` | enum | Yes | pending or processing | Current order status |
| `created_at` | timestamp | Yes | ISO 8601 timestamp | When order was created |
| `elapsed_minutes` | integer | Yes | > 15 | Minutes since order creation |
| `total_amount` | decimal | Yes | > 0 | Order total value |

### Validation Rules

- `order_number` must be unique per tenant
- `status` must be "pending" or "processing" (not completed/cancelled)
- `created_at` must be at least 15 minutes in the past
- `elapsed_minutes` = floor((NOW - created_at) / 60 seconds)
- `elapsed_minutes` must be > 15

### Relationships

- Links to `orders` table via `order_id`
- Filtered by `tenant_id`
- Real-time query (not cached)

### Business Rules

- Alert threshold: 15 minutes from `created_at`
- Only orders still in pending/processing status
- Ordered by `created_at` ASC (oldest first)
- Limit to top 10 most delayed orders
- Count total delayed orders for badge notification

---

## 7. RestockAlert

**Purpose**: Represents a product requiring restocking (operational task alert).

### Attributes

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `product_id` | UUID | Yes | Valid UUID | Product identifier |
| `product_name` | string | Yes | 1-255 chars | Product name |
| `current_quantity` | integer | Yes | >= 0 | Current stock level |
| `low_stock_threshold` | integer | Yes | >= 0 | Configured threshold for this product |
| `units_needed` | integer | Yes | > 0 | Recommended restock quantity |
| `category_name` | string | No | 1-100 chars | Product category |
| `last_restock_date` | date | No | ISO 8601 date | Last time product was restocked |
| `status` | enum | Yes | active | Product must be active |

### Validation Rules

- `current_quantity` must be non-negative
- `low_stock_threshold` must be non-negative
- Alert triggered when: `current_quantity` = 0 OR `current_quantity` < `low_stock_threshold`
- `units_needed` = `low_stock_threshold` - `current_quantity` (minimum 1)

### Relationships

- Links to `products` table via `product_id`
- Filtered by `tenant_id` and `status = 'active'`
- Threshold configurable per tenant in settings

### Business Rules

- Out of stock (quantity = 0): highest priority
- Low stock (quantity < threshold): normal priority
- Only active products generate alerts
- Ordered by priority (out of stock first), then by units_needed DESC
- Default threshold: 10 units (configurable per tenant)
- Limit to top 10 products needing restock

---

## Database Schema Requirements

### Existing Tables Used

```sql
-- orders table (already exists)
-- Need indexes on:
CREATE INDEX IF NOT EXISTS idx_orders_analytics 
ON orders(tenant_id, status, created_at) 
WHERE status IN ('completed', 'paid');

CREATE INDEX IF NOT EXISTS idx_orders_customer_analytics
ON orders(tenant_id, customer_phone, created_at)
WHERE status IN ('completed', 'paid');

-- products table (already exists)
-- Need indexes on:
CREATE INDEX IF NOT EXISTS idx_products_stock_alerts
ON products(tenant_id, status, current_quantity, low_stock_threshold)
WHERE status = 'active';
```

### New Configuration Tables

```sql
-- Tenant-specific dashboard settings (if needed)
CREATE TABLE IF NOT EXISTS tenant_dashboard_settings (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    low_stock_default_threshold INTEGER DEFAULT 10,
    delayed_order_threshold_minutes INTEGER DEFAULT 15,
    default_date_range VARCHAR(20) DEFAULT 'current-month',
    default_granularity VARCHAR(20) DEFAULT 'daily',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## Validation Summary

### Cross-Entity Validation

- All entities must enforce tenant isolation via `tenant_id`
- Dates must be validated against business rules (no future dates for historical data)
- Monetary values must use consistent currency per tenant
- Rankings must be consistent (no duplicate ranks unless tied)
- Time series must be complete (no gaps in date ranges)

### Performance Considerations

- Indexes on tenant_id + timestamp columns for fast filtering
- Partial indexes on frequently queried status values
- Cache aggregated metrics in Redis (TTL 1-5 minutes)
- Query timeouts: 5 seconds max per analytics query

### Security Considerations

- Tenant ID validated against JWT claim
- RLS enforced at database level
- Customer phone numbers encrypted at rest, decrypted for display
- No cross-tenant data leakage (verified in integration tests)
- Rate limiting on analytics endpoints: 60 requests/minute per tenant

---

## State Transitions

### SalesMetrics State

- **Stale** → **Calculating** → **Cached** → **Stale** (TTL expires)
- Trigger recalculation on cache miss or manual refresh

### Task Alerts State

- **Active** → **Resolved** (order processed or product restocked)
- Real-time query, no persistent state

### Rankings State

- **Outdated** → **Calculating** → **Current** → **Outdated**
- Cached with short TTL (5 minutes for current month)

---

## Summary

All entities designed for:
- **Tenant isolation**: Every query filtered by tenant_id from JWT
- **Performance**: Indexed columns, caching strategy, optimized queries
- **Accuracy**: Validation rules, business logic enforcement
- **Security**: Encryption, RLS, no cross-tenant leakage
- **Testability**: Clear validation rules, deterministic calculations

Next steps: Generate API contracts with these entities as request/response schemas.
