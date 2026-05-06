# Development Setup & Management Scripts

This directory contains scripts to setup and manage POS system development environment.

## 🚀 Quick Start (New Developers)

For first time setup on a new device:

```bash
./scripts/quick-setup.sh
```

This provides an interactive menu to choose setup options.

---

## Setup Scripts

### `quick-setup.sh` 🎯 (Recommended for First Time)

Interactive setup wizard for development environment.

**Usage:**

```bash
./scripts/quick-setup.sh
```

**Menu Options:**

1. Full setup (recommended) - Docker + Vault + Migrations
2. Setup without Docker - Manage Docker separately
3. Setup without Vault - Lightweight development
4. View setup options - See CLI commands
5. Reset everything - Clean start
6. Exit

**Use this if:** First time setup, unsure which options to use

---

### `setup-dev-environment.sh` ⚙️ (Main Setup)

Automated setup script orchestrating all components.

**Usage:**

```bash
# Full setup (default)
./scripts/setup-dev-environment.sh

# Skip Docker infrastructure
./scripts/setup-dev-environment.sh --skip-docker

# Skip Vault setup
./scripts/setup-dev-environment.sh --skip-vault

# Skip both
./scripts/setup-dev-environment.sh --skip-docker --skip-vault
```

**What it does:**

- Verify prerequisites (Docker, docker-compose, openssl)
- Generate Vault TLS certificates
- Create all .env files from examples
- Start Docker containers
- Initialize Vault with encryption keys
- Run database migrations
- Initialize MinIO buckets

**Use this if:** Scriptable setup, CI/CD automation, know what you need

---

### `dev-helper.sh` 🔧 (Daily Development)

Helper utility for development environment management.

**Usage:**

```bash
# Show environment status
./scripts/dev-helper.sh status

# View Vault credentials
./scripts/dev-helper.sh vault-token

# Set Vault CLI environment
source <(./scripts/dev-helper.sh vault-login)

# Check all services healthy
./scripts/dev-helper.sh check-services

# View service logs
./scripts/dev-helper.sh logs postgres
./scripts/dev-helper.sh logs vault
./scripts/dev-helper.sh logs redis

# Restart a service
./scripts/dev-helper.sh restart vault

# Rebuild containers
./scripts/dev-helper.sh rebuild

# Clean up (stop containers, remove volumes)
./scripts/dev-helper.sh clean

# Verify encryption setup
./scripts/dev-helper.sh verify-encryption

# Verify environment configuration
./scripts/dev-helper.sh verify-env

# Help
./scripts/dev-helper.sh help
```

**Common tasks:**

```bash
# Check everything is running
./scripts/dev-helper.sh status

# Debug postgres issue
./scripts/dev-helper.sh logs postgres

# Restart vault after config change
./scripts/dev-helper.sh restart vault

# Access Vault CLI
source <(./scripts/dev-helper.sh vault-login)
vault status
```

**Use this for:** Daily development, troubleshooting, checking status

---

### `generate-env-values.sh` 📝 (Custom Configuration)

Generate .env files with custom values.

**Usage:**

```bash
./scripts/generate-env-values.sh
```

**Options:**

1. Generate all with custom values (interactive prompts)
2. Only root .env (advanced)
3. Only vault .env (advanced)
4. Use defaults (quick)

**Prompts include:**

- Database host, port, user, password
- Redis configuration
- Kafka broker
- Service ports
- JWT secret (auto-generate or custom)
- Environment (development/staging/production)

**Use this if:** Need custom environment values, specific ports, custom JWT secret

---

## Original Scripts

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

**Note:** This is called automatically by setup-dev-environment.sh

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

Starts local development services using their `.env` configuration files.

**Usage:**

```bash
./scripts/start-all.sh
```

**What it does:**

1. Loads environment variables from root `.env`
2. Checks all service `.env` files exist
3. Starts PostgreSQL, Redis, Kafka, MinIO, and MailHog in Docker
4. Runs database migrations
5. Builds all Go services
6. Starts each service with its own `.env` file:
   - API Gateway (port 8080)
   - Auth Service (port 8082)
   - User Service (port 8083)
   - Tenant Service (port 8084)
   - Notification Service (port 8085)
   - Product Service (port 8086)
   - Order Service (port 8087)
   - Audit Service (port 8088)
   - Analytics Service (port 8089)
7. Starts the Next.js frontend (port 3000)
8. Stores PIDs in `/tmp/pos-services.pid`
9. Creates log files in `/tmp/`

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
- `/tmp/analytics-service.log`
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
- Analytics Service: `${ANALYTICS_SERVICE_PORT:-8089}`
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
- `/tmp/analytics-service.log`
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
vim backend/analytics-service/.env
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

- `.env` (root)
- `api-gateway/.env`
- `backend/auth-service/.env`
- `backend/user-service/.env`
- `backend/tenant-service/.env`
- `backend/notification-service/.env`
- `backend/product-service/.env`
- `backend/order-service/.env`
- `backend/audit-service/.env`
- `backend/analytics-service/.env`
- `observability/.env`
- `vault/.env`
- `frontend/.env.local`

Run `./scripts/setup-env.sh` to create all files.

## Updated Workflow

### First Time Setup (Recommended)

```bash
# 1. Interactive setup (choose your preferred setup mode)
./scripts/quick-setup.sh

# Or if you know exactly what you want:
./scripts/setup-dev-environment.sh

# 2. Verify everything is set up
./scripts/dev-helper.sh status

# 3. Check all services are healthy
./scripts/dev-helper.sh check-services
```

### Daily Development

```bash
# 1. Start services
./scripts/start-all.sh

# 2. Code and develop (services auto-reload)

# 3. Check logs if needed
./scripts/dev-helper.sh logs <service>

# 4. Restart service if config changed
./scripts/dev-helper.sh restart <service>

# 5. Stop when done
./scripts/stop-all.sh
```

### Troubleshooting

```bash
# Check environment status
./scripts/dev-helper.sh status

# View service logs
./scripts/dev-helper.sh logs postgres
./scripts/dev-helper.sh logs vault
./scripts/dev-helper.sh logs redis

# Verify configuration
./scripts/dev-helper.sh verify-env
./scripts/dev-helper.sh verify-encryption

# Health check
./scripts/dev-helper.sh check-services

# Clean and restart
./scripts/dev-helper.sh clean
./scripts/setup-dev-environment.sh
```

### Reset Environment

```bash
# Option 1: Interactive reset
./scripts/quick-setup.sh
# Choose option 5: Reset everything

# Option 2: Manual reset
./scripts/dev-helper.sh clean
rm -rf vault/tls vault/data
./scripts/setup-dev-environment.sh
```

## Notes

- `quick-setup.sh` and `setup-dev-environment.sh` handle everything (Docker, Vault, migrations)
- Use `dev-helper.sh` for day-to-day development tasks
- All .env files are created automatically from .env.example
- Vault TLS certificates are generated automatically
- Vault is initialized with encryption keys for development
- Use `./scripts/stop-all.sh` to cleanly stop all services
- Docker services (PostgreSQL, Redis, Vault) managed via `docker-compose`

## Tips & Best Practices

### 1. Vault Credentials

```bash
# Always backup vault/data/init-keys.txt
# This file contains the root token and unseal keys
# Keep it secure - don't share with others
cp vault/data/init-keys.txt ~/backups/vault-keys-backup.txt
```

### 2. Check Status Frequently

```bash
# During development, check status regularly
./scripts/dev-helper.sh status

# This shows:
# - Running Docker containers
# - Created .env files
# - TLS certificate status
# - Vault initialization status
```

### 3. View Logs

```bash
# Most common debugging command
./scripts/dev-helper.sh logs postgres

# Or check all logs
docker-compose logs -f
```

### 4. Environment Variables

```bash
# Review what's configured
cat .env
cat vault/.env
cat api-gateway/.env

# Don't commit .env files
# Use .env.example as template only
```

### 5. Fresh Start

```bash
# Complete reset when something breaks
./scripts/dev-helper.sh clean    # Stop & remove volumes
rm -rf vault/tls vault/data      # Remove TLS & data
./scripts/quick-setup.sh         # Full setup from scratch
```

## Configuration Files

Auto-generated during setup:

- ✅ `.env` (root)
- ✅ `vault/.env`
- ✅ `api-gateway/.env`
- ✅ `backend/auth-service/.env`
- ✅ `backend/user-service/.env`
- ✅ `backend/tenant-service/.env`
- ✅ `backend/notification-service/.env`
- ✅ `backend/product-service/.env`
- ✅ `backend/order-service/.env`
- ✅ `backend/audit-service/.env`
- ✅ `backend/analytics-service/.env`
- ✅ `backend/billing-service/.env`

TLS certificates auto-generated:

- ✅ `vault/tls/ca.key`
- ✅ `vault/tls/ca.crt`
- ✅ `vault/tls/vault.key`
- ✅ `vault/tls/vault.crt`

## See Also

- [DEVELOPMENT_SETUP.md](./DEVELOPMENT_SETUP.md) - Comprehensive setup guide
- [Environment Configuration](../docs/ENVIRONMENT.md) - Detailed variable documentation
- [Backend Conventions](../docs/BACKEND_CONVENTIONS.md) - Backend best practices
- [Vault Token Management](../vault/VAULT_TOKEN_MANAGEMENT.md) - Vault management
- [README](../README.md) - Project overview
