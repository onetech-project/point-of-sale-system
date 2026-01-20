# Bug Fixes: NULL Handling and Midtrans Encryption

## Date: January 5, 2026

## Issues Fixed

### 1. SQL NULL Handling Error in Order Service

**Error:**
```
sql: Scan error on column index 18, name "ip_address": converting NULL to string is unsupported
```

**Root Cause:**
- In `backend/order-service/src/repository/order_repository.go`, nullable encrypted fields (ip_address, user_agent, customer name, phone, email) were declared as `string` type
- When scanning NULL values from database, Go's SQL package cannot convert NULL to string

**Solution:**
- Changed variable declarations from `string` to `sql.NullString` for all nullable encrypted fields:
  ```go
  // Before:
  var encryptedName, encryptedPhone, encryptedEmail, encryptedIP, encryptedUA string
  
  // After:
  var encryptedName, encryptedPhone sql.NullString
  var encryptedEmail, encryptedIP, encryptedUA sql.NullString
  ```

- Updated NULL checks from `!= ""` to `.Valid`:
  ```go
  // Before:
  if encryptedIP != "" {
      order.CustomerIP, err = encryptionClient.Decrypt(ctx, encryptedIP)
  }
  
  // After:
  if encryptedIP.Valid {
      order.CustomerIP, err = encryptionClient.Decrypt(ctx, encryptedIP.String)
  }
  ```

**Files Modified:**
- `backend/order-service/src/repository/order_repository.go` (lines ~296-340)

---

### 2. Midtrans Keys Not Encrypted When Created

**Error:**
```
midtrans_server_key and midtrans_client_key is not decrypted after load from table
```

**Root Cause:**
- In `backend/tenant-service/src/repository/tenant_config_repository.go`:
  - ✅ `Update()` method: Encrypts Midtrans keys before saving
  - ❌ `Create()` method: Did NOT encrypt Midtrans keys, and didn't include them in INSERT query
  - ✅ `GetByTenantID()` method: Decrypts Midtrans keys after loading

- When new tenant configs were created, Midtrans keys were stored **unencrypted** in the database
- When retrieved, the service attempted to decrypt them, causing failures

**Solution:**
- Added encryption logic to `Create()` method to match `Update()` method:
  ```go
  func (r *TenantConfigRepository) Create(ctx context.Context, config *TenantConfig) error {
      // Encrypt Midtrans keys
      var encryptedServerKey, encryptedClientKey string
      var err error

      if config.MidtransServerKey != "" {
          encryptedServerKey, err = r.encryptor.Encrypt(ctx, config.MidtransServerKey)
          if err != nil {
              return fmt.Errorf("failed to encrypt midtrans_server_key: %w", err)
          }
      }

      if config.MidtransClientKey != "" {
          encryptedClientKey, err = r.encryptor.Encrypt(ctx, config.MidtransClientKey)
          if err != nil {
              return fmt.Errorf("failed to encrypt midtrans_client_key: %w", err)
          }
      }
      
      // ... rest of the method
  }
  ```

- Updated INSERT query to include Midtrans fields:
  ```sql
  INSERT INTO tenant_configs (
      tenant_id,
      enabled_delivery_types,
      service_area_data,
      delivery_fee_config,
      enable_delivery_fee_calculation,
      midtrans_server_key,        -- Added
      midtrans_client_key,        -- Added
      midtrans_merchant_id,       -- Added
      midtrans_environment        -- Added
  ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
  ```

**Files Modified:**
- `backend/tenant-service/src/repository/tenant_config_repository.go` (lines ~131-165)

---

## Encryption Flow (Now Consistent)

### Create Flow:
1. User creates/updates Midtrans config via API
2. Service receives plaintext keys
3. **Repository encrypts keys** before INSERT/UPDATE
4. Database stores encrypted keys

### Read Flow:
1. Service queries database for config
2. Database returns encrypted keys
3. **Repository decrypts keys** after SELECT
4. Service returns plaintext keys to API caller

---

## Deployment

Both services were rebuilt and redeployed:

```bash
# Order Service (NULL handling fix)
docker-compose build order-service
docker-compose stop order-service && docker-compose rm -f order-service
docker-compose up -d order-service

# Tenant Service (Midtrans encryption fix)
docker-compose build tenant-service
docker-compose stop tenant-service && docker-compose rm -f tenant-service
docker-compose up -d tenant-service
```

---

## Verification

### NULL Handling Verification:
- Order listing now works with NULL ip_address and user_agent values
- Guest orders (no user_agent/ip_address) can be listed successfully
- Encrypted fields are properly decrypted when present, skipped when NULL

### Midtrans Encryption Verification:
- New tenant configs encrypt Midtrans keys on creation
- Existing configs continue to work (Update already encrypted keys)
- Keys are properly decrypted when retrieved via API
- Payment flow works correctly with decrypted keys

---

## Impact

- ✅ Order listing no longer crashes on guest orders
- ✅ Midtrans payment gateway integration works correctly
- ✅ All sensitive data (keys, customer info) remains encrypted at rest
- ✅ Consistent encryption behavior across Create and Update operations

---

## Related Documentation

- [Vault Persistent Storage](./VAULT_PERSISTENT_STORAGE.md)
- [Vault Quick Start](./VAULT_QUICK_START.md)
- [Encryption Repository Pattern](./ENCRYPTION_REPOSITORY_PATTERN.md)
