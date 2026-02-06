# Vault Persistent Storage - Quick Start

## Summary

Vault is now configured with **automatic persistent storage** instead of dev mode.

### What Changed?

| Before (Dev Mode) | After (Persistent Mode) |
|-------------------|------------------------|
| ❌ In-memory storage | ✅ File-based storage (Docker volume) |
| ❌ Data lost on restart | ✅ Data persists across restarts |
| ❌ Manual setup every time | ✅ **Fully automatic** initialization |
| ❌ Not production-ready | ✅ Production-ready (with TLS) |

### Benefits

1. **Zero Manual Configuration**: Transit engine and encryption key created automatically on first start
2. **Data Persistence**: Encrypted data remains accessible after container restarts
3. **Production Ready**: Uses file storage backend suitable for development and staging

---

## Quick Start

### 1. Start Vault
```bash
docker-compose up -d vault
```

**First time only**: Vault will automatically:
- Initialize itself
- Create unseal keys
- Enable transit secrets engine
- Create `pos-encryption-key`
- Store root token in `/vault/data/init-keys.txt`

### 2. Check Initialization (Optional)
```bash
docker logs pos-vault
```

Expected output:
```
=====================================
Vault Configuration Complete
=====================================
✓ Vault initialized and unsealed
✓ Transit engine enabled
✓ Encryption key created

Transit engine: http://vault:8200/v1/transit/
Encryption key: pos-encryption-key
Root token stored in: /vault/data/init-keys.txt
=====================================
```

### 3. Get Root Token (If Needed)
```bash
docker exec pos-vault cat /vault/data/init-keys.txt | grep 'Initial Root Token'
```

### 4. Test Encryption
```bash
# Get root token
TOKEN=$(docker exec pos-vault cat /vault/data/init-keys.txt | grep 'Initial Root Token' | awk '{print $NF}')

# Encrypt test data
docker exec pos-vault sh -c "VAULT_ADDR=http://0.0.0.0:8200 VAULT_TOKEN=$TOKEN \
  vault write -field=ciphertext transit/encrypt/pos-encryption-key \
  plaintext=\$(echo -n 'hello world' | base64)"

# Output: vault:v1:abc123def456...
```

---

## How It Works

### Architecture

```
┌─────────────────────────────────────────┐
│  docker-compose.yml                     │
├─────────────────────────────────────────┤
│  vault:                                 │
│    entrypoint: vault-entrypoint.sh ─────┼─► Starts Vault + runs init script
│    volumes:                             │
│      - vault_data:/vault/data ──────────┼─► Persistent storage (survives restarts)
│      - vault-config.hcl ────────────────┼─► Server configuration
│      - vault-init.sh ───────────────────┼─► Auto-initialization logic
└─────────────────────────────────────────┘
```

### Initialization Flow

**First Start** (fresh volume):
```
1. vault-entrypoint.sh starts Vault server
2. vault-init.sh runs:
   ✓ Detects Vault is not initialized
   ✓ Runs: vault operator init
   ✓ Saves keys to /vault/data/init-keys.txt
   ✓ Unseals Vault
   ✓ Enables transit engine
   ✓ Creates pos-encryption-key
```

**Subsequent Starts** (existing volume):
```
1. vault-entrypoint.sh starts Vault server
2. vault-init.sh runs:
   ✓ Detects Vault is already initialized
   ✓ Reads keys from /vault/data/init-keys.txt
   ✓ Unseals Vault (if sealed)
   ✓ Verifies transit engine exists (skips if exists)
   ✓ Verifies encryption key exists (skips if exists)
```

---

## Files

### `/config/vault-config.hcl`
Server configuration with file storage backend:
```hcl
storage "file" {
  path = "/vault/data"  # Persistent storage location
}

listener "tcp" {
  address = "0.0.0.0:8200"
  tls_disable = true    # Enable TLS for production!
}
```

### `/scripts/vault-init.sh`
Auto-initialization script that:
- Initializes Vault (first run only)
- Unseals Vault automatically
- Enables transit secrets engine
- Creates pos-encryption-key
- Displays configuration summary

### `/scripts/vault-entrypoint.sh`
Container startup that:
- Starts Vault server in background
- Runs initialization script
- Keeps container running

---

## Verification

### Check Vault Status
```bash
docker exec pos-vault vault status
```

Expected output:
```
Key             Value
---             -----
Initialized     true   ✅
Sealed          false  ✅
Storage Type    file   ✅
```

### List Secret Engines
```bash
TOKEN=$(docker exec pos-vault cat /vault/data/init-keys.txt | grep 'Initial Root Token' | awk '{print $NF}')
docker exec -e VAULT_TOKEN=$TOKEN pos-vault vault secrets list
```

Expected output:
```
Path          Type         Description
----          ----         -----------
cubbyhole/    cubbyhole    per-token private secret storage
identity/     identity     identity store
sys/          system       system endpoints used for control
transit/      transit      ✅ Transit secrets engine enabled
```

### Check Encryption Key
```bash
TOKEN=$(docker exec pos-vault cat /vault/data/init-keys.txt | grep 'Initial Root Token' | awk '{print $NF}')
docker exec -e VAULT_TOKEN=$TOKEN pos-vault vault read transit/keys/pos-encryption-key
```

Expected output:
```
Key                       Value
---                       -----
name                      pos-encryption-key  ✅
type                      aes256-gcm96       ✅
latest_version            1
supports_encryption       true
supports_decryption       true
```

---

## Testing Persistence

Run this test to verify data survives restart:

```bash
# 1. Get root token
TOKEN=$(docker exec pos-vault cat /vault/data/init-keys.txt | grep 'Initial Root Token' | awk '{print $NF}')

# 2. Encrypt data
CIPHER=$(docker exec -e VAULT_TOKEN=$TOKEN pos-vault vault write -field=ciphertext \
  transit/encrypt/pos-encryption-key plaintext=$(echo -n "test-persistence" | base64))
echo "Ciphertext: $CIPHER"

# 3. Restart Vault
docker-compose restart vault
sleep 10

# 4. Decrypt after restart (should work!)
docker exec -e VAULT_TOKEN=$TOKEN pos-vault vault write -field=plaintext \
  transit/decrypt/pos-encryption-key ciphertext="$CIPHER" | base64 -d

# Output: test-persistence ✅
```

---

## Troubleshooting

### Vault is Sealed After Restart
```bash
# Get unseal key
UNSEAL_KEY=$(docker exec pos-vault cat /vault/data/init-keys.txt | grep 'Unseal Key 1' | awk '{print $NF}')

# Unseal manually
docker exec pos-vault vault operator unseal $UNSEAL_KEY
```

### Transit Engine Not Found
```bash
# Enable manually
TOKEN=$(docker exec pos-vault cat /vault/data/init-keys.txt | grep 'Initial Root Token' | awk '{print $NF}')
docker exec -e VAULT_TOKEN=$TOKEN pos-vault vault secrets enable transit
```

### Encryption Key Missing
```bash
# Create manually
TOKEN=$(docker exec pos-vault cat /vault/data/init-keys.txt | grep 'Initial Root Token' | awk '{print $NF}')
docker exec -e VAULT_TOKEN=$TOKEN pos-vault vault write -f transit/keys/pos-encryption-key
```

### Complete Reset (⚠️ Destroys All Data)
```bash
docker-compose stop vault
docker rm -f pos-vault
docker volume rm point-of-sale-system_vault_data
docker-compose up -d vault
```

---

## Production Deployment

### 1. Enable TLS
Update `config/vault-config.hcl`:
```hcl
listener "tcp" {
  address = "0.0.0.0:8200"
  tls_disable = false
  tls_cert_file = "/vault/tls/vault.crt"
  tls_key_file = "/vault/tls/vault.key"
}
```

### 2. Use Consul Backend (High Availability)
```hcl
storage "consul" {
  address = "consul:8500"
  path = "vault/"
}
```

### 3. Enable Auto-Unseal (AWS KMS)
```hcl
seal "awskms" {
  region = "us-east-1"
  kms_key_id = "alias/vault-unseal-key"
}
```

### 4. Configure Key Rotation
```bash
# Rotate encryption key every 90 days
vault write transit/keys/pos-encryption-key/config auto_rotate_period=2160h
```

### 5. Backup Strategy
```bash
# Backup vault data
docker run --rm \
  -v point-of-sale-system_vault_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/vault-backup-$(date +%Y%m%d).tar.gz /data

# Restore vault data
docker run --rm \
  -v point-of-sale-system_vault_data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/vault-backup-YYYYMMDD.tar.gz -C /
```

---

## FAQ

**Q: Do I need to run any manual commands after `docker-compose up`?**  
A: No! Everything is automatic. Just start Vault and it's ready to use.

**Q: Where is the root token stored?**  
A: In the Docker volume at `/vault/data/init-keys.txt`. Access it with:
```bash
docker exec pos-vault cat /vault/data/init-keys.txt
```

**Q: What happens if I delete the vault_data volume?**  
A: Vault will reinitialize automatically with NEW keys. Old encrypted data becomes unrecoverable.

**Q: Can I use the same Vault instance for multiple environments?**  
A: For development: yes (use namespaces). For production: use separate Vault instances.

**Q: How do I upgrade Vault version?**  
A: Update image in `docker-compose.yml`, then:
```bash
docker-compose pull vault
docker-compose up -d vault
```
Data persists through upgrades.

---

## Next Steps

1. ✅ Vault is now persistent and automatic
2. 📋 Re-encrypt existing data if migrating from dev mode
3. 🔐 Enable TLS for production
4. 🏗️ Consider Consul backend for HA
5. 🔄 Set up key rotation policy
6. 💾 Implement backup strategy

For detailed documentation, see: [docs/VAULT_PERSISTENT_STORAGE.md](./VAULT_PERSISTENT_STORAGE.md)
