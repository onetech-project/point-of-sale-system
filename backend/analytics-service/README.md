# Analytics Service

Business insights dashboard analytics service for tenant owners.

## Overview

The Analytics Service provides aggregated metrics, product/customer rankings, and operational task alerts for the POS system dashboard.

## Features

- **Sales Metrics**: Total sales, order count, net profit, average order value
- **Product Rankings**: Top 5 best/worst performers by quantity and sales
- **Customer Rankings**: Top 5 spending customers (by encrypted phone number)
- **Operational Tasks**: Delayed orders (>15min) and low stock alerts
- **Time-Series Data**: Sales trends with daily/weekly/monthly/quarterly/yearly granularity

## Architecture

- **Language**: Go 1.24+
- **Framework**: Echo v4
- **Database**: PostgreSQL 14+ (shared with other services)
- **Cache**: Redis 7+ (for aggregated metrics)
- **Encryption**: Vault Transit Engine (for customer PII)
- **Authentication**: Handled by API Gateway (tenant_id in request headers)

## API Endpoints

All endpoints require tenant owner authentication via API Gateway:

- `GET /health` - Health check
- `GET /analytics/overview` - Dashboard overview metrics
- `GET /analytics/sales-trend` - Time-series sales data
- `GET /analytics/top-products` - Product rankings
- `GET /analytics/top-customers` - Customer spending rankings
- `GET /analytics/tasks` - Operational task alerts

See [contracts/analytics-api.yaml](../../specs/007-business-insights-dashboard/contracts/analytics-api.yaml) for full API specification.

## Setup

### Prerequisites

- Go 1.24+
- PostgreSQL 14+
- Redis 7+
- Vault (for encryption)

### Installation

```bash
# Install dependencies
go mod download

# Copy environment file
cp .env.example .env

# Update .env with your configuration
```

### Database Setup

```sql
-- Create indexes for analytics queries
CREATE INDEX CONCURRENTLY idx_orders_tenant_status_created
  ON orders(tenant_id, status, created_at);

CREATE INDEX CONCURRENTLY idx_order_items_product_quantity
  ON order_items(product_id, quantity);

CREATE INDEX CONCURRENTLY idx_products_low_stock
  ON products(tenant_id)
  WHERE quantity <= low_stock_threshold;
```

### Running Locally

```bash
# Development mode
go run main.go

# With hot reload (requires air)
air
```

### Running with Docker

```bash
# Build image
docker build -t analytics-service .

# Run container
docker run -p 8089:8089 --env-file .env analytics-service
```

## Development

### Project Structure

```
backend/analytics-service/
├── api/                      # HTTP handlers
│   ├── health_handler.go
│   ├── analytics_handler.go
│   └── tasks_handler.go
├── src/
│   ├── config/              # Configuration
│   │   ├── database.go
│   │   └── redis.go
│   ├── models/              # Data structures
│   │   ├── time_range.go
│   │   ├── sales_metrics.go
│   │   ├── product_ranking.go
│   │   ├── customer_ranking.go
│   │   ├── delayed_order.go
│   │   ├── restock_alert.go
│   │   └── time_series.go
│   ├── repository/          # Database queries
│   │   ├── sales_repository.go
│   │   ├── product_repository.go
│   │   ├── customer_repository.go
│   │   └── task_repository.go
│   ├── services/            # Business logic
│   │   ├── analytics_service.go
│   │   └── cache_service.go
│   ├── middleware/          # HTTP middleware
│   │   └── tenant_auth.go
│   └── utils/               # Utilities
│       ├── time_series.go
│       ├── formatting.go
│       ├── encryption.go
│       └── masker.go
├── tests/                   # Tests
│   ├── unit/
│   ├── integration/
│   └── contract/
├── main.go                  # Entry point
├── go.mod
└── .env.example
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test ./tests/unit -run TestSalesMetrics
```

## Performance

- **Target Latency**: <200ms p95 for all queries
- **Dashboard Load**: <2 seconds
- **Chart Rendering**: <3 seconds
- **Data Points**: Supports up to 365 daily data points

## Security

- **Encryption**: Customer PII (phone, email, name) encrypted with Vault
- **Log Masking**: All PII masked in logs per UU PDP compliance
- **Tenant Isolation**: Row-Level Security enforced on all queries
- **Authentication**: Delegated to API Gateway

## Monitoring

- **Health Endpoint**: `/health`
- **Metrics**: Prometheus metrics on request duration, cache hit rate
- **Logging**: Structured JSON logs with zerolog
- **Tracing**: OpenTelemetry integration

## Documentation

- [Feature Specification](../../specs/007-business-insights-dashboard/spec.md)
- [API Contracts](../../specs/007-business-insights-dashboard/contracts/analytics-api.yaml)
- [Data Model](../../specs/007-business-insights-dashboard/data-model.md)
- [Implementation Plan](../../specs/007-business-insights-dashboard/plan.md)

## License

Proprietary - All Rights Reserved
