# POS System Operational Runbooks

**Purpose**: Step-by-step procedures for common operational tasks and incident response.

**Audience**: DevOps engineers, SREs, on-call engineers

**Last Updated**: January 16, 2026

---

## Table of Contents

1. [Vault Key Rotation](#vault-key-rotation)
2. [Audit Log Partition Management](#audit-log-partition-management)
3. [Data Cleanup Job Troubleshooting](#data-cleanup-job-troubleshooting)
4. [Data Breach Response](#data-breach-response)
5. [Database Migration Procedures](#database-migration-procedures)
6. [Service Health Checks](#service-health-checks)
7. [Emergency Procedures](#emergency-procedures)

---

## Vault Key Rotation

**Frequency**: Quarterly (every 90 days)  
**Duration**: ~30 minutes  
**Risk Level**: MEDIUM (requires careful coordination)

### Prerequisites

- [ ] Vault admin access (root token or admin policy)
- [ ] Database backup completed within last 24 hours
- [ ] All services healthy (check Prometheus alerts)
- [ ] Maintenance window scheduled (low traffic period)

### Procedure

#### 1. Pre-Rotation Verification

```bash
# Check Vault health
curl http://vault:8200/v1/sys/health
# Expected: {"initialized":true,"sealed":false,"standby":false}

# Check current key version
vault read transit/keys/pos-keys
# Note the "latest_version" value (e.g., 2)

# Test encryption with current key
vault write transit/encrypt/pos-keys plaintext=$(echo "test" | base64)
# Expected: {"ciphertext":"vault:v2:..."}
```

#### 2. Create New Key Version

```bash
# Rotate the encryption key (creates new version)
vault write -f transit/keys/pos-keys/rotate
# Expected: Success! Data written to: transit/keys/pos-keys/rotate

# Verify new version created
vault read transit/keys/pos-keys
# "latest_version" should now be 3 (incremented by 1)
```

#### 3. Update Services Configuration

**Option A: Rolling Restart (Zero Downtime)**

```bash
# Services automatically pick up new key version on next encryption
# No restart needed - Vault handles versioning transparently

# Test that new encryptions use new key
vault write transit/encrypt/pos-keys plaintext=$(echo "test" | base64)
# Expected: {"ciphertext":"vault:v3:..."} ← Note new version number
```

**Option B: Immediate Key Enforcement (Optional)**

```bash
# Set minimum decryption version (force migration)
vault write transit/keys/pos-keys/config min_decryption_version=3

# WARNING: Old ciphertexts (vault:v1:..., vault:v2:...) will no longer decrypt
# Only use after data migration (see step 4)
```

#### 4. Data Migration (Optional)

If you want to re-encrypt existing data with the new key:

```bash
# Connect to database
psql -U pos_user -d pos_db

# Count records with old key versions
SELECT
  COUNT(*) FILTER (WHERE email_encrypted LIKE 'vault:v1:%') AS v1_count,
  COUNT(*) FILTER (WHERE email_encrypted LIKE 'vault:v2:%') AS v2_count,
  COUNT(*) FILTER (WHERE email_encrypted LIKE 'vault:v3:%') AS v3_count
FROM users;

# Run migration script (processes 1000 records per batch)
./scripts/migrate-encryption-keys.sh --table users --field email_encrypted --batch-size 1000

# Verify migration completion
SELECT COUNT(*) FROM users WHERE email_encrypted NOT LIKE 'vault:v3:%';
# Expected: 0
```

#### 5. Post-Rotation Verification

```bash
# Test encryption with new key
NEW_CIPHERTEXT=$(vault write -field=ciphertext transit/encrypt/pos-keys plaintext=$(echo "test_data" | base64))
echo "New ciphertext: $NEW_CIPHERTEXT"
# Expected: vault:v3:...

# Test decryption (should work for all versions)
vault write -field=plaintext transit/decrypt/pos-keys ciphertext="$NEW_CIPHERTEXT" | base64 -d
# Expected: test_data

# Check application logs for encryption errors
docker logs user-service --tail 100 | grep -i "encryption error"
# Expected: No errors

# Monitor Prometheus metrics
curl http://prometheus:9090/api/v1/query?query=encryption_errors_total
# Expected: {"data":{"result":[]}}
```

#### 6. Documentation

```bash
# Record rotation in audit log
vault audit log -format=json | grep "transit/keys/pos-keys/rotate" | tail -1 > /logs/vault_rotation_$(date +%Y%m%d).json

# Update rotation tracking spreadsheet
echo "$(date +%Y-%m-%d),Key rotation completed,v3" >> /docs/vault_rotations.csv
```

### Rollback Procedure

If issues occur during rotation:

```bash
# Vault keeps all key versions - no rollback needed
# Old ciphertexts (vault:v1:, vault:v2:) remain decryptable

# To revert minimum decryption version (if set):
vault write transit/keys/pos-keys/config min_decryption_version=2

# Check services are using older version successfully
vault write transit/decrypt/pos-keys ciphertext="vault:v2:old_ciphertext"
# Should succeed
```

### Common Issues

**Issue**: Services cannot decrypt data after rotation

**Solution**:

```bash
# Check minimum decryption version
vault read transit/keys/pos-keys/config

# If too restrictive, lower it
vault write transit/keys/pos-keys/config min_decryption_version=1
```

**Issue**: Migration script fails midway

**Solution**:

```bash
# Migration is idempotent - safe to re-run
./scripts/migrate-encryption-keys.sh --table users --field email_encrypted --resume

# Check progress
SELECT email_encrypted, COUNT(*) FROM users GROUP BY LEFT(email_encrypted, 10);
```

---

## Audit Log Partition Management

**Frequency**: Monthly (automated with manual verification)  
**Duration**: ~10 minutes  
**Risk Level**: LOW (read-only verification)

### Overview

Audit events are partitioned by month for performance. The `PartitionService` automatically creates partitions 7 days before month end.

### Automated Partition Creation

The system runs `StartMonitor()` every 6 hours to check if next month's partition exists.

**Verify Automation**:

```bash
# Check partition service logs
docker logs audit-service | grep "Partition created successfully"

# List existing partitions
psql -U pos_user -d pos_db -c "\d+ audit_events*"
# Expected output:
# audit_events_2026_01 (partition for January 2026)
# audit_events_2026_02 (partition for February 2026)
# ...
```

### Manual Partition Creation

If automation fails or you need to create partitions ahead of time:

```sql
-- Connect to database
psql -U pos_user -d pos_db

-- Create partition for specific month
SELECT create_partition_for_audit_events(2026, 3); -- March 2026
-- Expected: Partition created successfully

-- Verify partition exists
\d+ audit_events_2026_03

-- Check partition boundaries
SELECT
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE tablename LIKE 'audit_events_%'
ORDER BY tablename DESC;
```

### Partition Archival (After 7 Years)

Per UU PDP Article 56, audit logs must be retained for 7 years (2555 days).

**Monthly Archival Procedure**:

```bash
# Calculate date 7 years ago
ARCHIVE_DATE=$(date -d "7 years ago" +%Y-%m)
echo "Archiving partitions older than: $ARCHIVE_DATE"

# Example: Archive January 2019 (in January 2026)
PARTITION="audit_events_2019_01"

# 1. Export partition to cold storage
pg_dump -U pos_user -d pos_db -t $PARTITION --data-only | gzip > /backups/archives/$PARTITION.sql.gz

# 2. Verify export
gunzip -c /backups/archives/$PARTITION.sql.gz | head -20
# Expected: SQL INSERT statements

# 3. Drop partition (only after export verified)
psql -U pos_user -d pos_db -c "DROP TABLE IF EXISTS $PARTITION;"

# 4. Record archival
echo "$(date +%Y-%m-%d),$PARTITION,archived" >> /docs/audit_archival.csv
```

### Partition Health Checks

**Weekly verification script**:

```bash
#!/bin/bash
# File: scripts/verify-audit-partitions.sh

# Check for missing partitions (next 3 months)
for i in {0..2}; do
  MONTH=$(date -d "+$i month" +%Y_%m)
  PARTITION="audit_events_$MONTH"

  EXISTS=$(psql -U pos_user -d pos_db -tAc "SELECT to_regclass('$PARTITION');")

  if [ "$EXISTS" == "" ]; then
    echo "⚠️  WARNING: Missing partition $PARTITION"
    # Auto-create
    YEAR=$(echo $MONTH | cut -d_ -f1)
    MONTH_NUM=$(echo $MONTH | cut -d_ -f2)
    psql -U pos_user -d pos_db -c "SELECT create_partition_for_audit_events($YEAR, $MONTH_NUM);"
  else
    echo "✅ Partition $PARTITION exists"
  fi
done

# Check partition sizes
psql -U pos_user -d pos_db -c "
SELECT
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size,
  (SELECT COUNT(*) FROM pg_tables WHERE tablename = tablename) AS row_count
FROM pg_tables
WHERE tablename LIKE 'audit_events_%'
ORDER BY tablename DESC
LIMIT 12;
"
```

**Schedule in crontab**:

```bash
# Run every Monday at 9 AM
0 9 * * 1 /opt/pos/scripts/verify-audit-partitions.sh >> /var/log/partition_checks.log 2>&1
```

### Prometheus Alerts

Monitor partition health with alerts:

```yaml
# observability/prometheus/audit_partition_alerts.yml
groups:
  - name: audit_partitions
    rules:
      - alert: MissingAuditPartition
        expr: audit_partition_exists{month_offset="1"} == 0
        for: 24h
        annotations:
          summary: 'Audit partition missing for next month'
          description: 'Partition for {{ $labels.year }}-{{ $labels.month }} does not exist'

      - alert: AuditPartitionFull
        expr: audit_partition_size_bytes > 10737418240 # 10GB
        for: 1h
        annotations:
          summary: 'Audit partition exceeding size threshold'
          description: 'Partition {{ $labels.partition }} is {{ $value | humanize }} bytes'
```

---

## Data Cleanup Job Troubleshooting

**When**: Alert `CleanupErrorsHigh` or `CleanupJobsStalled` fires  
**Duration**: 15-45 minutes  
**Risk Level**: MEDIUM (data deletion involved)

### Common Failure Scenarios

#### Scenario 1: Redis Lock Stuck

**Symptoms**:

- Cleanup jobs not running
- Logs show "Failed to acquire lock"
- `CleanupJobsStalled` alert firing

**Diagnosis**:

```bash
# Check for stuck locks
redis-cli KEYS "cleanup:lock:*"
# Example output:
# 1) "cleanup:lock:users:deleted"
# 2) "cleanup:lock:sessions:expired"

# Check lock TTL
redis-cli TTL "cleanup:lock:users:deleted"
# If shows large value (>2 hours) or -1 (no expiry), lock is stuck
```

**Resolution**:

```bash
# Option 1: Wait for TTL to expire (2 hours)
# Option 2: Manually release lock
redis-cli DEL "cleanup:lock:users:deleted"
# Expected: (integer) 1

# Verify lock released
redis-cli KEYS "cleanup:lock:*"
# Should not show the deleted key

# Trigger cleanup manually
curl -X POST http://user-service:8081/admin/cleanup/run-now
```

#### Scenario 2: Database Connection Pool Exhausted

**Symptoms**:

- Cleanup fails with timeout errors
- Logs show "no available connections"
- Other database operations slow

**Diagnosis**:

```bash
# Check active database connections
psql -U pos_user -d pos_db -c "
SELECT
  datname,
  count(*) as connections,
  max_conn.setting as max_connections
FROM pg_stat_activity,
     (SELECT setting FROM pg_settings WHERE name='max_connections') max_conn
WHERE datname = 'pos_db'
GROUP BY datname, max_conn.setting;
"

# Check connection pool metrics
curl http://user-service:8081/metrics | grep db_connections_active
```

**Resolution**:

```bash
# Increase max connections (temporarily)
psql -U postgres -c "ALTER SYSTEM SET max_connections = 200;"
psql -U postgres -c "SELECT pg_reload_conf();"

# Or restart service to reset connection pool
docker restart user-service

# Long-term: Tune connection pool settings in .env
MAX_DB_CONNECTIONS=50
DB_POOL_MAX_IDLE=10
DB_POOL_MAX_LIFETIME=1h
```

#### Scenario 3: Batch Processing Too Slow

**Symptoms**:

- Cleanup duration exceeding 2 hours (SLA)
- `CleanupDurationHigh` alert firing
- Large number of expired records

**Diagnosis**:

```bash
# Check expired record counts
curl http://user-service:8081/admin/retention-policies/users_deleted/expired-count
# Example: {"expired_count": 50000}

# Check cleanup duration metric
curl http://prometheus:9090/api/v1/query?query=cleanup_duration_seconds{table="users"}
```

**Resolution**:

```bash
# Option 1: Increase batch size (edit code)
# File: backend/user-service/src/jobs/cleanup_orchestrator.go
# Change: batchSize := 100 → batchSize := 500

# Option 2: Run cleanup multiple times per day
# Edit: backend/user-service/src/scheduler/cleanup_scheduler.go
# Change: targetHour := 2 → Multiple scheduled times

# Option 3: Manually run cleanup during low-traffic period
curl -X POST http://user-service:8081/admin/cleanup/run-now

# Monitor progress
watch -n 30 'curl -s http://user-service:8081/admin/retention-policies/users_deleted/expired-count'
```

#### Scenario 4: Notification Job Failing

**Symptoms**:

- Users not receiving deletion warning emails
- `DeletionNotificationJob` errors in logs

**Diagnosis**:

```bash
# Check notification job logs
docker logs user-service | grep "DeletionNotificationJob"

# Check flagged users (should be notified 30 days before deletion)
psql -U pos_user -d pos_db -c "
SELECT
  id,
  email_encrypted,
  deleted_at,
  notified_of_deletion,
  deleted_at + INTERVAL '90 days' AS permanent_deletion_date
FROM users
WHERE deleted_at IS NOT NULL
  AND deleted_at + INTERVAL '90 days' - INTERVAL '30 days' <= NOW()
  AND notified_of_deletion = FALSE
ORDER BY deleted_at;
"
```

**Resolution**:

```bash
# Option 1: Check SMTP configuration
docker exec -it notification-service env | grep SMTP
# Verify SMTP_HOST, SMTP_USERNAME, SMTP_PASSWORD set correctly

# Option 2: Test email sending
curl -X POST http://notification-service:8083/api/v1/notifications/test \
  -H "Content-Type: application/json" \
  -d '{"recipient": "test@example.com", "subject": "Test", "body": "Test email"}'

# Option 3: Manually mark as notified (if email confirmed sent)
psql -U pos_user -d pos_db -c "
UPDATE users SET notified_of_deletion = TRUE
WHERE id = 'user-uuid-here';
"
```

### Manual Cleanup Trigger

If automated cleanup fails, trigger manually:

```bash
# Run all cleanup jobs
curl -X POST http://user-service:8081/admin/cleanup/run-now \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Run specific cleanup job
curl -X POST http://user-service:8081/admin/cleanup/run-now?table=users&type=deleted \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Monitor progress via logs
docker logs -f user-service | grep "CleanupOrchestrator"
```

### Cleanup Job Metrics

Monitor cleanup health with Prometheus:

```promql
# Records processed per table
cleanup_records_processed_total{table="users", record_type="deleted"}

# Cleanup duration by table
cleanup_duration_seconds{table="sessions"}

# Cleanup errors
rate(cleanup_errors_total[1h])

# Last successful run timestamp
time() - cleanup_last_run_timestamp_seconds < 172800  # 48 hours
```

---

## Data Breach Response

**Trigger**: Security incident detected (unauthorized access, data leak, etc.)  
**Duration**: Varies (hours to days)  
**Risk Level**: CRITICAL

### Immediate Response (0-1 Hour)

#### 1. Contain the Breach

```bash
# STEP 1: Isolate affected services
docker stop user-service auth-service

# STEP 2: Block suspicious IP addresses (if identified)
iptables -A INPUT -s <suspicious_ip> -j DROP

# STEP 3: Revoke all active sessions
psql -U pos_user -d pos_db -c "UPDATE user_sessions SET expires_at = NOW();"

# STEP 4: Rotate Vault encryption keys immediately
vault write -f transit/keys/pos-keys/rotate
vault write transit/keys/pos-keys/config min_decryption_version=<new_version>

# STEP 5: Change all database passwords
psql -U postgres -c "ALTER USER pos_user WITH PASSWORD '<new_password>';"
# Update .env files with new password

# STEP 6: Disable API access temporarily
# Edit: api-gateway/middleware/rate_limit.go
# Set global rate limit to 0 or add maintenance mode
```

#### 2. Assess Impact

```bash
# Identify affected users/tenants
psql -U pos_user -d pos_db -c "
SELECT
  tenant_id,
  COUNT(DISTINCT user_id) AS affected_users,
  MIN(timestamp) AS first_suspicious_activity,
  MAX(timestamp) AS last_suspicious_activity
FROM audit_events
WHERE ip_address = '<suspicious_ip>'
  OR (user_agent LIKE '%suspicious_pattern%')
GROUP BY tenant_id;
"

# Check for unauthorized data access
psql -U pos_user -d pos_db -c "
SELECT * FROM audit_events
WHERE event_type IN ('USER_ACCESSED', 'DATA_EXPORTED')
  AND timestamp > '<breach_start_time>'
  AND actor_id NOT IN (SELECT id FROM users WHERE is_admin = TRUE)
ORDER BY timestamp;
"

# Export audit trail for forensics
pg_dump -U pos_user -d pos_db -t audit_events \
  --data-only --where="timestamp >= '<breach_start_time>'" \
  | gzip > /forensics/audit_trail_$(date +%Y%m%d_%H%M%S).sql.gz
```

### Investigation Phase (1-24 Hours)

#### 3. Forensic Analysis

```bash
# Analyze access patterns
psql -U pos_user -d pos_db -c "
SELECT
  actor_email,
  COUNT(*) AS action_count,
  array_agg(DISTINCT event_type) AS event_types,
  MIN(timestamp) AS first_seen,
  MAX(timestamp) AS last_seen
FROM audit_events
WHERE timestamp > '<breach_start_time>'
GROUP BY actor_email
ORDER BY action_count DESC
LIMIT 50;
"

# Check for privilege escalation
psql -U pos_user -d pos_db -c "
SELECT * FROM audit_events
WHERE event_type = 'ROLE_CHANGED'
  AND timestamp > '<breach_start_time>'
  AND after_value LIKE '%OWNER%';
"

# Review application logs for anomalies
docker logs user-service --since $(date -d "1 day ago" +%Y-%m-%dT%H:%M:%S) \
  | grep -E "ERROR|FATAL|unauthorized|failed login" \
  > /forensics/user_service_errors.log
```

#### 4. Determine Data Exposure

```bash
# List potentially exposed PII
psql -U pos_user -d pos_db -c "
SELECT
  'users' AS table_name,
  COUNT(*) AS record_count
FROM users
WHERE id IN (
  SELECT resource_id FROM audit_events
  WHERE event_type = 'USER_ACCESSED'
    AND timestamp > '<breach_start_time>'
)
UNION ALL
SELECT
  'guest_orders' AS table_name,
  COUNT(*) AS record_count
FROM guest_orders
WHERE id IN (
  SELECT resource_id FROM audit_events
  WHERE event_type = 'ORDER_ACCESSED'
    AND timestamp > '<breach_start_time>'
);
"

# Check if encryption keys were accessed
docker logs vault --since $(date -d "1 day ago" +%Y-%m-%dT%H:%M:%S) \
  | grep "transit/keys/pos-keys"
```

### Notification Phase (24-72 Hours)

#### 5. Regulatory Notification

**Legal Requirements** (UU PDP Article 57):

- Notify Indonesian Data Protection Authority (Kominfo) within **72 hours**
- Include: nature of breach, affected data categories, number of affected individuals, mitigation steps

**Notification Template**:

```
To: ditjen.apt@kominfo.go.id
Subject: Personal Data Breach Notification - [Company Name]

Dear Sir/Madam,

We are writing to notify you of a personal data breach that occurred on [DATE].

**Breach Details:**
- Date/Time Detected: [TIMESTAMP]
- Breach Type: [Unauthorized Access / Data Leak / Ransomware / etc.]
- Attack Vector: [SQL Injection / Phishing / Malware / etc.]

**Affected Data:**
- Number of Individuals: [COUNT]
- Data Categories: [Email, Names, Phone Numbers, etc.]
- Encryption Status: [Encrypted with AES-256-GCM / Partially Encrypted / etc.]

**Mitigation Steps Taken:**
- [List steps from containment phase]
- [Encryption keys rotated]
- [Affected users notified]

**Contact Person:**
Name: [Security Officer]
Email: [security@company.com]
Phone: [+62-xxx-xxxx-xxxx]

Sincerely,
[Company Name]
[Registration Number]
```

#### 6. User Notification

**Email Template** (bilingual Indonesian + English):

```html
<!-- File: backend/notification-service/templates/breach_notification.html -->
<html>
  <body>
    <div style="max-width: 600px; margin: 0 auto; font-family: Arial, sans-serif;">
      <h2 style="color: #d32f2f;">🔒 Important Security Notice / Pemberitahuan Keamanan Penting</h2>

      <h3>English</h3>
      <p>
        We are writing to inform you that we recently detected unauthorized access to our systems on
        {{.breach_date}}.
      </p>

      <p>
        <strong>What Happened:</strong><br />
        {{.breach_description}}
      </p>

      <p>
        <strong>What Information Was Involved:</strong><br />
        {{.affected_data}}
      </p>

      <p>
        <strong>What We Are Doing:</strong><br />
        - Implemented additional security measures<br />
        - Rotated all encryption keys<br />
        - Notified relevant authorities<br />
        - Conducting forensic investigation
      </p>

      <p>
        <strong>What You Should Do:</strong><br />
        - Change your password immediately<br />
        - Enable two-factor authentication (coming soon)<br />
        - Monitor your account for suspicious activity<br />
        - Contact us if you notice anything unusual
      </p>

      <hr />

      <h3>Bahasa Indonesia</h3>
      <p>
        Kami menginformasikan bahwa kami baru-baru ini mendeteksi akses tidak sah ke sistem kami
        pada {{.breach_date}}.
      </p>

      <p>
        <strong>Apa yang Terjadi:</strong><br />
        {{.breach_description_id}}
      </p>

      <p>
        <strong>Informasi Apa yang Terlibat:</strong><br />
        {{.affected_data_id}}
      </p>

      <p>
        <strong>Apa yang Kami Lakukan:</strong><br />
        - Menerapkan langkah keamanan tambahan<br />
        - Merotasi semua kunci enkripsi<br />
        - Memberitahu otoritas terkait<br />
        - Melakukan investigasi forensik
      </p>

      <p>
        <strong>Apa yang Harus Anda Lakukan:</strong><br />
        - Ubah kata sandi Anda segera<br />
        - Aktifkan autentikasi dua faktor (segera hadir)<br />
        - Pantau akun Anda untuk aktivitas mencurigakan<br />
        - Hubungi kami jika Anda melihat sesuatu yang tidak biasa
      </p>

      <p style="margin-top: 30px; color: #666; font-size: 12px;">
        If you have questions, contact: security@company.com<br />
        Jika Anda memiliki pertanyaan, hubungi: security@company.com
      </p>
    </div>
  </body>
</html>
```

**Send Notifications**:

```bash
# Get affected user emails
psql -U pos_user -d pos_db -t -c "
SELECT email_encrypted FROM users WHERE id IN (
  SELECT DISTINCT resource_id FROM audit_events
  WHERE timestamp > '<breach_start_time>'
);
" > /tmp/affected_users_encrypted.txt

# Decrypt emails (using script)
while read encrypted_email; do
  psql -U pos_user -d pos_db -c "SELECT decrypt_field('$encrypted_email');"
done < /tmp/affected_users_encrypted.txt > /tmp/affected_users.txt

# Send breach notifications
cat /tmp/affected_users.txt | while read email; do
  curl -X POST http://notification-service:8083/api/v1/notifications/breach \
    -H "Content-Type: application/json" \
    -d "{
      \"recipient\": \"$email\",
      \"breach_date\": \"2026-01-15\",
      \"breach_description\": \"Unauthorized access detected\",
      \"affected_data\": \"Email addresses, names\"
    }"
done
```

### Recovery Phase (72+ Hours)

#### 7. System Hardening

```bash
# Enable additional security features
# 1. Implement IP whitelisting
# 2. Add rate limiting (stricter)
# 3. Enable audit logging for all API calls
# 4. Implement anomaly detection

# 2FA Implementation (if not already done)
# See: docs/2FA_IMPLEMENTATION_GUIDE.md

# Security audit
./scripts/security-audit.sh > /reports/security_audit_post_breach.txt
```

#### 8. Post-Incident Review

**Document Lessons Learned**:

```markdown
# Breach Post-Mortem - [DATE]

## Timeline

- [TIME]: Breach detected
- [TIME]: Services isolated
- [TIME]: Keys rotated
- [TIME]: Authority notified
- [TIME]: Users notified

## Root Cause

[Detailed analysis of how breach occurred]

## Impact

- Affected Users: [COUNT]
- Data Exposed: [DESCRIPTION]
- Duration: [TIMESPAN]

## What Went Well

- [Response steps that worked]

## What Went Wrong

- [Issues encountered during response]

## Action Items

- [ ] Implement [security improvement 1]
- [ ] Update [procedure/documentation]
- [ ] Train team on [topic]
```

---

## Database Migration Procedures

**When**: Deploying schema changes  
**Duration**: 5-30 minutes  
**Risk Level**: MEDIUM-HIGH (depends on migration)

### Pre-Migration Checklist

- [ ] Migration tested in staging environment
- [ ] Database backup completed
- [ ] Rollback script prepared
- [ ] Maintenance window scheduled (if downtime required)
- [ ] Team notified

### Migration Execution

```bash
# 1. Backup database
pg_dump -U pos_user -d pos_db --clean --if-exists | gzip > /backups/pre_migration_$(date +%Y%m%d_%H%M%S).sql.gz

# 2. Verify backup
gunzip -c /backups/pre_migration_*.sql.gz | head -50

# 3. Apply migration (using migrate tool)
cd backend/migrations
migrate -path . -database "postgresql://pos_user:password@localhost:5432/pos_db?sslmode=disable" up

# 4. Verify migration applied
psql -U pos_user -d pos_db -c "SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 5;"

# 5. Test application
curl http://localhost:8080/health
docker logs user-service --tail 50

# 6. Monitor for errors
docker logs -f user-service | grep -i error
```

### Rollback Procedure

If migration fails:

```bash
# 1. Stop services
docker-compose down

# 2. Restore database from backup
gunzip -c /backups/pre_migration_*.sql.gz | psql -U pos_user -d pos_db

# 3. Verify restoration
psql -U pos_user -d pos_db -c "\dt"

# 4. Restart services
docker-compose up -d

# 5. Verify health
curl http://localhost:8080/health
```

---

## Service Health Checks

**Frequency**: Continuous (automated monitoring)  
**Duration**: 2-5 minutes  
**Risk Level**: LOW

### Quick Health Check Script

```bash
#!/bin/bash
# File: scripts/health-check.sh

echo "=== POS System Health Check ==="
echo ""

# API Gateway
echo "1. API Gateway:"
GATEWAY_HEALTH=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health)
if [ "$GATEWAY_HEALTH" == "200" ]; then
  echo "   ✅ Healthy (HTTP $GATEWAY_HEALTH)"
else
  echo "   ❌ Unhealthy (HTTP $GATEWAY_HEALTH)"
fi

# User Service
echo "2. User Service:"
USER_HEALTH=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/health)
if [ "$USER_HEALTH" == "200" ]; then
  echo "   ✅ Healthy (HTTP $USER_HEALTH)"
else
  echo "   ❌ Unhealthy (HTTP $USER_HEALTH)"
fi

# Database
echo "3. Database:"
DB_HEALTH=$(psql -U pos_user -d pos_db -tAc "SELECT 1;" 2>/dev/null)
if [ "$DB_HEALTH" == "1" ]; then
  echo "   ✅ Healthy (Connected)"
else
  echo "   ❌ Unhealthy (Cannot connect)"
fi

# Vault
echo "4. Vault:"
VAULT_HEALTH=$(curl -s http://localhost:8200/v1/sys/health | jq -r '.initialized')
if [ "$VAULT_HEALTH" == "true" ]; then
  echo "   ✅ Healthy (Initialized and unsealed)"
else
  echo "   ❌ Unhealthy (Not initialized or sealed)"
fi

# Redis
echo "5. Redis:"
REDIS_HEALTH=$(redis-cli ping 2>/dev/null)
if [ "$REDIS_HEALTH" == "PONG" ]; then
  echo "   ✅ Healthy (Responding)"
else
  echo "   ❌ Unhealthy (Not responding)"
fi

# Kafka
echo "6. Kafka:"
KAFKA_HEALTH=$(kafka-topics --bootstrap-server localhost:9092 --list 2>/dev/null | wc -l)
if [ "$KAFKA_HEALTH" -gt 0 ]; then
  echo "   ✅ Healthy ($KAFKA_HEALTH topics)"
else
  echo "   ❌ Unhealthy (Cannot list topics)"
fi

echo ""
echo "=== End of Health Check ==="
```

---

## Emergency Procedures

### System-Wide Outage

```bash
# 1. Check all services
docker ps -a

# 2. Restart all services
docker-compose restart

# 3. If restart fails, full reset
docker-compose down
docker-compose up -d

# 4. Monitor logs
docker-compose logs -f
```

### Database Corruption

```bash
# 1. Stop all services immediately
docker-compose down

# 2. Restore from most recent backup
gunzip -c /backups/latest_backup.sql.gz | psql -U pos_user -d pos_db

# 3. Verify restoration
psql -U pos_user -d pos_db -c "SELECT COUNT(*) FROM users;"

# 4. Restart services
docker-compose up -d
```

### Vault Sealed

```bash
# Check Vault status
vault status

# Unseal Vault (requires 3 of 5 key shares)
vault operator unseal <key_share_1>
vault operator unseal <key_share_2>
vault operator unseal <key_share_3>

# Verify unsealed
vault status
# Expected: Sealed: false
```

---

## Contact Information

**On-Call Engineer**: [phone/email]  
**DevOps Lead**: [phone/email]  
**Security Officer**: security@company.com  
**Escalation**: [manager phone/email]

---

_Last Updated: January 16, 2026_  
_Next Review: April 16, 2026_
