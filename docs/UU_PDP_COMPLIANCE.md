# Indonesian Data Protection Compliance (UU PDP) - Implementation Guide

**Feature**: 006-uu-pdp-compliance  
**Status**: ✅ COMPLETE  
**Last Updated**: January 16, 2026  
**Legal Framework**: UU PDP No.27 Tahun 2022 (Indonesian Personal Data Protection Law)

---

## Table of Contents

1. [Overview](#overview)
2. [Encryption Architecture](#encryption-architecture)
3. [Consent Management](#consent-management)
4. [Audit Trail](#audit-trail)
5. [Data Rights](#data-rights)
6. [Data Retention](#data-retention)
7. [Privacy Policy](#privacy-policy)
8. [Troubleshooting](#troubleshooting)
9. [Compliance Verification](#compliance-verification)

---

## Overview

### What is UU PDP?

**UU PDP (Undang-Undang Pelindungan Data Pribadi)** is Indonesia's Personal Data Protection Law enacted in 2022. It establishes comprehensive data protection requirements similar to EU GDPR, including:

- **Article 5**: Data minimization (don't keep data longer than necessary)
- **Article 6**: Transparency (clear privacy policy, timely responses)
- **Article 20**: Consent requirement (explicit opt-in for data processing)
- **Article 21**: Right to revoke consent
- **Articles 3-6**: Data subject rights (access, correction, deletion)
- **Article 56**: Audit trail retention (7 years)

**Penalties**: Non-compliance can result in fines up to **IDR 6 billion** (~$400,000 USD).

### Implementation Status

| User Story              | Status      | Tasks     | Key Features                                           |
| ----------------------- | ----------- | --------- | ------------------------------------------------------ |
| US1: Encryption at Rest | ✅ COMPLETE | T033-T060 | Field-level encryption, Vault integration, log masking |
| US2: Tenant Data Rights | ✅ COMPLETE | T119-T138 | View/export/delete data, 90-day retention              |
| US3: Guest Data Rights  | ✅ COMPLETE | T139-T157 | Guest data access, anonymization                       |
| US4: Audit Trail        | ✅ COMPLETE | T098-T118 | Immutable logging, partition management                |
| US5: Consent Collection | ✅ COMPLETE | T061-T085 | Purpose-based consent, versioning                      |
| US6: Privacy Policy     | ✅ COMPLETE | T086-T097 | Bilingual policy, SSR for SEO                          |
| US7: Consent Revocation | ✅ COMPLETE | T158-T170 | Revoke optional consents                               |
| US8: Data Retention     | ✅ COMPLETE | T171-T189 | Automated cleanup, retention policies                  |

---

## Encryption Architecture

### Overview

All personally identifiable information (PII) is encrypted at rest using **HashiCorp Vault Transit Engine** with **AES-256-GCM** encryption.

### What is Encrypted?

**User Data**:

- Email addresses
- Full names
- Phone numbers

**Guest Orders**:

- Customer name
- Customer phone
- Customer email
- IP address
- User agent

**Tenant Configurations**:

- Midtrans credentials (Server Key, Client Key)

**Notification Metadata**:

- Recipient email in notification metadata

### Encryption Flow

```
┌─────────────┐
│ Application │
└──────┬──────┘
       │ Encrypt(plaintext)
       ▼
┌─────────────────┐
│ Encryption      │
│ Service         │
│ (per service)   │
└──────┬──────────┘
       │ HTTP POST /transit/encrypt/pos-keys
       ▼
┌─────────────────┐
│ HashiCorp Vault │
│ Transit Engine  │
└──────┬──────────┘
       │ Returns: "vault:v1:encrypted_data"
       ▼
┌─────────────────┐
│ PostgreSQL      │
│ (ciphertext)    │
└─────────────────┘
```

### Key Management

**Production**:

- Vault Transit Engine manages encryption keys
- Keys never leave Vault
- Automatic key rotation supported
- Ciphertext format includes version: `vault:v1:base64_ciphertext`

**Development**:

- File-based keys stored in `encryption_keys/` (permissions: 400)
- **WARNING**: Never commit keys to git (in .gitignore)
- Generate keys: `./scripts/setup-env.sh`

### Encryption Service API

Located in each service: `src/utils/encryption.go`

```go
// Encrypt plaintext to ciphertext
ciphertext, err := encryptionService.Encrypt(ctx, "sensitive_data")

// Decrypt ciphertext to plaintext
plaintext, err := encryptionService.Decrypt(ctx, "vault:v1:...")

// Batch operations for performance
ciphertexts, err := encryptionService.EncryptBatch(ctx, []string{"data1", "data2"})
plaintexts, err := encryptionService.DecryptBatch(ctx, []string{"vault:v1:...", "vault:v1:..."})
```

### Log Masking

All structured logging automatically masks PII:

```go
// Before masking (NEVER logged)
log.Info().Str("email", "user@example.com").Msg("User registered")

// After masking (actual log output)
log.Info().Str("email", "***@***.com").Msg("User registered")
```

**Masked Fields**: email, phone, password, token, api_key, credit_card, ssn

**Implementation**: `src/observability/logger.go` - uses `MaskSensitiveData()` hook

### Performance

**Benchmarks** (per `encryption_bench_test.go`):

- Encrypt operation: ~2-5ms
- Decrypt operation: ~2-5ms
- Batch operations: ~10ms for 10 items
- **Overhead**: <10% for business operations ✅

---

## Consent Management

### Consent Purposes

| Purpose Code                  | Required? | Description                                                 | Legal Basis                            |
| ----------------------------- | --------- | ----------------------------------------------------------- | -------------------------------------- |
| `operational`                 | ✅ Yes    | Essential operations (order processing, account management) | UU PDP Art 20 (necessary for contract) |
| `payment_processing_midtrans` | ✅ Yes    | Payment processing via Midtrans                             | UU PDP Art 20 (necessary for service)  |
| `analytics`                   | ❌ No     | Usage analytics, service improvement                        | UU PDP Art 20 (consent required)       |
| `advertising`                 | ❌ No     | Promotional communications, marketing                       | UU PDP Art 20 (consent required)       |

### Consent Collection Flow

1. **Tenant Registration**:

   - All purposes displayed with checkboxes
   - Required purposes pre-checked and disabled
   - Optional purposes require explicit opt-in
   - Cannot proceed without accepting required consents

2. **Guest Checkout**:

   - Same consent collection mechanism
   - Stored with order reference
   - Accessible via `/guest/data/[order_reference]`

3. **API**:
   ```bash
   POST /api/v1/consent/grant
   {
     "subject_type": "tenant",
     "subject_id": "uuid",
     "consents": [
       {"purpose_code": "operational", "granted": true},
       {"purpose_code": "analytics", "granted": true}
     ]
   }
   ```

### Consent Versioning

Consent purposes can be updated over time:

**Example**:

```sql
-- Version 1 (original)
INSERT INTO consent_purposes (code, version) VALUES ('analytics', 1);

-- Version 2 (updated description)
INSERT INTO consent_purposes (code, version) VALUES ('analytics', 2);
```

**Behavior**:

- Users see latest version when granting consent
- Historical consent records track which version was accepted
- Re-consent triggered when major updates occur

### Consent Revocation

**UI**: Settings → Privacy Settings → Toggle switches for optional consents

**API**:

```bash
POST /api/v1/consent/revoke
{
  "purpose_code": "analytics"
}
```

**Effects**:

- `revoked_at` timestamp set on consent record
- System stops processing data for that purpose
- Can be re-granted at any time

**Audit**: All consent changes logged to audit trail with `ConsentGrantedEvent` or `ConsentRevokedEvent`

---

## Audit Trail

### What is Logged?

**All operations on PII-containing entities**:

- User creation/update/deletion
- Guest order creation
- Tenant configuration changes
- Login success/failure
- Session creation/expiration
- Consent grant/revoke
- Data cleanup operations

### Event Structure

```json
{
  "event_id": "uuid",
  "event_type": "USER_CREATED",
  "tenant_id": "uuid",
  "actor_id": "uuid",
  "actor_email": "admin@tenant.com",
  "action": "create",
  "resource_type": "user",
  "resource_id": "uuid",
  "before_value": null,
  "after_value": "{encrypted_json}",
  "ip_address": "192.168.1.1",
  "user_agent": "Mozilla/5.0...",
  "compliance_tag": "UU_PDP_Article_56",
  "timestamp": "2026-01-16T10:00:00Z"
}
```

### Immutability

**Enforcement**:

- Database-level: `REVOKE UPDATE, DELETE ON audit_events FROM ALL`
- Trigger: `prevent_audit_modification()` blocks updates/deletes
- Application: No update/delete methods in `AuditRepository`

**Verification**:

```bash
# Try to modify audit record (should fail)
psql -c "UPDATE audit_events SET action='hacked' WHERE id='...';"
# ERROR: UPDATE on table "audit_events" violates row-level security policy
```

### Partition Management

Audit events are partitioned by month for performance:

**Automatic Creation**:

- `PartitionService` creates next month's partition 7 days before month end
- Runs every 6 hours via `StartMonitor()`

**Manual Creation**:

```bash
# Create partition for February 2026
psql -c "SELECT create_partition_for_audit_events(2026, 2);"
```

### Querying Audit Trail

**API**:

```bash
GET /api/v1/audit/tenant?date_range=2026-01-01,2026-01-31&action_type=create&limit=100
```

**Database**:

```sql
-- Find all user deletions in last 90 days
SELECT * FROM audit_events
WHERE resource_type = 'user'
  AND action = 'delete'
  AND timestamp > NOW() - INTERVAL '90 days'
ORDER BY timestamp DESC;
```

### Retention

**Legal Requirement**: 7 years (2555 days) per UU PDP Article 56

**Implementation**:

- Retention policy configured in `retention_policies` table
- Automatic cleanup after 7 years
- Never delete audit trail manually

---

## Data Rights

### Tenant Data Rights

**Access**: `/settings/tenant-data` (OWNER role only)

**Features**:

1. **View All Data**:

   - Business profile
   - Team members
   - Configurations
   - Consent records

2. **Export Data** (JSON format):

   ```bash
   curl -H "Authorization: Bearer $TOKEN" \
        /api/v1/tenant/data/export > tenant_data.json
   ```

3. **Delete Team Member**:
   - **Soft Delete**: 90-day grace period, data retained
   - **Hard Delete**: Permanent removal after 90 days or force deletion

### Guest Data Rights

**Access**: `/guest/order-lookup` (no authentication required)

**Verification**: Order reference + (email OR phone)

**Features**:

1. **View Order Data**:

   - Customer information
   - Order details
   - Delivery address

2. **Request Deletion**:
   - Anonymizes PII: name, email, phone, address
   - Preserves order record for merchant
   - Irreversible action

**Example**:

```
Before Deletion:
- Name: "John Doe"
- Email: "john@example.com"
- Phone: "+628123456789"

After Deletion:
- Name: "Deleted User"
- Email: null
- Phone: null
- is_anonymized: true
- anonymized_at: "2026-01-16T10:00:00Z"
```

---

## Data Retention

### Retention Policies

| Table/Type                  | Retention Period    | Legal Minimum | Cleanup Method | Notification   |
| --------------------------- | ------------------- | ------------- | -------------- | -------------- |
| `email_verification_tokens` | 2 days              | 0             | hard_delete    | -              |
| `password_reset_tokens`     | 1 day               | 0             | hard_delete    | -              |
| `user_invitations`          | 30 days             | 0             | hard_delete    | -              |
| `user_sessions`             | 7 days              | 0             | hard_delete    | -              |
| `users` (deleted)           | 90 days             | 0             | anonymize      | 30 days before |
| `orders` (guest)            | 1825 days (5 years) | 1825          | hard_delete    | -              |
| `audit_events`              | 2555 days (7 years) | 2555          | hard_delete    | -              |

### Automated Cleanup

**Schedule**: Daily at 2 AM UTC

**Process**:

1. `CleanupScheduler` triggers `CleanupOrchestrator.RunAllCleanups()`
2. Orchestrator loads active policies from `retention_policies` table
3. For each policy:
   - Acquires Redis distributed lock (prevents concurrent execution)
   - Counts expired records
   - Processes in batches (100 records per transaction)
   - Publishes `CleanupCompletedEvent` to audit trail
   - Releases lock
4. Prometheus metrics updated

**Monitoring**:

- Metric: `cleanup_records_processed_total{table}`
- Alert: `CleanupErrorsHigh` (>5 errors in 24 hours)
- Alert: `CleanupJobsStalled` (no run in 48 hours)

### Deletion Notification

**Scenario**: Soft-deleted user account

**Timeline**:

- Day 0: User requests deletion, `deleted_at` timestamp set
- Day 60: System sends email warning ("30 days remaining")
- Day 90: Account permanently anonymized

**Email**: Bilingual (Indonesian + English), includes:

- Days remaining countdown
- Deletion date
- Login button to cancel deletion
- UU PDP Article 5 compliance notice

### Admin UI

**Location**: `/admin/retention-policies` (OWNER role only)

**Features**:

- View all retention policies
- Edit retention periods (must meet legal minimums)
- Change cleanup methods
- Enable/disable policies
- Preview expired record counts

**Validation**: Frontend prevents setting retention period below legal minimum

---

## Privacy Policy

### Access

**Public URL**: `/privacy-policy` (no authentication required)

**Location**: Also linked from:

- Footer (all pages)
- Registration forms
- Checkout forms

### Content

**Sections** (bilingual Indonesian + English):

1. Data Collected
2. Purposes of Processing
3. Legal Basis (UU PDP references)
4. Retention Periods
5. Third-Party Sharing (Midtrans)
6. Security Measures
7. User Rights
8. Contact Information

### SEO Optimization

- Server-Side Rendering (SSR) for fast load times
- Meta tags: title, description, Open Graph
- Target: <2s page load time (p95) ✅

### Updates

Privacy policy is versioned in `privacy_policy_versions` table:

```sql
SELECT version, effective_date, content
FROM privacy_policy_versions
ORDER BY version DESC LIMIT 1;
```

When updating policy:

1. Insert new version with incremented version number
2. Update `effective_date`
3. Users see latest version automatically
4. Historical versions preserved for audit

---

## Troubleshooting

### Encryption Failures

**Symptom**: `Failed to encrypt field` errors

**Causes**:

1. Vault server unreachable
2. Invalid encryption key
3. Network timeout

**Solutions**:

```bash
# Check Vault health
curl http://vault:8200/v1/sys/health

# Verify transit engine mounted
vault secrets list | grep transit

# Test encryption directly
vault write transit/encrypt/pos-keys plaintext=$(echo "test" | base64)

# Check service logs
docker logs user-service | grep "encryption error"
```

### Consent Validation Failures

**Symptom**: User cannot complete registration/checkout

**Causes**:

1. Missing required consent
2. Consent purpose disabled
3. Database constraint violation

**Solutions**:

```sql
-- Check consent purposes configuration
SELECT * FROM consent_purposes WHERE is_required = TRUE;

-- Verify consent records
SELECT * FROM consent_records WHERE subject_id = 'uuid';

-- Check for conflicting records
SELECT subject_id, purpose_code, COUNT(*)
FROM consent_records
WHERE revoked_at IS NULL
GROUP BY subject_id, purpose_code
HAVING COUNT(*) > 1;
```

### Audit Trail Issues

**Symptom**: Kafka consumer lag increasing

**Check**:

```bash
# Kafka lag
kafka-consumer-groups --bootstrap-server localhost:9092 \
  --describe --group audit-consumer

# Audit service metrics
curl http://localhost:8080/metrics | grep audit_events_persisted_total

# Alert status
curl http://prometheus:9090/api/v1/alerts | grep AuditKafkaConsumerLag
```

**Solutions**:

- Scale audit-service horizontally
- Increase Kafka partition count
- Check database write performance

### Cleanup Job Failures

**Symptom**: `CleanupErrorsHigh` alert firing

**Check**:

```bash
# View cleanup logs
docker logs user-service | grep "ERROR: Cleanup"

# Check Redis locks
redis-cli KEYS "cleanup:lock:*"

# Manually trigger cleanup
curl -X POST http://localhost:8081/admin/cleanup/run-now

# Check expired record counts
curl http://localhost:8081/admin/retention-policies/{policy_id}/expired-count
```

**Solutions**:

```bash
# Release stuck lock
redis-cli DEL "cleanup:lock:users:deleted"

# Increase batch size (if processing too slow)
# Edit: backend/user-service/src/jobs/cleanup_orchestrator.go
# batchSize: 100 → 500
```

---

## Compliance Verification

### Manual Checks

**1. Verify Encryption**:

```sql
-- All user emails should be encrypted (starts with "vault:v1:")
SELECT email_encrypted FROM users WHERE email_encrypted NOT LIKE 'vault:v1:%';
-- Expected: 0 rows

-- Guest order customer data encrypted
SELECT customer_email_encrypted FROM guest_orders
WHERE customer_email_encrypted NOT LIKE 'vault:v1:%'
  AND is_anonymized = FALSE;
-- Expected: 0 rows
```

**2. Verify Log Masking**:

```bash
# Search logs for plaintext PII (should find nothing)
grep -r "user@example.com" /var/log/pos/*.log
# Expected: No matches

# Verify masking works
grep -r "***@***.com" /var/log/pos/*.log
# Expected: Multiple matches
```

**3. Verify Audit Immutability**:

```sql
-- Try to modify audit record (should fail)
UPDATE audit_events SET action = 'tampered' WHERE id = 'some-uuid';
-- Expected: ERROR - permission denied

-- Try to delete audit record (should fail)
DELETE FROM audit_events WHERE id = 'some-uuid';
-- Expected: ERROR - permission denied
```

**4. Verify Consent Records**:

```sql
-- All tenants should have operational consent
SELECT t.id, t.business_name
FROM tenants t
LEFT JOIN consent_records cr ON t.id = cr.subject_id AND cr.purpose_code = 'operational'
WHERE cr.id IS NULL;
-- Expected: 0 rows

-- All guest orders should have consent records
SELECT go.order_reference
FROM guest_orders go
LEFT JOIN consent_records cr ON go.id = cr.subject_id AND cr.subject_type = 'guest'
WHERE cr.id IS NULL;
-- Expected: 0 rows
```

**5. Verify Retention Policies**:

```sql
-- Check legal minimums enforced
SELECT table_name, retention_period_days, legal_minimum_days
FROM retention_policies
WHERE retention_period_days < legal_minimum_days;
-- Expected: 0 rows

-- Verify audit events retention ≥7 years
SELECT * FROM retention_policies
WHERE table_name = 'audit_events'
  AND retention_period_days < 2555;
-- Expected: 0 rows
```

### Automated Verification

**Script**: `scripts/verify-uu-pdp-compliance.sh`

```bash
#!/bin/bash
# Usage: ./scripts/verify-uu-pdp-compliance.sh

echo "Running UU PDP compliance verification..."

# 1. Check encryption
echo "Checking encryption..."
UNENCRYPTED=$(psql -t -c "SELECT COUNT(*) FROM users WHERE email_encrypted NOT LIKE 'vault:v1:%';")
if [ "$UNENCRYPTED" -gt 0 ]; then
  echo "❌ FAIL: $UNENCRYPTED unencrypted email records found"
  exit 1
fi
echo "✅ PASS: All emails encrypted"

# 2. Check audit immutability
echo "Checking audit immutability..."
psql -c "UPDATE audit_events SET action='test' WHERE id='00000000-0000-0000-0000-000000000000';" 2>&1 | grep -q "permission denied"
if [ $? -eq 0 ]; then
  echo "✅ PASS: Audit events are immutable"
else
  echo "❌ FAIL: Audit events can be modified"
  exit 1
fi

# 3. Check consent records
echo "Checking consent records..."
MISSING_CONSENT=$(psql -t -c "SELECT COUNT(*) FROM tenants t LEFT JOIN consent_records cr ON t.id = cr.subject_id WHERE cr.id IS NULL;")
if [ "$MISSING_CONSENT" -gt 0 ]; then
  echo "❌ FAIL: $MISSING_CONSENT tenants without consent records"
  exit 1
fi
echo "✅ PASS: All tenants have consent records"

# 4. Check log masking
echo "Checking log masking..."
if grep -r "@example.com" /var/log/pos/*.log 2>/dev/null; then
  echo "❌ FAIL: Plaintext PII found in logs"
  exit 1
fi
echo "✅ PASS: No plaintext PII in logs"

echo ""
echo "✅ All compliance checks passed!"
```

### Compliance Report

**Endpoint**: `GET /admin/compliance/report`

**Response**:

```json
{
  "report_date": "2026-01-16T10:00:00Z",
  "encrypted_records": {
    "users": 1250,
    "guest_orders": 8940,
    "tenant_configs": 150
  },
  "active_consents": {
    "operational": 1400,
    "analytics": 890,
    "advertising": 450
  },
  "audit_events": {
    "total": 145680,
    "last_30_days": 12450
  },
  "retention_coverage": {
    "users": "100%",
    "guest_orders": "100%",
    "audit_events": "100%"
  },
  "compliance_status": "✅ COMPLIANT"
}
```

---

## Additional Resources

- **Specification**: [spec.md](../specs/006-uu-pdp-compliance/spec.md)
- **Implementation Plan**: [plan.md](../specs/006-uu-pdp-compliance/plan.md)
- **Quick Start Guide**: [quickstart.md](../specs/006-uu-pdp-compliance/quickstart.md)
- **API Documentation**: [API.md](./API.md)
- **Runbooks**: [RUNBOOKS.md](./RUNBOOKS.md)

---

## Legal Disclaimer

This implementation guide provides technical documentation for UU PDP compliance features. It does not constitute legal advice. Organizations should:

1. Consult with legal counsel for compliance interpretation
2. Conduct regular privacy impact assessments
3. Maintain documentation of data processing activities
4. Review and update privacy policies as regulations evolve
5. Train staff on data protection procedures

**Last Legal Review**: [PENDING - Document in tasks.md T097]

---

_For questions or issues, contact the development team or refer to the troubleshooting section above._
