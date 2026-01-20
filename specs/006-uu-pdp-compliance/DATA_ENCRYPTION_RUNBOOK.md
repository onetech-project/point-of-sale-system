# Data Encryption Deployment Runbook

**Feature:** 006-uu-pdp-compliance (User Story 1)  
**Tasks:** T069a (Schema Migration) + T069 (Data Migration)  
**Estimated Duration:** 10-15 minutes (schema: ~1 min, data migration: 5-10 min depending on data volume)  
**Risk Level:** MEDIUM (requires database schema changes and data transformation)

---

## Overview

This runbook covers the deployment of PII encryption at rest using HashiCorp Vault Transit Engine. The deployment consists of two sequential steps:

1. **T069a**: Database schema migration to increase column sizes for encrypted data
2. **T069**: Data migration to encrypt existing plaintext PII in the database

**Why two steps?** Vault Transit Engine ciphertext is 8-10x larger than plaintext (e.g., "john@example.com" → "vault:v1:XXXX..." ~100-150 characters). Existing columns (VARCHAR(50)) are too small and must be increased first.

---

## Prerequisites

### 1. Vault Transit Engine Configuration ✅
Verify Vault is running and configured:

```bash
# Check Vault status
docker exec vault vault status

# Verify transit engine enabled
docker exec vault vault secrets list | grep transit

# Verify encryption key exists
docker exec vault vault list transit/keys
# Expected output: pos-encryption-key
```

### 2. Database Backup (CRITICAL) ⚠️
Create a full database backup before proceeding:

```bash
# Production backup
pg_dump -h <prod-db-host> -U pos_user -d pos_db -F c -f pos_db_backup_$(date +%Y%m%d_%H%M%S).dump

# Development backup
docker exec postgres-db pg_dump -U pos_user -d pos_db -F c > pos_db_backup_dev_$(date +%Y%m%d_%H%M%S).dump
```

**Verify backup file size and integrity:**
```bash
ls -lh pos_db_backup_*.dump
pg_restore --list pos_db_backup_*.dump | head -20
```

### 3. Downtime Window
- **Estimated downtime**: 10-15 minutes
- **User impact**: System unavailable during migration
- **Recommended window**: Off-peak hours (e.g., 2-4 AM local time)

### 4. Environment Variables
Verify required environment variables in data-migration module:

```bash
cd scripts/data-migration
cat .env
```

Required variables:
```env
VAULT_ADDR=http://vault:8200
VAULT_TOKEN=<root-token>
DB_HOST=postgres-db
DB_PORT=5432
DB_USER=pos_user
DB_PASSWORD=<password>
DB_NAME=pos_db
DB_SSLMODE=disable
```

---

## Step 1: Schema Migration (T069a)

**Duration:** ~1 minute  
**Risk:** LOW (no data modification, only metadata changes)

### 1.1 Run Migration 000042

```bash
# Navigate to migrations directory
cd backend/migrations

# Apply migration (development)
docker exec postgres-db psql -U pos_user -d pos_db -f /migrations/000042_increase_column_sizes_for_encryption.up.sql

# OR using migrate tool
migrate -path ./backend/migrations -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" up 1
```

### 1.2 Verify Column Sizes

```bash
# Check users table
docker exec postgres-db psql -U pos_user -d pos_db -c "\d users" | grep -E "(email|first_name|last_name)"

# Expected output:
# email            | character varying(512)
# first_name       | character varying(512)
# last_name        | character varying(512)

# Check guest_orders table
docker exec postgres-db psql -U pos_user -d pos_db -c "\d guest_orders" | grep -E "(customer_name|customer_phone|customer_email|ip_address)"

# Expected output:
# customer_name    | character varying(512)
# customer_phone   | character varying(100)
# customer_email   | character varying(512)
# ip_address       | character varying(100)

# Check tenant_configs table
docker exec postgres-db psql -U pos_user -d pos_db -c "\d tenant_configs" | grep -E "(midtrans_server_key|midtrans_client_key)"

# Expected output:
# midtrans_server_key  | character varying(512)
# midtrans_client_key  | character varying(512)
```

### 1.3 Verification Checkpoint

✅ **Proceed to Step 2 if:**
- All column sizes increased to VARCHAR(512) or VARCHAR(100)
- No errors in migration output
- Database accessible and responsive

❌ **STOP and rollback if:**
- Migration failed with errors
- Column sizes not updated correctly
- Database performance degraded

**Rollback Step 1:**
```bash
# Revert schema changes
docker exec postgres-db psql -U pos_user -d pos_db -f /migrations/000042_increase_column_sizes_for_encryption.down.sql

# OR using migrate tool
migrate -path ./backend/migrations -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" down 1
```

---

## Step 2: Data Migration (T069)

**Duration:** 5-10 minutes (depends on data volume)  
**Risk:** MEDIUM (modifies existing data)

### 2.1 Build Data Migration Image (if not already built)

```bash
cd scripts/data-migration

# Build Docker image
docker build -t pos-data-migration:latest .

# Verify image exists
docker images | grep pos-data-migration
```

### 2.2 Pre-Migration Data Snapshot

```bash
# Count total records to encrypt
docker exec postgres-db psql -U pos_user -d pos_db << EOF
SELECT 
  'users' as table_name, 
  COUNT(*) as total_records 
FROM users WHERE email IS NOT NULL AND email NOT LIKE 'vault:v1:%'
UNION ALL
SELECT 
  'guest_orders' as table_name, 
  COUNT(*) as total_records 
FROM guest_orders WHERE customer_email IS NOT NULL AND customer_email NOT LIKE 'vault:v1:%'
UNION ALL
SELECT 
  'tenant_configs' as table_name, 
  COUNT(*) as total_records 
FROM tenant_configs WHERE midtrans_server_key IS NOT NULL AND midtrans_server_key NOT LIKE 'vault:v1:%';
EOF
```

**Save output for verification:**
```text
Example:
   table_name    | total_records
-----------------+---------------
 users           |           42
 guest_orders    |          156
 tenant_configs  |            3
```

### 2.3 Run Data Migration (All Types)

```bash
# Run migration for all types (users, guest_orders, tenant_configs)
docker run --rm \
  --network pos-network \
  --env-file .env \
  pos-data-migration:latest \
  -type=all

# OR run individually (recommended for debugging)
docker run --rm --network pos-network --env-file .env pos-data-migration:latest -type=users
docker run --rm --network pos-network --env-file .env pos-data-migration:latest -type=guest_orders
docker run --rm --network pos-network --env-file .env pos-data-migration:latest -type=tenant_configs
```

**Expected output (successful migration):**
```text
2026/01/05 10:30:15 Starting data migration (type=all)
2026/01/05 10:30:15 Connected to Vault at http://vault:8200
2026/01/05 10:30:16 Connected to database: pos_db
2026/01/05 10:30:16 Migrating users... (batch size: 100)
2026/01/05 10:30:18 Encrypted 42 users records
2026/01/05 10:30:18 Migrating guest_orders... (batch size: 100)
2026/01/05 10:30:22 Encrypted 156 guest_orders records
2026/01/05 10:30:22 Migrating tenant_configs... (batch size: 100)
2026/01/05 10:30:23 Encrypted 3 tenant_configs records
2026/01/05 10:30:23 Migration completed successfully
```

### 2.4 Post-Migration Verification

#### A. Check Encryption Format

```bash
# Verify users table encrypted
docker exec postgres-db psql -U pos_user -d pos_db -c "
SELECT 
  id, 
  LEFT(email, 20) as email_prefix, 
  LEFT(first_name, 20) as firstname_prefix 
FROM users 
LIMIT 5;"

# Expected output (encrypted):
# id | email_prefix          | firstname_prefix
# ---+-----------------------+-------------------
# 1  | vault:v1:XXXXXX...   | vault:v1:YYYYY...

# Verify guest_orders table encrypted
docker exec postgres-db psql -U pos_user -d pos_db -c "
SELECT 
  id, 
  LEFT(customer_email, 20) as email_prefix, 
  LEFT(customer_name, 20) as name_prefix 
FROM guest_orders 
LIMIT 5;"

# Verify tenant_configs encrypted
docker exec postgres-db psql -U pos_user -d pos_db -c "
SELECT 
  tenant_id, 
  LEFT(midtrans_server_key, 20) as server_key_prefix 
FROM tenant_configs 
LIMIT 3;"
```

#### B. Verify Record Counts Match

```bash
# Count encrypted records
docker exec postgres-db psql -U pos_user -d pos_db << EOF
SELECT 
  'users' as table_name, 
  COUNT(*) as encrypted_records 
FROM users WHERE email LIKE 'vault:v1:%'
UNION ALL
SELECT 
  'guest_orders' as table_name, 
  COUNT(*) as encrypted_records 
FROM guest_orders WHERE customer_email LIKE 'vault:v1:%'
UNION ALL
SELECT 
  'tenant_configs' as table_name, 
  COUNT(*) as encrypted_records 
FROM tenant_configs WHERE midtrans_server_key LIKE 'vault:v1:%';
EOF
```

**Compare with pre-migration snapshot:**
- Total records before: 42 users, 156 guest_orders, 3 tenant_configs
- Encrypted records after: **MUST MATCH** pre-migration counts

#### C. Test Decryption (Spot Check)

```bash
# Get one encrypted value
ENCRYPTED_EMAIL=$(docker exec postgres-db psql -U pos_user -d pos_db -t -c "SELECT email FROM users LIMIT 1" | xargs)

# Test manual decryption via Vault
docker exec vault vault write -field=plaintext transit/decrypt/pos-encryption-key ciphertext="$ENCRYPTED_EMAIL" | base64 -d
# Expected: Decrypted plaintext email (e.g., "john@example.com")
```

#### D. Application Integration Test

```bash
# Test user login (decryption happens transparently)
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# Expected: Successful login (HTTP 200) - proves decryption works
```

### 2.5 Verification Checkpoint

✅ **Migration successful if:**
- All records show "vault:v1:" prefix in database
- Encrypted record counts match pre-migration totals
- Manual decryption via Vault works
- Application login/queries work normally
- No plaintext PII visible in database

❌ **STOP and restore backup if:**
- Record counts don't match
- Application errors accessing encrypted data
- Decryption fails
- Performance severely degraded

---

## Step 3: Validation and Testing

### 3.1 Functional Testing Checklist

- [ ] User login works (tests email decryption)
- [ ] User profile display works (tests name decryption)
- [ ] Guest order creation works (tests encryption on write)
- [ ] Guest order retrieval works (tests decryption on read)
- [ ] Tenant config update works (tests payment credential encryption)
- [ ] Email notifications work (tests recipient decryption)

### 3.2 Security Verification

```bash
# Verify no plaintext PII in database dumps
docker exec postgres-db pg_dump -U pos_user -d pos_db -t users | grep -E "(john|alice|bob|example\.com)" | wc -l
# Expected: 0 (no plaintext matches)

# Verify backup files contain encrypted data
pg_restore --data-only --table=users pos_db_backup_*.dump | grep "vault:v1:" | wc -l
# Expected: >0 (encrypted data in backup)
```

### 3.3 Performance Baseline

```bash
# Measure query performance (before vs after encryption)
docker exec postgres-db psql -U pos_user -d pos_db -c "\timing on" -c "SELECT * FROM users LIMIT 100;"

# Expected: <50ms (minimal overhead from decryption)
```

---

## Step 4: Rollback Plan (Emergency)

**When to rollback:**
- Data migration fails mid-process
- Application cannot decrypt data
- Performance unacceptable
- Critical bugs discovered

### 4.1 Stop Application Services

```bash
# Stop all backend services
docker-compose -f docker-compose.yml down
```

### 4.2 Restore Database Backup

```bash
# Drop current database (CAUTION: destructive)
docker exec postgres-db psql -U pos_user -d postgres -c "DROP DATABASE pos_db;"

# Recreate database
docker exec postgres-db psql -U pos_user -d postgres -c "CREATE DATABASE pos_db;"

# Restore from backup
docker exec -i postgres-db pg_restore -U pos_user -d pos_db < pos_db_backup_*.dump

# OR for production
pg_restore -h <prod-db-host> -U pos_user -d pos_db pos_db_backup_*.dump
```

### 4.3 Revert Schema Migration (if needed)

```bash
# Rollback migration 000042
docker exec postgres-db psql -U pos_user -d pos_db -f /migrations/000042_increase_column_sizes_for_encryption.down.sql

# Verify rollback
docker exec postgres-db psql -U pos_user -d pos_db -c "\d users" | grep email
# Expected: email VARCHAR(255) or original size
```

### 4.4 Restart Services

```bash
# Restart application stack
docker-compose -f docker-compose.yml up -d

# Verify services healthy
docker-compose ps
```

---

## Step 5: Post-Deployment Tasks

### 5.1 Update Documentation

- [ ] Mark T069a as completed in `specs/006-uu-pdp-compliance/tasks.md`
- [ ] Mark T069 as completed in `specs/006-uu-pdp-compliance/tasks.md`
- [ ] Update `CHANGELOG.md` with encryption feature
- [ ] Update deployment notes in `docs/DEPLOYMENT_CHECKLIST.md`

### 5.2 Monitoring Setup

```bash
# Monitor Vault encryption/decryption metrics
docker exec vault vault read sys/metrics | grep transit

# Monitor database query performance
docker exec postgres-db psql -U pos_user -d pos_db -c "SELECT * FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"
```

### 5.3 Security Audit Log

```bash
# Log deployment completion for audit trail
docker exec postgres-db psql -U pos_user -d pos_db << EOF
INSERT INTO audit_events (event_type, actor_type, actor_id, resource_type, action, metadata)
VALUES (
  'SYSTEM_EVENT',
  'platform_admin',
  'deployment-script',
  'DATABASE',
  'PII_ENCRYPTION_DEPLOYED',
  '{"feature":"006-uu-pdp-compliance","tasks":"T069a+T069","timestamp":"$(date -u +%Y-%m-%dT%H:%M:%SZ)","encrypted_tables":["users","guest_orders","tenant_configs"]}'
);
EOF
```

---

## Troubleshooting

### Issue 1: Migration fails with "column does not exist"
**Symptom:** Error mentions non-existent column in migration scripts  
**Cause:** Migration script references column that doesn't exist in actual schema  
**Solution:** Review migration scripts and ensure they only reference existing columns. The current implementation uses in-place encryption (no separate encrypted columns needed).

### Issue 2: Migration fails with "value too long"
**Symptom:** Error: `pq: value too long for type character varying(50)`  
**Cause:** Schema migration (T069a) not run yet  
**Solution:** Complete Step 1 (schema migration) before Step 2 (data migration)

### Issue 3: Vault connection timeout
**Symptom:** `dial tcp: lookup vault on 127.0.0.11:53: no such host`  
**Cause:** Data migration container not on same Docker network as Vault  
**Solution:** Ensure `--network pos-network` in docker run command

### Issue 4: Decryption fails in application
**Symptom:** Application cannot read user data, errors in logs  
**Cause:** Vault token expired or encryption key not found  
**Solution:**
```bash
# Verify Vault access from application
docker exec user-service curl -H "X-Vault-Token: $VAULT_TOKEN" http://vault:8200/v1/transit/keys/pos-encryption-key

# Renew token if expired
docker exec vault vault token renew <token>
```

### Issue 5: Performance degradation
**Symptom:** Queries taking >200ms (2-4x slower than before)  
**Cause:** Excessive decryption calls, missing indexes, or Vault overload  
**Solution:**
```bash
# Check Vault performance
docker stats vault

# Add caching layer (future enhancement)
# Review query patterns and add selective decryption
```

---

## Success Criteria

- [x] T069a: Schema migration completed (column sizes increased)
- [x] T069: Data migration completed (all PII encrypted)
- [x] 100% of user records encrypted (email/first_name/last_name start with "vault:v1:")
- [x] 100% of guest order records encrypted (customer PII starts with "vault:v1:")
- [x] 100% of tenant config records encrypted (payment keys start with "vault:v1:")
- [x] Application functional (login, order creation, tenant management work)
- [x] No plaintext PII in database or backups
- [x] Decryption works transparently (no user-facing changes)
- [x] Performance acceptable (<50ms overhead per query)

---

## Contact Information

**Deployment Owner:** Platform Admin  
**Technical Contact:** Backend Team  
**Escalation:** Security Team (for data breach concerns)

**Rollback Authority:** Requires approval from Platform Owner if production deployment

---

## Appendix: Migration Script Details

### Migration 000042 (Schema Changes)

**File:** `backend/migrations/000042_increase_column_sizes_for_encryption.up.sql`

**Changes:**
- `users`: email/first_name/last_name → VARCHAR(512)
- `guest_orders`: customer_name/email → VARCHAR(512), customer_phone/ip_address → VARCHAR(100)
- `tenant_configs`: midtrans_server_key/client_key → VARCHAR(512)

**Performance Impact:** None (metadata-only change in PostgreSQL)

### Data Migration Module

**Location:** `scripts/data-migration/`

**Components:**
- `config.go`: Vault and database connection setup
- `migrate_users.go`: Encrypt user PII (email/first_name/last_name)
- `migrate_guest_orders.go`: Encrypt guest order PII (customer_name/phone/email/ip_address)
- `migrate_tenant_configs.go`: Encrypt payment credentials (midtrans keys)
- `main.go`: CLI entry point with `-type` flag (all/users/guest_orders/tenant_configs)

**Idempotency:** Safe to re-run (skips already encrypted records)

**Batch Size:** 100 records per batch (configurable in code)

---

**Document Version:** 1.0  
**Last Updated:** 2026-01-05  
**Review Date:** After first production deployment
