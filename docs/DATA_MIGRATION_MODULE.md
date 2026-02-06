# Data Migration Module - Implementation Summary

## Overview

Created an independent Go module for encrypting existing plaintext PII data using Vault Transit Engine. This module is designed for VPS deployment and can run as a standalone binary or Docker container.

## Architecture

**Module Location**: `scripts/data-migration/`

**Design Principles**:
- **Independent**: No dependencies on service packages
- **Self-contained**: Own configuration and Vault client
- **CLI-based**: Single entry point with argument-based routing
- **Docker-ready**: Can build and run as container
- **Idempotent**: Safe to run multiple times

## Module Structure

```
scripts/data-migration/
├── go.mod                      # Independent module: github.com/pos/data-migration
├── go.sum                      # Dependency lock file
├── config.go                   # Configuration loader and VaultClient
├── main.go                     # CLI entry point with -type flag
├── migrate_users.go            # User PII encryption logic
├── migrate_guest_orders.go     # Guest order PII encryption logic
├── migrate_tenant_configs.go   # Tenant credentials encryption logic
├── Dockerfile                  # Multi-stage build (golang:1.21-alpine → alpine:latest)
├── .gitignore                  # Ignores data-migration binary
└── README.md                   # Comprehensive deployment guide
```

## Key Features

### config.go - Configuration & Vault Client

```go
type Config struct {
    DatabaseURL      string
    VaultAddr        string
    VaultToken       string
    VaultTransitKey  string
}

type VaultClient struct {
    client     *vault.Client
    transitKey string
}

func LoadConfig() (*Config, error)
func NewVaultClient(cfg *Config) (*VaultClient, error)
func (vc *VaultClient) Encrypt(plaintext string) (string, error)
func (vc *VaultClient) Decrypt(ciphertext string) (string, error)
```

**Features**:
- Environment variable validation
- Singleton Vault client pattern
- Base64 encoding/decoding for binary-safe operations
- Error handling with context

### main.go - CLI Entry Point

```go
Usage: go run main.go -type=<migration-type>

Available types:
  users           - Encrypt user PII (email, first_name, last_name)
  guest-orders    - Encrypt guest order PII (customer_name, phone, email, ip_address)
  tenant-configs  - Encrypt tenant payment credentials (midtrans keys)
  all             - Run all migrations sequentially
```

**Features**:
- Flag-based argument parsing
- Migration type routing
- Sequential execution for "all" type
- Usage help message

### Migration Scripts

#### migrate_users.go
- **Target**: `users` table
- **Fields**: `email`, `first_name`, `last_name`
- **Filter**: Skip already encrypted (email LIKE 'vault:v1:%')
- **Batch Size**: 100 records

#### migrate_guest_orders.go
- **Target**: `guest_orders` table
- **Fields**: `customer_name`, `customer_phone`, `customer_email`, `ip_address`
- **Filter**: 
  - Skip already encrypted (customer_name LIKE 'vault:v1:%')
  - Only non-anonymized orders (is_anonymized = FALSE)
- **Batch Size**: 100 records

#### migrate_tenant_configs.go
- **Target**: `tenant_configs` table
- **Fields**: `midtrans_server_key`, `midtrans_client_key`
- **Filter**: Skip already encrypted (key LIKE 'vault:v1:%')
- **Special**: Handles NULL values, preserves already-encrypted keys
- **Batch Size**: 100 records

## Safety Features

1. **Idempotency**: Already encrypted values (starting with `vault:v1:`) are skipped
2. **Batch Processing**: Records processed in batches of 100 for memory efficiency
3. **Transaction Safety**: Each batch committed separately
4. **Null Handling**: NULL values preserved without encryption
5. **Guest Order Filter**: Only encrypts non-anonymized guest orders

## Deployment Options

### Option 1: Local Development

```bash
cd scripts/data-migration
go run main.go -type=users
go run main.go -type=all
```

### Option 2: Build Binary

```bash
cd scripts/data-migration
go build -o data-migration .
./data-migration -type=users
./data-migration -type=all
```

### Option 3: Docker Container

```bash
# Build image
cd scripts/data-migration
docker build -t pos-data-migration:latest .

# Run migration
docker run --rm \
  --network pos-network \
  -e DATABASE_URL="postgresql://user:pass@postgres:5432/dbname" \
  -e VAULT_ADDR="http://vault:8200" \
  -e VAULT_TOKEN="your-vault-token" \
  -e VAULT_TRANSIT_KEY="pos-encryption-key" \
  pos-data-migration -type=all
```

### Option 4: VPS with docker-compose

Add to `docker-compose.yml`:

```yaml
services:
  data-migration:
    build:
      context: ./scripts/data-migration
      dockerfile: Dockerfile
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - VAULT_ADDR=${VAULT_ADDR}
      - VAULT_TOKEN=${VAULT_TOKEN}
      - VAULT_TRANSIT_KEY=${VAULT_TRANSIT_KEY}
    networks:
      - pos-network
    profiles:
      - migration  # Only run when explicitly called
```

Run with:

```bash
docker-compose run --rm data-migration -type=all
```

## Dependencies

```go
require (
    github.com/hashicorp/vault/api v1.10.0  // Vault Transit Engine
    github.com/lib/pq v1.10.9               // PostgreSQL driver
)
```

**Note**: No dependencies on service packages (user-service, order-service, tenant-service)

## Verification

After running migrations, verify encryption coverage:

```bash
# Check users encryption (should return 0)
psql $DATABASE_URL -c "SELECT COUNT(*) FROM users WHERE email NOT LIKE 'vault:v1:%';"

# Check guest orders encryption (should return 0)
psql $DATABASE_URL -c "SELECT COUNT(*) FROM guest_orders WHERE is_anonymized = FALSE AND customer_name NOT LIKE 'vault:v1:%';"

# Check tenant configs encryption (should return 0)
psql $DATABASE_URL -c "SELECT COUNT(*) FROM tenant_configs WHERE midtrans_server_key IS NOT NULL AND midtrans_server_key NOT LIKE 'vault:v1:%';"
```

## Production Best Practices

1. **Backup First**: Always backup database before running migrations
2. **Test in Staging**: Verify migrations work in staging environment
3. **Monitor Vault**: Ensure Vault has sufficient capacity for encryption operations
4. **Run During Off-Peak**: Schedule migrations during low-traffic periods
5. **Verify Environment**: Double-check all environment variables before execution
6. **Log Output**: Redirect output to file for audit trail:
   ```bash
   ./data-migration -type=all 2>&1 | tee migration-$(date +%Y%m%d-%H%M%S).log
   ```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "connection refused" | Ensure Vault server is running and accessible at `$VAULT_ADDR` |
| "permission denied" | Check `$VAULT_TOKEN` has Transit Engine permissions |
| "database connection failed" | Validate `$DATABASE_URL` format and credentials |
| "already encrypted" warnings | Normal behavior for re-running migrations (idempotent) |
| Memory issues | Migration processes in batches of 100, increase container memory if needed |

## Task Completion Status

**Completed Tasks**:
- ✅ T066: Created independent data-migration module with go.mod, config.go, migrate_users.go
- ✅ T067: Created migrate_guest_orders.go for guest order PII encryption
- ✅ T068: Created migrate_tenant_configs.go, main.go CLI, and Dockerfile

**Pending**:
- ⏳ T069: Run migrations and verify 100% encryption coverage

## Migration Log Format

Expected output when running migrations:

```
Starting User PII Migration...
Environment check: ✓
Vault connection: ✓
Database connection: ✓

Processing users table...
Batch 1: 100 records encrypted
Batch 2: 100 records encrypted
Batch 3: 47 records encrypted

Migration complete!
Total encrypted: 247 records
Skipped (already encrypted): 0 records
Failed: 0 records
```

## Files Removed

The following files were deprecated and removed (superseded by new module structure):
- `encrypt_existing_users.go` → `migrate_users.go`
- `encrypt_existing_guest_orders.go` → `migrate_guest_orders.go`
- `encrypt_existing_tenant_configs.go` → `migrate_tenant_configs.go`

## Build Verification

✅ Module initialized: `go mod tidy` completed successfully  
✅ Binary builds: `go build -o data-migration .` successful  
✅ CLI works: `./data-migration` shows usage help  
✅ Dependencies resolved: vault/api v1.10.0, lib/pq v1.10.9  

**Status**: Module is production-ready for deployment testing

## Next Steps

1. Deploy to staging VPS
2. Run migrations with test data
3. Verify encryption coverage
4. Monitor Vault performance
5. Document any issues or edge cases
6. Prepare production deployment plan

---

**Implementation Date**: 2024-01-XX  
**Status**: Ready for Testing  
**Related Tasks**: T066-T069  
**Related User Story**: US1 (Encryption at Rest & Log Masking)
