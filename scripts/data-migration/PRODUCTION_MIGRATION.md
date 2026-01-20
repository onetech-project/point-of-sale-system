# Production Migration Tool (Standalone)

## Overview

The file `migrate_context_encryption_production_standalone.txt` contains a complete standalone production migration tool. It has been renamed to `.txt` to prevent compilation conflicts with the main data-migration module.

## Purpose

This script is designed for **production environments** where you need to migrate from old encryption (without context) to context-based deterministic encryption. It's more sophisticated than the regular migration tools because it:

1. **Supports both encryption schemes**: Can decrypt with old method, re-encrypt with new method
2. **Batch processing**: Processes records in configurable batch sizes
3. **Dry-run mode**: Preview changes before applying
4. **Table filtering**: Migrate specific tables or all tables
5. **Progress tracking**: Detailed statistics and logging
6. **HMAC verification**: Validates data integrity during migration

## ⚠️ Important Requirement

**This script requires the OLD encryption key to be accessible in Vault.**

If the old encryption key has been deleted or is no longer accessible, this script will fail. In that case, you must use the simpler `encrypt-plaintext` migration instead (which assumes data is plaintext).

## When to Use

Use this standalone script when:
- Migrating from old encryption to context-based encryption in **production**
- You have the **old encryption key** still available
- You need **fine-grained control** over the migration process
- You want to migrate **specific tables** rather than all at once
- You need **batch processing** for large datasets

## How to Use

### Step 1: Rename Back to .go

```bash
cd scripts/data-migration
mv migrate_context_encryption_production_standalone.txt migrate_context_encryption_production.go
```

### Step 2: Run Standalone (Outside Module)

```bash
# Move to parent directory to avoid module conflicts
cd ..

# Dry run first (preview only)
go run data-migration/migrate_context_encryption_production.go \
  --vault-addr=http://localhost:8200 \
  --vault-token=$VAULT_TOKEN \
  --db="$DATABASE_URL" \
  --hmac-secret="$ENCRYPTION_HMAC_SECRET" \
  --tables=users,invitations \
  --batch-size=100 \
  --dry-run=true

# Review the output carefully

# Run live migration
go run data-migration/migrate_context_encryption_production.go \
  --vault-addr=http://localhost:8200 \
  --vault-token=$VAULT_TOKEN \
  --db="$DATABASE_URL" \
  --hmac-secret="$ENCRYPTION_HMAC_SECRET" \
  --tables=users,invitations \
  --batch-size=100 \
  --dry-run=false
```

### Step 3: Rename Back to .txt

```bash
cd data-migration
mv migrate_context_encryption_production.go migrate_context_encryption_production_standalone.txt
```

## Command-Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `--db` | `$DATABASE_URL` | PostgreSQL connection string |
| `--vault-addr` | `$VAULT_ADDR` | Vault server address |
| `--vault-token` | `$VAULT_TOKEN` | Vault authentication token |
| `--vault-key` | `pos-encryption-key` | Vault transit key name |
| `--hmac-secret` | `$ENCRYPTION_HMAC_SECRET` | HMAC secret for integrity |
| `--tables` | `all` | Comma-separated list of tables or "all" |
| `--batch-size` | `100` | Records to process per batch |
| `--dry-run` | `true` | Preview mode (no changes) |

## Supported Tables

The script can migrate the following tables:

1. **users**: email, first_name, last_name
2. **invitations**: email, token
3. **guest_orders**: customer_name, customer_phone, customer_email, ip_address, user_agent
4. **delivery_addresses**: full_address, geocoding_result
5. **notifications**: recipient, body
6. **sessions**: session_id, ip_address
7. **consent_records**: ip_address

## Migration Process

For each record:
1. Read encrypted data from database
2. **Decrypt** using old key (no context)
3. **Re-encrypt** using new key (with context)
4. Update database with new ciphertext
5. Track statistics (success/failure counts)

## Production Best Practices

### Before Migration

1. **Backup the database**:
   ```bash
   pg_dump $DATABASE_URL > backup_$(date +%Y%m%d_%H%M%S).sql
   ```

2. **Backup Vault encryption keys**:
   ```bash
   vault write transit/keys/pos-encryption-key/config exportable=true
   vault read -format=json transit/backup/pos-encryption-key > vault_key_backup.json
   ```

3. **Test in staging first**: Run migration on staging environment with production-like data

4. **Schedule maintenance window**: Plan for downtime if needed

5. **Verify old key is accessible**:
   ```bash
   vault read transit/keys/pos-encryption-key
   ```

### During Migration

1. **Start with dry-run**: Always run with `--dry-run=true` first
2. **Migrate one table at a time**: Test each table before moving to next
3. **Monitor progress**: Watch logs for errors
4. **Keep services running**: If using key rotation, services can keep running

### After Migration

1. **Verify data**: Query some records and decrypt to verify correctness
2. **Test application**: Ensure services can read/write encrypted data
3. **Monitor for 24-48 hours**: Watch for decryption errors
4. **Update application config**: Switch to new encryption key if needed
5. **Keep old key for 30 days**: Don't delete old key immediately in case rollback needed

## Troubleshooting

### Error: "missing 'context' for key derivation"

**Problem**: Vault key requires context but none was provided during old encryption

**Solution**: Your old data may already use context-based encryption. Use the regular migration tools instead.

### Error: "invalid ciphertext: no prefix"

**Problem**: Data is plaintext, not encrypted

**Solution**: Use `encrypt-plaintext` migration type instead:
```bash
cd scripts/data-migration
go run main.go -type=encrypt-plaintext
```

### Error: "HMAC verification failed"

**Problem**: HMAC secret doesn't match what was used during encryption

**Solution**: Find the correct HMAC secret from your `.env` files or environment config

### Error: "permission denied"

**Problem**: Vault token doesn't have sufficient permissions

**Solution**: Ensure token has `transit` policy with encrypt/decrypt capabilities:
```hcl
path "transit/encrypt/*" {
  capabilities = ["update"]
}
path "transit/decrypt/*" {
  capabilities = ["update"]
}
```

## Alternative: Use Main Module

If you don't need the advanced features of the standalone script, you can use the main data-migration module:

```bash
cd scripts/data-migration
go run main.go -type=encrypt-plaintext
```

This is simpler and assumes data is already plaintext (not encrypted with old key).

## See Also

- `/scripts/data-migration/README.md` - Main migration module documentation
- `/docs/DATA_MIGRATION_COMPLETE.md` - Complete migration guide and lessons learned
- `/docs/DETERMINISTIC_ENCRYPTION_REFACTOR.md` - Technical design documentation
