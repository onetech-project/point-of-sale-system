# Order Service

Guest ordering service for the POS system.

## Features

- **Cart Management**: Redis-backed shopping cart with 24hr TTL
- **Order Processing**: Guest order creation and tracking
- **Payment Integration**: Midtrans QRIS payment processing
- **Delivery Management**: Geocoding and service area validation
- **Inventory Reservations**: Temporary inventory holds during checkout

## Environment Variables

See `.env.example` for all required configuration.

## Development

```bash
# Install dependencies
go mod download

# Run service
go run main.go

# Run tests
go test ./...
```

## API Endpoints

### Public Cart (Guest)
- `GET /api/v1/public/:tenantId/cart` - Get cart
- `POST /api/v1/public/:tenantId/cart/items` - Add item
- `PATCH /api/v1/public/:tenantId/cart/items/:productId` - Update item
- `DELETE /api/v1/public/:tenantId/cart/items/:productId` - Remove item
- `DELETE /api/v1/public/:tenantId/cart` - Clear cart

Headers: `X-Session-Id` (required for cart operations)

## Architecture

- **Handlers** (`api/`): HTTP request handling
- **Services** (`src/services/`): Business logic
- **Repository** (`src/repository/`): Data access
- **Models** (`src/models/`): Domain entities
- **Middleware** (`src/middleware/`): Request processing
- **Config** (`src/config/`): Service configuration
- **Utils** (`src/utils/`): Helper functions
