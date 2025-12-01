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
- `GET /api/v1/products` - List products (with filters)
- `GET /api/v1/products/:id` - Get product by ID
- `PUT /api/v1/products/:id` - Update product
- `DELETE /api/v1/products/:id` - Delete product
- `PATCH /api/v1/products/:id/archive` - Archive product
- `PATCH /api/v1/products/:id/restore` - Restore archived product
- `POST /api/v1/products/:id/photo` - Upload product photo

### Categories

- `POST /api/v1/categories` - Create category
- `GET /api/v1/categories` - List categories
- `GET /api/v1/categories/:id` - Get category by ID
- `PUT /api/v1/categories/:id` - Update category
- `DELETE /api/v1/categories/:id` - Delete category (if no products assigned)

### Health

- `GET /health` - Health check
- `GET /ready` - Readiness check

## Features

### Multi-Tenant Isolation

All data is tenant-scoped using PostgreSQL Row-Level Security (RLS). The tenant context is set automatically via middleware from JWT tokens.

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

### Category Management

- Flat category structure
- Manual display ordering
- Prevent deletion of categories with assigned products

### Inventory Tracking

- Real-time stock quantities
- Low stock filtering
- Inventory summary dashboard
- Support for negative stock (backorders)

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

Build for production:
```bash
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o product-service .
```

Docker build:
```bash
docker build -t product-service:latest .
```

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
