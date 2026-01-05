# Data Model: Indonesian Data Protection Compliance (UU PDP)

**Feature**: 006-uu-pdp-compliance  
**Date**: 2026-01-02  
**Phase**: 1 - Data Model Design

## Overview

This document defines the data model for UU PDP No.27 Tahun 2022 compliance features. All entities support encryption at rest, audit trails, consent management, and data retention automation as researched in Phase 0.

---

## New Entities

### 1. Consent Purposes

**Purpose**: Define reusable consent types (operational, analytics, advertising, third_party) with required/optional flags

**Fields**:
- `id`: UUID, primary key
- `purpose_code`: VARCHAR(50), unique, not null - Machine-readable code ('operational', 'analytics', 'advertising', 'third_party_midtrans')
- `purpose_name_en`: VARCHAR(100), not null - English display name
- `purpose_name_id`: VARCHAR(100), not null - Indonesian display name (primary)
- `description_en`: TEXT, not null - English description
- `description_id`: TEXT, not null - Indonesian description (primary, legal language)
- `is_required`: BOOLEAN, not null, default false - Whether consent is mandatory
- `display_order`: INT, not null - UI display order (1, 2, 3, ...)
- `created_at`: TIMESTAMPTZ, not null, default NOW()

**Relationships**:
- One-to-many with `consent_records` (one purpose → many consent grants)

**Indexes**:
- Unique index on `purpose_code` (primary lookup)
- Index on `is_required, display_order` (query UI display order)

**Constraints**:
- CHECK: `purpose_code` matches pattern `^[a-z_]+$` (lowercase with underscores only)
- CHECK: `display_order > 0`

**Business Rules**:
- Required purposes ('operational', 'third_party_midtrans') cannot be revoked while account/order is active
- Purpose codes are immutable (cannot UPDATE `purpose_code`, only soft delete and create new)
- Indonesian descriptions (`*_id` fields) are legally binding, English is for convenience

---

### 2. Privacy Policies

**Purpose**: Track versioned privacy policy text with effective dates for consent versioning

**Fields**:
- `id`: UUID, primary key
- `version`: VARCHAR(20), unique, not null - Semantic version ('1.0.0', '1.1.0', '2.0.0')
- `policy_text_id`: TEXT, not null - Full policy text in Indonesian (Bahasa Indonesia)
- `policy_text_en`: TEXT, nullable - Optional English translation
- `effective_date`: TIMESTAMPTZ, not null - When this version becomes active
- `change_summary_id`: TEXT, nullable - Description of changes from previous version (Indonesian)
- `change_summary_en`: TEXT, nullable - Description of changes (English)
- `is_major_update`: BOOLEAN, not null, default false - Material changes requiring re-consent
- `is_current`: BOOLEAN, not null, default false - Only one current version
- `created_at`: TIMESTAMPTZ, not null, default NOW()

**Relationships**:
- One-to-many with `consent_records` (one policy version → many consents)

**Indexes**:
- Unique index on `version`
- Index on `is_current` WHERE `is_current = TRUE` (find current policy)
- Index on `effective_date DESC` (policy history queries)

**Constraints**:
- CHECK: Only one row can have `is_current = TRUE` (enforce via unique partial index or trigger)
- CHECK: `version` matches semantic versioning pattern `^\d+\.\d+\.\d+$`
- CHECK: `effective_date <= NOW()` for `is_current = TRUE` (cannot set future policy as current)

**Business Rules**:
- Major version updates (e.g., 1.x.x → 2.0.0) trigger blocking re-consent workflow
- Minor/patch updates (e.g., 1.0.0 → 1.0.1) show non-blocking notification banner
- Policy text must be reviewed by legal counsel before setting `is_current = TRUE`
- Previous policy versions retained indefinitely for legal proof (never DELETE)

---

### 3. Consent Records

**Purpose**: Track individual consent grants and revocations with full metadata for legal proof

**Fields**:
- `id`: UUID, primary key
- `tenant_id`: UUID, not null, foreign key to `tenants(id)` - Which tenant this consent relates to
- `subject_type`: VARCHAR(10), not null - 'tenant' or 'guest' (who gave consent)
- `subject_id`: UUID, nullable, foreign key to `users(id)` - User ID for tenant consents, NULL for guests
- `guest_order_id`: UUID, nullable, foreign key to `guest_orders(id)` - Order ID for guest consents, NULL for tenants
- `purpose_id`: UUID, not null, foreign key to `consent_purposes(id)` - What consent is for
- `granted`: BOOLEAN, not null - TRUE = consent granted, FALSE = denied/revoked
- `policy_version`: VARCHAR(20), not null, foreign key to `privacy_policies(version)` - Policy version when consented
- `granted_at`: TIMESTAMPTZ, not null, default NOW() - When consent was granted
- `revoked_at`: TIMESTAMPTZ, nullable - When consent was revoked (NULL if still active)
- `ip_address`: INET, not null - IP address of consent action (legal proof)
- `user_agent`: TEXT, not null - Browser user agent (legal proof)
- `session_id`: UUID, nullable - Session ID at time of consent
- `consent_method`: VARCHAR(20), not null - 'registration', 'checkout', 'settings_update'
- `created_at`: TIMESTAMPTZ, not null, default NOW()

**Relationships**:
- Many-to-one with `tenants` (many consents → one tenant)
- Many-to-one with `users` (many consents → one user, if tenant consent)
- Many-to-one with `guest_orders` (many consents → one order, if guest consent)
- Many-to-one with `consent_purposes` (many consents → one purpose)
- Many-to-one with `privacy_policies` (many consents → one policy version)

**Indexes**:
- Index on `(subject_type, subject_id, purpose_id, revoked_at)` WHERE `revoked_at IS NULL` (query active consents)
- Index on `(guest_order_id, purpose_id, revoked_at)` WHERE `revoked_at IS NULL` (guest consent lookup)
- Index on `(tenant_id, granted_at DESC)` (tenant consent history)
- Index on `(purpose_id, granted_at DESC)` (consent analytics)

**Constraints**:
- CHECK: `(subject_type = 'tenant' AND subject_id IS NOT NULL AND guest_order_id IS NULL) OR (subject_type = 'guest' AND subject_id IS NULL AND guest_order_id IS NOT NULL)` (exactly one subject identity)
- CHECK: `subject_type IN ('tenant', 'guest')`
- CHECK: `consent_method IN ('registration', 'checkout', 'settings_update', 'api')`
- CHECK: `revoked_at IS NULL OR revoked_at >= granted_at` (cannot revoke before granting)

**Business Rules**:
- Consent is active when `granted = TRUE AND revoked_at IS NULL`
- Revoking consent sets `revoked_at = NOW()`, does NOT update `granted = FALSE` (preserve grant history)
- Re-granting consent after revocation inserts new row (full history preserved)
- Required consents cannot have `granted = FALSE` during registration/checkout (validation before insert)
- IP address and user agent stored for legal proof per UU PDP Article 6 (transparency requirement)

---

### 4. Audit Events

**Purpose**: Immutable audit trail of all data access and modifications for UU PDP compliance investigations

**Fields**:
- `id`: UUID, primary key
- `event_id`: VARCHAR(100), unique, not null - Idempotency key for Kafka retries
- `tenant_id`: UUID, not null - Tenant isolation
- `timestamp`: TIMESTAMPTZ, not null, default NOW() - Event timestamp
- `actor_type`: VARCHAR(20), not null - 'user', 'system', 'guest', 'admin'
- `actor_id`: UUID, nullable - User ID for users/admins, NULL for guests/system
- `actor_email`: VARCHAR(255), nullable - Email (encrypted), for data subject requests
- `session_id`: UUID, nullable - Session ID for user actions
- `action`: VARCHAR(50), not null - 'CREATE', 'READ', 'UPDATE', 'DELETE', 'ACCESS', 'EXPORT', 'ANONYMIZE'
- `resource_type`: VARCHAR(50), not null - 'product', 'order', 'user', 'tenant_config', etc.
- `resource_id`: VARCHAR(255), not null - ID of affected resource
- `ip_address`: INET, nullable - IP address of actor
- `user_agent`: TEXT, nullable - Browser user agent
- `request_id`: VARCHAR(50), nullable - Request ID for distributed tracing correlation
- `before_value`: JSONB, nullable - State before change (encrypted sensitive fields)
- `after_value`: JSONB, nullable - State after change (encrypted sensitive fields)
- `metadata`: JSONB, nullable - Additional context (error messages, query params, etc.)
- `purpose`: VARCHAR(100), nullable - Legal basis for data access (UU PDP Article 20)
- `consent_id`: UUID, nullable - Link to consent record if applicable

**Partitioning**:
- **Partitioned by**: `timestamp` (range partitioning, monthly partitions)
- **Partition naming**: `audit_events_YYYY_MM` (e.g., `audit_events_2026_01`)
- **Retention**: Hot (0-1 year), Warm (1-3 years), Cold/S3 (3-7 years)

**Relationships**:
- Many-to-one with `tenants` (many audit events → one tenant)
- Many-to-one with `consent_records` (many audit events → one consent, optional)

**Indexes** (on parent table, inherited by partitions):
- Unique index on `event_id` (Kafka idempotency)
- Index on `(tenant_id, timestamp DESC)` (tenant audit queries)
- Index on `(actor_id, timestamp DESC)` WHERE `actor_id IS NOT NULL` (data subject access requests)
- Index on `(resource_type, resource_id, timestamp DESC)` (resource history queries)
- Index on `(tenant_id, action, timestamp DESC)` (find all DELETE operations)
- GIN index on `metadata` using `jsonb_path_ops` (flexible metadata queries)

**Constraints**:
- CHECK: `actor_type IN ('user', 'system', 'guest', 'admin')`
- CHECK: `action IN ('CREATE', 'READ', 'UPDATE', 'DELETE', 'ACCESS', 'EXPORT', 'ANONYMIZE')`
- **Immutability**: REVOKE UPDATE, DELETE grants from all users (append-only, enforced at database level)

**Business Rules**:
- Audit events are immutable (no UPDATE or DELETE operations allowed)
- Sensitive fields in `before_value`/`after_value` JSONB must be encrypted before storage
- `event_id` prevents duplicate events from Kafka retries (idempotency)
- Partition management automated (create next month's partition 7 days before month end)
- Old partitions detached after 1 year, archived to S3, dropped after successful archive
- 7-year retention requirement per UU PDP (Indonesian record retention law)

---

### 5. Retention Policies

**Purpose**: Define automated data retention rules per table/record type with legal minimums

**Fields**:
- `id`: UUID, primary key
- `table_name`: VARCHAR(50), not null - Database table name ('users', 'guest_orders', 'sessions')
- `record_type`: VARCHAR(50), nullable - Optional record subtype ('verification_token', 'completed_order')
- `retention_period_days`: INT, not null - Days to retain after `retention_field` timestamp
- `retention_field`: VARCHAR(50), not null - Timestamp field to check ('created_at', 'expired_at', 'deleted_at')
- `grace_period_days`: INT, nullable - Soft delete grace period before hard delete (e.g., 90 days)
- `legal_minimum_days`: INT, nullable - Minimum retention by law (5 years = 1825 days for tax, 7 years = 2555 days for audit)
- `cleanup_method`: VARCHAR(20), not null - 'soft_delete', 'hard_delete', 'anonymize'
- `notification_days_before`: INT, nullable - Send notification N days before deletion (e.g., 30 days)
- `is_active`: BOOLEAN, not null, default true - Enable/disable policy without deleting
- `created_at`: TIMESTAMPTZ, not null, default NOW()
- `updated_at`: TIMESTAMPTZ, not null, default NOW()

**Indexes**:
- Index on `(table_name, is_active)` (query active policies)
- Index on `(retention_period_days)` (find expired records)

**Constraints**:
- CHECK: `cleanup_method IN ('soft_delete', 'hard_delete', 'anonymize')`
- CHECK: `retention_period_days > 0`
- CHECK: `legal_minimum_days IS NULL OR retention_period_days >= legal_minimum_days` (cannot configure retention below legal minimum)
- Unique constraint on `(table_name, record_type)` (one policy per table/type combination)

**Business Rules**:
- Legal minimums enforced at application layer (5 years for tax records, 7 years for audit logs)
- Grace period applies only to soft-deleted records (e.g., tenant accounts marked as deleted)
- Notification sent N days before hard delete (e.g., email warning 30 days before permanent deletion)
- Policies can be temporarily disabled (`is_active = FALSE`) without deleting configuration
- Policy updates trigger validation against legal minimums (prevent accidental compliance violations)

---

## Modified Entities

### 1. Users Table (Encryption Fields)

**Changes**:
- Add encrypted storage for existing PII fields
- Add consent tracking fields

**Schema Changes**:
- NO NEW COLUMNS ADDED - Encryption is handled at application layer
- Existing columns store encrypted values: `email`, `first_name`, `last_name`, `verification_token`
- Vault Transit Engine ciphertext format ("vault:v1:...") includes key version automatically
- Additional columns for consent tracking:
  - `accepted_policy_version`: VARCHAR(20), nullable - Last accepted privacy policy version
  - `notified_of_deletion`: BOOLEAN, not null, default false - Whether user notified before hard delete

**Migration Strategy**:
- Phase 1: Add new `*_encrypted` columns (nullable)
- Phase 2: Background job encrypts existing plaintext data, populates encrypted columns
- Phase 3: Application reads from encrypted columns (fallback to plaintext if NULL)
- Phase 4: After full migration, drop plaintext columns (or keep for debugging with `*_legacy` suffix)

**Business Rules**:
- `email` field remains unencrypted for login lookup (hash-based lookup considered but rejected for MVP)
- `password_hash` NOT encrypted (bcrypt already protects passwords)
- Encryption transparent to business logic (repository layer handles encrypt/decrypt)

---

### 2. Guest Orders Table (Encryption Fields)

**Changes**:
- Add encrypted storage for customer PII
- Link to consent records

**New Fields**:
**Schema Changes**:
- Existing columns store encrypted values: `customer_name`, `customer_phone`, `customer_email`, `ip_address`
- New anonymization tracking columns:
  - `is_anonymized`: BOOLEAN, not null, default false - Whether personal data has been deleted per guest request
  - `anonymized_at`: TIMESTAMPTZ, nullable - When data was anonymized
- Vault Transit Engine handles encryption with version embedded in ciphertext ("vault:v1:...")

**Business Rules**:
- When guest requests data deletion, set `is_anonymized = TRUE`, encrypt/null all PII fields
- Order record retained for merchant (transaction history, tax compliance) but customer data removed
- Anonymization is irreversible (cannot restore guest PII after deletion)

---

### 3. Delivery Addresses Table (Encryption Fields)

**Changes**:
- Encrypt all address components and geographic data

**Schema Changes**:
- NO NEW COLUMNS - Encryption at application layer
- Existing columns store encrypted values: `address`, `latitude`, `longitude`, `geocoded_address`
- Vault Transit Engine ciphertext includes version ("vault:v1:...")

**Business Rules**:
- Coordinates encrypted to prevent location tracking (even approximate location is PII under UU PDP)
- Address search requires decryption (performance trade-off, acceptable for delivery management use case)

---

### 4. Password Reset Tokens Table (Encryption Fields)

**Changes**:
- Encrypt token value

**Schema Changes**:
- NO NEW COLUMNS - Encryption at application layer
- Existing `token` column stores encrypted value ("vault:v1:...")

**Business Rules**:
- Tokens are already high-entropy random values, but encryption adds defense-in-depth
- Token comparison requires decryption (minimal performance impact, reset is infrequent operation)

---

### 5. Invitations Table (Encryption Fields)

**Changes**:
- Encrypt invitation tokens and recipient email

**Schema Changes**:
- NO NEW COLUMNS - Encryption at application layer
- Existing columns store encrypted values: `email`, `token`
- Vault Transit Engine ciphertext includes version ("vault:v1:...")

---

### 6. Sessions Table (Encryption Fields)

**Changes**:
- Encrypt session identifiers and IP addresses

**Schema Changes**:
- NO NEW COLUMNS - Encryption at application layer
- Existing columns store encrypted values: `session_id`, `ip_address`
- Vault Transit Engine ciphertext includes version ("vault:v1:...")

---

### 7. Notifications Table (Encryption Fields)

**Changes**:
- Encrypt recipient information and message content containing PII

**Schema Changes**:
- NO NEW COLUMNS - Conditional encryption at application layer
- Existing columns store encrypted values when PII detected: `recipient`, `message_body`, `metadata`
- Vault Transit Engine ciphertext includes version ("vault:v1:...")

**Business Rules**:
- Not all notifications contain PII (e.g., "New order received" may not include customer name)
- Encrypt only if message body or metadata contains PII (conditional encryption)
- Non-PII notifications remain unencrypted for performance

---

### 8. Tenant Configs Table (Encryption Fields)

**Changes**:
- Encrypt payment gateway credentials

**Schema Changes**:
- NO NEW COLUMNS - Encryption at application layer
- Existing columns store encrypted values: `midtrans_server_key`, `midtrans_client_key`
- Vault Transit Engine ciphertext includes version ("vault:v1:...")

**Business Rules**:
- Payment credentials are extremely sensitive (financial risk, not just privacy risk)
- All payment-related config encrypted, no exceptions
- Key rotation must re-encrypt all tenant configs

---

## Entity Relationships Diagram

```
┌─────────────────┐
│ Privacy         │
│ Policies        │
│                 │
│ - version (PK)  │
│ - policy_text_id│
│ - is_current    │
└────────┬────────┘
         │
         │ 1:N (policy version → consent)
         │
         ▼
┌────────────────────────────────────────┐
│ Consent Records                        │
│                                        │
│ - id (PK)                              │
│ - tenant_id (FK → tenants)             │
│ - subject_type                         │
│ - subject_id (FK → users, nullable)    │
│ - guest_order_id (FK → guest_orders)   │
│ - purpose_id (FK → consent_purposes)   │
│ - policy_version (FK → privacy_policies)│
│ - granted                              │
│ - granted_at                           │
│ - revoked_at                           │
└────────┬───────────────────────────────┘
         │
         ├─ N:1 (consent → tenant)
         │
         ├─ N:1 (consent → user, if tenant consent)
         │
         ├─ N:1 (consent → guest order, if guest consent)
         │
         └─ N:1 (consent → purpose)
              │
              ▼
         ┌────────────────┐
         │ Consent        │
         │ Purposes       │
         │                │
         │ - id (PK)      │
         │ - purpose_code │
         │ - is_required  │
         └────────────────┘


┌────────────────────────────────────────┐
│ Audit Events (Partitioned by timestamp)│
│                                        │
│ - id (PK)                              │
│ - event_id (unique)                    │
│ - tenant_id (FK → tenants)             │
│ - timestamp                            │
│ - actor_type                           │
│ - actor_id                             │
│ - action                               │
│ - resource_type                        │
│ - resource_id                          │
│ - before_value (JSONB)                 │
│ - after_value (JSONB)                  │
│ - consent_id (FK → consent_records)    │
└────────┬───────────────────────────────┘
         │
         ├─ N:1 (audit event → tenant)
         │
         └─ N:1 (audit event → consent, optional)


┌────────────────────────────────────────┐
│ Retention Policies                     │
│                                        │
│ - id (PK)                              │
│ - table_name                           │
│ - record_type                          │
│ - retention_period_days                │
│ - legal_minimum_days                   │
│ - cleanup_method                       │
│ - notification_days_before             │
└────────────────────────────────────────┘
(No foreign keys - configuration table)


Modified Entities (Encryption Fields Added):

┌─────────────────┐       ┌──────────────────┐
│ Users           │       │ Guest Orders     │
│                 │       │                  │
│ - email_enc     │       │ - customer_name_enc
│ - first_name_enc│       │ - customer_email_enc
│ - last_name_enc │       │ - is_anonymized  │
└─────────────────┘       └──────────────────┘

┌──────────────────┐      ┌──────────────────┐
│ Delivery         │      │ Password Reset   │
│ Addresses        │      │ Tokens           │
│                  │      │                  │
│ - address_enc    │      │ - token_enc      │
│ - lat_enc        │      └──────────────────┘
│ - long_enc       │
└──────────────────┘

┌──────────────────┐      ┌──────────────────┐
│ Invitations      │      │ Sessions         │
│                  │      │                  │
│ - email_enc      │      │ - session_id_enc │
│ - token_enc      │      │ - ip_address_enc │
└──────────────────┘      └──────────────────┘

┌──────────────────┐      ┌──────────────────┐
│ Notifications    │      │ Tenant Configs   │
│                  │      │                  │
│ - recipient_enc  │      │ - midtrans_*_enc │
│ - message_enc    │      └──────────────────┘
└──────────────────┘
```

---

## Data Access Patterns

### 1. Check Tenant Consent Status

**Query**: Determine if tenant has active consent for specific purpose

```sql
SELECT granted
FROM consent_records
WHERE subject_type = 'tenant'
  AND subject_id = :user_id
  AND purpose_id = (SELECT id FROM consent_purposes WHERE purpose_code = :purpose)
  AND revoked_at IS NULL
ORDER BY granted_at DESC
LIMIT 1;
```

**Performance**: Index on `(subject_type, subject_id, purpose_id, revoked_at)` ensures fast lookup (index-only scan)

---

### 2. Query Tenant Audit Trail

**Query**: Retrieve all audit events for tenant in date range

```sql
SELECT id, timestamp, actor_type, action, resource_type, resource_id
FROM audit_events
WHERE tenant_id = :tenant_id
  AND timestamp >= :start_date
  AND timestamp < :end_date
ORDER BY timestamp DESC
LIMIT 100;
```

**Performance**: Partition pruning limits scan to relevant months, index on `(tenant_id, timestamp DESC)` accelerates query

---

### 3. Guest Data Access (Order Lookup)

**Query**: Retrieve guest order with decrypted PII

```sql
SELECT 
    id,
    order_reference,
    pgp_sym_decrypt(customer_name_encrypted, :encryption_key) AS customer_name,
    pgp_sym_decrypt(customer_email_encrypted, :encryption_key) AS customer_email,
    pgp_sym_decrypt(customer_phone_encrypted, :encryption_key) AS customer_phone,
    is_anonymized
FROM guest_orders
WHERE order_reference = :order_ref
  AND pgp_sym_decrypt(customer_email_encrypted, :encryption_key) = :email;
```

**Note**: In application-layer encryption (not pgcrypto), decryption happens in Go code, not SQL

---

### 4. Find Expired Records for Cleanup

**Query**: Identify records eligible for deletion per retention policy

```sql
-- Example: Find expired verification tokens
SELECT id
FROM users
WHERE verification_token_expires_at < NOW() - INTERVAL '2 days'
  AND verification_token IS NOT NULL
LIMIT 100;
```

**Performance**: Batch LIMIT 100 prevents long-running transactions, index on `verification_token_expires_at` accelerates query

---

### 5. Re-consent Required Check

**Query**: Determine if user needs to re-consent to updated policy

```sql
SELECT 
    u.id,
    u.accepted_policy_version,
    pp.version AS current_version,
    pp.is_major_update
FROM users u
CROSS JOIN privacy_policies pp
WHERE u.id = :user_id
  AND pp.is_current = TRUE
  AND u.accepted_policy_version != pp.version;
```

**Performance**: Unique partial index on `is_current = TRUE` ensures fast lookup of current policy

---

## Storage Estimates

### New Tables

| Table | Estimated Row Count (7 years) | Avg Row Size | Total Size |
|-------|-------------------------------|--------------|------------|
| `consent_purposes` | 10 (stable config) | 500 bytes | 5 KB |
| `privacy_policies` | 20 versions | 50 KB each | 1 MB |
| `consent_records` | 500K tenants × 4 purposes × 1.2 changes = 2.4M | 400 bytes | 960 MB |
| `audit_events` | 30K events/day × 365 × 7 = 76.6M | 2 KB | 153 GB (75 GB with partitioning + compression) |
| `retention_policies` | 20 (config) | 300 bytes | 6 KB |

**Total new storage**: ~76 GB (with partitioning and archival)

### Modified Tables (Encryption Overhead)

| Table | Current Row Count (estimate) | Encryption Overhead per Row | Additional Storage |
|-------|------------------------------|-----------------------------|--------------------|
| `users` | 10K tenants × 5 users = 50K | ~200 bytes (3 encrypted fields) | 10 MB |
| `guest_orders` | 1K orders/day × 365 × 3 years = 1M | ~300 bytes (5 encrypted fields) | 300 MB |
| `delivery_addresses` | 1M orders × 0.8 delivery = 800K | ~250 bytes (4 encrypted fields) | 200 MB |
| `password_reset_tokens` | 100 active at any time | ~100 bytes | negligible |
| `invitations` | 500 active at any time | ~150 bytes | negligible |
| `sessions` | 1K concurrent × 1.5x = 1.5K | ~150 bytes | negligible |
| `notifications` | 10K/day × 90 days retention = 900K | ~200 bytes | 180 MB |
| `tenant_configs` | 10K tenants | ~200 bytes | 2 MB |

**Total encryption overhead**: ~692 MB

**Grand total storage impact**: ~77 GB over 7 years

---

## Migration Plan

### Phase 1: Schema Changes (Non-breaking)

1. Create new tables: `consent_purposes`, `privacy_policies`, `consent_records`, `audit_events`, `retention_policies`
2. Add encrypted columns to existing tables (all nullable)
3. Add indexes for new tables
4. Seed `consent_purposes` with initial purposes (operational, analytics, advertising, third_party_midtrans)
5. Seed `privacy_policies` with version 1.0.0 (initial policy)

**Rollback strategy**: DROP new tables and columns (no data loss, feature not enabled yet)

---

### Phase 2: Data Migration (In-Place Encryption)

1. Data migration tool encrypts existing PII in-place (same columns)
2. Read plaintext from columns like `email`, `first_name`, `customer_name`
3. Encrypt via Vault Transit Engine (produces "vault:v1:..." ciphertext)
4. Update same columns with encrypted values
5. Monitor progress via Prometheus metrics (`migration_records_encrypted_total`)

**Duration**: Estimated 2-4 hours for 1M records at 500 records/second encryption rate

**Rollback strategy**: Application reads from plaintext columns if encrypted columns are NULL (dual-read fallback)

---

### Phase 3: Application Cutover (Feature Flags)

1. Enable encryption writes (new records encrypted immediately)
2. Enable encryption reads (prefer encrypted columns, fallback to plaintext)
3. Enable consent collection UI (registration and checkout forms)
4. Enable audit trail publishing (Kafka events)
5. Enable data retention cleanup jobs

**Feature flags**:
- `ENCRYPTION_ENABLED`: Enable encrypt/decrypt operations
- `CONSENT_REQUIRED`: Block registration/checkout without consents
- `AUDIT_TRAIL_ENABLED`: Publish audit events to Kafka
- `RETENTION_CLEANUP_ENABLED`: Run automated cleanup jobs

**Rollback strategy**: Disable feature flags, application reverts to plaintext reads

---

### Phase 4: Plaintext Deprecation (Optional)

1. After 100% migration verified, drop plaintext columns from tables
2. Or rename to `*_legacy` suffix for forensic debugging (recommended approach)
3. Update application to remove plaintext fallback logic

**Risk**: Irreversible (cannot easily roll back encryption after dropping plaintext columns)

---

## Testing Strategy

### Unit Tests
- Encryption/decryption round-trip correctness
- Consent validation logic (required vs optional)
- Retention policy legal minimum enforcement
- Audit event serialization

### Integration Tests
- Encrypted data stored correctly in database
- Consent records linked to orders/users
- Audit events published to Kafka and persisted
- Cleanup jobs delete expired records

### Contract Tests
- Consent API endpoints enforce validation
- Audit trail query API returns correct fields
- Data deletion API anonymizes PII

### Performance Tests
- Encryption overhead <10% on typical operations
- Audit trail can handle 1000 events/second burst
- Cleanup jobs complete within 2 hours for 1M records

### Security Tests
- No plaintext PII in database after migration
- No plaintext PII in application logs
- Audit logs immutable (UPDATE/DELETE fail)
- Consent revocation prevents data processing

---

## Open Questions

1. **Encryption key rotation**: How frequently? Automated or manual trigger? Re-encryption strategy for 1M+ records?
   - **Answer deferred to Phase 2 (tasks.md)**: Quarterly rotation, manual trigger for MVP, background re-encryption job

2. **Audit log archival to S3**: Which S3-compatible storage? MinIO (self-hosted) or cloud provider?
   - **Answer deferred to infrastructure team**: MVP uses PostgreSQL only, S3 archival in future phase

3. **Consent UI language**: Indonesian only or bilingual (Indonesian + English)?
   - **Answer from spec**: Indonesian is legally required (primary), English optional for convenience

4. **Guest data deletion**: Email notification to guest after anonymization? Or silent operation?
   - **Answer deferred to Phase 2 (tasks.md)**: Send email confirmation to guest (user experience requirement)

5. **Tenant-specific retention policies**: Do enterprise tenants need custom retention periods?
   - **Answer**: Not in MVP scope (use Out of Scope section), global policies only for Phase 1

---

**Next**: Generate API contracts in `/contracts/` directory
