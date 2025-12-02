# Product Service

Product & Inventory Management microservice for the POS system.

## Overview

This service handles:
- Product CRUD operations (create, read, update, delete, archive/restore)
- Category management for organizing products
- Inventory tracking and stock level monitoring
- Product photo upload and management
- Stock adjustments with audit trail (future enhancement)

## Architecture

- **Language**: Go 1.23.0
- **Framework**: Echo v4
- **Database**: PostgreSQL 14+ with Row-Level Security (RLS)
- **Cache**: Redis v9
- **Image Processing**: imaging library for photo resizing

## Project Structure

```
backend/product-service/
├── api/                    # HTTP handlers
│   ├── product_handler.go
│   └── category_handler.go
├── src/
│   ├── models/             # Data models
│   ├── repository/         # Database layer
│   ├── services/           # Business logic
│   ├── config/            # Database, Redis setup
│   ├── middleware/        # Tenant context
│   └── utils/             # Logging, errors
├── tests/
│   ├── contract/          # API contract tests
│   ├── integration/       # Integration tests
│   └── unit/             # Unit tests
├── go.mod
└── main.go
```

## Getting Started

### Prerequisites

- Go 1.23.0+
- PostgreSQL 14+
- Redis 7+
- Running auth/tenant services

### Setup

1. Copy environment file:
```bash
cp .env.example .env
```

2. Update .env with your configuration:
```env
PORT=8085
DATABASE_URL=postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable
REDIS_HOST=localhost:6379
JWT_SECRET=your-secret-key
UPLOAD_DIR=./uploads
MAX_PHOTO_SIZE_MB=5
```

3. Install dependencies:
```bash
go mod tidy
```

4. Run database migrations:
```bash
migrate -path ../migrations \
        -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
        up
```

5. Build and run:
```bash
go build -o product-service
./product-service
```

## API Endpoints

### Products

- `POST /api/v1/products` - Create product
- `GET /api/v1/products` - List products (with filters: search, category, low_stock, archived)
- `GET /api/v1/products/:id` - Get product by ID
- `PUT /api/v1/products/:id` - Update product
- `DELETE /api/v1/products/:id` - Delete product (only if no sales history)
- `PATCH /api/v1/products/:id/archive` - Archive product
- `PATCH /api/v1/products/:id/restore` - Restore archived product
- `POST /api/v1/products/:id/photo` - Upload product photo
- `DELETE /api/v1/products/:id/photo` - Delete product photo

### Inventory & Stock

- `GET /api/v1/inventory/summary` - Get inventory summary statistics
- `POST /api/v1/products/:id/stock` - Adjust product stock quantity
- `GET /api/v1/products/:id/adjustments` - Get stock adjustment history for product
- `GET /api/v1/inventory/adjustments` - Get all stock adjustments with filters

### Categories

- `POST /api/v1/categories` - Create category
- `GET /api/v1/categories` - List categories (cached)
- `GET /api/v1/categories/:id` - Get category by ID
- `PUT /api/v1/categories/:id` - Update category
- `DELETE /api/v1/categories/:id` - Delete category (if no products assigned)

### Health

- `GET /health` - Health check (basic status)
- `GET /ready` - Readiness check (verifies database connectivity)

## Features

### Multi-Tenant Isolation

All data is tenant-scoped using PostgreSQL Row-Level Security (RLS). The tenant context is set automatically via middleware from JWT tokens passed through the API Gateway.

### Product Management

- SKU uniqueness validation per tenant
- Category assignment (optional)
- Price and tax rate tracking
- Stock quantity management
- Archive/restore for discontinued products
- Permanent deletion (only if no sales history)

### Photo Management

- Upload product photos (max 5MB)
- Automatic resizing to 800px width
- Formats: JPEG, PNG, WebP
- File system storage with database metadata
- Tenant-isolated storage paths

### Category Management

- Flat category structure
- Manual display ordering
- Prevent deletion of categories with assigned products
- Redis caching with 5-minute TTL

### Inventory Tracking

- Real-time stock quantities
- Low stock filtering
- Inventory summary dashboard
- Stock adjustments with audit trail
- Support for negative stock (backorders)

### Stock Adjustments with Audit Trail

- Manual stock adjustments (restocks, corrections, shrinkage, damage)
- Complete audit logging: user, timestamp, reason, notes
- Transaction-based updates ensuring consistency
- Adjustment history per product
- Reason codes: supplier_delivery, physical_count, shrinkage, damage, return, correction

### Observability

- Structured logging for all operations
- Request ID tracking for distributed tracing
- Response time metrics
- Rate limiting (100 requests/minute per IP)
- Health and readiness checks
- Graceful shutdown handling

## Testing

Run unit tests:
```bash
go test ./src/... -v
```

Run all tests:
```bash
go test ./... -v
```

## Development

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `go vet` for static analysis

### Database Migrations

Create new migration:
```bash
migrate create -ext sql -dir ../migrations -seq migration_name
```

## Deployment

### Build for Production

```bash
# Build binary with .bin extension
go build -o product-service.bin main.go

# Or with optimizations
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o product-service.bin .
```

### Docker Deployment

```bash
docker build -t product-service:latest .
docker run -p 8086:8086 --env-file .env product-service:latest
```

### Kubernetes Deployment

Health and readiness probes configuration:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8086
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8086
  initialDelaySeconds: 5
  periodSeconds: 5
```

### Graceful Shutdown

The service handles SIGINT/SIGTERM signals gracefully:
- Stops accepting new requests
- Waits up to 10 seconds for active requests to complete
- Closes database and Redis connections
- Exits cleanly

### Performance Considerations

- Rate limiting: 100 requests/minute per IP (configurable)
- Category caching: 5-minute TTL in Redis
- Database connection pooling
- Request timeout: 10 seconds for graceful shutdown
- Photo resizing: Max 800px width to optimize storage

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| PORT | No | 8085 | HTTP server port |
| DATABASE_URL | Yes | - | PostgreSQL connection string |
| REDIS_HOST | No | localhost:6379 | Redis host:port |
| REDIS_PASSWORD | No | "" | Redis password |
| JWT_SECRET | Yes | - | JWT validation secret |
| UPLOAD_DIR | No | ./uploads | Photo storage directory |
| MAX_PHOTO_SIZE_MB | No | 5 | Max photo size in MB |

## Troubleshooting

### Database Connection Failed

Ensure PostgreSQL is running and migrations are applied:
```bash
docker-compose ps
migrate -path ../migrations -database $DATABASE_URL up
```

### Redis Connection Failed

Check Redis is running:
```bash
docker-compose exec redis redis-cli ping
```

### SKU Already Exists Error

SKUs must be unique per tenant. Check for duplicates:
```sql
SELECT tenant_id, sku, COUNT(*) 
FROM products 
GROUP BY tenant_id, sku 
HAVING COUNT(*) > 1;
```

## License

Proprietary - POS System
