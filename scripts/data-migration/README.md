# Data Migration Scripts

Independent Go module for encrypting existing plaintext PII data in the database using Vault Transit Engine.

## Architecture

This migration tool is a standalone module with:
- **Independent module**: Own `go.mod`, no dependencies on service code
- **Single entry point**: `main.go` with argument-based routing
- **Self-contained configuration**: Vault and database config in `config.go`
- **Docker support**: Can run as container or standalone binary
- **Modular design**: Each migration type has its own file + wrapper

## Prerequisites

1. Vault server running and accessible
2. Database with existing data
3. Required environment variables:
   - `DATABASE_URL` - PostgreSQL connection string
   - `VAULT_ADDR` - Vault server address
   - `VAULT_TOKEN` - Vault authentication token
   - `VAULT_TRANSIT_KEY` - Transit encryption key name
   - `ENCRYPTION_HMAC_SECRET` - (Optional) HMAC secret for integrity checking

## Available Migration Types

- **users**: Encrypt user PII (email, first_name, last_name)
- **guest-orders**: Encrypt guest order PII (customer_name, phone, email, ip_address)
- **tenant-configs**: Encrypt tenant payment credentials (midtrans keys)
- **notifications**: Encrypt notification recipient, body, and metadata sensitive fields
- **invitations**: Encrypt invitation email and token
- **search-hashes**: Populate searchable HMAC hashes for encrypted fields
- **encrypt-plaintext**: Encrypt plaintext PII data with context-based encryption
- **all**: Run all migrations sequentially

## Running Migrations

### Option A: Local Development (with Go installed)

```bash
cd scripts/data-migration

# Initialize dependencies
go mod download

# Run specific migration
go run main.go -type=users
go run main.go -type=guest-orders
go run main.go -type=encrypt-plaintext

# Run all migrations
go run main.go -type=all
```

### Option B: Build and Run Binary

```bash
cd scripts/data-migration

# Build
go build -o data-migration .

# Run
./data-migration -type=users
./data-migration -type=guest-orders
./data-migration -type=tenant-configs
./data-migration -type=all
```

### Option C: Using Docker

#### Build the Docker image

```bash
cd scripts/data-migration
docker build -t pos-data-migration .
```

#### Run migrations in Docker

```bash
# Run single migration
docker run --rm \
  --network your-docker-network \
  -e DATABASE_URL="postgresql://user:pass@postgres:5432/dbname" \
  -e VAULT_ADDR="http://vault:8200" \
  -e VAULT_TOKEN="your-vault-token" \
  -e VAULT_TRANSIT_KEY="pos-encryption-key" \
  pos-data-migration -type=users

# Run all migrations
docker run --rm \
  --network your-docker-network \
  -e DATABASE_URL="postgresql://user:pass@postgres:5432/dbname" \
  -e VAULT_ADDR="http://vault:8200" \
  -e VAULT_TOKEN="your-vault-token" \
  -e VAULT_TRANSIT_KEY="pos-encryption-key" \
  pos-data-migration -type=all
```

#### Using docker-compose

Add to your `docker-compose.yml`:

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
# Run specific migration
docker-compose run --rm data-migration -type=users

# Run all migrations
docker-compose run --rm data-migration -type=all
```

### Option D: VPS with Existing Containers (No Rebuilding)

If you have services already deployed and don't want to rebuild images:

```bash
# Copy migration files to VPS
scp -r scripts/data-migration user@vps:/path/to/project/scripts/

# SSH into VPS
ssh user@vps

# Run migration with Docker
cd /path/to/project/scripts/data-migration
docker build -t pos-data-migration .
docker run --rm \
  --network pos-network \
  -e DATABASE_URL="$DATABASE_URL" \
  -e VAULT_ADDR="$VAULT_ADDR" \
  -e VAULT_TOKEN="$VAULT_TOKEN" \
  -e VAULT_TRANSIT_KEY="$VAULT_TRANSIT_KEY" \
  pos-data-migration -type=all
```

## Migration Types

| Type | Target Table | Encrypted Fields |
|------|-------------|------------------|
| `users` | `users` | `email`, `first_name`, `last_name` |
| `guest-orders` | `guest_orders` | `customer_name`, `customer_phone`, `customer_email`, `ip_address` |
| `tenant-configs` | `tenant_configs` | `midtrans_server_key`, `midtrans_client_key` |
| `all` | All tables | All fields above (sequential execution) |

## Safety Features

- **Idempotency**: Already encrypted values (starting with `vault:v1:`) are skipped
- **Batch Processing**: Records processed in batches of 100 for memory efficiency
- **Transaction Safety**: Each batch committed separately
- **Null Handling**: NULL values preserved without encryption
- **Guest Order Filter**: Only encrypts non-anonymized guest orders (`is_anonymized = FALSE`)

## Verification

After running migrations, verify encryption coverage:

```bash
# Check users encryption
psql $DATABASE_URL -c "SELECT COUNT(*) FROM users WHERE email NOT LIKE 'vault:v1:%';"
# Expected: 0

# Check guest orders encryption
psql $DATABASE_URL -c "SELECT COUNT(*) FROM guest_orders WHERE is_anonymized = FALSE AND customer_name NOT LIKE 'vault:v1:%';"
# Expected: 0

# Check tenant configs encryption
psql $DATABASE_URL -c "SELECT COUNT(*) FROM tenant_configs WHERE midtrans_server_key IS NOT NULL AND midtrans_server_key NOT LIKE 'vault:v1:%';"
# Expected: 0
```

## Production Deployment Best Practices

1. **Backup First**: Always backup database before running migrations
2. **Test in Staging**: Verify migrations work in staging environment
3. **Monitor Vault**: Ensure Vault has sufficient capacity for encryption operations
4. **Run During Off-Peak**: Schedule migrations during low-traffic periods
5. **Verify Environment**: Double-check all environment variables before execution
6. **Log Output**: Redirect output to file for audit trail:
   ```bash
   ./data-migration -type=all 2>&1 | tee migration-$(date +%Y%m%d-%H%M%S).log
   ```

## Rollback Plan

If encryption fails or causes issues:

1. **Restore from Backup**: Use database backup to restore pre-migration state
2. **Revert Application Changes**: Temporarily disable encryption repositories
3. **Investigate**: Check logs for specific error messages
4. **Fix and Retry**: Address issues and re-run migration

## Module Structure

```
scripts/data-migration/
├── go.mod                      # Independent module definition
├── main.go                     # CLI entry point with flag parsing
├── config.go                   # Configuration loader and Vault client
├── migrate_users.go            # User PII encryption logic
├── migrate_guest_orders.go     # Guest order PII encryption logic
├── migrate_tenant_configs.go   # Tenant credentials encryption logic
├── Dockerfile                  # Container build instructions
└── README.md                   # This file
```

## Troubleshooting

**"connection refused" errors**:
- Ensure Vault server is running and accessible at `$VAULT_ADDR`
- Verify network connectivity between migration tool and Vault

**"permission denied" errors**:
- Check `$VAULT_TOKEN` has permissions for Transit Engine operations
- Verify token is not expired

**"database connection failed"**:
- Validate `$DATABASE_URL` format and credentials
- Ensure PostgreSQL server is accessible

**"already encrypted" warnings**:
- Normal behavior for re-running migrations
- Migration is idempotent, safe to run multiple times

**Memory issues with large datasets**:
- Migration processes in batches of 100 records
- Consider increasing container memory limits if needed

## Verification

After each migration, verify 100% encryption coverage:

```sql
-- Users
SELECT COUNT(*) FROM users WHERE email NOT LIKE 'vault:v1:%';

-- Guest Orders
SELECT COUNT(*) FROM guest_orders 
WHERE is_anonymized = FALSE 
  AND customer_name NOT LIKE 'vault:v1:%';

-- Tenant Configs
SELECT COUNT(*) FROM tenant_configs 
WHERE (midtrans_server_key IS NOT NULL AND midtrans_server_key != '' AND midtrans_server_key NOT LIKE 'vault:v1:%')
   OR (midtrans_client_key IS NOT NULL AND midtrans_client_key != '' AND midtrans_client_key NOT LIKE 'vault:v1:%');
```

Expected result for all queries: `0`

## Features

- **Batch Processing**: Processes 100 records at a time
- **Progress Logging**: Reports progress every 10 records
- **Error Handling**: Continues on individual record failures, reports at end
- **Idempotent**: Skips already encrypted records (checks for `vault:v1:` prefix)
- **Rate Limiting**: 100ms delay between batches to avoid overwhelming Vault

## Important Notes for VPS/Production

1. **Container must have Go installed**: The migration scripts use `go run`. Ensure your service images include the Go runtime, or use Option C to build a dedicated migration container.

2. **Network connectivity**: Ensure the container can reach both the database and Vault:
   - Database: Usually via internal Docker network
   - Vault: Check firewall rules and network policies

3. **Environment variables**: All required env vars must be available in the container:
   ```bash
   # Verify environment variables
   docker exec user-service env | grep -E 'DATABASE_URL|VAULT_'
   ```

4. **Dry run first**: Test on a staging environment with production-like data before running on production.

5. **Monitor logs**: Watch migration progress in real-time:
   ```bash
   docker exec -it user-service sh -c "cd /app && go run /tmp/encrypt_existing_users.go" 2>&1 | tee migration.log
   ```

## Rollback

⚠️ **WARNING**: These migrations are destructive - they replace plaintext with ciphertext in-place.

Before running in production:
1. Take a database backup
2. Test on staging environment first
3. Verify encryption/decryption works correctly in application code
