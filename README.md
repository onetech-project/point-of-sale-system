# Point of Sale System - Multi-Tenant with QRIS Guest Ordering

A modern, scalable Point of Sale (POS) system with multi-tenancy, user authentication, and guest QRIS ordering capabilities.

## 🏗️ Architecture

**Microservices Architecture:**
- **API Gateway** (Port 8080): Entry point for all client requests, handles routing, authentication, rate limiting
- **Auth Service** (Port 8082): User authentication, session management, JWT token generation
- **Tenant Service** (Port 8081): Tenant registration and management
- **User Service** (Port 8083): User management, invitations
- **Order Service** (Port 8084): **NEW** Guest ordering, cart, payments, inventory reservations
- **Frontend** (Port 3000): Next.js React application with i18n support (EN/ID)

**Data Layer:**
- **PostgreSQL 14**: Primary database with Row-Level Security for tenant isolation
- **Redis 7**: Session storage, rate limiting, and cart persistence

**External Services:**
- **Midtrans**: QRIS payment gateway (sandbox/production)
- **Google Maps API**: Geocoding for delivery addresses

## 🚀 Quick Start

### Prerequisites

- Go 1.23+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL 14+ (via Docker)
- Redis 7+ (via Docker)
- **Midtrans Account**: For QRIS payment processing (sandbox for testing)
- **Google Maps API Key**: For geocoding delivery addresses (optional for basic features)

### Installation

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd point-of-sale-system
   ```

2. **Set up environment variables:**
   ```bash
   ./scripts/setup-env.sh
   ```
   
   Or manually:
   ```bash
   cp .env.example .env
   cp api-gateway/.env.example api-gateway/.env
   cp backend/auth-service/.env.example backend/auth-service/.env
   cp backend/order-service/.env.example backend/order-service/.env  # NEW
   # ... repeat for other services
   ```
   
   ⚠️ **Important:** Review and update the `.env` files with your configuration.
   
   **For Order Service**, add these required variables:
   ```bash
   # backend/order-service/.env
   MIDTRANS_SERVER_KEY=your_midtrans_server_key
   MIDTRANS_CLIENT_KEY=your_midtrans_client_key
   MIDTRANS_ENVIRONMENT=sandbox  # or production
   GOOGLE_MAPS_API_KEY=your_google_maps_api_key  # optional
   ```
   
   See [docs/ENVIRONMENT.md](docs/ENVIRONMENT.md) for details.

3. **Install frontend dependencies:**
   ```bash
   cd frontend
   npm install
   cd ..
   ```

4. **Start Docker services (PostgreSQL & Redis):**
   ```bash
   docker-compose up -d
   ```

5. **Run database migrations:**
   ```bash
   # Install golang-migrate if not already installed
   # macOS: brew install golang-migrate
   # Linux: See https://github.com/golang-migrate/migrate
   
   migrate -path backend/migrations \
           -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
           up
   ```

5. **Start all services:**
   ```bash
   ./scripts/start-all.sh
   ```

6. **Access the application:**
   - Frontend: http://localhost:3000
   - API Gateway: http://localhost:8080
   - **Guest Menu**: http://localhost:3000/menu/{tenant_id}  (NEW - No login required!)
   - **Admin Orders**: http://localhost:3000/admin/orders  (NEW)

### Stop All Services

```bash
./scripts/stop-all.sh
```

## 📁 Project Structure

```
point-of-sale-system/
├── api-gateway/              # API Gateway service
│   ├── middleware/           # JWT auth, tenant scope, rate limiting, CORS, logging
│   └── main.go
├── backend/
│   ├── auth-service/         # Authentication service
│   │   ├── api/              # HTTP handlers
│   │   ├── src/
│   │   │   ├── models/       # Data models
│   │   │   ├── repository/   # Database operations
│   │   │   └── services/     # Business logic
│   │   └── main.go
│   ├── order-service/        # 🆕 Guest ordering service
│   │   ├── api/              # Cart, checkout, payment webhook handlers
│   │   ├── src/
│   │   │   ├── models/       # Order, cart, payment, reservation models
│   │   │   ├── repository/   # Order, payment, reservation repositories
│   │   │   ├── services/     # Cart, payment, inventory, geocoding services
│   │   │   └── middleware/   # Rate limiting
│   │   ├── tests/
│   │   │   ├── contract/     # API schema validation tests
│   │   │   ├── integration/  # End-to-end flow tests
│   │   │   └── unit/         # Service logic tests
│   │   └── main.go
│   ├── tenant-service/       # Tenant management service
│   │   ├── api/
│   │   ├── src/
│   │   └── main.go
│   ├── user-service/         # User management service
│   │   ├── api/
│   │   ├── src/
│   │   └── main.go
│   ├── src/
│   │   ├── config/           # Database & Redis configuration
│   │   ├── i18n/             # Backend translations (EN/ID)
│   │   ├── middleware/       # Shared middleware
│   │   ├── repository/       # Base repository pattern
│   │   └── utils/            # Utilities (password, slug, token, response)
│   └── migrations/           # Database migrations (8 files)
├── frontend/
│   ├── pages/                # Next.js pages (login, signup, 🆕 menu, orders, admin)
│   ├── src/
│   │   ├── components/       # React components
│   │   │   ├── guest/        # 🆕 Guest ordering components (cart, checkout, menu)
│   │   │   └── admin/        # 🆕 Admin order management
│   │   ├── i18n/             # i18n configuration & translations
│   │   ├── services/         # API client, auth service, 🆕 order service
│   │   ├── store/            # State management
│   │   └── utils/            # Validation utilities
│   └── package.json
├── scripts/
│   ├── start-all.sh          # Start all services
│   └── stop-all.sh           # Stop all services
├── docker-compose.yml        # PostgreSQL, Redis, 🆕 Order Service containers
└── specs/                    # Feature specifications and documentation
    ├── 001-guest-qris-ordering/  # 🆕 QRIS guest ordering spec
    │   ├── plan.md
    │   ├── tasks.md
    │   ├── quickstart.md
    │   └── VALIDATION_RESULTS.md
    └── ...
```

## 🌟 Features

### Implemented (Phase 1 & 2 - Foundation)

✅ **Project Setup**
- Microservices architecture with Go backend
- Next.js frontend with TypeScript
- Docker containerization for PostgreSQL & Redis
- Complete i18n support (English & Indonesian)

✅ **Authentication Infrastructure**
- JWT-based authentication
- Session management with Redis
- Password hashing with bcrypt (cost factor 12)
- Rate limiting for login attempts
- Secure token generation for invitations

✅ **Multi-Tenancy**
- Tenant isolation with Row-Level Security (RLS)
- Tenant-scoped queries in all services
- Automatic tenant context injection via middleware

✅ **API Gateway**
- Centralized routing to microservices
- JWT authentication middleware
- Tenant scope middleware
- CORS configuration
- Structured logging
- Rate limiting (login endpoints)

✅ **Database Schema**
- Tenants, Users, Sessions, Invitations tables
- 🆕 Guest Orders, Order Items, Payment Transactions, Inventory Reservations, Delivery Addresses, Tenant Configs
- RLS policies for complete data isolation
- Automatic timestamps and triggers
- Comprehensive indexes for performance

✅ **Frontend**
- Login and Signup pages
- Form validation
- API service layer
- Authentication state management
- Language switcher component
- Protected routes
- 🆕 **Guest ordering flow** (no login required)
- 🆕 **Admin order management dashboard**

### 🎉 NEW: Guest QRIS Ordering System (v1.0.0)

✅ **Session-Based Shopping Cart**
- Add/update/remove items without authentication
- 24-hour cart persistence in Redis
- Real-time inventory validation
- Automatic cart cleanup

✅ **QRIS Payment Integration**
- Midtrans payment gateway
- QR code scanning for instant payment
- Webhook-based status updates
- Idempotent payment processing

✅ **Delivery Options**
- Pickup: Walk-in order collection
- Delivery: Address input with geocoding validation
- Dine-in: Table number assignment
- Distance-based delivery fee calculation

✅ **Inventory Management**
- 15-minute inventory reservations on checkout
- Auto-release on expiration or payment failure
- Convert to permanent on payment success
- Race condition protection with SELECT FOR UPDATE

✅ **Order Tracking**
- Public order status page (no login)
- Auto-refresh for pending/paid orders
- Order history with timestamps
- Print-friendly order confirmation

✅ **Admin Order Management**
- View all orders with filters (status, date)
- Update order status (PENDING → PAID → COMPLETE)
- Add courier tracking notes
- Real-time order count display

✅ **Multi-Tenant Configuration**
- Per-tenant delivery settings
- Service area configuration (radius/polygon)
- Delivery fee rules (distance/zone/flat)
- Minimum order amounts
- Inventory reservation TTL customization

**Documentation**: See [docs/QRIS_GUEST_ORDERING.md](docs/QRIS_GUEST_ORDERING.md)

### In Progress

🚧 **Polish & Production Readiness**
- Input sanitization across all handlers
- Comprehensive monitoring and metrics
- Performance optimization
- Security hardening

### Planned

⏳ **Additional Payment Methods**: GoPay, OVO, Bank Transfer  
⏳ **Guest Order History**: Email-based order lookup  
⏳ **Push Notifications**: Real-time order updates  
⏳ **Promotions & Discounts**: Coupon codes and special offers  
⏳ **Scheduled Orders**: Pre-order for later delivery  
⏳ **Advanced Analytics**: Sales reports and insights

## 🧪 Testing

### Run Backend Tests

```bash
# Unit tests
cd backend/auth-service
go test ./...

# Integration tests (requires Docker)
cd backend/auth-service/tests/integration
go test -v

# Contract tests
cd backend/auth-service/tests/contract
go test -v
```

### Run Frontend Tests

```bash
cd frontend
npm test                  # Run all tests
npm test -- --watch      # Watch mode
npm test -- --coverage   # With coverage
```

### Run Order Service Tests

```bash
cd backend/order-service

# Unit tests (fast, mocked dependencies)
go test ./src/services/... -v

# Integration tests (requires Redis & PostgreSQL)
go test ./tests/integration/... -v

# Contract tests (API schema validation)
go test ./tests/contract/... -v

# All tests
go test ./... -v
```

## 🛒 Guest Ordering Quick Start

### For Customers (No Login Required!)

1. **Browse Menu**: Visit `http://localhost:3000/menu/{tenant_id}`
2. **Add to Cart**: Click items to add, adjust quantities
3. **Checkout**: Choose delivery type (pickup/delivery/dine-in)
4. **Pay**: Scan QRIS code via Midtrans
5. **Track Order**: View status at `http://localhost:3000/orders/{order_reference}`

### For Restaurant Admins

1. **Login**: Use your admin credentials
2. **View Orders**: Visit `http://localhost:3000/admin/orders`
3. **Manage**: Filter by status, update order status, add courier notes
4. **Complete**: Mark orders as COMPLETE when fulfilled

### Test Guest Ordering Flow

```bash
# 1. Get a tenant ID from your database
TENANT_ID=$(docker-compose exec -T postgres psql -U pos_user -d pos_db -t -c "SELECT id FROM tenants LIMIT 1" | tr -d ' \n')

# 2. Get a product ID
PRODUCT_ID=$(docker-compose exec -T postgres psql -U pos_user -d pos_db -t -c "SELECT id FROM products WHERE tenant_id='$TENANT_ID' LIMIT 1" | tr -d ' \n')

# 3. Add item to cart
curl -X POST http://localhost:8084/api/v1/public/$TENANT_ID/cart/items \
  -H "Content-Type: application/json" \
  -H "X-Session-Id: test-session-123" \
  -d "{\"product_id\":\"$PRODUCT_ID\",\"product_name\":\"Test Product\",\"quantity\":2,\"unit_price\":50000}"

# 4. View cart
curl http://localhost:8084/api/v1/public/$TENANT_ID/cart \
  -H "X-Session-Id: test-session-123"

# 5. Run validation script
cd specs/001-guest-qris-ordering
./validate-quickstart.sh
```

### Midtrans Sandbox Testing

1. Set `MIDTRANS_ENVIRONMENT=sandbox` in `backend/order-service/.env`
2. Use Midtrans sandbox credentials
3. Test payment with sandbox QRIS: https://docs.midtrans.com/en/technical-reference/sandbox-test
4. Check webhook notifications in order-service logs

**Documentation**: [docs/QRIS_GUEST_ORDERING.md](docs/QRIS_GUEST_ORDERING.md)  
**Quickstart Guide**: [specs/001-guest-qris-ordering/quickstart.md](specs/001-guest-qris-ordering/quickstart.md)

## 🔐 Security Features

- **Password Security**: bcrypt hashing with cost factor 12
- **Session Management**: HTTP-only cookies, Redis-backed sessions with TTL
- **Multi-Tenancy**: Row-Level Security policies enforce complete data isolation
- **JWT Tokens**: Signed tokens with configurable expiration
- **Rate Limiting**: Login attempt throttling per email/tenant
- **CORS**: Configurable cross-origin resource sharing
- **Input Validation**: Server-side and client-side validation

## 🌍 Internationalization (i18n)

**Supported Languages:**
- English (en)
- Indonesian (id)

**Coverage:**
- All UI text and labels
- Error messages
- Success messages
- Form validation messages
- Authentication flows

## 📊 Database Migrations

Migrations are located in `backend/migrations/` and use the `golang-migrate` tool.

**Apply all migrations:**
```bash
migrate -path backend/migrations \
        -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
        up
```

**Rollback last migration:**
```bash
migrate -path backend/migrations \
        -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
        down 1
```

**Check migration status:**
```bash
migrate -path backend/migrations \
        -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
        version
```

## 🔧 Configuration

### Environment Variables

**API Gateway:**
- `PORT`: Server port (default: 8080)
- `TENANT_SERVICE_URL`: Tenant service URL (default: http://localhost:8081)
- `AUTH_SERVICE_URL`: Auth service URL (default: http://localhost:8082)
- `USER_SERVICE_URL`: User service URL (default: http://localhost:8083)
- `ORDER_SERVICE_URL`: Order service URL (default: http://localhost:8084)  🆕

**Auth Service:**
- `PORT`: Server port (default: 8082)
- `DATABASE_URL`: PostgreSQL connection string
- `REDIS_HOST`: Redis host and port (default: localhost:6379)
- `REDIS_PASSWORD`: Redis password
- `JWT_SECRET`: Secret key for JWT signing (required in production)
- `JWT_EXPIRATION_MINUTES`: JWT token expiration (default: 15)
- `SESSION_TTL_MINUTES`: Session TTL in Redis (default: 15)
- `RATE_LIMIT_LOGIN_MAX`: Max login attempts (default: 5)
- `RATE_LIMIT_LOGIN_WINDOW`: Rate limit window in seconds (default: 900)

**Order Service:** 🆕
- `PORT`: Server port (default: 8084)
- `DATABASE_URL`: PostgreSQL connection string
- `REDIS_URL`: Redis connection string (default: redis://localhost:6379/0)
- `MIDTRANS_SERVER_KEY`: Midtrans server key (required)
- `MIDTRANS_CLIENT_KEY`: Midtrans client key (required)
- `MIDTRANS_ENVIRONMENT`: sandbox or production (default: sandbox)
- `GOOGLE_MAPS_API_KEY`: Google Maps API key for geocoding (optional)
- `CORS_ALLOWED_ORIGINS`: Allowed CORS origins (default: http://localhost:3000)
- `CART_TTL_HOURS`: Cart session TTL (default: 24)
- `INVENTORY_RESERVATION_TTL_MINUTES`: Reservation TTL (default: 15)
- `LOG_LEVEL`: Logging level (default: info)

**Tenant Service:**
- `PORT`: Server port (default: 8081)
- `DATABASE_URL`: PostgreSQL connection string

**User Service:**
- `PORT`: Server port (default: 8083)
- `DATABASE_URL`: PostgreSQL connection string

## 📝 API Documentation

### Health Checks

All services provide health check endpoints:

```bash
# API Gateway
curl http://localhost:8080/health
curl http://localhost:8080/ready

# Individual services
curl http://localhost:8081/health  # Tenant Service
curl http://localhost:8082/health  # Auth Service
curl http://localhost:8083/health  # User Service
```

### Authentication Endpoints

**Tenant Registration:**
```bash
POST http://localhost:8080/api/tenants/register
Content-Type: application/json

{
  "business_name": "My Business",
  "owner_email": "owner@example.com",
  "owner_password": "SecurePassword123!",
  "owner_full_name": "John Doe"
}
```

**User Login:**
```bash
POST http://localhost:8080/api/auth/login
Content-Type: application/json

{
  "email": "owner@example.com",
  "password": "SecurePassword123!",
  "tenant_id": "uuid-here"
}
```

**Get Session (requires JWT):**
```bash
GET http://localhost:8080/api/auth/session
Authorization: Bearer <jwt-token>
```

## 🐛 Troubleshooting

### Docker not running
If you see "Cannot connect to the Docker daemon", start Docker:
```bash
# Linux
sudo systemctl start docker

# macOS
open -a Docker
```

### Database connection failed
Check if PostgreSQL is running:
```bash
docker-compose ps
```

### Redis connection failed
Verify Redis is accessible:
```bash
docker-compose exec redis redis-cli ping
# Should return: PONG
```

### Port already in use
Find and kill the process using the port:
```bash
lsof -ti:8080 | xargs kill -9
```

## 📈 Implementation Status

**Overall Progress:** ~45% Complete

- ✅ Phase 1 (Setup): 100% Complete
- ✅ Phase 2 (Foundation): 100% Complete
- 🚧 Phase 3 (User Story 1): Backend & Frontend Complete, Tests Pending
- 🚧 Phase 4 (User Story 2): Backend & Frontend Complete, Tests Pending
- ⏳ Phases 5-8: Not Started

See `IMPLEMENTATION_STATUS.md` and `IMPLEMENTATION_SUMMARY.md` for detailed progress.

## 📚 Documentation

Detailed documentation is available in the `specs/001-auth-multitenancy/` directory:
- `spec.md`: Feature specification
- `plan.md`: Implementation plan
- `data-model.md`: Database design
- `contracts/`: OpenAPI specifications
- `tasks.md`: Task breakdown

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests first (TDD approach)
4. Implement the feature
5. Submit a pull request

## 📄 License

[Add your license here]

## 👥 Authors

[Add authors/contributors here]
