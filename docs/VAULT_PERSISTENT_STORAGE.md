# Vault Persistent Storage Setup

## Overview

Vault is now configured with **persistent file-based storage** instead of dev mode. This means:

✅ **Data persists across restarts** - Transit engine and encryption keys survive container restarts  
✅ **Automatic initialization** - Transit engine and pos-encryption-key created automatically on first start  
✅ **Production-ready configuration** - Uses file storage backend suitable for development and staging  

## Architecture

### Previous (Dev Mode)
```
vault:
  command: server -dev  ❌ In-memory storage
  # No volumes          ❌ Data lost on restart
```

**Problems**:
- All data stored in RAM
- Transit engine lost on restart
- Encryption keys lost on restart
- Manual setup required every time

### Current (Persistent Mode)
```
vault:
  entrypoint: vault-entrypoint.sh  ✅ Custom initialization
  volumes:
    - vault_data:/vault/data        ✅ Persistent storage
    - vault-config.hcl              ✅ Production config
    - vault-init.sh                 ✅ Auto-setup script
```

**Benefits**:
- Data stored on disk (Docker volume)
- Survives container restarts
- Transit engine auto-enabled
- Encryption key auto-created
- Zero manual intervention

## Files Structure

```
point-of-sale-system/
├── config/
│   └── vault-config.hcl           # Vault server configuration
├── scripts/
│   ├── vault-entrypoint.sh        # Container entrypoint
│   └── vault-init.sh              # Auto-initialization logic
└── docker-compose.yml             # Updated Vault service config
```

### 1. `config/vault-config.hcl`
```hcl
storage "file" {
  path = "/vault/data"  # Persistent storage location
}

listener "tcp" {
  address = "0.0.0.0:8200"
  tls_disable = true    # For dev/staging (enable TLS in production)
}
```

### 2. `scripts/vault-init.sh`
Auto-initialization script that:
- ✅ Waits for Vault to be ready
- ✅ Initializes Vault (if first run)
- ✅ Unseals Vault automatically
- ✅ Enables transit secrets engine
- ✅ Creates `pos-encryption-key`
- ✅ Displays configuration summary

### 3. `scripts/vault-entrypoint.sh`
Container startup script that:
- Starts Vault server with config file
- Runs initialization script
- Keeps container running

## Usage

### First Time Setup

1. **Start Vault**:
   ```bash
   docker-compose up -d vault
   ```

2. **Check logs** (initialization happens automatically):
   ```bash
   docker logs pos-vault
   ```

   Expected output:
   ```
   =====================================
   Vault Initialization Script
   =====================================
   ✓ Vault is ready
   Initializing Vault...
   Unsealing Vault...
   ✓ Vault initialized and unsealed
   Enabling transit secrets engine...
   ✓ Transit engine enabled
   Creating pos-encryption-key...
   ✓ Encryption key created
   =====================================
   Vault Configuration Complete
   =====================================
   name                      pos-encryption-key
   type                      aes256-gcm96
   latest_version            1
   ```

3. **Verify setup**:
   ```bash
   docker exec -e VAULT_TOKEN=dev-root-token pos-vault vault read transit/keys/pos-encryption-key
   ```

### Subsequent Starts

After the first initialization, Vault will:
- ✅ Use existing data from `vault_data` volume
- ✅ Auto-unseal using stored keys
- ✅ Verify transit engine exists
- ✅ Verify encryption key exists
- ✅ Skip initialization if already complete

```bash
# Restart Vault - data persists!
docker-compose restart vault

# Check logs - should show existing config detected
docker logs pos-vault
```

Expected output:
```
✓ Vault already initialized
✓ Transit engine already enabled
✓ Encryption key already exists
```

## Root Token Location

The Vault root token is stored in:
```
Docker Volume: vault_data
File: /vault/data/init-keys.txt
```

To retrieve it:
```bash
docker exec pos-vault cat /vault/data/init-keys.txt
```

## Vault Status

Check Vault health:
```bash
docker exec pos-vault vault status
```

Expected output:
```
Key             Value
---             -----
Seal Type       shamir
Initialized     true
Sealed          false      ✅ Unsealed and ready
Total Shares    1
Threshold       1
Version         1.15.x
Storage Type    file       ✅ Persistent storage
```

## Testing Encryption

Test that encryption persists across restarts:

```bash
# 1. Encrypt some data
docker exec -e VAULT_TOKEN=dev-root-token pos-vault vault write -field=ciphertext \
  transit/encrypt/pos-encryption-key plaintext=$(echo -n "test data" | base64)

# Output: vault:v1:abc123def456...

# 2. Restart Vault
docker-compose restart vault

# 3. Decrypt the same data (should work!)
docker exec -e VAULT_TOKEN=dev-root-token pos-vault vault write -field=plaintext \
  transit/decrypt/pos-encryption-key ciphertext="vault:v1:abc123def456..." | base64 -d

# Output: test data ✅
```

## Troubleshooting

### Issue: Vault is sealed
```bash
# Check seal status
docker exec pos-vault vault status | grep Sealed

# If sealed, check logs for unseal key
docker logs pos-vault | grep "Unseal Key"

# Manual unseal
docker exec -e VAULT_TOKEN=dev-root-token pos-vault vault operator unseal <UNSEAL_KEY>
```

### Issue: Transit engine not found
```bash
# List all secret engines
docker exec -e VAULT_TOKEN=dev-root-token pos-vault vault secrets list

# If transit missing, enable it
docker exec -e VAULT_TOKEN=dev-root-token pos-vault vault secrets enable transit
```

### Issue: Encryption key missing
```bash
# List all transit keys
docker exec -e VAULT_TOKEN=dev-root-token pos-vault vault list transit/keys

# If pos-encryption-key missing, create it
docker exec -e VAULT_TOKEN=dev-root-token pos-vault vault write -f transit/keys/pos-encryption-key
```

### Complete Reset

To start fresh (⚠️ **destroys all encrypted data**):
```bash
# Stop Vault
docker-compose stop vault

# Delete volume
docker volume rm point-of-sale-system_vault_data

# Start Vault (will reinitialize)
docker-compose up -d vault
```

## Production Considerations

For production deployment, enhance security:

### 1. Enable TLS
```hcl
# config/vault-config.hcl
listener "tcp" {
  address = "0.0.0.0:8200"
  tls_disable = false
  tls_cert_file = "/vault/tls/vault.crt"
  tls_key_file = "/vault/tls/vault.key"
}
```

### 2. Use External Storage
Replace file storage with:
- **Consul** (recommended for HA)
- **AWS S3** + DynamoDB
- **Azure Storage**
- **Google Cloud Storage**

Example Consul backend:
```hcl
storage "consul" {
  address = "consul:8500"
  path = "vault/"
}
```

### 3. Enable Auto-Unseal
Use cloud KMS for automatic unsealing:
```hcl
seal "awskms" {
  region = "us-east-1"
  kms_key_id = "alias/vault-unseal-key"
}
```

### 4. Implement Key Rotation
```bash
# Rotate encryption key (creates new version)
docker exec -e VAULT_TOKEN=<token> pos-vault vault write -f transit/keys/pos-encryption-key/rotate

# Configure auto-rotation
docker exec -e VAULT_TOKEN=<token> pos-vault vault write transit/keys/pos-encryption-key/config \
  auto_rotate_period=2160h  # 90 days
```

### 5. Backup Strategy
```bash
# Backup Vault data
docker run --rm -v point-of-sale-system_vault_data:/data -v $(pwd):/backup \
  alpine tar czf /backup/vault-backup-$(date +%Y%m%d).tar.gz /data

# Restore Vault data
docker run --rm -v point-of-sale-system_vault_data:/data -v $(pwd):/backup \
  alpine tar xzf /backup/vault-backup-20260105.tar.gz -C /
```

## Migration from Dev Mode

If you have existing data encrypted with dev-mode Vault:

1. **Option A: Keep old key** (if you have the unseal key):
   - Not possible - dev mode doesn't use file storage

2. **Option B: Re-encrypt all data** (recommended):
   ```bash
   # 1. Stop all services
   docker-compose stop
   
   # 2. Start only Vault and database
   docker-compose up -d vault postgres
   
   # 3. Run data migration to re-encrypt
   docker run --rm --network pos-network \
     -e DATABASE_URL="postgresql://pos_user:pos_password@postgres-db:5432/pos_db?sslmode=disable" \
     -e VAULT_ADDR="http://vault:8200" \
     -e VAULT_TOKEN="dev-root-token" \
     -e VAULT_TRANSIT_KEY="pos-encryption-key" \
     pos-data-migration -type=all
   
   # 4. Start all services
   docker-compose up -d
   ```

## Summary

| Feature | Dev Mode ❌ | Persistent Mode ✅ |
|---------|-------------|-------------------|
| Storage | In-memory (RAM) | File-based (disk) |
| Survives restart | No | Yes |
| Auto-initialization | No | Yes |
| Production-ready | No | Yes (with TLS) |
| Manual setup | Every restart | Once only |
| Transit engine | Lost on restart | Persists |
| Encryption keys | Lost on restart | Persists |

**Next Steps**:
1. Test the new persistent setup
2. Verify encryption/decryption works after restart
3. Document backup procedures
4. Plan for production hardening (TLS, Consul, auto-unseal)
