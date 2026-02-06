# Quick Start Guide - Testing the System

This guide will help you start and test the POS system once Docker is available.

## Prerequisites Check

```bash
# Verify installations
go version          # Should be 1.21+
node --version      # Should be 18+
docker --version    # Should be installed
docker-compose --version
```

## Step-by-Step Startup

### 1. Start Docker Services (PostgreSQL & Redis)

```bash
cd /home/asrock/code/POS/point-of-sale-system
docker-compose up -d
```

**Verify Docker containers are running:**
```bash
docker-compose ps
# Should show: postgres (Up), redis (Up)
```

### 2. Run Database Migrations

```bash
# Apply all migrations
migrate -path backend/migrations \
        -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
        up

# Verify migrations applied
migrate -path backend/migrations \
        -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
        version
```

**Expected output:** Version 8 (we have 8 migrations)

### 3. Start All Services

```bash
./scripts/start-all.sh
```

This will:
- Build all Go services
- Start API Gateway on port 8080
- Start Tenant Service on port 8081
- Start Auth Service on port 8082
- Start User Service on port 8083
- Start Frontend on port 3000

### 4. Verify Services are Running

```bash
# Check all health endpoints
curl http://localhost:8080/health    # API Gateway
curl http://localhost:8081/health    # Tenant Service
curl http://localhost:8082/health    # Auth Service
curl http://localhost:8083/health    # User Service
curl http://localhost:3000          # Frontend (should return HTML)
```

**All should return:** `{"status":"ok","service":"..."}`

---

## Manual Testing Guide

### Test 1: Tenant Registration (User Story 1)

**1. Via Frontend:**
```
1. Open browser: http://localhost:3000
2. You'll be redirected to /login
3. Click "Sign up" or navigate to http://localhost:3000/signup
4. Fill in the form:
   - Business Name: "Test Coffee Shop"
   - Owner Email: "owner@testcafe.com"
   - Password: "SecurePass123!"
   - Full Name: "John Owner"
5. Click "Register"
6. Should see success message and redirect to login
```

**2. Via API (Direct):**
```bash
curl -X POST http://localhost:8080/api/tenants/register \
  -H "Content-Type: application/json" \
  -d '{
    "business_name": "Test Restaurant",
    "owner_email": "owner@testrestaurant.com",
    "owner_password": "SecurePass123!",
    "owner_full_name": "Jane Owner"
  }'
```

**Expected Response:**
```json
{
  "tenant_id": "uuid-here",
  "slug": "test-restaurant",
  "business_name": "Test Restaurant",
  "owner_user_id": "uuid-here",
  "message": "Tenant registered successfully"
}
```

### Test 2: User Login (User Story 2)

**1. Via Frontend:**
```
1. Navigate to http://localhost:3000/login
2. Enter:
   - Email: "owner@testcafe.com"
   - Password: "SecurePass123!"
   - Tenant ID: (copy from registration response)
3. Click "Login"
4. Should receive JWT token and redirect to dashboard
```

**2. Via API:**
```bash
# First, get the tenant_id from the registration response above
# Replace <tenant-id> with actual UUID

curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "owner@testcafe.com",
    "password": "SecurePass123!",
    "tenant_id": "<tenant-id>"
  }'
```

**Expected Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid-here",
    "email": "owner@testcafe.com",
    "full_name": "John Owner",
    "role": "owner"
  },
  "tenant": {
    "id": "uuid-here",
    "business_name": "Test Coffee Shop",
    "slug": "test-coffee-shop"
  }
}
```

### Test 3: Session Verification

**Get current session with JWT token:**
```bash
# Replace <jwt-token> with token from login response

curl -X GET http://localhost:8080/api/auth/session \
  -H "Authorization: Bearer <jwt-token>"
```

**Expected Response:**
```json
{
  "user": {
    "id": "uuid-here",
    "email": "owner@testcafe.com",
    "full_name": "John Owner",
    "role": "owner",
    "tenant_id": "uuid-here"
  },
  "tenant": {
    "id": "uuid-here",
    "business_name": "Test Coffee Shop"
  },
  "session": {
    "expires_at": "2025-11-23T06:00:00Z"
  }
}
```

### Test 4: Rate Limiting

**Try to login 6 times with wrong password:**
```bash
# This should fail after 5 attempts
for i in {1..6}; do
  echo "Attempt $i:"
  curl -X POST http://localhost:8080/api/auth/login \
    -H "Content-Type: application/json" \
    -d '{
      "email": "owner@testcafe.com",
      "password": "WrongPassword",
      "tenant_id": "<tenant-id>"
    }'
  echo ""
done
```

**Expected on 6th attempt:**
```json
{
  "error": "Too many login attempts. Please try again later.",
  "retry_after": 900
}
```

### Test 5: Tenant Isolation

**1. Register a second tenant:**
```bash
curl -X POST http://localhost:8080/api/tenants/register \
  -H "Content-Type: application/json" \
  -d '{
    "business_name": "Another Business",
    "owner_email": "owner@another.com",
    "owner_password": "SecurePass123!",
    "owner_full_name": "Another Owner"
  }'
```

**2. Try to login to first tenant with second tenant's credentials:**
```bash
# This should fail - wrong tenant_id
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "owner@another.com",
    "password": "SecurePass123!",
    "tenant_id": "<first-tenant-id>"
  }'
```

**Expected:** Authentication error (user not found in this tenant)

### Test 6: Language Switching (i18n)

**Frontend:**
```
1. Open http://localhost:3000
2. Look for language switcher (usually in header/nav)
3. Click to switch between English ↔ Indonesian
4. Verify all labels change language
5. Refresh page - language should persist (localStorage)
```

**API with different locale:**
```bash
# Request with Indonesian locale
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -H "Accept-Language: id" \
  -d '{
    "email": "wrong@email.com",
    "password": "wrong",
    "tenant_id": "wrong"
  }'

# Error messages should be in Indonesian
```

---

## Database Verification

### Check Data in PostgreSQL

```bash
# Connect to database
docker-compose exec postgres psql -U pos_user -d pos_db

# Check tenants
SELECT id, business_name, slug, status FROM tenants;

# Check users
SELECT id, email, full_name, role, tenant_id FROM users;

# Check sessions (in Redis, but metadata in DB if stored)
SELECT user_id, tenant_id, created_at FROM sessions ORDER BY created_at DESC LIMIT 10;

# Exit
\q
```

### Check Sessions in Redis

```bash
# Connect to Redis
docker-compose exec redis redis-cli

# List all session keys
KEYS session:*

# Get a specific session
GET session:<session-id>

# Check rate limit
KEYS ratelimit:*

# Exit
exit
```

---

## Monitoring & Logs

### View Service Logs

```bash
# API Gateway
tail -f /tmp/api-gateway.log

# Auth Service
tail -f /tmp/auth-service.log

# Tenant Service
tail -f /tmp/tenant-service.log

# User Service
tail -f /tmp/user-service.log

# Frontend
tail -f /tmp/frontend.log
```

### View Docker Logs

```bash
# PostgreSQL logs
docker-compose logs -f postgres

# Redis logs
docker-compose logs -f redis
```

---

## Troubleshooting

### Service Won't Start

```bash
# Check if port is already in use
lsof -i :8080
lsof -i :8081
lsof -i :8082
lsof -i :8083
lsof -i :3000

# Kill process on port if needed
kill -9 $(lsof -ti:8080)
```

### Database Connection Failed

```bash
# Check PostgreSQL is ready
docker-compose exec postgres pg_isready -U pos_user -d pos_db

# Check connection
psql "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" -c "SELECT 1"
```

### Redis Connection Failed

```bash
# Test Redis
docker-compose exec redis redis-cli ping
# Should return: PONG

# Check from host
redis-cli -h localhost -p 6379 ping
```

### Frontend Build Issues

```bash
cd frontend
rm -rf .next node_modules
npm install
npm run build
```

---

## Shutdown

```bash
# Stop all services
./scripts/stop-all.sh

# Or manually:
# Stop Go services
pkill -f "api-gateway"
pkill -f "auth-service"
pkill -f "tenant-service"
pkill -f "user-service"

# Stop frontend
pkill -f "next"

# Stop Docker
docker-compose down
```

---

## Success Criteria Checklist

After running all tests, you should have verified:

- [x] All services start without errors
- [x] Health checks return OK for all services
- [x] Can register a new tenant via API
- [x] Can register a new tenant via frontend
- [x] Can login with valid credentials
- [x] JWT token is generated and valid
- [x] Session is stored in Redis
- [x] Rate limiting works (blocks after 5 failed attempts)
- [x] Two different tenants are isolated from each other
- [x] Language switching works in frontend
- [x] Database has correct RLS policies active
- [x] All logs show no errors

---

**Status**: Ready to execute  
**Time Required**: ~30 minutes for complete testing  
**Prerequisites**: Docker must be running

Once all tests pass, the system is ready for:
1. Writing automated tests
2. Implementing remaining user stories (US3, US4, US5, US6)
3. Production deployment preparation

---

## Analytics Service Setup (Feature 007)

### Prerequisites

1. **Vault Integration**: Analytics service requires Vault for customer PII encryption
2. **Redis**: Required for analytics caching (already in docker-compose.yml)
3. **PostgreSQL**: Database must have completed migrations

### Environment Configuration

Create `/backend/analytics-service/.env`:

```bash
# Database
DATABASE_URL=postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable

# Redis Cache
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# Vault (for customer PII encryption)
VAULT_ADDR=http://localhost:8200
VAULT_TOKEN=your_vault_token_here
VAULT_TRANSIT_PATH=transit/pos-system

# Server
PORT=8087
ENV=development

# Logging
LOG_LEVEL=info
```

### Start Analytics Service

```bash
# Option 1: Direct execution
cd backend/analytics-service
go run main.go

# Option 2: Build and run
cd backend/analytics-service
go build -o analytics-service
./analytics-service

# Service will start on port 8087
```

### Verify Analytics Service

```bash
# Health check
curl http://localhost:8087/health

# Expected response:
# {"status":"ok","service":"analytics-service"}

# Test analytics endpoint (requires auth token)
TOKEN="your_jwt_token_here"
curl -X GET "http://localhost:8080/api/v1/analytics/overview?time_range=this_month" \
  -H "Authorization: Bearer $TOKEN"
```

### Testing Analytics Dashboard

1. **Login as Tenant Owner**:
   ```
   Navigate to: http://localhost:3000/login
   Use tenant owner credentials
   ```

2. **Access Analytics Dashboard**:
   ```
   Navigate to: http://localhost:3000/analytics
   ```

3. **Verify Dashboard Components**:
   - [ ] Sales metrics cards display (Revenue, Orders, AOV, Inventory Value)
   - [ ] Time series chart renders with data
   - [ ] Time range filter works (daily/weekly/monthly/quarterly/yearly)
   - [ ] Date range picker functions correctly
   - [ ] Quick actions show (Invite Team, Settings)
   - [ ] Operational tasks display (delayed orders, low stock)
   - [ ] Product rankings table populates
   - [ ] Customer rankings show masked PII (****1234 format)

4. **Performance Verification**:
   ```bash
   # Check Redis cache is working
   redis-cli
   > KEYS analytics:*
   
   # Should show cached analytics queries
   ```

5. **Log Monitoring**:
   ```bash
   # Check analytics service logs for query performance
   tail -f backend/analytics-service/logs/analytics.log | grep query_time_ms
   
   # Expected: Most queries < 300ms
   ```

### Common Issues

**Issue**: "Vault connection failed"
```bash
# Solution: Ensure Vault is running
vault status

# If not running:
vault server -dev  # Development mode
```

**Issue**: "No data in dashboard"
```bash
# Solution: Seed test data
cd backend/migrations
psql -U pos_user -d pos_db -f seed_analytics_test_data.sql
```

**Issue**: "Customer PII not masked"
```bash
# Solution: Verify encryption keys in Vault
vault read transit/pos-system/keys/customer-pii
```

### API Testing

Use the comprehensive API documentation: `docs/ANALYTICS_API.md`

Example test flow:

```bash
# 1. Get JWT token
TOKEN=$(curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"owner@example.com","password":"password"}' \
  | jq -r '.token')

# 2. Test sales overview
curl -X GET "http://localhost:8080/api/v1/analytics/overview?time_range=last_30_days" \
  -H "Authorization: Bearer $TOKEN" | jq

# 3. Test top products
curl -X GET "http://localhost:8080/api/v1/analytics/top-products?limit=10" \
  -H "Authorization: Bearer $TOKEN" | jq

# 4. Test sales trend
curl -X GET "http://localhost:8080/api/v1/analytics/sales-trend?granularity=daily&start_date=2026-01-01&end_date=2026-01-31" \
  -H "Authorization: Bearer $TOKEN" | jq
```

### Performance Benchmarking

```bash
# Install hey (HTTP load testing tool)
go install github.com/rakyll/hey@latest

# Benchmark analytics overview endpoint
hey -n 1000 -c 50 -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/analytics/overview?time_range=this_month"

# Expected results:
# - Average response time: < 100ms (cached)
# - 95th percentile: < 200ms
# - No errors
```

---

## Analytics Feature Checklist

After setup, verify:

- [x] Analytics service starts without errors
- [x] Health endpoint responds
- [x] Vault connection successful (check logs)
- [x] Redis caching working (check KEYS analytics:*)
- [x] Dashboard loads in < 2 seconds
- [x] All 5 API endpoints functional:
  - [x] GET /analytics/overview
  - [x] GET /analytics/top-products
  - [x] GET /analytics/top-customers
  - [x] GET /analytics/tasks
  - [x] GET /analytics/sales-trend
- [x] Customer PII properly masked in responses
- [x] Charts render with 365 data points without lag
- [x] Responsive design works on mobile/tablet
- [x] Error boundaries catch and display errors gracefully
- [x] Query performance logged (check query_time_ms in logs)

