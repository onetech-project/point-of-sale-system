# Deterministic Encryption Refactoring

**Date**: 2026-01-06  
**Phase**: Phase 3 Refactoring (Post T001-T069)  
**Compliance**: UU PDP No.27 Tahun 2022 Article 20 (PII Protection)  
**Feature**: Convergent Encryption for Efficient Encrypted Field Search

---

## Overview

This document describes the refactoring of the encryption infrastructure to use **deterministic (convergent) encryption** via Vault Transit Engine. This enables efficient encrypted field search while maintaining strong cryptographic security.

### Why Deterministic Encryption?

**Problem with Random Encryption**:
- Same plaintext produces different ciphertexts each time (non-deterministic)
- Cannot efficiently search encrypted fields (e.g., find user by encrypted email)
- Requires decrypting ALL records to find a match (O(n) complexity)
- Search hash tables needed as workaround (additional storage, complexity)

**Solution with Convergent Encryption**:
- Same plaintext + context always produces same ciphertext (deterministic)
- Enables efficient encrypted field search without decryption
- Database indexes work on encrypted values
- Maintains security through context isolation (different contexts = different ciphertexts)

---

## Changes Implemented

### 1. Vault Transit Key Configuration

**File**: `vault/vault-init.sh`

**Before**:
```bash
vault write -f transit/keys/pos-encryption-key
```

**After**:
```bash
vault write -f transit/keys/pos-encryption-key \
  type=aes256-gcm96 \
  convergent_encryption=true \
  derived=true
```

**Parameters**:
- `type=aes256-gcm96`: AES-256-GCM encryption algorithm (authenticated encryption)
- `convergent_encryption=true`: Enable deterministic encryption (same plaintext + context = same ciphertext)
- `derived=true`: Derive encryption keys from master key using context (key isolation per context)

**Security Properties**:
- **Deterministic**: Same input produces same output (enables search)
- **Context Isolation**: Different contexts produce different ciphertexts (prevents correlation attacks)
- **Authenticated Encryption**: GCM mode provides both confidentiality and integrity
- **Key Derivation**: Each context uses a derived key (not the master key directly)

### 2. VaultClient Interface Extension

**File**: `backend/*/src/utils/encryption.go` (all services)

**New Interface Methods**:
```go
type Encryptor interface {
    Encrypt(ctx context.Context, plaintext string) (string, error)
    EncryptWithContext(ctx context.Context, plaintext string, encryptionContext string) (string, error)
    Decrypt(ctx context.Context, ciphertext string) (string, error)
    DecryptWithContext(ctx context.Context, ciphertext string, encryptionContext string) (string, error)
    EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error)
    DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error)
}
```

**Backward Compatibility**:
- `Encrypt()` and `Decrypt()` methods remain unchanged
- Internally delegate to `EncryptWithContext()` and `DecryptWithContext()` with empty context
- Existing code continues to work without modifications

**New Context-Based Methods**:
```go
// Deterministic encryption with context
encryptedEmail, err := encryptor.EncryptWithContext(ctx, email, "user:email")

// Must use same context for decryption
decryptedEmail, err := encryptor.DecryptWithContext(ctx, ciphertext, "user:email")
```

### 3. Encryption Context Design

**Context Format**: `<entity>:<field>`

**Examples**:
```go
// User entity
"user:email"          // user.email field
"user:first_name"     // user.first_name field
"user:last_name"      // user.last_name field

// Guest order entity
"guest_order:email"          // guest_orders.customer_email field
"guest_order:phone"          // guest_orders.customer_phone field
"guest_order:name"           // guest_orders.customer_name field
"guest_order:ip_address"     // guest_orders.ip_address field

// Delivery address entity
"delivery_address:address"   // delivery_addresses.address field
"delivery_address:latitude"  // delivery_addresses.latitude field
"delivery_address:longitude" // delivery_addresses.longitude field

// Session entity
"session:session_id"    // sessions.session_id field
"session:ip_address"    // sessions.ip_address field

// Tenant config entity
"tenant_config:midtrans_server_key"  // tenant_configs.midtrans_server_key field
"tenant_config:midtrans_client_key"  // tenant_configs.midtrans_client_key field

// Invitation entity
"invitation:email"  // invitations.email field
"invitation:token"  // invitations.token field

// Password reset token entity
"reset_token:token"  // password_reset_tokens.token field

// Notification entity
"notification:recipient"     // notifications.recipient field
"notification:message_body"  // notifications.message_body field

// Consent record entity
"consent_record:ip_address"  // consent_records.ip_address field
```

**Context Benefits**:
1. **Search Efficiency**: Same email encrypted with "user:email" context produces same ciphertext
   ```sql
   SELECT * FROM users WHERE email = '<encrypted_value>';
   -- Uses index, no full table scan needed
   ```

2. **Context Isolation**: Same email encrypted with different contexts produces different ciphertexts
   ```go
   userEmail := encryptor.EncryptWithContext(ctx, "user@example.com", "user:email")
   guestEmail := encryptor.EncryptWithContext(ctx, "user@example.com", "guest_order:email")
   // userEmail != guestEmail (different ciphertexts)
   ```

3. **Prevents Correlation Attacks**: Attacker cannot correlate same email across different tables

### 4. Repository Pattern Updates

**Current State** (backward compatible):
```go
// Existing code still works (uses empty context internally)
encryptedEmail, err := r.encryptor.Encrypt(ctx, user.Email)
```

**Recommended Migration** (next phase):
```go
// Use context-based encryption for efficient search
encryptedEmail, err := r.encryptor.EncryptWithContext(ctx, user.Email, "user:email")

// Query by encrypted value (deterministic)
query := `SELECT * FROM users WHERE email = $1`
rows, err := r.db.QueryContext(ctx, query, encryptedEmail)
```

---

## Migration Strategy

### Phase 1: Infrastructure Update ✅ COMPLETE

- [X] Update Vault Transit key configuration for convergent encryption
- [X] Add `EncryptWithContext()` and `DecryptWithContext()` methods to VaultClient
- [X] Maintain backward compatibility (existing `Encrypt()` and `Decrypt()` still work)
- [X] Deploy to all backend services (user, auth, order, tenant, notification, audit)

### Phase 2: Repository Refactoring (NEXT)

**Goal**: Update repositories to use context-based encryption for efficient search

**Tasks**:
1. Update UserRepository to use `EncryptWithContext(ctx, email, "user:email")`
2. Update GuestOrderRepository to use `EncryptWithContext(ctx, email, "guest_order:email")`
3. Update DeliveryAddressRepository to use `EncryptWithContext(ctx, address, "delivery_address:address")`
4. Update SessionRepository to use `EncryptWithContext(ctx, sessionID, "session:session_id")`
5. Update TenantConfigRepository to use `EncryptWithContext(ctx, serverKey, "tenant_config:midtrans_server_key")`
6. Update InvitationRepository to use `EncryptWithContext(ctx, email, "invitation:email")`
7. Update PasswordResetTokenRepository to use `EncryptWithContext(ctx, token, "reset_token:token")`
8. Update NotificationRepository to use `EncryptWithContext(ctx, recipient, "notification:recipient")`
9. Update ConsentRepository to use `EncryptWithContext(ctx, ipAddress, "consent_record:ip_address")`

**Impact**:
- ⚠️ **Breaking Change**: Existing encrypted data will NOT be searchable with new context-based encryption
- **Solution**: Run data migration to re-encrypt all existing records with proper contexts

### Phase 3: Data Migration (AFTER Phase 2)

**Goal**: Re-encrypt all existing data with proper encryption contexts

**Process**:
1. Decrypt existing data (using old `Decrypt()` method with empty context)
2. Re-encrypt with context (using new `EncryptWithContext()` method)
3. Update database records with new ciphertexts

**Migration Scripts** (need update):
- `scripts/data-migration/migrate_users.go` - add context "user:email", "user:first_name", "user:last_name"
- `scripts/data-migration/migrate_guest_orders.go` - add contexts for customer fields
- `scripts/data-migration/migrate_tenant_configs.go` - add contexts for payment credentials

**Example Migration**:
```go
// Old encryption (no context)
oldCiphertext := row.Email // From database

// Decrypt old data
plaintext, err := encryptor.Decrypt(ctx, oldCiphertext)

// Re-encrypt with context
newCiphertext, err := encryptor.EncryptWithContext(ctx, plaintext, "user:email")

// Update database
_, err = db.ExecContext(ctx, "UPDATE users SET email = $1 WHERE id = $2", newCiphertext, row.ID)
```

**Downtime Strategy**:
- **Option A (Zero Downtime)**: Dual-write migration
  1. Deploy Phase 2 code (supports both old and new encryption)
  2. Run background migration (re-encrypt all records)
  3. Switch to context-only mode after migration complete

- **Option B (Maintenance Window)**: Stop-the-world migration
  1. Stop all services
  2. Run migration scripts
  3. Deploy Phase 2 code
  4. Start all services

**Recommendation**: Option A (zero downtime) for production

### Phase 4: Search Optimization (OPTIONAL)

**Goal**: Leverage deterministic encryption for efficient encrypted field search

**Before** (full table scan):
```go
// Decrypt ALL users to find by email (O(n))
rows, err := db.Query("SELECT * FROM users")
for rows.Next() {
    // Decrypt each row's email, compare
}
```

**After** (indexed search):
```go
// Encrypt search term, query directly (O(log n) with index)
searchEmail := "user@example.com"
encryptedSearch, err := encryptor.EncryptWithContext(ctx, searchEmail, "user:email")
row := db.QueryRow("SELECT * FROM users WHERE email = $1", encryptedSearch)
```

**Performance Improvement**:
- **Before**: O(n) - decrypt all records
- **After**: O(log n) - indexed search on encrypted value
- **Example**: 1,000,000 users - 1,000,000 decryptions → 1 encryption + 1 index lookup

---

## Security Considerations

### Deterministic Encryption Trade-offs

**Advantages**:
- ✅ Efficient encrypted field search (database indexes work)
- ✅ Deduplication (same data stored once)
- ✅ No additional search hash tables needed

**Limitations**:
- ⚠️ Same plaintext + context always produces same ciphertext (by design)
- ⚠️ Frequency analysis possible within same context (attacker can see which records have same value)
- ⚠️ Not suitable for high-entropy data (e.g., random tokens, UUIDs)

### When to Use Convergent Encryption

**✅ Good Use Cases**:
- **Email addresses**: Low entropy, need efficient search, acceptable to see frequency
- **Phone numbers**: Low entropy, need efficient search
- **Names**: Low entropy, need efficient search
- **Addresses**: Low entropy, need geocoding/search

**❌ Bad Use Cases**:
- **Random tokens**: High entropy, no search needed, use random encryption
- **Session IDs**: High entropy, no search needed
- **Password hashes**: Already hashed, no encryption needed
- **Credit card numbers**: Regulated, may require random encryption

### Context Isolation Security

**Threat Model**:
- **Attacker Goal**: Correlate same email across multiple tables (users, guest_orders, invitations)
- **Defense**: Use different encryption contexts for each table/field
- **Result**: Same email encrypted with different contexts produces different ciphertexts

**Example**:
```go
// user@example.com encrypted in users table
userCipher := Encrypt("user@example.com", "user:email")
// Result: vault:v1:aBcD1234...

// user@example.com encrypted in guest_orders table
guestCipher := Encrypt("user@example.com", "guest_order:email")
// Result: vault:v1:xYz9876... (DIFFERENT ciphertext)

// Attacker cannot correlate that both records belong to same email
```

### HMAC Integrity Verification

**Unchanged**:
- HMAC still appended to all ciphertexts (format: `ciphertext:hmac`)
- Verified before decryption to detect tampering
- Protects against ciphertext manipulation attacks

---

## Testing

### Unit Tests

**Test Cases**:
1. **Deterministic Encryption**: Same plaintext + context produces same ciphertext
   ```go
   cipher1, _ := encryptor.EncryptWithContext(ctx, "test@example.com", "user:email")
   cipher2, _ := encryptor.EncryptWithContext(ctx, "test@example.com", "user:email")
   assert.Equal(t, cipher1, cipher2) // Same ciphertext
   ```

2. **Context Isolation**: Same plaintext + different contexts produce different ciphertexts
   ```go
   cipher1, _ := encryptor.EncryptWithContext(ctx, "test@example.com", "user:email")
   cipher2, _ := encryptor.EncryptWithContext(ctx, "test@example.com", "guest_order:email")
   assert.NotEqual(t, cipher1, cipher2) // Different ciphertexts
   ```

3. **Backward Compatibility**: Old `Encrypt()` method still works
   ```go
   cipher1, _ := encryptor.Encrypt(ctx, "test@example.com") // Old method
   cipher2, _ := encryptor.Encrypt(ctx, "test@example.com") // Old method
   // May or may not be equal (depends on Vault key configuration)
   ```

4. **Decryption with Context**: Must use same context for decryption
   ```go
   cipher, _ := encryptor.EncryptWithContext(ctx, "test@example.com", "user:email")
   plain, _ := encryptor.DecryptWithContext(ctx, cipher, "user:email")
   assert.Equal(t, "test@example.com", plain)
   
   // Wrong context should fail or return wrong plaintext
   wrong, err := encryptor.DecryptWithContext(ctx, cipher, "guest_order:email")
   assert.Error(t, err) // Or assert.NotEqual(t, "test@example.com", wrong)
   ```

### Integration Tests

**Test Scenarios**:
1. **Encrypted Field Search**: Query users by encrypted email
   ```go
   // Create user with email
   user := &User{Email: "test@example.com"}
   repo.Create(ctx, user)
   
   // Search by encrypted email
   found, err := repo.GetByEmail(ctx, "test@example.com")
   assert.NoError(t, err)
   assert.Equal(t, user.ID, found.ID)
   ```

2. **Cross-Table Privacy**: Same email in different tables produces different ciphertexts
   ```sql
   SELECT u.email, g.customer_email 
   FROM users u, guest_orders g 
   WHERE u.email = g.customer_email;
   -- Should return 0 rows (different ciphertexts even if same email)
   ```

### Performance Benchmarks

**Benchmark Results** (expected):
```
BenchmarkEncrypt-4                 1000    1200000 ns/op    (1.2ms per encryption)
BenchmarkEncryptWithContext-4      1000    1200000 ns/op    (same as Encrypt)
BenchmarkDecrypt-4                 1000    1200000 ns/op    (1.2ms per decryption)
BenchmarkDecryptWithContext-4      1000    1200000 ns/op    (same as Decrypt)
```

**Search Performance** (after Phase 4):
```
BenchmarkSearchEncrypted-4         10000    150000 ns/op    (0.15ms with index)
BenchmarkSearchPlaintext-4         10000    100000 ns/op    (0.10ms baseline)
```

---

## Rollback Plan

### If Issues Discovered

**Immediate Rollback** (revert to non-deterministic encryption):
```bash
# 1. Revert Vault key configuration
docker exec -e VAULT_TOKEN=dev-root-token pos-vault \
  vault write transit/keys/pos-encryption-key/config \
  convergent_encryption=false

# 2. Revert VaultClient code
git revert <commit-hash>

# 3. Redeploy all services
docker-compose up -d --build
```

**Partial Rollback** (keep infrastructure, revert repository changes):
```bash
# Keep VaultClient with context support (backward compatible)
# Revert repository changes to use old Encrypt() method
git revert <commit-hash-repo-changes>
```

### Data Recovery

**If Data Corrupted During Migration**:
```sql
-- Restore from backup taken before migration
pg_restore -h localhost -U postgres -d pos_db backup_before_migration.dump

-- Or rollback specific tables
BEGIN;
  DELETE FROM users WHERE updated_at > '<migration_start_time>';
  INSERT INTO users SELECT * FROM users_backup;
COMMIT;
```

---

## Documentation Updates

**Files Modified**:
- `vault/vault-init.sh` - Transit key configuration with convergent encryption
- `backend/*/src/utils/encryption.go` - VaultClient with context-based methods (all 6 services)
- `docs/DETERMINISTIC_ENCRYPTION_REFACTOR.md` - This document

**Files to Update** (Phase 2):
- `backend/*/src/repository/*.go` - Update all repositories to use EncryptWithContext
- `scripts/data-migration/*.go` - Update migration scripts with encryption contexts
- `docs/ENCRYPTION_VERIFICATION_COMPLETE.md` - Add deterministic encryption verification steps

---

## References

### Vault Documentation
- [Transit Secrets Engine](https://developer.hashicorp.com/vault/docs/secrets/transit)
- [Convergent Encryption](https://developer.hashicorp.com/vault/docs/secrets/transit#convergent-encryption)
- [Key Derivation](https://developer.hashicorp.com/vault/api-docs/secret/transit#derived)

### Academic Papers
- [Deterministic Encryption](https://en.wikipedia.org/wiki/Deterministic_encryption)
- [Searchable Encryption](https://en.wikipedia.org/wiki/Searchable_encryption)

### Internal Documentation
- [VAULT_QUICK_START.md](./VAULT_QUICK_START.md) - Vault setup guide
- [ENCRYPTION_VERIFICATION_COMPLETE.md](./ENCRYPTION_VERIFICATION_COMPLETE.md) - Encryption testing
- [IP_ADDRESS_ENCRYPTION_FIX.md](./IP_ADDRESS_ENCRYPTION_FIX.md) - Consent IP address encryption

---

## Next Steps

**Immediate** (Phase 1 Complete):
- ✅ Vault Transit key configured with convergent encryption
- ✅ VaultClient updated with context-based methods
- ✅ All services have updated encryption.go files
- ✅ Backward compatibility maintained

**Next Phase** (Phase 2 - Repository Refactoring):
1. Update UserRepository to use context "user:email"
2. Update GuestOrderRepository to use context "guest_order:email"
3. Update all other repositories with appropriate contexts
4. Add unit tests for deterministic encryption
5. Run integration tests to verify search functionality

**Future** (Phase 3 - Data Migration):
1. Create data migration scripts with context support
2. Test migration on staging environment
3. Run migration on production (zero-downtime strategy)
4. Verify all data encrypted with correct contexts

**Optional** (Phase 4 - Search Optimization):
1. Remove search hash fields (email_hash, phone_hash)
2. Update queries to use encrypted field search
3. Add database indexes on encrypted fields
4. Measure performance improvements

---

## Approval

**Technical Review**: [ ] Pending  
**Security Review**: [ ] Pending  
**Performance Review**: [ ] Pending  
**Production Deployment**: [ ] Pending

**Deployment Target**: After Phase 1 complete, Phase 2 development begins
