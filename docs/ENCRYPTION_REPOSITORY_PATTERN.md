# Implementation Guide: Repository Encryption Pattern (T052-T059)

## Overview
This guide documents the Dependency Injection pattern established for VaultClient encryption across all repositories. Use this pattern for remaining tasks T052-T059.

## Established Pattern (Reference: T050-T051)

### 1. Interface Definition
```go
// src/utils/encryption.go
type Encryptor interface {
    Encrypt(ctx context.Context, plaintext string) (string, error)
    Decrypt(ctx context.Context, ciphertext string) (string, error)
    EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error)
    DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error)
}
```

### 2. Repository Structure
```go
type XRepository struct {
    db        *sql.DB
    encryptor utils.Encryptor  // Interface, not concrete type
}

// For testing - accepts mock
func NewXRepository(db *sql.DB, encryptor utils.Encryptor) *XRepository {
    return &XRepository{
        db:        db,
        encryptor: encryptor,
    }
}

// For production - creates real VaultClient
func NewXRepositoryWithVault(db *sql.DB) (*XRepository, error) {
    vaultClient, err := utils.NewVaultClient()
    if err != nil {
        return nil, fmt.Errorf("failed to initialize VaultClient: %w", err)
    }
    return NewXRepository(db, vaultClient), nil
}
```

### 3. Helper Methods for Pointer Fields
```go
func (r *XRepository) encryptStringPtr(ctx context.Context, value *string) (string, error) {
    if value == nil || *value == "" {
        return "", nil
    }
    return r.encryptor.Encrypt(ctx, *value)
}

func (r *XRepository) decryptToStringPtr(ctx context.Context, encrypted string) (*string, error) {
    if encrypted == "" {
        return nil, nil
    }
    decrypted, err := r.encryptor.Decrypt(ctx, encrypted)
    if err != nil {
        return nil, err
    }
    return &decrypted, nil
}
```

### 4. Encryption on Write
```go
func (r *XRepository) Create(ctx context.Context, entity *models.Entity) error {
    // Encrypt PII fields before database write
    encryptedField1, err := r.encryptor.Encrypt(ctx, entity.Field1)
    if err != nil {
        return err
    }
    
    encryptedField2, err := r.encryptStringPtr(ctx, entity.Field2) // For pointers
    if err != nil {
        return err
    }
    
    // Execute query with encrypted values
    _, err = r.db.ExecContext(ctx, query, encryptedField1, encryptedField2, ...)
    return err
}
```

### 5. Decryption on Read
```go
func (r *XRepository) GetByID(ctx context.Context, id string) (*models.Entity, error) {
    var entity models.Entity
    var encryptedField1, encryptedField2 string
    
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &entity.ID,
        &encryptedField1,
        &encryptedField2,
        ...
    )
    
    if err != nil {
        return nil, err
    }
    
    // Decrypt PII fields after database read
    entity.Field1, err = r.encryptor.Decrypt(ctx, encryptedField1)
    if err != nil {
        return nil, err
    }
    
    entity.Field2, err = r.decryptToStringPtr(ctx, encryptedField2)
    if err != nil {
        return nil, err
    }
    
    return &entity, nil
}
```

## Task-Specific Implementation Notes

### T052-T053: GuestOrderRepository (order-service)
**Status**: Needs repository extraction from checkout_handler.go

**Current State**: Guest order insertion is in `CheckoutHandler.insertOrder()` method
**Required Action**:
1. Create `backend/order-service/src/repository/guest_order_repository.go`
2. Move SQL logic from handler to repository
3. Encrypt: customer_name, customer_phone, customer_email, ip_address
4. Add GetByReference method for order lookup

**Files to modify**:
- `backend/order-service/api/checkout_handler.go` - refactor to use repository
- Create new repository file

### T054: DeliveryAddressRepository (order-service)
**Status**: Check if repository exists

**Fields to encrypt**:
- address (string)
- latitude (*float64)
- longitude (*float64)
- geocoded_address (*string)

**Note**: Latitude/longitude encryption may impact geocoding queries - consider trade-offs

### T055: PasswordResetTokenRepository (auth-service)
**Status**: Check if repository exists

**Fields to encrypt**:
- token (string)

**Implementation**: Standard pattern, token is non-nullable string

### T056: InvitationRepository (user-service)
**Status**: Repository exists at `backend/user-service/src/repository/invitation_repository.go`

**Fields to encrypt**:
- email (string)
- token (string)

**Action**: Add encryptor to existing repository, update Create/Find methods

### T057: SessionRepository (auth-service)
**Status**: Check if repository exists

**Fields to encrypt**:
- session_id (string)
- ip_address (*string)

### T058: NotificationRepository (notification-service)
**Status**: Check if repository exists

**Fields to encrypt** (conditionally):
- recipient (string) - if contains PII
- message_body (*string) - if contains PII

**Special handling**: Check content before encryption (e.g., only encrypt if email/phone detected)

### T059: TenantConfigRepository (tenant-service)
**Status**: Check if repository exists

**Fields to encrypt**:
- midtrans_server_key (string)
- midtrans_client_key (string)

**Note**: These are API credentials, not PII, but still sensitive data requiring encryption

## Testing Strategy

### Unit Tests (per repository)
```go
// tests/repository_test.go
func TestXRepository_Create_WithMockEncryptor(t *testing.T) {
    mockEncryptor := &utils.MockEncryptor{
        EncryptFunc: func(ctx context.Context, plaintext string) (string, error) {
            return "encrypted:" + plaintext, nil
        },
    }
    
    repo := NewXRepository(testDB, mockEncryptor)
    // Test business logic without Vault dependency
}
```

### Integration Tests
```go
func TestXRepository_Create_WithRealVault(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    repo, err := NewXRepositoryWithVault(testDB)
    // Test with real Vault in integration environment
}
```

## Service Layer Updates

When repository signature changes, update service constructors:

```go
// Old (breaks after DI pattern)
func NewXService(db *sql.DB) *XService {
    return &XService{
        repo: repository.NewXRepository(db), // ERROR: missing encryptor parameter
    }
}

// New (compatible with DI pattern)
func NewXService(db *sql.DB) (*XService, error) {
    repo, err := repository.NewXRepositoryWithVault(db)
    if err != nil {
        return nil, fmt.Errorf("failed to create repository: %w", err)
    }
    return &XService{repo: repo}, nil
}

// For testing
func NewXServiceWithRepository(db *sql.DB, repo *repository.XRepository) *XService {
    return &XService{repo: repo}
}
```

## Checklist for Each Task

- [ ] Check if repository exists
- [ ] If not, create repository with standard structure
- [ ] Add `encryptor utils.Encryptor` field to struct
- [ ] Create `NewXRepository(db, encryptor)` constructor
- [ ] Create `NewXRepositoryWithVault(db)` convenience constructor
- [ ] Add helper methods for pointer fields if needed
- [ ] Update Create/Insert methods to encrypt before write
- [ ] Update Get/Find methods to decrypt after read
- [ ] Update Update methods to encrypt before write
- [ ] Update service layer to use new constructor signature
- [ ] Create test file in `tests/` subdirectory
- [ ] Add example tests with MockEncryptor
- [ ] Verify build succeeds: `go build`
- [ ] Verify tests pass: `go test ./...`
- [ ] Mark task complete in tasks.md

## Common Pitfalls

1. **Forgetting pointer helper methods** - Use encryptStringPtr/decryptToStringPtr for optional fields
2. **Not updating service layer** - Service constructors must handle error returns
3. **Missing test directories** - Create `tests/` subdirectory for each package
4. **Hardcoded VaultClient** - Always use interface, never create VaultClient in repository
5. **Batch operations** - Use EncryptBatch/DecryptBatch for performance on multiple records

## Performance Considerations

- Encrypt/decrypt operations add latency (~1-5ms per field with Vault)
- Use batch operations when processing multiple records
- Consider caching frequently accessed encrypted data if read-heavy
- Monitor Vault API rate limits in production

## Security Notes

- Encrypted values start with "vault:v1:" prefix (Vault Transit Engine format)
- HMAC integrity verification is built into VaultClient
- Never log plaintext PII values
- Always use context.Context for proper timeout/cancellation

## Next Steps After T052-T059

Once all repository encryption is complete:
1. Proceed to T060-T065: Log masking middleware integration
2. Then T066-T069: Data migration scripts for existing records
3. Complete User Story 1 (Encryption at Rest)
4. Begin User Story 5 (Consent Collection)

---

**Reference Implementation**: `backend/user-service/src/repository/user_repository.go`
**Test Reference**: `backend/user-service/src/repository/tests/user_repository_test.go`
