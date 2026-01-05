# Quickstart Guide: UU PDP Compliance Implementation

**Feature**: 006-uu-pdp-compliance  
**Date**: 2026-01-02  
**Audience**: Developers implementing Indonesian data protection compliance

## Prerequisites

- Go 1.24+ installed
- PostgreSQL 15+ running locally or via Docker
- Redis 7+ (for distributed locking in retention service)
- Kafka (existing docker-compose setup)
- HashiCorp Vault (dev mode for local, production Vault for staging/prod)
- Node.js 18+ and npm (for frontend)

## Quick Start (30 minutes)

### 1. Database Setup (5 minutes)

```bash
# Navigate to migrations directory
cd backend/migrations

# Run new migrations for UU PDP compliance
migrate -path . -database "postgresql://postgres:postgres@localhost:5432/pos?sslmode=disable" up

# Migrations will create:
# - consent_purposes table
# - privacy_policies table
# - consent_records table
# - audit_events (partitioned) table
# - retention_policies table
# - Add *_encrypted columns to existing tables
```

**Verify migration**:
```sql
-- Check tables exist
\dt consent_*
\dt privacy_*
\dt audit_events*
\dt retention_policies

-- Check encrypted columns (encryption happens at application layer, no schema changes)
\d users
-- Note: email, first_name, last_name columns store encrypted values (vault:v1:...)

-- Check audit_events partitions
\dt audit_events_*
```

### 2. Seed Initial Data (3 minutes)

```bash
# Run seed script to populate consent purposes and initial privacy policy
cd backend/scripts
./seed-compliance-data.sh

# Or manually via SQL:
psql -d pos << EOF
-- Insert consent purposes
INSERT INTO consent_purposes (purpose_code, purpose_name_en, purpose_name_id, description_en, description_id, is_required, display_order) VALUES
('operational', 'Operational Data Processing', 'Pemrosesan Data Operasional', 
 'We process your business data to manage orders, inventory, and team members.', 
 'Kami memproses data bisnis Anda untuk mengelola pesanan, inventaris, dan tim.',
 TRUE, 1),
('analytics', 'Service Analysis and Improvement', 'Analisis dan Peningkatan Layanan',
 'We analyze system usage to improve features and performance.',
 'Kami menganalisis penggunaan sistem untuk meningkatkan fitur dan kinerja.',
 TRUE, 2),
('advertising', 'Advertising and Promotion', 'Promosi dan Iklan',
 'We may send promotional offers via email.',
 'Kami dapat mengirimkan penawaran promosi melalui email.',
 FALSE, 3),
('third_party_midtrans', 'Midtrans Payment Integration', 'Integrasi Pembayaran Midtrans',
 'We share payment data with Midtrans to process transactions.',
 'Kami berbagi data pembayaran dengan Midtrans untuk memproses transaksi.',
 TRUE, 4);

-- Insert initial privacy policy
INSERT INTO privacy_policies (version, policy_text_id, effective_date, is_current) VALUES
('1.0.0', 
 '# Kebijakan Privasi\n\n## 1. Data yang Dikumpulkan\nKami mengumpulkan email, nama, nomor telepon...',
 '2026-01-01T00:00:00Z',
 TRUE);
EOF
```

### 3. HashiCorp Vault Setup (5 minutes)

**Local Development (Vault Dev Server)**:

```bash
# Start Vault dev server (Docker)
docker run --rm --name vault-dev -p 8200:8200 -e VAULT_DEV_ROOT_TOKEN_ID=dev-token hashicorp/vault:latest

# In another terminal, configure Vault
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='dev-token'

# Enable Transit secrets engine
vault secrets enable transit

# Create encryption key for PII
vault write -f transit/keys/pos-pii-key

# Verify key created
vault read transit/keys/pos-pii-key
```

**Environment Configuration**:

```bash
# Add to .env or backend/.env
VAULT_ADDR=http://127.0.0.1:8200
VAULT_TOKEN=dev-token
VAULT_TRANSIT_KEY=pos-pii-key
ENCRYPTION_ENABLED=true
```

**Alternative: File-based keys (testing only)**:

```bash
# Generate random encryption key (32 bytes for AES-256)
openssl rand -hex 32 > ~/.pos/encryption.key
chmod 400 ~/.pos/encryption.key

# Add to .env
ENCRYPTION_KEY_FILE=~/.pos/encryption.key
ENCRYPTION_ENABLED=true
VAULT_ENABLED=false  # Disable Vault, use file-based
```

### 4. Backend Service Updates (10 minutes)

**Install new dependencies**:

```bash
cd backend/user-service  # Or any service needing encryption
go get github.com/hashicorp/vault/api@latest

# If using shared encryption library
cd backend/shared
go mod init github.com/pos/shared
go mod tidy
```

**Create encryption service** (if not exists):

```bash
mkdir -p backend/shared/encryption
cat > backend/shared/encryption/service.go << 'EOF'
package encryption

import (
    "context"
    "encoding/base64"
    "fmt"
    "github.com/hashicorp/vault/api"
)

type Service interface {
    Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
    Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
}

type VaultEncryptionService struct {
    client    *api.Client
    keyName   string
}

func NewVaultEncryptionService(vaultAddr, vaultToken, keyName string) (*VaultEncryptionService, error) {
    config := api.DefaultConfig()
    config.Address = vaultAddr
    client, err := api.NewClient(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create Vault client: %w", err)
    }
    client.SetToken(vaultToken)
    
    return &VaultEncryptionService{
        client:  client,
        keyName: keyName,
    }, nil
}

func (s *VaultEncryptionService) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
    data := map[string]interface{}{
        "plaintext": base64.StdEncoding.EncodeToString(plaintext),
    }
    
    secret, err := s.client.Logical().Write(fmt.Sprintf("transit/encrypt/%s", s.keyName), data)
    if err != nil {
        return nil, fmt.Errorf("encryption failed: %w", err)
    }
    
    ciphertext := secret.Data["ciphertext"].(string)
    return []byte(ciphertext), nil
}

func (s *VaultEncryptionService) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
    data := map[string]interface{}{
        "ciphertext": string(ciphertext),
    }
    
    secret, err := s.client.Logical().Write(fmt.Sprintf("transit/decrypt/%s", s.keyName), data)
    if err != nil {
        return nil, fmt.Errorf("decryption failed: %w", err)
    }
    
    plaintextB64 := secret.Data["plaintext"].(string)
    plaintext, err := base64.StdEncoding.DecodeString(plaintextB64)
    if err != nil {
        return nil, fmt.Errorf("base64 decode failed: %w", err)
    }
    
    return plaintext, nil
}
EOF
```

**Create log masking utility** (if not exists):

```bash
mkdir -p backend/shared/masker
cat > backend/shared/masker/masker.go << 'EOF'
package masker

import (
    "regexp"
    "strings"
)

var (
    emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    phonePattern = regexp.MustCompile(`^(\+?[0-9]{1,3})[0-9]{6,}([0-9]{4})$`)
)

// Email masks email: user@example.com -> us***@example.com
func Email(email string) string {
    if len(email) < 3 || !emailPattern.MatchString(email) {
        return "***INVALID***"
    }
    parts := strings.Split(email, "@")
    localPart := parts[0]
    if len(localPart) <= 2 {
        return "**@" + parts[1]
    }
    return localPart[:2] + "***@" + parts[1]
}

// Phone masks phone: +628123456789 -> +62******789
func Phone(phone string) string {
    matches := phonePattern.FindStringSubmatch(phone)
    if matches == nil {
        return "***INVALID***"
    }
    return matches[1] + "******" + matches[2]
}

// Token masks tokens: abc123xyz789 -> abc***789
func Token(token string) string {
    if len(token) < 10 {
        return "***REDACTED***"
    }
    return token[:3] + "***" + token[len(token)-3:]
}

// IP masks IP address: 192.168.1.100 -> 192.168.*.*
func IP(ip string) string {
    parts := strings.Split(ip, ".")
    if len(parts) == 4 {
        return parts[0] + "." + parts[1] + ".*.*"
    }
    return "***INVALID_IP***"
}

// Name masks personal name: John Doe -> J*** D***
func Name(name string) string {
    words := strings.Fields(name)
    masked := make([]string, len(words))
    for i, word := range words {
        if len(word) == 0 {
            masked[i] = "***"
        } else {
            masked[i] = string(word[0]) + "***"
        }
    }
    return strings.Join(masked, " ")
}

// Redact completely hides sensitive data
func Redact() string {
    return "***REDACTED***"
}
EOF
```

**Update logging in services**:

```go
// Before (unsafe)
logger.Info().Str("email", user.Email).Msg("User registered")

// After (safe)
import "github.com/pos/shared/masker"
logger.Info().Str("email", masker.Email(user.Email)).Msg("User registered")
```

### 5. Frontend Setup (5 minutes)

**Install dependencies** (if needed):

```bash
cd frontend
npm install axios  # Already installed
```

**Add consent components**:

```bash
mkdir -p frontend/src/components/Consent
cat > frontend/src/components/Consent/ConsentCheckbox.tsx << 'EOF'
import React from 'react';

interface ConsentCheckboxProps {
  purpose: string;
  label: string;
  description: string;
  required: boolean;
  checked: boolean;
  onChange: (purpose: string, granted: boolean) => void;
  link?: string;
}

export const ConsentCheckbox: React.FC<ConsentCheckboxProps> = ({
  purpose,
  label,
  description,
  required,
  checked,
  onChange,
  link
}) => {
  return (
    <div className="consent-checkbox-group mb-4">
      <label className="flex items-start">
        <input
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange(purpose, e.target.checked)}
          className="mt-1 mr-3"
          required={required}
        />
        <div>
          <div className="font-medium">
            {label} {required && <span className="text-red-500">*</span>}
          </div>
          <div className="text-sm text-gray-600 mt-1">{description}</div>
          {link && (
            <a href={link} className="text-sm text-blue-600 hover:underline mt-1 inline-block">
              Pelajari lebih lanjut →
            </a>
          )}
        </div>
      </label>
    </div>
  );
};
EOF
```

**Add privacy policy page**:

```bash
mkdir -p frontend/src/pages
cat > frontend/src/pages/privacy-policy.tsx << 'EOF'
import React, { useEffect, useState } from 'react';
import axios from 'axios';

const PrivacyPolicyPage: React.FC = () => {
  const [policy, setPolicy] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    axios.get('/api/v1/privacy-policy')
      .then(res => {
        setPolicy(res.data.data);
        setLoading(false);
      })
      .catch(err => {
        console.error('Failed to load privacy policy', err);
        setLoading(false);
      });
  }, []);

  if (loading) return <div>Loading...</div>;

  return (
    <div className="container mx-auto px-4 py-8 max-w-4xl">
      <h1 className="text-3xl font-bold mb-4">Kebijakan Privasi</h1>
      <div className="text-sm text-gray-600 mb-6">
        Versi {policy?.version} | Berlaku sejak {new Date(policy?.effective_date).toLocaleDateString('id-ID')}
      </div>
      <div className="prose prose-lg" dangerouslySetInnerHTML={{ __html: policy?.policy_text }} />
    </div>
  );
};

export default PrivacyPolicyPage;
EOF
```

### 6. Run Services (2 minutes)

**Start backend services**:

```bash
# Terminal 1: Vault (if not already running)
docker run --rm --name vault-dev -p 8200:8200 -e VAULT_DEV_ROOT_TOKEN_ID=dev-token hashicorp/vault:latest

# Terminal 2: User service (with encryption enabled)
cd backend/user-service
export ENCRYPTION_ENABLED=true
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=dev-token
export VAULT_TRANSIT_KEY=pos-pii-key
go run main.go

# Terminal 3: Audit service (if implemented as separate service)
cd backend/audit-service
go run main.go

# Terminal 4: Frontend
cd frontend
npm run dev
```

**Verify setup**:

```bash
# Check consent purposes API
curl http://localhost:8080/v1/consent/purposes

# Check privacy policy API
curl http://localhost:8080/v1/privacy-policy

# Check Vault encryption (manual test)
curl -X POST http://127.0.0.1:8200/v1/transit/encrypt/pos-pii-key \
  -H "X-Vault-Token: dev-token" \
  -d '{"plaintext":"dGVzdEBleGFtcGxlLmNvbQ=="}'  # base64("test@example.com")
```

## Development Workflow

### Test Consent Collection

1. **Navigate to registration page**: `http://localhost:3000/register`
2. **Verify consent checkboxes displayed**:
   - ✅ Pemrosesan data operasional (wajib)
   - ✅ Analisis dan peningkatan layanan (wajib)
   - ☐ Promosi dan iklan (opsional)
   - ✅ Integrasi pembayaran Midtrans (wajib)
3. **Try to submit without required consents**: Should show validation error
4. **Check all required consents and submit**
5. **Verify consent records created**:
   ```sql
   SELECT * FROM consent_records WHERE subject_id = '<new_user_id>' ORDER BY granted_at DESC;
   ```

### Test Encryption

1. **Create a new user** (via registration or API)
2. **Check encrypted fields in database**:
   ```sql
   SELECT id, email, email_encrypted, first_name_encrypted FROM users WHERE id = '<user_id>';
   -- email_encrypted should be bytea (encrypted), not plaintext
   ```
3. **Verify decryption works**:
   ```bash
   # Via API
   curl -H "Authorization: Bearer <token>" http://localhost:8080/v1/user/me
   # Should return decrypted email, first_name, last_name
   ```

### Test Log Masking

1. **Trigger user registration** (logs email during registration)
2. **Check application logs**:
   ```bash
   tail -f backend/user-service/logs/app.log | grep "User registered"
   # Should show: "User registered" email="us***@example.com"
   # NOT: "User registered" email="user@example.com"
   ```
3. **Verify no plaintext PII in logs**:
   ```bash
   grep -E '[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}' backend/*/logs/*.log
   # Should NOT find unmasked emails (or very few false positives)
   ```

### Test Audit Trail

1. **Perform data modification** (update user profile)
2. **Check audit events published to Kafka**:
   ```bash
   kafka-console-consumer --bootstrap-server localhost:9092 --topic audit.events --from-beginning
   ```
3. **Verify audit events persisted in database**:
   ```sql
   SELECT * FROM audit_events WHERE tenant_id = '<tenant_id>' ORDER BY timestamp DESC LIMIT 10;
   ```

### Test Guest Data Deletion

1. **Create guest order** via checkout (without account)
2. **Access guest order data**:
   ```bash
   curl "http://localhost:8080/v1/guest/order/ORD-20260102-ABC123/data?email=guest@example.com"
   ```
3. **Request data deletion**:
   ```bash
   curl -X POST http://localhost:8080/v1/guest/order/ORD-20260102-ABC123/delete \
     -H "Content-Type: application/json" \
     -d '{"email_or_phone": "guest@example.com", "confirmation": true}'
   ```
4. **Verify anonymization**:
   ```sql
   SELECT customer_name, customer_email, is_anonymized FROM guest_orders WHERE order_reference = 'ORD-20260102-ABC123';
   -- customer_name should be "Deleted User", customer_email should be NULL, is_anonymized should be TRUE
   ```

## Testing

### Unit Tests

```bash
# Test encryption service
cd backend/shared/encryption
go test -v

# Test masker utility
cd backend/shared/masker
go test -v

# Test consent validation
cd backend/user-service
go test -v ./handlers/consent_test.go
```

### Integration Tests

```bash
# Run integration test suite (requires test database)
cd backend
./scripts/run-integration-tests.sh
```

### Contract Tests

```bash
# Validate OpenAPI contracts
cd specs/006-uu-pdp-compliance/contracts
npx @stoplight/spectral-cli lint consent-api.yaml
npx @stoplight/spectral-cli lint data-rights-api.yaml
```

## Troubleshooting

### Encryption Not Working

**Symptom**: Encrypted fields are NULL or plaintext remains

**Solutions**:
1. Check `ENCRYPTION_ENABLED=true` in environment
2. Verify Vault is running: `curl -H "X-Vault-Token: dev-token" http://127.0.0.1:8200/v1/sys/health`
3. Check Vault key exists: `vault read transit/keys/pos-pii-key`
4. Check service logs for encryption errors

### Log Masking Not Applied

**Symptom**: Plaintext PII appears in logs

**Solutions**:
1. Verify `masker` package imported in service
2. Check all log statements use `masker.Email(email)`, not raw `email`
3. Run linter: `golangci-lint run --enable=all`
4. Add CI/CD check: `grep -r 'Str("email"' backend/ --include="*.go"` (should find `masker.Email` calls only)

### Consent Validation Fails

**Symptom**: Cannot register/checkout even with all consents checked

**Solutions**:
1. Check consent purposes seeded: `SELECT * FROM consent_purposes;`
2. Verify frontend sends correct payload: Check browser Network tab
3. Check backend validation logic: Review `handlers/consent.go`
4. Verify required consents match: `SELECT purpose_code FROM consent_purposes WHERE is_required = TRUE;`

### Audit Events Not Persisted

**Symptom**: Audit events published to Kafka but not in database

**Solutions**:
1. Check audit service is running and consuming Kafka
2. Verify Kafka topic exists: `kafka-topics --bootstrap-server localhost:9092 --list | grep audit`
3. Check audit service logs for errors
4. Verify partition exists: `SELECT tablename FROM pg_tables WHERE tablename LIKE 'audit_events_%' ORDER BY tablename DESC LIMIT 1;`

## Next Steps

1. **Review research.md**: Understand design decisions
2. **Review data-model.md**: Understand database schema
3. **Review contracts**: OpenAPI specs in `contracts/` directory
4. **Implement Phase 2 tasks**: See `tasks.md` (generated by `/speckit.tasks` command, not yet created)
5. **Write tests**: Follow Test-First Development principle from constitution

## Additional Resources

- **UU PDP No.27 Tahun 2022**: Indonesian data protection law (full text available from government portal)
- **HashiCorp Vault Docs**: https://www.vaultproject.io/docs
- **PostgreSQL Partitioning**: https://www.postgresql.org/docs/current/ddl-partitioning.html
- **Go crypto/cipher**: https://pkg.go.dev/crypto/cipher
- **Zerolog Logging**: https://github.com/rs/zerolog

## Support

For questions or issues during implementation, contact:
- **Tech Lead**: (Add contact info)
- **Security Team**: (Add contact info for encryption key management)
- **Compliance Officer**: (Add contact info for legal questions)
