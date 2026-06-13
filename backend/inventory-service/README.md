# Inventory Service

Ingredient inventory foundation for the POS system.

## Responsibilities

- Unit of measure master data
- Tenant-scoped custom UoMs
- Ingredient/raw material CRUD
- Ingredient-specific unit conversions
- Stock ledger movements
- Initial stock, purchase-in, manual adjustment, and waste
- Ingredient movement history, low-stock query, and valuation

This service is separate from `product-service` product stock. Product stock routes remain backward compatible.

## API

- `GET /api/v1/uoms`
- `POST /api/v1/uoms`
- `PUT /api/v1/uoms/:id`
- `DELETE /api/v1/uoms/:id`
- `GET /api/v1/ingredients`
- `GET /api/v1/ingredients/:id`
- `POST /api/v1/ingredients`
- `PUT /api/v1/ingredients/:id`
- `DELETE /api/v1/ingredients/:id`
- `GET /api/v1/ingredients/:id/conversions`
- `POST /api/v1/ingredients/:id/conversions`
- `PUT /api/v1/ingredients/:id/conversions/:conversionId`
- `DELETE /api/v1/ingredients/:id/conversions/:conversionId`
- `POST /api/v1/inventory/initial-stock`
- `POST /api/v1/inventory/purchases`
- `POST /api/v1/inventory/adjustments`
- `POST /api/v1/inventory/waste`
- `GET /api/v1/inventory/ingredients/:ingredientId/movements`
- `GET /api/v1/inventory/low-stock`
- `GET /api/v1/inventory/valuation`

All API routes require `X-Tenant-ID`. Mutation routes use `X-User-ID` when present for stock movement audit fields.

## Development

```bash
cp .env.example .env
go mod tidy
go test ./...
go build ./...
```

Database migrations live in `backend/migrations`.
