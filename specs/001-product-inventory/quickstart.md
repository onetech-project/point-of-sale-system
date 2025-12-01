# Quickstart Guide: Product & Inventory Management

**Feature Branch**: `001-product-inventory`  
**Last Updated**: 2025-12-01

## Overview

This guide provides a quick reference for developers working on the Product & Inventory Management feature. It covers setup, common operations, and troubleshooting.

## Prerequisites

- Go 1.23.0+
- Node.js 18+
- PostgreSQL 14+ (via Docker)
- Redis 7+ (via Docker)
- Existing auth/tenant services running

## Quick Setup

### 1. Start Dependencies

```bash
cd /home/asrock/code/POS/point-of-sale-system
docker-compose up -d
```

### 2. Run Database Migrations

```bash
# Apply new migrations for products, categories, stock_adjustments
migrate -path backend/migrations \
        -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
        up
```

### 3. Install Product Service Dependencies

```bash
cd backend/product-service
go mod init github.com/pos/backend/product-service
go mod tidy
```

### 4. Start Product Service

```bash
cd backend/product-service
go run main.go
# Service runs on port 8084 by default
```

### 5. Update API Gateway Routes

Add product service routing to `api-gateway/main.go`:
```go
productGroup := e.Group("/api/v1/products")
productGroup.Use(authMiddleware, tenantMiddleware)
productGroup.Any("/*", proxyHandler("http://localhost:8084"))

categoryGroup := e.Group("/api/v1/categories")
categoryGroup.Use(authMiddleware, tenantMiddleware)
categoryGroup.Any("/*", proxyHandler("http://localhost:8084"))

inventoryGroup := e.Group("/api/v1/inventory")
inventoryGroup.Use(authMiddleware, tenantMiddleware)
inventoryGroup.Any("/*", proxyHandler("http://localhost:8084"))
```

### 6. Start Frontend Development Server

```bash
cd frontend
npm run dev
# Frontend runs on port 3000
```

## Project Structure

```
backend/product-service/
├── api/                    # HTTP handlers
│   ├── product_handler.go  # Product CRUD endpoints
│   ├── category_handler.go # Category management
│   └── stock_handler.go    # Inventory adjustments
├── src/
│   ├── models/             # Data structures
│   ├── repository/         # Database layer
│   └── services/           # Business logic
├── tests/
│   ├── contract/           # API contract tests
│   ├── integration/        # Integration tests
│   └── unit/              # Unit tests
├── go.mod
└── main.go

frontend/pages/products/
├── index.tsx               # Product catalog page
├── [id].tsx               # Product detail/edit page
├── new.tsx                # Create product page
└── categories.tsx         # Category management page
```

## Common Tasks

### Create a Product via API

```bash
curl -X POST http://localhost:8080/api/v1/products \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "sku": "PROD-001",
    "name": "Coffee Beans - Medium Roast",
    "description": "Premium Arabica coffee beans",
    "category_id": "category-uuid-here",
    "selling_price": 15.99,
    "cost_price": 8.50,
    "tax_rate": 10.00,
    "stock_quantity": 50
  }'
```

### List Products with Filtering

```bash
# Get all active products
curl http://localhost:8080/api/v1/products \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Search by name
curl "http://localhost:8080/api/v1/products?search=coffee" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Filter by category
curl "http://localhost:8080/api/v1/products?category_id=CATEGORY_UUID" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Get low stock products (below 10 units)
curl "http://localhost:8080/api/v1/products?low_stock=10" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Adjust Stock Quantity

```bash
curl -X POST http://localhost:8080/api/v1/products/PRODUCT_UUID/stock \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "new_quantity": 150,
    "reason": "supplier_delivery",
    "notes": "Received shipment from ABC Supplier"
  }'
```

### Upload Product Photo

```bash
curl -X POST http://localhost:8080/api/v1/products/PRODUCT_UUID/photo \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "photo=@/path/to/image.jpg"
```

### Create a Category

```bash
curl -X POST http://localhost:8080/api/v1/categories \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Beverages",
    "display_order": 1
  }'
```

### Get Stock Adjustment History

```bash
# For a specific product
curl http://localhost:8080/api/v1/products/PRODUCT_UUID/adjustments \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# All adjustments with filters
curl "http://localhost:8080/api/v1/inventory/adjustments?reason=shrinkage&start_date=2025-11-01T00:00:00Z" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## Running Tests

### Unit Tests

```bash
cd backend/product-service
go test ./src/... -v
```

### Contract Tests

```bash
cd backend/product-service/tests/contract
go test -v
```

### Integration Tests

```bash
# Requires PostgreSQL running
cd backend/product-service/tests/integration
go test -v
```

### Frontend Tests

```bash
cd frontend
npm test
```

## Development Workflow

### Test-First Development (TDD)

1. **Write the test first**:
   ```bash
   cd backend/product-service/tests/unit
   # Create test file: product_service_test.go
   ```

2. **Run test (should fail)**:
   ```bash
   go test -v
   ```

3. **Implement the code**:
   ```bash
   cd backend/product-service/src/services
   # Implement in product_service.go
   ```

4. **Run test (should pass)**:
   ```bash
   go test -v
   ```

5. **Refactor if needed**

### Database Migrations

**Create a new migration**:
```bash
# Install golang-migrate if needed
migrate create -ext sql -dir backend/migrations -seq create_table_name
```

**Apply migrations**:
```bash
migrate -path backend/migrations \
        -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
        up
```

**Rollback last migration**:
```bash
migrate -path backend/migrations \
        -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
        down 1
```

## Key Concepts

### Multi-Tenant Isolation

All queries automatically scoped to tenant via Row-Level Security:
```go
// Middleware sets tenant context
db.Exec("SET LOCAL app.current_tenant_id = $1", tenantID)

// Queries automatically filtered by RLS
products, err := repo.FindAll(ctx)  // Only returns current tenant's products
```

### Inventory Updates

Two types of stock changes:
1. **Manual Adjustments**: Via `/products/{id}/stock` endpoint with audit logging
2. **Sales Deductions**: Triggered by sales service (future integration)

Both are logged in `stock_adjustments` table for full audit trail.

### Product Photos

Stored in file system, served via API:
- **Upload**: `POST /products/{id}/photo` (multipart/form-data)
- **Retrieve**: `GET /products/{id}/photo` (returns image binary)
- **Delete**: `DELETE /products/{id}/photo`

File path stored in database: `uploads/{tenant_id}/{product_id}/photo.jpg`

### Archive vs Delete

- **Archive**: Soft delete, preserves data for historical reports
- **Delete**: Permanent removal, only allowed if no sales history

## Troubleshooting

### Product Service Won't Start

**Check PostgreSQL connection**:
```bash
docker-compose ps
psql postgresql://pos_user:pos_password@localhost:5432/pos_db
```

**Check Redis connection**:
```bash
docker-compose exec redis redis-cli ping
# Should return: PONG
```

### SKU Already Exists Error (409)

SKUs must be unique per tenant. Check existing products:
```bash
curl "http://localhost:8080/api/v1/products?search=PROD-001" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Cannot Delete Category (403)

Category has products assigned. Reassign products first:
```bash
# Get products in category
curl "http://localhost:8080/api/v1/products?category_id=CATEGORY_UUID" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Update each product to different category
curl -X PUT http://localhost:8080/api/v1/products/PRODUCT_UUID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"category_id": "NEW_CATEGORY_UUID"}'
```

### Photo Upload Fails (400)

Check file size and format:
- Max size: 5MB
- Allowed formats: JPEG, PNG, WebP
- Min dimensions: 100x100px
- Max dimensions: 4000x4000px

### Stock Adjustment Not Logged

Ensure:
1. User is authenticated (JWT token valid)
2. Reason is one of: `supplier_delivery`, `physical_count`, `shrinkage`, `damage`, `return`, `correction`
3. `new_quantity` is provided in request body

## Environment Variables

Create `.env` in `backend/product-service/`:

```env
PORT=8084
DATABASE_URL=postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable
REDIS_HOST=localhost:6379
REDIS_PASSWORD=
JWT_SECRET=your-secret-key-here
UPLOAD_DIR=./uploads
MAX_PHOTO_SIZE_MB=5
```

## API Documentation

Full OpenAPI specification available at:
- File: `specs/001-product-inventory/contracts/product-api.yaml`
- Interactive: Import YAML into Swagger UI or Postman

## Next Steps

1. **Phase 2**: Generate detailed task breakdown (`/speckit.tasks`)
2. **Implementation**: Follow TDD approach per constitution
3. **Frontend**: Build product management UI in Next.js
4. **Integration**: Connect to sales service for inventory deductions
5. **Testing**: Achieve 80%+ code coverage

## Useful Commands

```bash
# View product service logs
tail -f backend/product-service/logs/service.log

# Check database tables
psql postgresql://pos_user:pos_password@localhost:5432/pos_db
\dt            # List tables
\d products    # Describe products table

# Clear Redis cache
docker-compose exec redis redis-cli FLUSHALL

# Rebuild Go service
cd backend/product-service
go build -o product-service

# Format Go code
go fmt ./...

# Run linter
golangci-lint run
```

## Resources

- **API Spec**: [contracts/product-api.yaml](./contracts/product-api.yaml)
- **Data Model**: [data-model.md](./data-model.md)
- **Research**: [research.md](./research.md)
- **Feature Spec**: [spec.md](./spec.md)
- **Constitution**: [../.specify/memory/constitution.md](../.specify/memory/constitution.md)

## Support

For questions or issues:
1. Check this guide first
2. Review feature spec and data model
3. Check existing tests for examples
4. Consult constitution for architecture decisions
