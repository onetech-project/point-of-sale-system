# Development Environment Setup Guide

Panduan lengkap untuk setup development environment POS system di device baru.

## Quick Start (Recommended)

Untuk setup pertama kali di device baru, cukup jalankan:

```bash
./scripts/quick-setup.sh
```

Script ini akan memberikan menu interaktif dengan pilihan:

1. **Full Setup** - Setup lengkap (recommended untuk first time)
2. **Setup tanpa Docker** - Jika Anda menjalankan Docker di local machine
3. **Setup tanpa Vault** - Untuk development lightweight tanpa encryption
4. **Reset** - Membersihkan semua untuk fresh start

## Detailed Setup Scripts

### 1. setup-dev-environment.sh (Main Setup Script)

Script utama yang mengotomatisasi seluruh setup:

```bash
# Full setup (semua infrastruktur)
./scripts/setup-dev-environment.sh

# Skip Docker infrastructure (Anda manage Docker sendiri)
./scripts/setup-dev-environment.sh --skip-docker

# Skip Vault (tanpa encryption vault)
./scripts/setup-dev-environment.sh --skip-vault

# Skip keduanya
./scripts/setup-dev-environment.sh --skip-docker --skip-vault
```

**Apa yang dilakukan script ini:**

- ✓ Memeriksa prerequisites (Docker, docker-compose, openssl)
- ✓ Generate Vault TLS certificates
- ✓ Membuat .env files untuk semua services
- ✓ Menjalankan Docker containers (postgres, redis, vault, kafka, dll)
- ✓ Initialize Vault dengan encryption keys
- ✓ Menjalankan database migrations
- ✓ Initialize MinIO buckets

**Output:**

```
✓ Prerequisites verified
✓ Vault TLS certificates generated at vault/tls/
✓ Environment files created
✓ Docker infrastructure started
✓ Vault initialized with root token
✓ Database migrations completed
✓ MinIO buckets initialized
```

### 2. quick-setup.sh (Interactive Setup)

Wrapper yang lebih user-friendly dengan menu interaktif:

```bash
./scripts/quick-setup.sh
```

Gunakan ini jika:

- Pertama kali setup (ingin menu interaktif)
- Tidak yakin dengan setup options
- Ingin reset environment dan setup dari awal

### 3. dev-helper.sh (Development Utilities)

Helper command untuk management development environment:

```bash
# Lihat status environment
./scripts/dev-helper.sh status

# Lihat Vault credentials
./scripts/dev-helper.sh vault-token

# Set Vault environment variables untuk CLI
source <(./scripts/dev-helper.sh vault-login)

# Lihat health semua services
./scripts/dev-helper.sh check-services

# Lihat logs untuk service tertentu
./scripts/dev-helper.sh logs postgres
./scripts/dev-helper.sh logs vault
./scripts/dev-helper.sh logs redis

# Restart service
./scripts/dev-helper.sh restart vault

# Rebuild containers
./scripts/dev-helper.sh rebuild

# Cleanup (stop containers, remove volumes, tapi keep config)
./scripts/dev-helper.sh clean

# Verify encryption setup
./scripts/dev-helper.sh verify-encryption

# Verify environment configuration
./scripts/dev-helper.sh verify-env

# Lihat semua available commands
./scripts/dev-helper.sh help
```

## Step-by-Step Setup untuk Device Baru

### Prerequisite

Pastikan sudah install di device:

- Docker 20.10+
- docker-compose 1.29+
- OpenSSL 1.1.1+
- Git
- Node.js 18+ (untuk frontend development)
- Go 1.20+ (untuk backend development)
- Make

### Setup Procedure

```bash
# 1. Clone repository
git clone <repo-url>
cd point-of-sale-system

# 2. Run setup (pilih salah satu)
# Option A: Interactive setup
./scripts/quick-setup.sh

# Option B: Full setup langsung
./scripts/setup-dev-environment.sh

# 3. Tunggu setup selesai
# Durasi: ~2-5 menit tergantung internet speed dan machine

# 4. Verify setup berhasil
./scripts/dev-helper.sh status

# 5. Start development
./scripts/start-all.sh
```

## Environment Files

Setup script secara otomatis membuat .env files dari .env.example untuk:

```
├── .env                                  # Root configuration
├── vault/
│   └── .env                              # Vault configuration
├── api-gateway/
│   └── .env                              # API Gateway config
└── backend/
    ├── auth-service/.env
    ├── user-service/.env
    ├── tenant-service/.env
    ├── notification-service/.env
    ├── product-service/.env
    ├── order-service/.env
    ├── audit-service/.env
    ├── analytics-service/.env
    └── billing-service/.env
```

### Customization

Setelah setup, Anda bisa customize .env files sesuai kebutuhan:

- Database credentials
- Service ports
- API endpoints
- External service keys (Midtrans, Email, dll)

Jangan commit .env files - gunakan .env.example sebagai template!

## Vault TLS Certificates

TLS certificates untuk Vault di-generate secara otomatis di `vault/tls/`:

```
vault/tls/
├── ca.key               # CA private key
├── ca.crt               # CA certificate
├── vault.key            # Vault server private key
└── vault.crt            # Vault server certificate
```

**Untuk regenerate certificates:**

```bash
# Remove existing certificates
rm -rf vault/tls/

# Re-run setup
./scripts/setup-dev-environment.sh
```

## Vault Initialization

Saat Vault pertama kali dijalankan, script akan:

1. Generate 3 unseal keys
2. Generate root token
3. Enable transit secrets engine
4. Create encryption keys untuk development, staging, production

**Vault credentials disimpan di:** `vault/data/init-keys.txt`

```
Unseal Key 1: xxx
Unseal Key 2: yyy
Unseal Key 3: zzz
Initial Root Token: root_token_xxx
```

**PENTING:** Keep file ini aman! Diperlukan untuk unseal Vault setelah restart.

### Vault CLI Access

Untuk akses Vault via CLI (development only):

```bash
# Option 1: Set environment variables
source <(./scripts/dev-helper.sh vault-login)

# Option 2: Manual set
export VAULT_ADDR="https://localhost:8200"
export VAULT_TOKEN="<root-token-dari-init-keys.txt>"
export VAULT_SKIP_VERIFY=true

# Verify connection
vault status

# List encryption keys
vault secrets list
vault read transit/keys/pos-development-key

# Encrypt data
vault write transit/encrypt/pos-development-key plaintext=@/tmp/data.txt
```

## Docker Services

Setup script menjalankan services berikut via docker-compose:

```
postgres:5432           - Database utama
redis:6379              - Cache & session storage
vault:8200              - Encryption vault
kafka:9092              - Event streaming
zookeeper:2181          - Kafka coordinator
minio:9000              - Object storage (S3-compatible)
```

### Manage Services

```bash
# Status semua services
docker-compose ps

# Start services
docker-compose up -d

# Stop services
docker-compose down

# Remove volumes (cleanup data)
docker-compose down -v

# Rebuild containers
docker-compose build --no-cache

# View logs
docker-compose logs -f postgres
docker-compose logs -f vault

# Access container
docker-compose exec postgres bash
```

## Common Issues & Troubleshooting

### Issue: Docker daemon not running

```bash
# Solution: Start Docker
# macOS
open /Applications/Docker.app

# Linux
sudo systemctl start docker

# Windows
# Open Docker Desktop application
```

### Issue: Port already in use

```bash
# Find what's using port
lsof -i :8200  # Vault
lsof -i :5432  # PostgreSQL
lsof -i :6379  # Redis

# Kill process
kill -9 <PID>

# Or change port di .env files
```

### Issue: Vault not initialized

```bash
# Check Vault status
./scripts/dev-helper.sh check-services

# View logs
./scripts/dev-helper.sh logs vault

# Manual re-initialize
docker-compose down -v
./scripts/setup-dev-environment.sh
```

### Issue: Database migration failed

```bash
# Check database connection
docker-compose exec postgres psql -U pos_user -d pos_db

# View migration logs
./scripts/dev-helper.sh logs postgres

# Re-run migrations
./scripts/run-migrations.sh
```

### Issue: Certificate verification failed

```bash
# Regenerate TLS certificates
rm -rf vault/tls/
./scripts/setup-dev-environment.sh

# Or skip verification (dev only)
export VAULT_SKIP_VERIFY=true
```

## Starting Development

Setelah setup selesai:

### Start all services

```bash
./scripts/start-all.sh
```

### Start specific services

```bash
./scripts/start-all.sh auth           # Auth service only
./scripts/start-all.sh frontend       # Frontend only
./scripts/start-all.sh auth user      # Multiple services
./scripts/start-all.sh all with-vault # All with Vault
```

### Stop all services

```bash
./scripts/stop-all.sh
```

## Verification

### Verify environment configuration

```bash
./scripts/verify-env.sh
```

### Verify encryption setup

```bash
./scripts/verify-deterministic-encryption.sh
```

### Check all services running

```bash
./scripts/dev-helper.sh check-services
```

## Development Workflow

```bash
# 1. Setup environment (first time only)
./scripts/quick-setup.sh

# 2. Start all services
./scripts/start-all.sh

# 3. Development
# - Edit code
# - Services auto-reload
# - View logs: ./scripts/dev-helper.sh logs <service>

# 4. View status
./scripts/dev-helper.sh status

# 5. Stop services
./scripts/stop-all.sh

# 6. Reset environment (jika perlu fresh start)
./scripts/quick-setup.sh
# Choose option 5: Reset everything
```

## Additional Resources

- [API Documentation](../docs/API.md)
- [Backend Conventions](../docs/BACKEND_CONVENTIONS.md)
- [Frontend Conventions](../docs/FRONTEND_CONVENTIONS.md)
- [Environment Configuration](../docs/ENVIRONMENT.md)
- [Vault TLS Management](../vault/VAULT_TOKEN_MANAGEMENT.md)

## Support

Jika mengalami issue:

1. Check logs: `./scripts/dev-helper.sh logs <service>`
2. Verify setup: `./scripts/dev-helper.sh verify-env`
3. Check status: `./scripts/dev-helper.sh status`
4. Reset & retry: `./scripts/quick-setup.sh` → option 5

---

**Last Updated:** May 2026
**Version:** 1.0
