# Research: Encryption at Rest for Go Applications with PostgreSQL

**Feature**: Indonesian Data Protection Compliance (UU PDP No.27 Tahun 2022)  
**Research Date**: January 2, 2026  
**Status**: Complete  
**Researcher**: GitHub Copilot

---

## Decision: Application-Layer Encryption with Go crypto/cipher

### Library/Tool Choice
**Primary**: Go standard library `crypto/cipher` with `crypto/aes`  
**Key Management**: HashiCorp Vault (production), file-based with restricted permissions (development)  
**Algorithm**: AES-256-GCM (Galois/Counter Mode)

### Encryption Algorithm Recommendation
**AES-256-GCM** for the following reasons:
- **FIPS 140-2 compliant** - meets government security standards
- **Authenticated encryption** - provides both confidentiality and integrity
- **Hardware acceleration** - CPU AES-NI support on modern processors
- **Standard library support** - `crypto/cipher.NewGCM()` with `crypto/aes`
- **Nonce handling** - Go 1.24+ introduces `NewGCMWithRandomNonce()` for automatic nonce management

**Alternative for high-throughput scenarios**: ChaCha20-Poly1305
- Better performance on systems without AES-NI hardware support
- Constant-time implementation (resistant to timing attacks)
- Available in `golang.org/x/crypto/chacha20poly1305`

### Key Management Strategy

**Production (Recommended)**: HashiCorp Vault
```
Vault Approach:
- Keys stored in Vault Transit Secrets Engine
- Application retrieves keys via API with AppRole authentication
- Keys never written to disk
- Automatic key rotation support
- Audit trail for all key access
- Centralized key lifecycle management
```

**Development**: File-based with restricted permissions
```
File-based Approach (dev/staging only):
- Keys stored in /etc/pos/keys/ directory
- Permissions: 400 (read-only by owner)
- Owner: application service user (pos-service)
- Keys loaded at startup from environment variable pointing to file path
- Keys in base64 format, 32 bytes for AES-256
```

**Key Rotation Strategy**:
- Implement versioned encryption keys (key_version column)
- New records encrypted with latest key version
- Background job re-encrypts old records on key rotation
- Support multiple active key versions during transition period
- Rotation frequency: Every 90 days (P1 data), 180 days (P2/P3 data)

---

## Rationale

### Security Properties

#### AES-256-GCM Advantages:
1. **Authenticated Encryption with Associated Data (AEAD)**
   - Prevents tampering - any modification to ciphertext is detected
   - Protects associated metadata (e.g., tenant_id, record_id) without encryption
   - Single operation for encryption + authentication (better than encrypt-then-MAC)

2. **Proven Security**
   - NIST approved (SP 800-38D)
   - No known practical attacks when used correctly
   - 256-bit key provides quantum-resistant security for foreseeable future

3. **Constant-Time Operations** (with hardware support)
   - AES-NI instruction set eliminates timing side-channels
   - Go's `crypto/aes` automatically uses AES-NI when available
   - Falls back to software implementation on older CPUs

4. **Nonce/IV Management**
   - 96-bit (12-byte) nonce standard for GCM mode
   - Go 1.24+ `NewGCMWithRandomNonce()` handles nonce automatically
   - Nonce prepended to ciphertext (no separate storage needed)
   - Critical: NEVER reuse nonce with same key

#### Application-Layer vs Database-Level Encryption:

**Application-layer (chosen)**:
- ✅ Fine-grained control (encrypt specific fields, not whole tables)
- ✅ Keys separate from database (principle of least privilege)
- ✅ Compatible with any database (portable across PostgreSQL versions)
- ✅ Can integrate with external key management (Vault, KMS)
- ✅ Encryption logic testable in unit tests
- ❌ Requires code changes for new encrypted fields
- ❌ Cannot query encrypted fields with SQL (must decrypt in app)

**Database-level (pgcrypto rejected)**:
- ✅ Transparent to application code
- ✅ Can use SQL functions to query encrypted data
- ❌ Keys often stored in database (same attack surface)
- ❌ PostgreSQL-specific (vendor lock-in)
- ❌ Performance overhead on all queries
- ❌ Cannot easily integrate with external KMS
- ❌ Complex key rotation (requires database triggers)

### Performance Characteristics

#### Benchmark Results (Expected):
Based on Go crypto benchmarks and POS system characteristics:

```
AES-256-GCM Encryption (with AES-NI):
- Small fields (email, phone): ~1-2 µs per operation
- Medium fields (address): ~5-10 µs per operation
- Large fields (JSON config): ~50-100 µs per field

ChaCha20-Poly1305 (software):
- ~20-30% faster than AES-GCM without AES-NI
- ~50-60% slower than AES-GCM with AES-NI
- Consistent performance regardless of CPU

Database Impact:
- Connection pool: No impact (encryption before query execution)
- Storage overhead: +28 bytes per encrypted field (nonce 12B + tag 16B)
- Index performance: Cannot index encrypted fields (design limitation)
- Query performance: +10-20% latency for reads with encrypted fields
```

#### Optimization Strategies:
1. **Selective Encryption** - Only encrypt PII fields, not entire records
2. **Batch Operations** - Encrypt multiple fields in single transaction
3. **Connection Pooling** - Reuse database connections (already implemented)
4. **Lazy Decryption** - Decrypt only when field is accessed in API response
5. **Caching Considerations** - NEVER cache decrypted PII in Redis/memory

#### Performance Testing Plan:
- Benchmark encryption/decryption operations (Go benchmarks)
- Load test with 1000 concurrent users (realistic POS traffic)
- Measure p50, p95, p99 latency with encryption enabled
- Compare encrypted vs non-encrypted field query performance
- Test key rotation impact on production traffic

### Integration Complexity

#### Implementation Effort Estimate:

**Low Complexity (2-3 days)**:
1. Create `crypto` utility package with encrypt/decrypt functions
2. Implement key loader (file-based for MVP)
3. Add struct tags for automatic encryption (e.g., `db:"email" encrypt:"pii"`)
4. Create repository layer interceptor for encrypt/decrypt
5. Write comprehensive unit tests

**Medium Complexity (3-5 days)**:
1. Integrate HashiCorp Vault client
2. Implement key rotation mechanism
3. Create database migration to add key_version columns
4. Background worker to re-encrypt old data
5. Monitoring and alerting for encryption failures

**High Complexity (5-7 days)**:
1. Implement transparent encryption with GORM hooks
2. Support multiple encryption key versions concurrently
3. Audit logging for all encryption/decryption operations
4. Performance optimization and caching strategy
5. Integration tests with real database and key rotation

#### Go Ecosystem Maturity:
- **crypto/cipher**: Standard library, stable since Go 1.2
- **crypto/aes**: Hardware acceleration widely available
- **HashiCorp Vault Go client**: Official library, well-maintained
- **GORM**: Popular ORM with hook support for transparent encryption
- **Testify**: Standard testing framework, easy to mock crypto operations

### Operational Maintainability

#### Deployment Considerations:

**Development Environment**:
```bash
# Generate encryption key
openssl rand -base64 32 > /tmp/encryption.key

# Set environment variable
export ENCRYPTION_KEY_PATH=/tmp/encryption.key

# Application loads key at startup
./pos-service
```

**Production Environment**:
```bash
# HashiCorp Vault setup
vault secrets enable transit
vault write -f transit/keys/pos-encryption

# Application authenticates with AppRole
export VAULT_ADDR=https://vault.example.com
export VAULT_ROLE_ID=<role-id>
export VAULT_SECRET_ID=<secret-id>

# Application retrieves keys from Vault at runtime
./pos-service
```

#### Monitoring and Alerting:
- **Metrics to track**:
  - Encryption/decryption operation latency (p50, p95, p99)
  - Encryption operation failure rate
  - Key rotation success/failure events
  - Vault connection errors
  - Number of records encrypted per key version
  
- **Alerts to configure**:
  - Encryption failure rate > 0.1%
  - Vault connection down for > 5 minutes
  - Key rotation failure
  - Decryption failure (indicates key corruption or tampering)

#### Operational Runbooks:
1. **Key Rotation Procedure**:
   - Generate new key in Vault
   - Update application config with new key version
   - Deploy application (zero downtime, supports multiple key versions)
   - Run background job to re-encrypt old records
   - Monitor re-encryption progress
   - Decommission old key after 100% migration

2. **Key Loss Recovery**:
   - Restore key from Vault backup
   - If key permanently lost: Encrypted data is unrecoverable
   - Mitigation: Regular Vault backups, key escrow for disaster recovery

3. **Performance Degradation**:
   - Check Vault connection latency
   - Review encryption operation metrics
   - Scale application horizontally if needed
   - Consider caching decrypted data (with TTL and security review)

---

## Alternatives Considered

### 1. PostgreSQL pgcrypto Extension

**Why Rejected**:
- **Vendor lock-in**: pgcrypto is PostgreSQL-specific, makes database migration difficult
- **Key storage**: Keys typically stored in database or config files, increases attack surface
- **Performance**: All encryption/decryption happens in database, adds latency to every query
- **Testing complexity**: Requires database for unit tests, cannot easily mock encryption
- **Key rotation**: Requires complex triggers and stored procedures, error-prone
- **External KMS integration**: No native support for Vault, AWS KMS, etc.

**When it makes sense**:
- Legacy applications where code changes are prohibitive
- DBAs have full control over encryption (not application developers)
- All queries need to filter on encrypted fields (pgcrypto supports encrypted indexes)

### 2. Database Transparent Data Encryption (TDE)

**Example**: PostgreSQL TDE patches, AWS RDS encryption at rest

**Why Rejected**:
- **Coarse-grained**: Encrypts entire database/tablespace, not individual fields
- **No field-level control**: Cannot selectively encrypt PII columns
- **Key management**: Keys managed by database/cloud provider, less control
- **UU PDP compliance**: Insufficient for field-level PII protection requirements
- **Audit trail**: No visibility into which fields were accessed/decrypted

**When it makes sense**:
- Compliance requires "encryption at rest" (disk encryption sufficient)
- Protecting against physical theft of storage devices
- No need for field-level encryption granularity
- Cloud provider manages key rotation automatically

### 3. External Library: github.com/gtank/cryptopasta

**Why Rejected**:
- **Maintenance**: Last updated 2019, not actively maintained
- **Standard library sufficient**: Go's crypto/cipher provides same functionality
- **Additional dependency**: Adds external dependency for no clear benefit
- **Testing overhead**: Need to audit third-party crypto code
- **Community support**: Smaller community than standard library

**When it makes sense**:
- Need convenience wrappers around crypto/cipher (but we can write our own)
- Team unfamiliar with cryptography primitives (but standard lib has good docs)

### 4. ChaCha20-Poly1305 (Alternative Algorithm)

**Why Not Primary Choice** (but valid alternative):
- **No hardware acceleration**: Relies on software implementation
- **Slightly larger nonce**: 192-bit vs 96-bit for GCM
- **Less widespread**: AES-GCM is more commonly used (more tooling/examples)
- **Not in standard library**: Requires `golang.org/x/crypto` (semi-official but external)

**When to use**:
- Systems without AES-NI support (ARM, older x86 CPUs)
- Mobile/edge devices where power efficiency matters
- Maximum resistance to timing side-channels (always constant-time)

**Recommendation**: Support both algorithms, let configuration choose:
```go
type EncryptionAlgorithm string
const (
    AlgorithmAESGCM       EncryptionAlgorithm = "aes-256-gcm"
    AlgorithmChaCha20Poly1305 EncryptionAlgorithm = "chacha20-poly1305"
)
```

### 5. AWS KMS / Google Cloud KMS

**Why Not Primary Choice**:
- **Cloud vendor lock-in**: Ties application to specific cloud provider
- **Network latency**: Every encrypt/decrypt operation calls external API (50-200ms)
- **Cost**: Per-operation pricing can be expensive at scale
- **Single point of failure**: Application down if KMS API unavailable
- **Self-hosted requirement**: UU PDP prefers data sovereignty (Indonesia-hosted)

**When it makes sense**:
- Already fully committed to AWS/GCP ecosystem
- Compliance requires FIPS 140-2 Level 3+ (KMS provides hardware security modules)
- Team lacks cryptography expertise (KMS handles key management complexity)
- Can tolerate higher latency for encryption operations

**Hybrid approach** (viable):
- Use KMS to encrypt data encryption keys (DEKs)
- Store encrypted DEKs in application database
- Decrypt DEKs at startup and cache in memory
- Use DEKs for actual data encryption (envelope encryption)
- Reduces KMS API calls to startup + key rotation only

---

## Implementation Notes

### Code Patterns and Examples

#### 1. Encryption Utility Package

**File**: `backend/src/utils/crypto.go`

```go
package utils

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "fmt"
    "io"
)

// EncryptionService handles field-level encryption
type EncryptionService struct {
    aead cipher.AEAD
}

// NewEncryptionService creates encryption service with AES-256-GCM
func NewEncryptionService(key []byte) (*EncryptionService, error) {
    if len(key) != 32 {
        return nil, errors.New("encryption key must be 32 bytes for AES-256")
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher: %w", err)
    }

    // Use NewGCMWithRandomNonce for Go 1.24+ (automatic nonce handling)
    // Falls back to NewGCM for older versions
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }

    return &EncryptionService{aead: aead}, nil
}

// Encrypt encrypts plaintext and returns base64-encoded ciphertext
// Format: nonce + ciphertext + tag (automatically handled by GCM)
func (s *EncryptionService) Encrypt(plaintext string) (string, error) {
    if plaintext == "" {
        return "", nil // Empty strings not encrypted
    }

    // Generate random nonce (12 bytes for GCM)
    nonce := make([]byte, s.aead.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", fmt.Errorf("failed to generate nonce: %w", err)
    }

    // Encrypt and authenticate
    // Seal appends nonce + ciphertext + tag
    ciphertext := s.aead.Seal(nonce, nonce, []byte(plaintext), nil)

    // Base64 encode for storage in VARCHAR fields
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext and returns plaintext
func (s *EncryptionService) Decrypt(encoded string) (string, error) {
    if encoded == "" {
        return "", nil // Empty strings not encrypted
    }

    // Decode from base64
    ciphertext, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil {
        return "", fmt.Errorf("failed to decode base64: %w", err)
    }

    // Check minimum size (nonce + tag)
    nonceSize := s.aead.NonceSize()
    if len(ciphertext) < nonceSize {
        return "", errors.New("ciphertext too short")
    }

    // Extract nonce and ciphertext
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

    // Decrypt and verify authentication tag
    plaintext, err := s.aead.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", fmt.Errorf("decryption failed (key mismatch or tampered data): %w", err)
    }

    return string(plaintext), nil
}

// EncryptWithAD encrypts with Additional Data (AD) for context binding
// AD is authenticated but not encrypted (e.g., record ID, tenant ID)
func (s *EncryptionService) EncryptWithAD(plaintext string, additionalData []byte) (string, error) {
    if plaintext == "" {
        return "", nil
    }

    nonce := make([]byte, s.aead.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", fmt.Errorf("failed to generate nonce: %w", err)
    }

    // Seal with additional data (prevents ciphertext from being moved to different context)
    ciphertext := s.aead.Seal(nonce, nonce, []byte(plaintext), additionalData)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptWithAD decrypts with Additional Data verification
func (s *EncryptionService) DecryptWithAD(encoded string, additionalData []byte) (string, error) {
    if encoded == "" {
        return "", nil
    }

    ciphertext, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil {
        return "", fmt.Errorf("failed to decode base64: %w", err)
    }

    nonceSize := s.aead.NonceSize()
    if len(ciphertext) < nonceSize {
        return "", errors.New("ciphertext too short")
    }

    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

    // Open with additional data verification
    plaintext, err := s.aead.Open(nil, nonce, ciphertext, additionalData)
    if err != nil {
        return "", fmt.Errorf("decryption failed: %w", err)
    }

    return string(plaintext), nil
}
```

#### 2. Repository Pattern with Transparent Encryption

**File**: `backend/user-service/src/repository/user_repository.go`

```go
package repository

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/pos/backend/src/utils"
)

type UserRepository struct {
    db         *sql.DB
    encryption *utils.EncryptionService
}

func NewUserRepository(db *sql.DB, encryption *utils.EncryptionService) *UserRepository {
    return &UserRepository{
        db:         db,
        encryption: encryption,
    }
}

type User struct {
    ID            string
    TenantID      string
    Email         string // Encrypted
    FirstName     string
    LastName      string
    Phone         string // Encrypted
    PasswordHash  string
    Role          string
    KeyVersion    int    // Track which encryption key was used
}

// Create inserts user with encrypted PII fields
func (r *UserRepository) Create(ctx context.Context, user *User) error {
    // Encrypt PII fields before insertion
    encryptedEmail, err := r.encryption.Encrypt(user.Email)
    if err != nil {
        return fmt.Errorf("failed to encrypt email: %w", err)
    }

    encryptedPhone, err := r.encryption.Encrypt(user.Phone)
    if err != nil {
        return fmt.Errorf("failed to encrypt phone: %w", err)
    }

    query := `
        INSERT INTO users (id, tenant_id, email_encrypted, first_name, last_name, 
                          phone_encrypted, password_hash, role, key_version)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `

    _, err = r.db.ExecContext(ctx, query,
        user.ID,
        user.TenantID,
        encryptedEmail,
        user.FirstName,
        user.LastName,
        encryptedPhone,
        user.PasswordHash,
        user.Role,
        1, // Current key version
    )

    return err
}

// GetByID retrieves user and decrypts PII fields
func (r *UserRepository) GetByID(ctx context.Context, id string, tenantID string) (*User, error) {
    query := `
        SELECT id, tenant_id, email_encrypted, first_name, last_name, 
               phone_encrypted, password_hash, role, key_version
        FROM users
        WHERE id = $1 AND tenant_id = $2
    `

    var user User
    var encryptedEmail, encryptedPhone string

    err := r.db.QueryRowContext(ctx, query, id, tenantID).Scan(
        &user.ID,
        &user.TenantID,
        &encryptedEmail,
        &user.FirstName,
        &user.LastName,
        &encryptedPhone,
        &user.PasswordHash,
        &user.Role,
        &user.KeyVersion,
    )

    if err != nil {
        return nil, err
    }

    // Decrypt PII fields after retrieval
    user.Email, err = r.encryption.Decrypt(encryptedEmail)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt email: %w", err)
    }

    user.Phone, err = r.encryption.Decrypt(encryptedPhone)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt phone: %w", err)
    }

    return &user, nil
}

// GetByEmailHash retrieves user by hashed email (for login)
// NOTE: Cannot query on encrypted email directly, use hash for lookups
func (r *UserRepository) GetByEmailHash(ctx context.Context, emailHash string, tenantID string) (*User, error) {
    query := `
        SELECT id, tenant_id, email_encrypted, first_name, last_name, 
               phone_encrypted, password_hash, role, key_version
        FROM users
        WHERE email_hash = $1 AND tenant_id = $2
    `

    // ... similar to GetByID with decryption ...
}
```

#### 3. Key Management with HashiCorp Vault

**File**: `backend/src/config/vault.go`

```go
package config

import (
    "context"
    "encoding/base64"
    "fmt"
    "os"

    vault "github.com/hashicorp/vault/api"
)

type VaultConfig struct {
    Address  string
    RoleID   string
    SecretID string
}

type KeyManager struct {
    client *vault.Client
    mountPath string
}

func NewKeyManager() (*KeyManager, error) {
    cfg := vault.DefaultConfig()
    cfg.Address = os.Getenv("VAULT_ADDR")

    client, err := vault.NewClient(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create vault client: %w", err)
    }

    // Authenticate with AppRole
    roleID := os.Getenv("VAULT_ROLE_ID")
    secretID := os.Getenv("VAULT_SECRET_ID")

    data := map[string]interface{}{
        "role_id":   roleID,
        "secret_id": secretID,
    }

    resp, err := client.Logical().Write("auth/approle/login", data)
    if err != nil {
        return nil, fmt.Errorf("vault login failed: %w", err)
    }

    client.SetToken(resp.Auth.ClientToken)

    return &KeyManager{
        client:    client,
        mountPath: "transit",
    }, nil
}

// GetEncryptionKey retrieves encryption key from Vault
func (km *KeyManager) GetEncryptionKey(keyName string) ([]byte, error) {
    path := fmt.Sprintf("%s/keys/%s", km.mountPath, keyName)

    secret, err := km.client.Logical().Read(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read key from vault: %w", err)
    }

    // For Transit engine, keys are not exported directly
    // Instead, use Vault's encrypt/decrypt endpoints
    // OR use datakey generation for local encryption
    
    // Generate data encryption key (DEK)
    dataKeyPath := fmt.Sprintf("%s/datakey/plaintext/%s", km.mountPath, keyName)
    dekResp, err := km.client.Logical().Write(dataKeyPath, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to generate DEK: %w", err)
    }

    // DEK is base64 encoded
    dekB64 := dekResp.Data["plaintext"].(string)
    dek, err := base64.StdEncoding.DecodeString(dekB64)
    if err != nil {
        return nil, fmt.Errorf("failed to decode DEK: %w", err)
    }

    return dek, nil
}

// RotateKey initiates key rotation in Vault
func (km *KeyManager) RotateKey(keyName string) error {
    path := fmt.Sprintf("%s/keys/%s/rotate", km.mountPath, keyName)
    _, err := km.client.Logical().Write(path, nil)
    return err
}
```

#### 4. Struct Tags for Automatic Encryption (Advanced)

**File**: `backend/src/repository/encrypted_repository.go`

```go
package repository

import (
    "database/sql"
    "fmt"
    "reflect"

    "github.com/pos/backend/src/utils"
)

// EncryptedField tag marks fields for automatic encryption
// Usage: `db:"email" encrypt:"pii"`
type EncryptedRepository struct {
    db         *sql.DB
    encryption *utils.EncryptionService
}

// ScanWithDecryption automatically decrypts tagged fields
func (r *EncryptedRepository) ScanWithDecryption(rows *sql.Rows, dest interface{}) error {
    v := reflect.ValueOf(dest).Elem()
    t := v.Type()

    // Build slice of scan targets
    scanTargets := make([]interface{}, t.NumField())
    encryptedFields := make(map[int]bool)

    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        encryptTag := field.Tag.Get("encrypt")

        if encryptTag == "pii" {
            // Encrypted field - scan into temporary string
            var temp string
            scanTargets[i] = &temp
            encryptedFields[i] = true
        } else {
            // Regular field - scan directly
            scanTargets[i] = v.Field(i).Addr().Interface()
        }
    }

    // Scan row
    if err := rows.Scan(scanTargets...); err != nil {
        return err
    }

    // Decrypt encrypted fields
    for i, isEncrypted := range encryptedFields {
        if isEncrypted {
            encryptedValue := scanTargets[i].(*string)
            decrypted, err := r.encryption.Decrypt(*encryptedValue)
            if err != nil {
                return fmt.Errorf("failed to decrypt field %s: %w", t.Field(i).Name, err)
            }
            v.Field(i).SetString(decrypted)
        }
    }

    return nil
}
```

### Gotchas to Avoid

#### 1. **CRITICAL: Never Reuse Nonces**
```go
// ❌ WRONG - Reusing same nonce breaks GCM security
nonce := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
ciphertext := aead.Seal(nonce, nonce, plaintext, nil)

// ✅ CORRECT - Generate random nonce for every encryption
nonce := make([]byte, aead.NonceSize())
io.ReadFull(rand.Reader, nonce)
ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
```

**Why**: Reusing nonces with same key completely breaks GCM's security guarantees. Attacker can recover plaintext and forge ciphertexts.

#### 2. **Cannot Query Encrypted Fields with SQL**
```go
// ❌ WRONG - Cannot search encrypted email
query := "SELECT * FROM users WHERE email_encrypted = $1"

// ✅ CORRECT - Use separate hash column for lookups
query := "SELECT * FROM users WHERE email_hash = $1"
// Hash email before querying: SHA-256(email + tenant_id)
```

**Mitigation**: Store hash of searchable encrypted fields separately. Use HMAC or keyed hash to prevent rainbow table attacks.

#### 3. **Storage Overhead**
```plaintext
Original email: "user@example.com" (16 bytes)
Encrypted storage:
  - Nonce: 12 bytes
  - Ciphertext: 16 bytes (same as plaintext)
  - Authentication tag: 16 bytes
  - Base64 encoding overhead: ~33% increase
  - Total: ~60 bytes in database

Use VARCHAR(255) for email fields (was VARCHAR(100))
```

#### 4. **Key Rotation Complexity**
```go
// ❌ WRONG - Rotate key without re-encrypting existing data
// Result: Old records cannot be decrypted with new key

// ✅ CORRECT - Multi-version key support
type EncryptionService struct {
    keys map[int]*cipher.AEAD // key_version -> AEAD
    currentVersion int
}

func (s *EncryptionService) Decrypt(ciphertext string, keyVersion int) (string, error) {
    aead := s.keys[keyVersion]
    if aead == nil {
        return "", errors.New("key version not found")
    }
    // Decrypt with appropriate key version
}
```

#### 5. **Performance: Don't Encrypt Everything**
```go
// ❌ WRONG - Encrypting non-sensitive fields
type Product struct {
    Name        string `encrypt:"pii"` // NOT PII!
    Description string `encrypt:"pii"` // NOT PII!
    Price       int64  `encrypt:"pii"` // NOT PII!
}

// ✅ CORRECT - Only encrypt PII
type User struct {
    ID       string
    Email    string `encrypt:"pii"` // PII - encrypt
    Phone    string `encrypt:"pii"` // PII - encrypt
    Name     string // Not sensitive - no encryption
}
```

**Rule**: Only encrypt fields that are:
- Personal Identifiable Information (PII)
- Financial data (credit cards, bank accounts)
- Health information
- Credentials/secrets

#### 6. **Concurrent Access with Same Key**
```go
// ✅ SAFE - cipher.AEAD is safe for concurrent use
// No mutex needed around Seal/Open operations
var encryptionService *EncryptionService

func Handler1() {
    encrypted := encryptionService.Encrypt("data1") // Safe
}

func Handler2() {
    encrypted := encryptionService.Encrypt("data2") // Safe
}
```

**Note**: Go's `cipher.AEAD` implementations are goroutine-safe. No need for locks.

#### 7. **Error Handling: Decryption Failures**
```go
// ❌ WRONG - Exposing decryption errors to users
func GetUser(id string) (*User, error) {
    user, err := repo.GetByID(id)
    if err != nil {
        return nil, err // Might expose "decryption failed: key mismatch"
    }
}

// ✅ CORRECT - Log detailed errors, return generic message
func GetUser(id string) (*User, error) {
    user, err := repo.GetByID(id)
    if err != nil {
        logger.Errorf("Failed to get user %s: %v", id, err) // Detailed logging
        return nil, errors.New("failed to retrieve user") // Generic error
    }
}
```

**Why**: Decryption failures might indicate:
- Key rotation in progress
- Data tampering (security incident)
- Key corruption

Log detailed errors for investigation, but don't expose to end users.

### Testing Strategies

#### 1. Unit Tests for Encryption Service

**File**: `backend/src/utils/crypto_test.go`

```go
package utils

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestEncryptionService(t *testing.T) {
    // Generate test key (32 bytes for AES-256)
    key := make([]byte, 32)
    for i := range key {
        key[i] = byte(i)
    }

    service, err := NewEncryptionService(key)
    require.NoError(t, err)

    t.Run("encrypt and decrypt successfully", func(t *testing.T) {
        plaintext := "user@example.com"

        ciphertext, err := service.Encrypt(plaintext)
        require.NoError(t, err)
        assert.NotEqual(t, plaintext, ciphertext)

        decrypted, err := service.Decrypt(ciphertext)
        require.NoError(t, err)
        assert.Equal(t, plaintext, decrypted)
    })

    t.Run("empty string not encrypted", func(t *testing.T) {
        ciphertext, err := service.Encrypt("")
        require.NoError(t, err)
        assert.Equal(t, "", ciphertext)
    })

    t.Run("decryption with wrong key fails", func(t *testing.T) {
        plaintext := "secret-data"
        ciphertext, _ := service.Encrypt(plaintext)

        // Create service with different key
        wrongKey := make([]byte, 32)
        wrongService, _ := NewEncryptionService(wrongKey)

        _, err := wrongService.Decrypt(ciphertext)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "decryption failed")
    })

    t.Run("tampered ciphertext detected", func(t *testing.T) {
        plaintext := "important-data"
        ciphertext, _ := service.Encrypt(plaintext)

        // Tamper with ciphertext
        tamperedCiphertext := ciphertext[:len(ciphertext)-5] + "XXXXX"

        _, err := service.Decrypt(tamperedCiphertext)
        assert.Error(t, err)
    })

    t.Run("encrypt with additional data", func(t *testing.T) {
        plaintext := "email@example.com"
        additionalData := []byte("tenant-123")

        ciphertext, err := service.EncryptWithAD(plaintext, additionalData)
        require.NoError(t, err)

        // Decrypt with correct AD
        decrypted, err := service.DecryptWithAD(ciphertext, additionalData)
        require.NoError(t, err)
        assert.Equal(t, plaintext, decrypted)

        // Decrypt with wrong AD fails
        wrongAD := []byte("tenant-456")
        _, err = service.DecryptWithAD(ciphertext, wrongAD)
        assert.Error(t, err)
    })
}

func TestKeyValidation(t *testing.T) {
    t.Run("reject invalid key length", func(t *testing.T) {
        invalidKey := make([]byte, 16) // AES-128, need AES-256
        _, err := NewEncryptionService(invalidKey)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "must be 32 bytes")
    })
}

func BenchmarkEncryption(b *testing.B) {
    key := make([]byte, 32)
    service, _ := NewEncryptionService(key)
    plaintext := "user@example.com"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = service.Encrypt(plaintext)
    }
}

func BenchmarkDecryption(b *testing.B) {
    key := make([]byte, 32)
    service, _ := NewEncryptionService(key)
    ciphertext, _ := service.Encrypt("user@example.com")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = service.Decrypt(ciphertext)
    }
}
```

#### 2. Integration Tests with Database

**File**: `backend/user-service/tests/integration/encryption_test.go`

```go
package integration

import (
    "context"
    "database/sql"
    "testing"

    "github.com/pos/backend/src/utils"
    "github.com/pos/backend/user-service/src/repository"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestEncryptedUserRepository(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()

    // Initialize encryption service
    testKey := make([]byte, 32)
    encryption, _ := utils.NewEncryptionService(testKey)

    repo := repository.NewUserRepository(db, encryption)
    ctx := context.Background()

    t.Run("create user with encrypted fields", func(t *testing.T) {
        user := &repository.User{
            ID:        "user-123",
            TenantID:  "tenant-456",
            Email:     "test@example.com",
            Phone:     "+6281234567890",
            FirstName: "John",
            LastName:  "Doe",
        }

        err := repo.Create(ctx, user)
        require.NoError(t, err)

        // Verify data is encrypted in database
        var encryptedEmail string
        err = db.QueryRow("SELECT email_encrypted FROM users WHERE id = $1", user.ID).Scan(&encryptedEmail)
        require.NoError(t, err)

        // Encrypted email should not match plaintext
        assert.NotEqual(t, user.Email, encryptedEmail)

        // Should be base64 encoded
        assert.Regexp(t, `^[A-Za-z0-9+/]+=*$`, encryptedEmail)
    })

    t.Run("retrieve user with decrypted fields", func(t *testing.T) {
        user, err := repo.GetByID(ctx, "user-123", "tenant-456")
        require.NoError(t, err)

        assert.Equal(t, "test@example.com", user.Email)
        assert.Equal(t, "+6281234567890", user.Phone)
    })

    t.Run("cannot decrypt with wrong key", func(t *testing.T) {
        // Create repo with different key
        wrongKey := make([]byte, 32)
        for i := range wrongKey {
            wrongKey[i] = 0xFF
        }
        wrongEncryption, _ := utils.NewEncryptionService(wrongKey)
        wrongRepo := repository.NewUserRepository(db, wrongEncryption)

        _, err := wrongRepo.GetByID(ctx, "user-123", "tenant-456")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "decryption failed")
    })
}

func setupTestDB(t *testing.T) *sql.DB {
    // Create test database connection
    // Run migrations
    // Return database handle
}
```

#### 3. Property-Based Tests (Fuzzing)

**File**: `backend/src/utils/crypto_fuzz_test.go`

```go
package utils

import (
    "testing"
)

func FuzzEncryptionRoundTrip(f *testing.F) {
    key := make([]byte, 32)
    service, _ := NewEncryptionService(key)

    // Seed corpus
    f.Add("test@example.com")
    f.Add("")
    f.Add("very long email address that might cause issues@example.com")
    f.Add("unicode-テスト@example.jp")

    f.Fuzz(func(t *testing.T, plaintext string) {
        ciphertext, err := service.Encrypt(plaintext)
        if err != nil {
            t.Skip() // Skip invalid inputs
        }

        decrypted, err := service.Decrypt(ciphertext)
        if err != nil {
            t.Fatalf("Decryption failed for input %q: %v", plaintext, err)
        }

        if decrypted != plaintext {
            t.Fatalf("Round-trip failed: got %q, want %q", decrypted, plaintext)
        }
    })
}
```

#### 4. Performance Tests

**File**: `backend/tests/performance/encryption_bench_test.go`

```go
package performance

import (
    "testing"

    "github.com/pos/backend/src/utils"
)

func BenchmarkEncryptionSizes(b *testing.B) {
    key := make([]byte, 32)
    service, _ := utils.NewEncryptionService(key)

    testCases := []struct {
        name string
        data string
    }{
        {"small-email", "user@example.com"},
        {"large-email", "very.long.email.address.with.many.characters@example.com"},
        {"phone", "+6281234567890"},
        {"address", "Jl. Sudirman No. 123, Jakarta Selatan, DKI Jakarta 12345"},
        {"json-config", `{"key1":"value1","key2":"value2","key3":"value3"}`},
    }

    for _, tc := range testCases {
        b.Run(tc.name, func(b *testing.B) {
            for i := 0; i < b.N; i++ {
                _, _ = service.Encrypt(tc.data)
            }
        })
    }
}

func BenchmarkConcurrentEncryption(b *testing.B) {
    key := make([]byte, 32)
    service, _ := utils.NewEncryptionService(key)
    plaintext := "test@example.com"

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, _ = service.Encrypt(plaintext)
        }
    })
}
```

---

## Summary

### Recommended Approach
- **Algorithm**: AES-256-GCM (Go standard library `crypto/cipher`)
- **Key Management**: HashiCorp Vault (production), file-based (development)
- **Pattern**: Application-layer encryption via repository pattern
- **Performance**: ~1-10 µs per field with hardware acceleration
- **Maintenance**: Well-supported, standard library, mature ecosystem

### Next Steps for Implementation
1. Create `backend/src/utils/crypto.go` with encryption service
2. Integrate key loading from environment/Vault
3. Modify repositories to encrypt/decrypt PII fields
4. Add `key_version` column to encrypted tables
5. Write comprehensive unit and integration tests
6. Benchmark encryption overhead with realistic load
7. Document key rotation procedures in runbook

### Key Decisions Summary
| Aspect | Decision | Justification |
|--------|----------|---------------|
| **Library** | Go crypto/cipher (stdlib) | Mature, hardware-accelerated, no external deps |
| **Algorithm** | AES-256-GCM | FIPS compliant, authenticated encryption, proven |
| **Alternative** | ChaCha20-Poly1305 | For non-AES-NI systems, constant-time |
| **Layer** | Application-layer | Fine-grained control, portable, testable |
| **Key Mgmt** | Vault (prod), file (dev) | Secure, auditable, rotation support |
| **Pattern** | Repository pattern | Transparent to business logic, testable |
| **Searchability** | Hash searchable fields | Cannot query encrypted data directly |
| **Rotation** | Multi-version keys | Support gradual migration, zero downtime |

---

**Research Complete**: Ready for implementation planning phase.
