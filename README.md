# Point of Sale System - Multi-Tenant with User Authentication

A modern, scalable Point of Sale (POS) system with multi-tenancy and comprehensive user authentication features.

## 🏗️ Architecture

**Microservices Architecture:**
- **API Gateway** (Port 8080): Entry point for all client requests, handles routing, authentication, rate limiting
- **Auth Service** (Port 8082): User authentication, session management, JWT token generation
- **Tenant Service** (Port 8081): Tenant registration and management
- **User Service** (Port 8083): User management, invitations
- **Frontend** (Port 3000): Next.js React application with i18n support (EN/ID)

**Data Layer:**
- **PostgreSQL 14**: Primary database with Row-Level Security for tenant isolation
- **Redis 7**: Session storage and rate limiting

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL 14+ (via Docker)
- Redis 7+ (via Docker)

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
   # ... repeat for other services
   ```
   
   ⚠️ **Important:** Review and update the `.env` files with your configuration.
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
│   ├── pages/                # Next.js pages (login, signup)
│   ├── src/
│   │   ├── components/       # React components
│   │   ├── i18n/             # i18n configuration & translations
│   │   ├── services/         # API client & auth service
│   │   ├── store/            # State management
│   │   └── utils/            # Validation utilities
│   └── package.json
├── scripts/
│   ├── start-all.sh          # Start all services
│   └── stop-all.sh           # Stop all services
├── docker-compose.yml        # PostgreSQL & Redis containers
└── specs/                    # Feature specifications and documentation
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

### In Progress

🚧 **User Story 1: Tenant Registration**
- Backend implementation complete
- Frontend implementation complete
- Tests pending (requires Docker)

🚧 **User Story 2: User Login**
- Backend implementation complete
- Frontend implementation complete
- Tests pending (requires Docker)

### Planned

⏳ **User Story 3: User Invitation System**
⏳ **User Story 4: Multi-User Tenant Management**
⏳ **User Story 5: Session Management & Logout**
⏳ **User Story 6: Language Preference**

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
