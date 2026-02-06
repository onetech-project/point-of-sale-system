# Standalone Migration Scripts

These are independent migration scripts that can be run directly without the main data-migration module. They were created for the Phase 2-3 deterministic encryption migration.

## Scripts

### 1. `encrypt_plaintext_data.go`
**Purpose**: Encrypt plaintext PII data with context-based encryption

**Use Case**: When you need to encrypt data that is currently stored as plaintext (not encrypted)

**Usage**:
```bash
go run encrypt_plaintext_data.go \
  --vault-addr=http://localhost:8200 \
  --vault-token=<your-token> \
  --db="postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
  --hmac-secret=<your-hmac-secret> \
  --dry-run=true
```

**Options**:
- `--vault-addr`: Vault server address (default: from VAULT_ADDR env var)
- `--vault-token`: Vault authentication token (default: from VAULT_TOKEN env var)
- `--vault-key`: Vault transit key name (default: "pos-encryption-key")
- `--db`: PostgreSQL connection string (default: from DATABASE_URL env var)
- `--hmac-secret`: HMAC secret for integrity (default: from ENCRYPTION_HMAC_SECRET env var)
- `--dry-run`: Preview changes without modifying data (default: true)

**Tables Processed**:
- `users`: email, first_name, last_name
- `invitations`: email, token
- `guest_orders`: customer_name, customer_phone, customer_email, ip_address, user_agent
- `notifications`: recipient, body

---

### 2. `migrate_context_encryption_production.go`
**Purpose**: Migrate encrypted data from old encryption (no context) to context-based encryption

**Use Case**: When you already have encrypted data and want to re-encrypt it with context for deterministic encryption

**⚠️ IMPORTANT**: This script requires the OLD encryption key to be accessible for decryption. If the old key is lost, this script will fail.

**Usage**:
```bash
go run migrate_context_encryption_production.go \
  --vault-addr=http://localhost:8200 \
  --vault-token=<your-token> \
  --db="postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
  --hmac-secret=<your-hmac-secret> \
  --tables=users,invitations \
  --batch-size=100 \
  --dry-run=true
```

**Options**:
- `--vault-addr`: Vault server address
- `--vault-token`: Vault authentication token
- `--vault-key`: Vault transit key name (default: "pos-encryption-key")
- `--db`: PostgreSQL connection string
- `--hmac-secret`: HMAC secret for integrity
- `--tables`: Comma-separated list of tables or "all" (default: "all")
- `--batch-size`: Records to process per batch (default: 100)
- `--dry-run`: Preview changes without modifying data (default: true)

**Migration Process**:
1. Read encrypted data from database
2. Decrypt using old method (no context)
3. Re-encrypt using new method (with context)
4. Update database with new ciphertext

**Tables Supported**:
- `users`: email, first_name, last_name → contexts: user:email, user:first_name, user:last_name
- `invitations`: email, token → contexts: invitation:email, invitation:token
- `guest_orders`: customer_name, customer_phone, customer_email, ip_address, user_agent
- `delivery_addresses`: full_address, geocoding_result
- `notifications`: recipient, body
- `sessions`: session_id, ip_address
- `consent_records`: ip_address

---

### 3. `migrate_to_context_encryption.go`
**Purpose**: Documentation/template script (not fully implemented)

**Use Case**: Reference implementation showing the migration structure

---

## Environment Variables

All scripts support environment variables as defaults:

```bash
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="hvs.xxxxxxxxxxxxxxxx"
export DATABASE_URL="postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable"
export ENCRYPTION_HMAC_SECRET="your-hmac-secret-key-min-32-chars-required"
```

Then you can run scripts without flags:
```bash
go run encrypt_plaintext_data.go --dry-run=false
```

## Typical Migration Workflow

### Scenario 1: Fresh Deployment (No Existing Encrypted Data)
```bash
# Just use the main data-migration module
cd ../data-migration
go run main.go -type=all
```

### Scenario 2: Migrating from Old Encryption to Context-Based
```bash
# Step 1: Backup Vault keys
vault read -format=json transit/backup/pos-encryption-key > key-backup.json

# Step 2: Create new key with convergent encryption
vault write -f transit/keys/pos-encryption-key-v2 \
  type=aes256-gcm96 \
  convergent_encryption=true \
  derived=true

# Step 3: Migrate data (dry-run first)
go run migrate_context_encryption_production.go --dry-run=true

# Step 4: Migrate data (live)
go run migrate_context_encryption_production.go --dry-run=false

# Step 5: Update application to use new key
# Update VAULT_TRANSIT_KEY=pos-encryption-key-v2 in all services
```

### Scenario 3: Data Lost Old Key (Development Only!)
```bash
# Step 1: Reset data to plaintext placeholders (SQL script)
# See: docs/DATA_MIGRATION_COMPLETE.md

# Step 2: Encrypt plaintext data
go run encrypt_plaintext_data.go --dry-run=false
```

## Testing

Always run with `--dry-run=true` first to preview changes:

```bash
# Preview what will be encrypted
go run encrypt_plaintext_data.go --dry-run=true

# Check the logs carefully
# Then run without dry-run
go run encrypt_plaintext_data.go --dry-run=false
```

## Dependencies

These scripts use:
- `github.com/lib/pq` - PostgreSQL driver
- Standard library (crypto, http, encoding)
- No external Vault SDK (direct HTTP API calls)

Install dependencies:
```bash
go mod download
```

## Security Notes

1. **HMAC Secret**: Should be at least 32 characters, randomly generated
2. **Vault Token**: Store securely, never commit to Git
3. **Database Credentials**: Use environment variables, not hardcoded
4. **Key Backup**: Always backup encryption keys before migration
5. **Test First**: Always test in staging before production

## Troubleshooting

### Error: "missing 'context' for key derivation"
- The Vault key requires context but none was provided
- Solution: Use `EncryptWithContext()` instead of `Encrypt()`

### Error: "invalid ciphertext: no prefix"
- Data is plaintext, not encrypted
- Solution: Use `encrypt_plaintext_data.go` instead

### Error: "HMAC verification failed"
- HMAC secret doesn't match what was used during encryption
- Solution: Use the correct HMAC secret from your .env file

### Error: "permission denied"
- Vault token doesn't have permission for transit operations
- Solution: Use a token with `transit` policy permissions

## See Also

- `/docs/DATA_MIGRATION_COMPLETE.md` - Complete migration documentation
- `/docs/DETERMINISTIC_ENCRYPTION_REFACTOR.md` - Technical design
- `/backend/*/src/utils/encryption.go` - VaultClient implementation in services
