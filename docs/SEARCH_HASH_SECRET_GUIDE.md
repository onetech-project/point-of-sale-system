# SEARCH_HASH_SECRET Configuration Guide

## Purpose

The `SEARCH_HASH_SECRET` is used to generate deterministic HMAC-SHA256 hashes for encrypted fields, enabling efficient O(1) database lookups without exposing plaintext data.

## Critical Requirements

### ⚠️ MUST BE IDENTICAL ACROSS ALL SERVICES

The same secret value must be configured in:
- `backend/user-service/.env`
- `backend/auth-service/.env`
- `backend/tenant-service/.env`
- `scripts/data-migration/.env`

**Why?** All services must generate identical hashes for the same plaintext value to enable cross-service lookups.

## Current Value (Development)

```env
SEARCH_HASH_SECRET=96278bfede09090c7d97b12e6b6de52c001eef5f1206bfcbd790e638ce25a0c9
```

## Generation (Production)

```bash
# Generate new secure random secret
openssl rand -hex 32

# Example output (DO NOT USE THIS EXAMPLE):
# f4a8b2c9e1d3a7f6b5c8d2e9a4b7c1f3e6d9a2b5c8e1d4a7b3c6f9e2d5a8b1c4
```

## Adding to Services

### Method 1: Manual Addition
```bash
# Add to each service's .env file
echo "SEARCH_HASH_SECRET=YOUR_SECRET_HERE" >> backend/user-service/.env
echo "SEARCH_HASH_SECRET=YOUR_SECRET_HERE" >> backend/auth-service/.env
echo "SEARCH_HASH_SECRET=YOUR_SECRET_HERE" >> backend/tenant-service/.env
echo "SEARCH_HASH_SECRET=YOUR_SECRET_HERE" >> scripts/data-migration/.env
```

### Method 2: Scripted Deployment
```bash
#!/bin/bash
# deploy-search-secret.sh

SEARCH_SECRET=$(openssl rand -hex 32)

echo "Generated SEARCH_HASH_SECRET: ${SEARCH_SECRET}"

# Add to all services
echo "" >> backend/user-service/.env
echo "SEARCH_HASH_SECRET=${SEARCH_SECRET}" >> backend/user-service/.env

echo "" >> backend/auth-service/.env
echo "SEARCH_HASH_SECRET=${SEARCH_SECRET}" >> backend/auth-service/.env

echo "" >> backend/tenant-service/.env
echo "SEARCH_HASH_SECRET=${SEARCH_SECRET}" >> backend/tenant-service/.env

echo "" >> scripts/data-migration/.env
echo "SEARCH_HASH_SECRET=${SEARCH_SECRET}" >> scripts/data-migration/.env

echo "✅ SEARCH_HASH_SECRET added to all services"
```

## Verification

```bash
# Verify all services have the same value
echo "=== User Service ==="
grep SEARCH_HASH_SECRET backend/user-service/.env

echo "=== Auth Service ==="
grep SEARCH_HASH_SECRET backend/auth-service/.env

echo "=== Tenant Service ==="
grep SEARCH_HASH_SECRET backend/tenant-service/.env

echo "=== Migration Tool ==="
grep SEARCH_HASH_SECRET scripts/data-migration/.env
```

## Security Considerations

### Protection
- ✅ Store in `.env` files (git-ignored)
- ✅ Never commit to version control
- ✅ Use different values for dev/staging/production
- ✅ Restrict file permissions: `chmod 600 .env`
- ✅ Use secret management in production (AWS Secrets Manager, etc.)

### Rotation Procedure

**Warning**: Rotating this secret requires re-hashing all existing data.

```bash
# 1. Generate new secret
NEW_SECRET=$(openssl rand -hex 32)

# 2. Update all service .env files with NEW_SECRET

# 3. Rebuild and restart services
docker-compose build user-service auth-service tenant-service
docker-compose restart user-service auth-service tenant-service

# 4. Re-run hash population migration
cd scripts/data-migration
# Update .env with NEW_SECRET
docker run --rm --network pos-network --env-file .env \
  pos-data-migration -type=search-hashes

# 5. Verify hash population
docker exec -it postgres-db psql -U pos_user -d pos_db -c "
  SELECT COUNT(*) as with_hash FROM users WHERE email_hash IS NOT NULL;
"
```

## Troubleshooting

### Symptom: Login returns "user not found" but user exists
**Cause**: SEARCH_HASH_SECRET mismatch between auth-service and user-service
**Fix**: Verify all services have identical SEARCH_HASH_SECRET values

### Symptom: Invitation token not found
**Cause**: Token hash was generated with different SEARCH_HASH_SECRET
**Fix**: Re-run hash population migration with correct secret

### Symptom: New users can't login
**Cause**: User was created with different SEARCH_HASH_SECRET than auth-service uses
**Fix**: Ensure all services restarted after SEARCH_HASH_SECRET update

### Symptom: Migration script fails with "environment variable not set"
**Cause**: SEARCH_HASH_SECRET missing from data-migration/.env
**Fix**: Add SEARCH_HASH_SECRET to scripts/data-migration/.env

## Testing

```bash
# Test that hash generation is consistent
# Run this in Go code:

package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "os"
)

func HashForSearch(value string) string {
    secret := os.Getenv("SEARCH_HASH_SECRET")
    h := hmac.New(sha256.New, []byte(secret))
    h.Write([]byte(value))
    return hex.EncodeToString(h.Sum(nil))
}

func main() {
    os.Setenv("SEARCH_HASH_SECRET", "96278bfede09090c7d97b12e6b6de52c001eef5f1206bfcbd790e638ce25a0c9")
    
    email := "test@example.com"
    hash1 := HashForSearch(email)
    hash2 := HashForSearch(email)
    
    fmt.Printf("Hash 1: %s\n", hash1)
    fmt.Printf("Hash 2: %s\n", hash2)
    fmt.Printf("Match: %v\n", hash1 == hash2)  // Should be true
}
```

Expected output:
```
Hash 1: adc8c43cabb7b8b0bb8266fdc2019f64a321c528aed91d24e687adf15907d897
Hash 2: adc8c43cabb7b8b0bb8266fdc2019f64a321c528aed91d24e687adf15907d897
Match: true
```

## Related Documentation

- [Encryption Performance Fix](./ENCRYPTION_PERFORMANCE_FIX.md)
- [Encryption Verification](./ENCRYPTION_VERIFICATION_COMPLETE.md)
- [Environment Variables](./ENVIRONMENT.md)

## Support

For issues or questions:
1. Check this documentation
2. Verify SEARCH_HASH_SECRET is identical in all .env files
3. Check service logs: `docker logs <service-name>`
4. Review [ENCRYPTION_PERFORMANCE_FIX.md](./ENCRYPTION_PERFORMANCE_FIX.md)
