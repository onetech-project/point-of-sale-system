# Quick Start Guide: User Authentication and Multi-Tenancy

**Feature**: User Authentication and Multi-Tenancy  
**Branch**: `001-auth-multitenancy`  
**Last Updated**: 2025-11-22

## Overview

This guide helps developers quickly understand and work with the authentication and multi-tenancy feature. It covers the architecture, local setup, testing, and common development workflows.

---

## Architecture Overview

### System Components

```
┌─────────────┐
│   Next.js   │  Frontend (port 3000)
│   Frontend  │  - Login/Signup UI
└──────┬──────┘  - Auth state management
       │         - HTTP-only cookie handling
       │
       │ HTTPS (cookies)
       │
┌──────▼──────────────────────────────────────────┐
│            API Gateway (port 8000)              │
│  - JWT validation                               │
│  - Rate limiting                                │
│  - Tenant context injection                     │
│  - Service routing                              │
└───┬──────────────┬──────────────┬───────────────┘
    │              │              │
    │              │              │
┌───▼────────┐ ┌───▼────────┐ ┌──▼─────────┐
│   Auth     │ │   User     │ │  Tenant    │
│  Service   │ │  Service   │ │  Service   │
│ (port 8001)│ │ (port 8002)│ │ (port 8003)│
│            │ │            │ │            │
│ - Login    │ │ - User CRUD│ │ - Register │
│ - Sessions │ │ - Invites  │ │ - Settings │
│ - Password │ │ - Roles    │ │ - Status   │
└─────┬──────┘ └─────┬──────┘ └──────┬─────┘
      │              │               │
      └──────┬───────┴───────┬───────┘
             │               │
    ┌────────▼────────┐  ┌───▼────────┐
    │   PostgreSQL    │  │   Redis    │
    │   (port 5432)   │  │ (port 6379)│
    │                 │  │            │
    │ - tenants       │  │ - sessions │
    │ - users         │  │ - rate     │
    │ - sessions      │  │   limits   │
    │ - invitations   │  │            │
    └─────────────────┘  └────────────┘
```

### Request Flow

**1. Login Request**:
```
User enters credentials → Next.js frontend → API Gateway
  → Gateway validates format
  → Gateway checks rate limit (Redis)
  → Gateway proxies to Auth Service
  → Auth Service queries PostgreSQL for user
  → Auth Service validates password (bcrypt)
  → Auth Service creates session in Redis (15-min TTL)
  → Auth Service creates session audit in PostgreSQL
  → Auth Service signs JWT with session ID
  → Auth Service returns JWT
  → Gateway sets HTTP-only cookie
  → Frontend receives success + user info
```

**2. Authenticated Request**:
```
User action → Next.js (cookie auto-sent) → API Gateway
  → Gateway validates JWT signature
  → Gateway checks session in Redis
  → Gateway extracts tenantId, userId, role
  → Gateway injects context to request
  → Gateway proxies to backend service
  → Service queries PostgreSQL with tenant_id filter
  → Service returns tenant-scoped data
  → Gateway proxies response to frontend
```

**3. Language Switch Flow**:
```
User clicks language switcher (EN/ID) → React updates i18n context
  → Updates localStorage for persistence
  → If authenticated: calls PATCH /users/{userId}/locale
  → User Service updates users.locale in PostgreSQL
  → UI re-renders with new translations
  → Subsequent API requests include Accept-Language header
  → Backend returns error messages in user's locale
```

### Language Support (i18n)

The system supports **English (EN)** and **Indonesian (ID)** languages:

**Frontend**:
- Language switcher component in navigation bar
- Translations stored in `frontend/locales/{en,id}/*.json`
- Uses `react-i18next` for seamless language switching
- Locale persisted in localStorage (guests) and database (authenticated users)
- Initial locale detection: User DB → localStorage → Browser → Default (EN)

**Backend**:
- API error messages localized based on `Accept-Language` header
- User locale preference stored in `users.locale` column
- Supports locale update via `PATCH /users/{userId}/locale` endpoint
- Translation files in `backend/locales/{en,id}.json`

**Example Usage**:
```jsx
// Frontend component
import { useTranslation } from 'react-i18next';

function LoginPage() {
  const { t, i18n } = useTranslation();
  
  return (
    <form>
      <h1>{t('auth.login.title')}</h1>
      <input placeholder={t('auth.login.email')} />
      <button>{t('auth.login.submit')}</button>
    </form>
  );
}
```

---

## Local Development Setup

### Prerequisites

- **Go**: 1.21 or later
- **Node.js**: 18 or later
- **Docker**: For PostgreSQL and Redis
- **Docker Compose**: For orchestration

### 1. Start Dependencies

```bash
# From repository root
docker-compose up -d postgresql redis

# Verify services are running
docker ps
```

**PostgreSQL**: `localhost:5432`  
**Redis**: `localhost:6379`

### 2. Start Backend Services

**Terminal 1 - Auth Service**:
```bash
cd backend/auth-service
go mod download
go run main.go
# Listening on :8001
```

**Terminal 2 - User Service**:
```bash
cd backend/user-service
go mod download
go run main.go
# Listening on :8002
```

**Terminal 3 - Tenant Service**:
```bash
cd backend/tenant-service
go mod download
go run main.go
# Listening on :8003
```

**Terminal 4 - API Gateway**:
```bash
cd api-gateway
go mod download
go run main.go
# Listening on :8000
```

### 3. Start Frontend

**Terminal 5 - Next.js**:
```bash
cd frontend
npm install
npm run dev
# Listening on :3000
```

### 4. Verify Setup

Open browser to: `http://localhost:3000`

Check health endpoints:
- Gateway: `http://localhost:8000/health`
- Auth Service: `http://localhost:8001/health`
- User Service: `http://localhost:8002/health`
- Tenant Service: `http://localhost:8003/health`

---

## Testing

### Running Tests

**Contract Tests** (validate OpenAPI specs):
```bash
cd backend/auth-service
go test ./tests/contract -v

cd backend/user-service
go test ./tests/contract -v

cd backend/tenant-service
go test ./tests/contract -v
```

**Unit Tests**:
```bash
# Backend services
cd backend/auth-service
go test ./... -v

# Frontend
cd frontend
npm test
```

**Integration Tests** (requires Docker):
```bash
# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
cd backend/auth-service
go test ./tests/integration -v

# Cleanup
docker-compose -f docker-compose.test.yml down
```

### Manual Testing Workflow

**1. Register New Tenant**:
```bash
curl -X POST http://localhost:8000/api/tenants/register \
  -H "Content-Type: application/json" \
  -d '{
    "businessName": "Test Coffee Shop",
    "slug": "test-coffee",
    "ownerEmail": "owner@test.com",
    "ownerPassword": "SecurePass123",
    "ownerProfile": {
      "firstName": "John",
      "lastName": "Doe"
    }
  }'
```

Response includes `tenant.id` (save this as `TENANT_ID`).

**2. Login**:
```bash
curl -X POST http://localhost:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -c cookies.txt \
  -d '{
    "email": "owner@test.com",
    "password": "SecurePass123",
    "tenantId": "<TENANT_ID>"
  }'
```

This saves the JWT cookie to `cookies.txt`.

**3. Get Session Info** (authenticated):
```bash
curl -X GET http://localhost:8000/api/auth/session \
  -b cookies.txt
```

**4. List Users** (authenticated):
```bash
curl -X GET http://localhost:8000/api/users \
  -b cookies.txt
```

**5. Invite Team Member** (authenticated):
```bash
curl -X POST http://localhost:8000/api/invitations \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "email": "manager@test.com",
    "role": "manager"
  }'
```

**6. Logout**:
```bash
curl -X POST http://localhost:8000/api/auth/logout \
  -b cookies.txt
```

---

## Common Development Tasks

### Adding New Endpoint

**1. Update OpenAPI Contract** (`specs/001-auth-multitenancy/contracts/<service>.yaml`)

**2. Write Contract Test**:
```go
// tests/contract/endpoint_test.go
func TestNewEndpoint(t *testing.T) {
    // Load OpenAPI spec
    // Validate request/response schemas
}
```

**3. Write Unit Tests**:
```go
// services/myservice_test.go
func TestMyServiceLogic(t *testing.T) {
    // Test business logic in isolation
}
```

**4. Implement Handler**:
```go
// api/handlers.go
func HandleNewEndpoint(c echo.Context) error {
    // Implementation
}
```

**5. Register Route**:
```go
// main.go
e.GET("/new-endpoint", HandleNewEndpoint, authMiddleware)
```

**6. Run Tests**:
```bash
go test ./... -v
```

### Debugging Authentication Issues

**Check JWT Token**:
```bash
# Extract token from cookie
echo "<token>" | base64 -d

# Decode JWT (use jwt.io or jwt-cli)
```

**Check Redis Session**:
```bash
redis-cli
> KEYS session:*
> GET session:<sessionId>
> TTL session:<sessionId>
```

**Check PostgreSQL User**:
```bash
psql -h localhost -U postgres -d pos
> SELECT * FROM users WHERE email = 'user@test.com';
```

**Check Rate Limit**:
```bash
redis-cli
> KEYS ratelimit:*
> GET ratelimit:login:user@test.com:<tenantId>
```

### Testing Tenant Isolation

**Verify Cross-Tenant Protection**:
```bash
# Login as Tenant A user
curl -X POST http://localhost:8000/api/auth/login \
  -c cookies_a.txt -d '{...tenant A...}'

# Try to access Tenant B data (should fail)
curl -X GET http://localhost:8000/api/users?tenantId=<TENANT_B_ID> \
  -b cookies_a.txt
# Expected: 403 Forbidden
```

**Check PostgreSQL Queries Include tenant_id**:
```bash
# Enable query logging in postgresql.conf
log_statement = 'all'

# Perform operations, then check logs
tail -f /var/log/postgresql/postgresql-*.log
# Verify all queries include WHERE tenant_id = ...
```

---

## Environment Configuration

### Backend Services (`.env`)

```env
# Database
DATABASE_URL=postgresql://postgres:password@localhost:5432/pos?sslmode=disable
DATABASE_MAX_CONNECTIONS=25

# Redis
REDIS_HOST=localhost:6379
REDIS_PASSWORD=

# JWT
JWT_SECRET=<generate-secure-secret>
JWT_EXPIRATION=900  # 15 minutes

# Service
PORT=8001
LOG_LEVEL=debug

# Rate Limiting
RATE_LIMIT_LOGIN=5
RATE_LIMIT_WINDOW=900  # 15 minutes
```

### API Gateway (`.env`)

```env
# Upstream Services
AUTH_SERVICE_URL=http://localhost:8001
USER_SERVICE_URL=http://localhost:8002
TENANT_SERVICE_URL=http://localhost:8003

# Redis (for rate limiting)
REDIS_HOST=localhost:6379

# JWT
JWT_SECRET=<same-as-services>

# Service
PORT=8000
LOG_LEVEL=debug
```

### Frontend (`.env.local`)

```env
# API Gateway
NEXT_PUBLIC_API_URL=http://localhost:8000/api

# App Config
NEXT_PUBLIC_APP_NAME=POS System
```

---

## Troubleshooting

### "Cannot connect to PostgreSQL"
```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Check connection
psql -h localhost -U postgres -c "SELECT version();"
```

### "Session not found in Redis"
```bash
# Check Redis is running
redis-cli ping
# Expected: PONG

# Check session exists
redis-cli
> KEYS session:*
```

### "CORS errors in browser"
Gateway must include CORS middleware:
```go
e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins: []string{"http://localhost:3000"},
    AllowCredentials: true,
}))
```

### "Cookie not sent with requests"
Frontend must include credentials:
```javascript
fetch(url, {
  credentials: 'include'  // Important!
})
```

### "Cross-tenant data leak"
Service layer MUST inject tenant_id filter:
```go
// Set RLS context variable
_, err := db.Exec("SET app.current_tenant_id = $1", tenantID)

// Query automatically filtered by RLS
rows, err := db.Query("SELECT * FROM users WHERE status = 'active'")
// RLS ensures only tenant's users returned
```

---

## API Reference

Full API documentation available in:
- [Auth Service API](./contracts/auth-service.yaml)
- [User Service API](./contracts/user-service.yaml)
- [Tenant Service API](./contracts/tenant-service.yaml)
- [API Gateway](./contracts/api-gateway.yaml)

---

## Next Steps

After completing Phase 1 (design):
1. Run `/speckit.tasks` to generate implementation tasks
2. Implement backend services (test-first)
3. Implement frontend components
4. Integration testing
5. Deploy to staging environment

For implementation details, see `tasks.md` (generated in Phase 2).
