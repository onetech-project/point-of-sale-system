# Start/Stop Scripts

This directory contains scripts to manage POS system services.

## Scripts

### `setup-env.sh`
Creates all `.env` files from `.env.example` templates.

**Usage:**
```bash
./scripts/setup-env.sh
```

**What it does:**
- Copies `.env.example` to `.env` for all services
- Checks if files already exist (won't overwrite)
- Provides guidance on next steps

### `verify-env.sh`
Verifies environment configuration is correct.

**Usage:**
```bash
./scripts/verify-env.sh
```

**What it checks:**
- All `.env` files exist (including audit-service, observability, vault)
- JWT_SECRET is consistent across services
- Database configuration is set
- Redis configuration is set
- Vault configuration is set
- Frontend API URL is configured

### `start-all.sh`
Starts all backend services using their `.env` configuration files.

**Usage:**
```bash
./scripts/start-all.sh
```

**What it does:**
1. Loads environment variables from root `.env`
2. Checks all service `.env` files exist
3. Builds all Go services
4. Starts each service with its own `.env` file:
   - API Gateway (port 8080)
   - Auth Service (port 8082)
   - User Service (port 8083)
   - Tenant Service (port 8084)
   - Notification Service (port 8085)
   - Product Service (port 8086)
   - Order Service (port 8087)
   - Audit Service (port 8088)
5. Stores PIDs in `/tmp/pos-services.pid`
6. Creates log files in `/tmp/`

**Environment Loading Order:**
1. Root `.env` (global defaults)
2. Service-specific `.env` (overrides root)
3. System environment variables (highest priority)

**Logs:**
- `/tmp/api-gateway.log`
- `/tmp/auth-service.log`
- `/tmp/user-service.log`
- `/tmp/tenant-service.log`
- `/tmp/notification-service.log`
- `/tmp/product-service.log`
- `/tmp/order-service.log`
- `/tmp/audit-service.log`
- `/tmp/frontend.log`

### `stop-all.sh`
Stops all running services and optionally cleans up log files.

**Usage:**
```bash
./scripts/stop-all.sh
```

**What it does:**
1. Loads port configuration from root `.env` file
2. Stops services using stored PIDs from `/tmp/pos-services.pid`
3. Falls back to stopping by port (uses environment variables or defaults)
4. Stops Next.js processes by name
5. Removes Next.js lock file
6. Prompts to remove log files (with 10-second timeout)
7. Shows summary of stopped services

**Ports stopped (from .env or defaults):**
- API Gateway: `${API_GATEWAY_PORT:-8080}`
- Auth Service: `${AUTH_SERVICE_PORT:-8082}`
- User Service: `${USER_SERVICE_PORT:-8083}`
- Tenant Service: `${TENANT_SERVICE_PORT:-8084}`
- Notification Service: `${NOTIFICATION_SERVICE_PORT:-8085}`
- Product Service: `${PRODUCT_SERVICE_PORT:-8086}`
- Order Service: `${ORDER_SERVICE_PORT:-8087}`
- Audit Service: `${AUDIT_SERVICE_PORT:-8088}`
- Frontend: `${FRONTEND_PORT:-3000}`

**Log files managed:**
- `/tmp/api-gateway.log`
- `/tmp/auth-service.log`
- `/tmp/user-service.log`
- `/tmp/tenant-service.log`
- `/tmp/notification-service.log`
- `/tmp/product-service.log`
- `/tmp/order-service.log`
- `/tmp/audit-service.log`
- `/tmp/frontend.log`

## Typical Workflow

### Initial Setup
```bash
# 1. Create environment files
./scripts/setup-env.sh

# 2. Review and update .env files
vim .env
vim api-gateway/.env
vim backend/auth-service/.env
vim backend/user-service/.env
vim backend/tenant-service/.env
vim backend/notification-service/.env
vim backend/product-service/.env
vim backend/order-service/.env
vim backend/audit-service/.env
vim observability/.env
vim vault/.env
vim frontend/.env.local

# 3. Verify configuration
./scripts/verify-env.sh
```

### Daily Development
```bash
# Start services
./scripts/start-all.sh

# Check logs
tail -f /tmp/auth-service.log

# Test endpoints
curl http://localhost:8080/health

# Stop services
./scripts/stop-all.sh
```

### Troubleshooting

#### Services won't start
```bash
# Check .env files exist
./scripts/verify-env.sh

# Check for port conflicts
lsof -i :8080
lsof -i :8082

# View service logs
tail -f /tmp/api-gateway.log
tail -f /tmp/auth-service.log
```

#### JWT token validation fails
```bash
# Verify JWT_SECRET is consistent
grep JWT_SECRET .env
grep JWT_SECRET api-gateway/.env
grep JWT_SECRET backend/auth-service/.env

# They should all match!
```

#### Database connection issues
```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Check DATABASE_URL in service .env
grep DATABASE_URL backend/auth-service/.env
```

## Environment Variables

Services load environment variables in this priority order:

1. **System environment** (highest)
2. **Service .env file** (middle)
3. **Root .env file** (lowest)
4. **Code defaults** (fallback)

Example:
```bash
# Root .env
JWT_SECRET=root-secret

# backend/auth-service/.env
JWT_SECRET=auth-secret

# System
export JWT_SECRET=system-secret

# Auth service will use: system-secret
```

## Configuration Files Required

Before running `start-all.sh`, ensure these exist:

- ✅ `.env` (root)
- ✅ `api-gateway/.env`
- ✅ `backend/auth-service/.env`
- ✅ `backend/user-service/.env`
- ✅ `backend/tenant-service/.env`
- ✅ `backend/notification-service/.env`
- ✅ `backend/product-service/.env`
- ✅ `backend/order-service/.env`
- ✅ `backend/audit-service/.env`
- ✅ `observability/.env`
- ✅ `vault/.env`
- ✅ `frontend/.env.local`

Run `./scripts/setup-env.sh` to create all files.

## Notes

- Services must be built before starting (script handles this)
- Each service runs in background with logs to `/tmp/`
- Frontend is not started automatically (run `npm run dev` separately)
- Docker services (PostgreSQL, Redis) must be started separately
- Use `./scripts/stop-all.sh` to cleanly stop all services

## See Also

- [Environment Configuration](../docs/ENVIRONMENT.md) - Detailed variable documentation
- [Development Guide](../docs/development.md) - Development workflow
- [README](../README.md) - Project overview
