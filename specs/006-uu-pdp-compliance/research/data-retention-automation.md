# Research: Automated Data Retention and Cleanup Implementation

**Feature**: Indonesian Data Protection Compliance (UU PDP No.27 Tahun 2022)  
**Research Focus**: Scheduled data cleanup, retention policies, soft delete grace periods  
**Date**: January 2, 2026  
**Researcher**: GitHub Copilot

---

## Decision: time.Ticker with Distributed Locking and Table-Driven Retention Policies

### Job Scheduling Method
**Primary**: Go `time.Ticker` for background goroutines  
**Distributed Locking**: Redis SETNX with TTL for multi-instance coordination  
**Fallback**: Kubernetes CronJob for critical cleanup tasks (hard delete finalization)

### Retention Policy Modeling
**Primary**: Table-driven configuration (`retention_policies` table)  
**Fallback**: Code constants for immutable core policies (e.g., 90-day grace period)

### Cleanup Strategy
**Soft Delete**: `deleted_at` timestamp with 90-day grace period  
**Hard Delete**: Batch deletion with `LIMIT` + `scheduled_deletion_at` column  
**Notifications**: 30-day warning via notification queue

---

## Rationale

### 1. Scheduled Job Pattern: time.Ticker + Distributed Locking

#### Why time.Ticker?

**Advantages**:
- ✅ **Native Go solution** - no external dependencies (robfig/cron adds 12KB, time.Ticker is stdlib)
- ✅ **Already proven in codebase** - existing pattern in `reservation_cleanup_job.go`, `retry_worker.go`, `retry_queue.go`
- ✅ **Context-aware** - integrates with context cancellation for graceful shutdown
- ✅ **Simple to test** - mock time.Ticker in unit tests
- ✅ **Low overhead** - single goroutine per cleanup job

**Codebase Evidence**:
```go
// From backend/order-service/src/services/reservation_cleanup_job.go
func (j *ReservationCleanupJob) Start(ctx context.Context) {
    log.Info().Msg("Starting reservation cleanup job")
    ticker := time.NewTicker(j.interval)  // 1 minute interval
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            j.cleanupExpiredReservations(ctx)
        case <-j.stopChan:
            return
        case <-ctx.Done():
            return
        }
    }
}
```

**Why NOT robfig/cron?**
- ❌ Not currently used in codebase (zero imports of `robfig/cron`)
- ❌ Adds external dependency for marginal benefit (cron syntax vs time.Duration)
- ❌ Overkill for simple periodic tasks (cron best for complex schedules like "every weekday at 3am")
- ❌ Less control over goroutine lifecycle

**Why NOT Kubernetes CronJob?**
- ❌ Requires K8s infrastructure (not always available in dev/staging)
- ❌ Slower startup overhead (spin up pod, run job, tear down)
- ❌ Less flexible for high-frequency jobs (reservation cleanup every 1 minute)
- ✅ **BUT**: Good for critical infrequent tasks (hard delete finalization once daily)

**Hybrid Approach (Recommended)**:
```
High-frequency (every 1-5 minutes):
- Soft delete marking: time.Ticker goroutine
- Expired reservation cleanup: time.Ticker goroutine
- Notification warnings: time.Ticker goroutine

Low-frequency critical (daily/weekly):
- Hard delete finalization: Kubernetes CronJob (if available) OR systemd timer
- Retention policy audit: Kubernetes CronJob
- Storage quota recalculation: Kubernetes CronJob
```

#### Distributed Locking for Multi-Instance Safety

**Problem**: Multiple service instances = race conditions

**Solution**: Redis SETNX (SET if Not eXists) with TTL

```go
// Acquire distributed lock before cleanup
func (s *CleanupService) acquireLock(ctx context.Context, lockKey string, ttl time.Duration) (bool, error) {
    result, err := s.redis.SetNX(ctx, lockKey, s.instanceID, ttl).Result()
    if err != nil {
        return false, fmt.Errorf("failed to acquire lock: %w", err)
    }
    return result, nil  // true if lock acquired, false if already held
}

// Example usage in cleanup job
func (s *CleanupService) runCleanup(ctx context.Context) {
    lockKey := "cleanup:soft_delete:lock"
    lockTTL := 5 * time.Minute  // Longer than job execution time
    
    acquired, err := s.acquireLock(ctx, lockKey, lockTTL)
    if err != nil {
        log.Error().Err(err).Msg("Failed to acquire lock")
        return
    }
    if !acquired {
        log.Debug().Msg("Lock already held by another instance, skipping")
        return
    }
    defer s.releaseLock(ctx, lockKey)
    
    // Proceed with cleanup...
}
```

**Why Redis SETNX?**
- ✅ **Already in codebase** - Redis used for sessions, cart, inventory cache
- ✅ **Atomic operation** - no race conditions between instances
- ✅ **Auto-expiry** - TTL prevents deadlock if instance crashes mid-cleanup
- ✅ **Simple API** - single command, no coordination protocol

**Alternative: Database Advisory Locks (Rejected)**:
- ❌ Requires persistent connection (resource waste if not actively cleaning)
- ❌ No auto-expiry (must manually release, risk of deadlock)
- ❌ PostgreSQL-specific (`pg_try_advisory_lock`)

**Idempotency Strategy**:
```go
// Mark records as "cleanup_attempted" to prevent duplicate processing
UPDATE users
SET cleanup_attempted_at = NOW()
WHERE status = 'deleted'
  AND deleted_at < NOW() - INTERVAL '90 days'
  AND cleanup_attempted_at IS NULL
  AND scheduled_deletion_at IS NULL
RETURNING id;

// Process returned IDs...
// If job crashes, cleanup_attempted_at persists, but scheduled_deletion_at=NULL
// Next run picks up where it left off
```

---

### 2. Retention Policy Modeling: Table-Driven Config

#### Schema Design

```sql
CREATE TABLE retention_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Policy identification
    policy_name VARCHAR(100) UNIQUE NOT NULL,  -- 'user_accounts', 'guest_orders', 'notifications'
    description TEXT,
    
    -- Retention rules
    retention_days INTEGER NOT NULL,           -- Days to keep after soft delete (e.g., 90)
    notification_days INTEGER,                 -- Days before hard delete to warn user (e.g., 30)
    enabled BOOLEAN NOT NULL DEFAULT true,
    
    -- Scope
    applies_to VARCHAR(50) NOT NULL,           -- 'users', 'guest_orders', 'consent_records', etc.
    tenant_specific BOOLEAN DEFAULT false,     -- Can tenants customize this policy?
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tenant-specific overrides (optional, for future extensibility)
CREATE TABLE tenant_retention_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    policy_id UUID NOT NULL REFERENCES retention_policies(id) ON DELETE CASCADE,
    
    -- Override values
    retention_days INTEGER,        -- NULL = use global policy
    notification_days INTEGER,     -- NULL = use global policy
    enabled BOOLEAN NOT NULL DEFAULT true,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT unique_tenant_policy UNIQUE(tenant_id, policy_id)
);
```

#### Seed Data

```sql
INSERT INTO retention_policies (policy_name, description, retention_days, notification_days, applies_to, tenant_specific)
VALUES
    ('user_accounts', 'User account data after account deletion', 90, 30, 'users', false),
    ('guest_orders', 'Guest order data after order completion/cancellation', 90, NULL, 'guest_orders', false),
    ('sessions', 'Historical session records for audit', 90, NULL, 'sessions', false),
    ('notifications', 'Notification history', 90, NULL, 'notifications', false),
    ('consent_records', 'Consent history after revocation', 365, NULL, 'consent_records', false),  -- Longer retention for legal compliance
    ('password_reset_tokens', 'Expired password reset tokens', 7, NULL, 'password_reset_tokens', false),
    ('invitations', 'Expired or accepted invitations', 30, NULL, 'invitations', false);
```

**Why Table-Driven?**
- ✅ **Runtime configuration** - no code deploy to change retention period
- ✅ **Audit trail** - `updated_at` tracks when policy changed
- ✅ **Per-tenant flexibility** - future feature: let premium tenants customize retention
- ✅ **Single source of truth** - no duplication between code and docs
- ✅ **Database-driven** - can query "which tables have 90-day retention?" easily

**Why NOT Code Constants?**
- ❌ Requires code deploy to change retention period
- ❌ Harder to audit changes (git history vs database audit log)
- ❌ No per-tenant customization
- ✅ **BUT**: Good for immutable legal requirements (e.g., "minimum 90 days")

**Hybrid Approach (Recommended)**:
```go
// Code constants for immutable legal minimums
const (
    MinRetentionDaysUUPDP = 90  // UU PDP requires minimum 90 days
)

// Query database for actual policy
func (s *CleanupService) getRetentionPolicy(policyName string) (*RetentionPolicy, error) {
    policy := &RetentionPolicy{}
    err := s.db.QueryRow(`
        SELECT retention_days, notification_days, enabled
        FROM retention_policies
        WHERE policy_name = $1 AND enabled = true
    `, policyName).Scan(&policy.RetentionDays, &policy.NotificationDays, &policy.Enabled)
    
    // Enforce legal minimum
    if policy.RetentionDays < MinRetentionDaysUUPDP {
        return nil, fmt.Errorf("retention_days (%d) below legal minimum (%d)", 
            policy.RetentionDays, MinRetentionDaysUUPDP)
    }
    
    return policy, err
}
```

---

### 3. Soft Delete Grace Period: 90-Day Pattern

#### Schema Changes

Add to all tables with soft delete:

```sql
-- Example: users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS scheduled_deletion_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS deletion_notified_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS cleanup_attempted_at TIMESTAMPTZ;

CREATE INDEX idx_users_scheduled_deletion ON users(scheduled_deletion_at) WHERE scheduled_deletion_at IS NOT NULL;
CREATE INDEX idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;
```

**Column Semantics**:
- `deleted_at`: When user initiated deletion (soft delete trigger)
- `scheduled_deletion_at`: When hard delete will occur (calculated: `deleted_at + 90 days`)
- `deletion_notified_at`: When we sent "30 days until deletion" warning
- `cleanup_attempted_at`: Idempotency marker (prevents duplicate processing)

#### Workflow

**Stage 1: User Deletes Account**
```go
func (s *UserService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
    // Soft delete
    query := `
        UPDATE users
        SET deleted_at = NOW(),
            scheduled_deletion_at = NOW() + INTERVAL '90 days',
            status = 'deleted'
        WHERE id = $1
        RETURNING deleted_at, scheduled_deletion_at
    `
    var deletedAt, scheduledDeletionAt time.Time
    err := s.db.QueryRowContext(ctx, query, userID).Scan(&deletedAt, &scheduledDeletionAt)
    
    log.Info().
        Str("user_id", userID.String()).
        Time("deleted_at", deletedAt).
        Time("scheduled_deletion_at", scheduledDeletionAt).
        Msg("User account soft deleted")
    
    return err
}
```

**Stage 2: Notification Job (runs daily)**
```go
func (s *CleanupService) sendDeletionWarnings(ctx context.Context) error {
    // Find users 60 days into grace period (30 days remaining)
    query := `
        UPDATE users
        SET deletion_notified_at = NOW()
        WHERE status = 'deleted'
          AND deleted_at IS NOT NULL
          AND scheduled_deletion_at > NOW()  -- Grace period not expired yet
          AND scheduled_deletion_at <= NOW() + INTERVAL '30 days'  -- 30 days or less remaining
          AND deletion_notified_at IS NULL  -- Not already notified
        RETURNING id, email, name, scheduled_deletion_at
    `
    
    rows, err := s.db.QueryContext(ctx, query)
    if err != nil {
        return fmt.Errorf("failed to query users for deletion warnings: %w", err)
    }
    defer rows.Close()
    
    count := 0
    for rows.Next() {
        var user struct {
            ID                    uuid.UUID
            Email                 string
            Name                  string
            ScheduledDeletionAt   time.Time
        }
        
        if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.ScheduledDeletionAt); err != nil {
            log.Error().Err(err).Msg("Failed to scan user")
            continue
        }
        
        // Publish notification event
        event := &NotificationEvent{
            EventType: "user.deletion_warning",
            TenantID:  "", // Fetch from user record if needed
            UserID:    user.ID.String(),
            Data: map[string]interface{}{
                "email":                 user.Email,
                "name":                  user.Name,
                "scheduled_deletion_at": user.ScheduledDeletionAt.Format(time.RFC3339),
                "days_remaining":        int(time.Until(user.ScheduledDeletionAt).Hours() / 24),
            },
        }
        
        if err := s.eventPublisher.Publish(ctx, event); err != nil {
            log.Error().Err(err).Str("user_id", user.ID.String()).Msg("Failed to publish deletion warning event")
            // Continue processing other users
        } else {
            count++
        }
    }
    
    log.Info().Int("count", count).Msg("Deletion warnings sent")
    return nil
}
```

**Stage 3: Hard Delete Job (runs daily)**
```go
func (s *CleanupService) executeHardDeletes(ctx context.Context) error {
    // Acquire distributed lock
    lockKey := "cleanup:hard_delete:lock"
    acquired, err := s.acquireLock(ctx, lockKey, 10*time.Minute)
    if err != nil || !acquired {
        return err
    }
    defer s.releaseLock(ctx, lockKey)
    
    // Batch processing (avoid long-running transactions)
    batchSize := 100
    totalDeleted := 0
    
    for {
        // Find users ready for hard delete
        query := `
            SELECT id, email, tenant_id
            FROM users
            WHERE status = 'deleted'
              AND scheduled_deletion_at <= NOW()
              AND cleanup_attempted_at IS NULL
            LIMIT $1
        `
        
        rows, err := s.db.QueryContext(ctx, query, batchSize)
        if err != nil {
            return fmt.Errorf("failed to query users for hard delete: %w", err)
        }
        
        var userIDs []uuid.UUID
        userMap := make(map[uuid.UUID]struct {
            Email    string
            TenantID uuid.UUID
        })
        
        for rows.Next() {
            var id, tenantID uuid.UUID
            var email string
            if err := rows.Scan(&id, &email, &tenantID); err != nil {
                log.Error().Err(err).Msg("Failed to scan user")
                continue
            }
            userIDs = append(userIDs, id)
            userMap[id] = struct {
                Email    string
                TenantID uuid.UUID
            }{email, tenantID}
        }
        rows.Close()
        
        if len(userIDs) == 0 {
            break  // No more users to delete
        }
        
        // Mark as cleanup attempted (idempotency)
        _, err = s.db.ExecContext(ctx, `
            UPDATE users
            SET cleanup_attempted_at = NOW()
            WHERE id = ANY($1)
        `, pq.Array(userIDs))
        if err != nil {
            log.Error().Err(err).Msg("Failed to mark cleanup attempted")
            // Continue anyway - will retry next run
        }
        
        // Hard delete in transaction
        for _, userID := range userIDs {
            if err := s.hardDeleteUser(ctx, userID); err != nil {
                log.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to hard delete user")
                // Continue processing other users
            } else {
                totalDeleted++
                
                // Log audit event
                s.auditLogger.LogDeletion(ctx, "users", userID.String(), userMap[userID].TenantID.String())
            }
        }
        
        // Rate limiting (avoid overloading database)
        time.Sleep(1 * time.Second)
    }
    
    log.Info().Int("total_deleted", totalDeleted).Msg("Hard delete job completed")
    return nil
}

func (s *CleanupService) hardDeleteUser(ctx context.Context, userID uuid.UUID) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    // Delete in order of foreign key dependencies
    tables := []string{
        "consent_records WHERE user_id = $1",
        "sessions WHERE user_id = $1",
        "password_reset_tokens WHERE user_id = $1",
        "notifications WHERE user_id = $1",  // Optional: keep for audit
        "users WHERE id = $1",
    }
    
    for _, table := range tables {
        query := fmt.Sprintf("DELETE FROM %s", table)
        result, err := tx.ExecContext(ctx, query, userID)
        if err != nil {
            return fmt.Errorf("failed to delete from %s: %w", table, err)
        }
        rows, _ := result.RowsAffected()
        log.Debug().Str("table", table).Int64("rows", rows).Msg("Deleted records")
    }
    
    return tx.Commit()
}
```

**Why 90-Day Grace Period?**
- ✅ **UU PDP compliance** - allows data subject to change mind
- ✅ **Business continuity** - prevents accidental deletion impact
- ✅ **Audit trail** - sufficient time for investigations/disputes
- ✅ **Industry standard** - GDPR often uses 30-90 days

**Why NOT Immediate Hard Delete?**
- ❌ No recovery if deletion was accidental
- ❌ Violates GDPR "right to rectification" (user can't undo)
- ❌ No time for business to react (e.g., open invoices, legal holds)

---

### 4. Cleanup Operations: Batch Deletion with LIMIT

#### Why Batch Processing?

**Problem**: Deleting 100,000 users in single transaction = table lock, replication lag, transaction log bloat

**Solution**: Batch of 100 users per iteration, 1-second pause between batches

```go
func (s *CleanupService) cleanupExpiredRecords(ctx context.Context, tableName string, retentionDays int) error {
    batchSize := 100
    totalDeleted := 0
    
    for {
        // Use LIMIT for batch processing
        query := fmt.Sprintf(`
            DELETE FROM %s
            WHERE id IN (
                SELECT id
                FROM %s
                WHERE created_at < NOW() - INTERVAL '%d days'
                LIMIT $1
            )
        `, tableName, tableName, retentionDays)
        
        result, err := s.db.ExecContext(ctx, query, batchSize)
        if err != nil {
            return fmt.Errorf("failed to delete expired records: %w", err)
        }
        
        rows, _ := result.RowsAffected()
        if rows == 0 {
            break  // No more records to delete
        }
        
        totalDeleted += int(rows)
        
        // Rate limiting (avoid overloading database)
        time.Sleep(1 * time.Second)
        
        // Check for context cancellation
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
    }
    
    log.Info().
        Str("table", tableName).
        Int("total_deleted", totalDeleted).
        Msg("Cleanup completed")
    
    return nil
}
```

**Why NOT Delete All at Once?**
- ❌ Long-running transaction blocks concurrent writes
- ❌ Replication lag spikes (millions of WAL records)
- ❌ VACUUM overhead (bloated dead tuple space)
- ❌ No graceful cancellation (can't stop mid-delete)

**Alternative: PostgreSQL Partitioning (Considered)**

For tables with high delete volume (e.g., `notifications`, `sessions`):

```sql
-- Create partitioned table
CREATE TABLE notifications_partitioned (
    LIKE notifications INCLUDING ALL
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE notifications_2026_01 PARTITION OF notifications_partitioned
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE notifications_2026_02 PARTITION OF notifications_partitioned
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');

-- Cleanup = instant partition drop (no DELETE overhead)
DROP TABLE IF EXISTS notifications_2025_10;  -- >90 days old
```

**Partitioning Trade-offs**:
- ✅ **Instant cleanup** - `DROP TABLE` faster than `DELETE` (no VACUUM needed)
- ✅ **Better query performance** - partition pruning for date-range queries
- ✅ **Simpler cleanup code** - no batch logic, just drop partition
- ❌ **Schema complexity** - requires migration to partitioned table
- ❌ **Partition management** - need automation to create future partitions
- ❌ **Not suitable for small tables** - overhead not worth it for <100K rows/month

**Decision**: Use batch deletion for now, consider partitioning if table growth exceeds 1M rows/month

---

### 5. User Notifications: Notification Queue + Scheduled Email

#### Notification Trigger

```sql
-- Migration: Add notification scheduling
CREATE TABLE scheduled_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Notification details
    notification_type VARCHAR(50) NOT NULL,  -- 'deletion_warning', 'deletion_reminder'
    scheduled_for TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    error_msg TEXT,
    
    -- Metadata
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT valid_send_state CHECK (
        (sent_at IS NULL AND failed_at IS NULL) OR
        (sent_at IS NOT NULL AND failed_at IS NULL) OR
        (sent_at IS NULL AND failed_at IS NOT NULL)
    )
);

CREATE INDEX idx_scheduled_notifications_scheduled_for ON scheduled_notifications(scheduled_for) WHERE sent_at IS NULL;
```

#### Notification Scheduler Job

```go
func (s *CleanupService) scheduleNotifications(ctx context.Context) error {
    // Find users entering "30 days remaining" window
    query := `
        INSERT INTO scheduled_notifications (user_id, tenant_id, notification_type, scheduled_for, metadata)
        SELECT 
            u.id,
            u.tenant_id,
            'deletion_warning',
            NOW() + INTERVAL '1 hour',  -- Send soon, but not immediately (avoid thundering herd)
            jsonb_build_object(
                'email', u.email,
                'name', u.name,
                'scheduled_deletion_at', u.scheduled_deletion_at,
                'days_remaining', EXTRACT(DAY FROM (u.scheduled_deletion_at - NOW()))::int
            )
        FROM users u
        WHERE u.status = 'deleted'
          AND u.scheduled_deletion_at > NOW()
          AND u.scheduled_deletion_at <= NOW() + INTERVAL '30 days'
          AND u.deletion_notified_at IS NULL
          AND NOT EXISTS (
              SELECT 1 
              FROM scheduled_notifications sn
              WHERE sn.user_id = u.id 
                AND sn.notification_type = 'deletion_warning'
          )
        ON CONFLICT DO NOTHING  -- Idempotency
    `
    
    result, err := s.db.ExecContext(ctx, query)
    if err != nil {
        return fmt.Errorf("failed to schedule notifications: %w", err)
    }
    
    rows, _ := result.RowsAffected()
    log.Info().Int64("scheduled", rows).Msg("Deletion warnings scheduled")
    
    return nil
}

// Separate job sends scheduled notifications
func (s *NotificationService) sendScheduledNotifications(ctx context.Context) error {
    query := `
        SELECT id, user_id, tenant_id, notification_type, metadata
        FROM scheduled_notifications
        WHERE scheduled_for <= NOW()
          AND sent_at IS NULL
          AND failed_at IS NULL
        ORDER BY scheduled_for ASC
        LIMIT 100
    `
    
    rows, err := s.db.QueryContext(ctx, query)
    if err != nil {
        return fmt.Errorf("failed to query scheduled notifications: %w", err)
    }
    defer rows.Close()
    
    for rows.Next() {
        var notification ScheduledNotification
        if err := rows.Scan(&notification.ID, &notification.UserID, &notification.TenantID, 
                            &notification.Type, &notification.Metadata); err != nil {
            log.Error().Err(err).Msg("Failed to scan notification")
            continue
        }
        
        // Send via existing notification service
        if err := s.sendDeletionWarning(ctx, &notification); err != nil {
            // Mark as failed
            _, _ = s.db.ExecContext(ctx, `
                UPDATE scheduled_notifications
                SET failed_at = NOW(), error_msg = $1
                WHERE id = $2
            `, err.Error(), notification.ID)
        } else {
            // Mark as sent
            _, _ = s.db.ExecContext(ctx, `
                UPDATE scheduled_notifications
                SET sent_at = NOW()
                WHERE id = $2
            `, notification.ID)
            
            // Update user record
            _, _ = s.db.ExecContext(ctx, `
                UPDATE users
                SET deletion_notified_at = NOW()
                WHERE id = $1
            `, notification.UserID)
        }
    }
    
    return nil
}
```

**User-Initiated "Restore Account" Flow**:

```go
func (s *UserService) RestoreDeletedAccount(ctx context.Context, userID uuid.UUID) error {
    query := `
        UPDATE users
        SET status = 'active',
            deleted_at = NULL,
            scheduled_deletion_at = NULL,
            deletion_notified_at = NULL,
            cleanup_attempted_at = NULL,
            updated_at = NOW()
        WHERE id = $1
          AND status = 'deleted'
          AND scheduled_deletion_at > NOW()  -- Grace period not expired
        RETURNING id
    `
    
    var restoredID uuid.UUID
    err := s.db.QueryRowContext(ctx, query, userID).Scan(&restoredID)
    if err == sql.ErrNoRows {
        return fmt.Errorf("account not found or grace period expired")
    }
    if err != nil {
        return fmt.Errorf("failed to restore account: %w", err)
    }
    
    // Cancel scheduled notifications
    _, _ = s.db.ExecContext(ctx, `
        DELETE FROM scheduled_notifications
        WHERE user_id = $1 AND sent_at IS NULL
    `, userID)
    
    log.Info().Str("user_id", userID.String()).Msg("User account restored")
    return nil
}
```

---

### 6. Idempotency: Multi-Layer Protection

#### Layer 1: Distributed Lock (Prevents Concurrent Execution)

```go
// Only one instance runs cleanup at a time
acquired, err := s.acquireLock(ctx, "cleanup:soft_delete:lock", 5*time.Minute)
if !acquired {
    return nil  // Another instance is running, skip
}
defer s.releaseLock(ctx, "cleanup:soft_delete:lock")
```

#### Layer 2: Idempotency Marker (`cleanup_attempted_at`)

```sql
-- Mark records before processing
UPDATE users
SET cleanup_attempted_at = NOW()
WHERE status = 'deleted'
  AND scheduled_deletion_at <= NOW()
  AND cleanup_attempted_at IS NULL
RETURNING id;
```

**Why?** If job crashes mid-processing:
- Records with `cleanup_attempted_at` != NULL won't be picked up again immediately
- Manual intervention can reset `cleanup_attempted_at` to retry failed records

#### Layer 3: Transaction Isolation

```go
// Each user deletion in separate transaction (failure doesn't rollback all)
for _, userID := range userIDs {
    err := s.hardDeleteUserInTransaction(ctx, userID)
    if err != nil {
        log.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to delete user")
        // Record failure, continue with next user
        s.recordFailure(ctx, userID, err)
    }
}
```

#### Layer 4: Unique Constraints

```sql
-- Prevent duplicate scheduled notifications
CREATE UNIQUE INDEX idx_scheduled_notifications_user_type ON scheduled_notifications(user_id, notification_type) 
    WHERE sent_at IS NULL;
```

#### Layer 5: Audit Trail

```sql
CREATE TABLE data_deletion_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name VARCHAR(100) NOT NULL,
    record_id UUID NOT NULL,
    tenant_id UUID,
    deleted_by VARCHAR(100),  -- 'system' | user_id
    deleted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB  -- Snapshot of deleted record (for legal compliance)
);

-- Before hard delete, log to audit
INSERT INTO data_deletion_audit (table_name, record_id, tenant_id, deleted_by, metadata)
SELECT 'users', id, tenant_id, 'system', row_to_json(users.*)
FROM users
WHERE id = $1;
```

---

### 7. Monitoring: Prometheus Metrics + Audit Trail

#### Prometheus Metrics

```go
var (
    cleanupJobDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "pos_cleanup_job_duration_seconds",
            Help:    "Duration of cleanup job execution",
            Buckets: prometheus.DefBuckets,
        },
        []string{"job_type", "table"},  // e.g., job_type="soft_delete", table="users"
    )
    
    cleanupRecordsProcessed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "pos_cleanup_records_processed_total",
            Help: "Total number of records processed by cleanup jobs",
        },
        []string{"job_type", "table", "result"},  // result="success"|"failure"
    )
    
    cleanupJobLastRun = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "pos_cleanup_job_last_run_timestamp",
            Help: "Unix timestamp of last cleanup job execution",
        },
        []string{"job_type"},
    )
)

func init() {
    prometheus.MustRegister(cleanupJobDuration, cleanupRecordsProcessed, cleanupJobLastRun)
}

func (s *CleanupService) runCleanupWithMetrics(ctx context.Context, jobType, table string) error {
    startTime := time.Now()
    defer func() {
        duration := time.Since(startTime).Seconds()
        cleanupJobDuration.WithLabelValues(jobType, table).Observe(duration)
        cleanupJobLastRun.WithLabelValues(jobType).SetToCurrentTime()
    }()
    
    err := s.cleanupExpiredRecords(ctx, table, 90)
    if err != nil {
        cleanupRecordsProcessed.WithLabelValues(jobType, table, "failure").Inc()
        return err
    }
    
    cleanupRecordsProcessed.WithLabelValues(jobType, table, "success").Inc()
    return nil
}
```

#### Grafana Alerts

```yaml
# Prometheus AlertManager rules
groups:
  - name: data_retention_alerts
    interval: 5m
    rules:
      - alert: CleanupJobFailed
        expr: |
          rate(pos_cleanup_records_processed_total{result="failure"}[5m]) > 0.1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Cleanup job failing frequently"
          description: "Cleanup job for {{ $labels.table }} has >10% failure rate"
      
      - alert: CleanupJobStale
        expr: |
          (time() - pos_cleanup_job_last_run_timestamp) > 86400
        for: 1h
        labels:
          severity: critical
        annotations:
          summary: "Cleanup job not running"
          description: "Cleanup job {{ $labels.job_type }} has not run in 24 hours"
      
      - alert: HighRetentionBacklog
        expr: |
          pos_cleanup_records_pending{table="users"} > 10000
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "High cleanup backlog"
          description: "{{ $labels.table }} has {{ $value }} records pending cleanup"
```

#### Audit Trail Queries

```sql
-- Daily report: Records deleted by cleanup jobs
SELECT 
    table_name,
    DATE(deleted_at) AS deletion_date,
    COUNT(*) AS records_deleted,
    ROUND(AVG(EXTRACT(EPOCH FROM (deleted_at - (metadata->>'deleted_at')::TIMESTAMPTZ))), 2) AS avg_grace_period_days
FROM data_deletion_audit
WHERE deleted_by = 'system'
  AND deleted_at >= NOW() - INTERVAL '7 days'
GROUP BY table_name, DATE(deleted_at)
ORDER BY deletion_date DESC, table_name;

-- Compliance report: Verify no premature deletions
SELECT 
    table_name,
    record_id,
    deleted_at,
    metadata->>'deleted_at' AS soft_deleted_at,
    EXTRACT(DAY FROM (deleted_at - (metadata->>'deleted_at')::TIMESTAMPTZ)) AS grace_period_days
FROM data_deletion_audit
WHERE deleted_by = 'system'
  AND deleted_at >= NOW() - INTERVAL '30 days'
  AND EXTRACT(DAY FROM (deleted_at - (metadata->>'deleted_at')::TIMESTAMPTZ)) < 90
ORDER BY deleted_at DESC;
```

---

## Alternatives Considered

### Alternative 1: robfig/cron for Job Scheduling

**Description**: Use `github.com/robfig/cron/v3` for cron-like job scheduling

**Pros**:
- ✅ Cron syntax familiar to ops teams (`0 2 * * *` = daily at 2am)
- ✅ Built-in timezone support
- ✅ Job chaining and dependencies

**Cons**:
- ❌ Not used anywhere in codebase (introduces new dependency)
- ❌ Overkill for simple periodic tasks (time.Ticker sufficient)
- ❌ Less flexible for dynamic intervals (retention policy changes)
- ❌ Requires additional error handling/retry logic

**Rejected because**: time.Ticker is simpler, proven in codebase, and sufficient for periodic tasks. Cron syntax not needed for "every N minutes" jobs.

---

### Alternative 2: PostgreSQL pg_cron Extension

**Description**: Schedule cleanup jobs directly in database using `pg_cron`

**Pros**:
- ✅ No application code for scheduling
- ✅ Database-native (no external dependencies)
- ✅ Jobs run even if application down

**Cons**:
- ❌ Requires PostgreSQL extension (not available on all managed databases)
- ❌ Hard to test (requires database setup in CI/CD)
- ❌ No distributed locking (risk of duplicate execution if database replicated)
- ❌ Limited observability (no Prometheus metrics, log aggregation)
- ❌ Hard to version control (SQL jobs stored in database, not git)

**Rejected because**: Violates "business logic in application layer" principle. Cleanup logic should be testable, versioned, and observable like other business logic.

---

### Alternative 3: Queue-Based Delayed Deletion

**Description**: Publish "delete user" event to message queue with 90-day delay

**Pros**:
- ✅ Event-driven (decouples soft delete from hard delete)
- ✅ Scalable (queue handles backpressure)
- ✅ Retry built-in (dead letter queue for failures)

**Cons**:
- ❌ Kafka doesn't support delayed messages natively (requires workaround)
- ❌ Complexity (need separate delayed message queue like RabbitMQ with plugins)
- ❌ State split (user in database, deletion in queue - harder to restore)
- ❌ Harder to query "when will this user be deleted?" (need to inspect queue)

**Rejected because**: Over-engineering for scheduled tasks. time.Ticker + database state simpler and more queryable.

---

### Alternative 4: Soft Delete Only (No Hard Delete)

**Description**: Keep deleted records forever with `status='deleted'`, never physically remove

**Pros**:
- ✅ Simplest implementation (no cleanup jobs)
- ✅ Complete audit trail
- ✅ Easy to restore deleted accounts

**Cons**:
- ❌ Violates UU PDP "right to erasure" (data must be deleted after retention period)
- ❌ Database bloat (indexes grow, VACUUM overhead increases)
- ❌ Storage costs accumulate
- ❌ Tenant isolation risk (deleted tenant data still in database)

**Rejected because**: Non-compliant with UU PDP Article 34 (right to deletion). Hard delete required after grace period.

---

## Implementation Notes

### Job Scheduling Patterns

#### Pattern 1: Single Cleanup Service (Recommended)

```go
// backend/cleanup-service/main.go
func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()
    
    cleanupService := services.NewCleanupService(db, redis, kafkaProducer)
    
    // Start all cleanup jobs
    go cleanupService.SoftDeleteMarker(ctx, 1*time.Hour)       // Mark records for deletion
    go cleanupService.NotificationScheduler(ctx, 1*time.Hour)  // Schedule deletion warnings
    go cleanupService.HardDeleteExecutor(ctx, 24*time.Hour)    // Execute hard deletes (daily)
    go cleanupService.RetentionAuditor(ctx, 168*time.Hour)     // Weekly audit
    
    <-ctx.Done()
    log.Info().Msg("Cleanup service shutting down gracefully")
}
```

**Advantages**:
- ✅ Centralized cleanup logic (single service to monitor)
- ✅ Shared database connection pool
- ✅ Single metrics endpoint (`/metrics`)
- ✅ Easy to scale horizontally (distributed locking prevents duplication)

**Disadvantages**:
- ❌ Single point of failure (if service crashes, all cleanups stop)
- ❌ Resource contention (all jobs share same service resources)

#### Pattern 2: Per-Service Cleanup Jobs (Alternative)

```go
// backend/user-service/main.go
func main() {
    // ... existing user service setup ...
    
    // Start user cleanup job within user-service
    cleanupJob := services.NewUserCleanupJob(db, redis)
    go cleanupJob.Start(ctx)
}

// backend/order-service/main.go
func main() {
    // ... existing order service setup ...
    
    // Start order cleanup job within order-service
    cleanupJob := services.NewOrderCleanupJob(db, redis)
    go cleanupJob.Start(ctx)
}
```

**Advantages**:
- ✅ Domain-driven (user-service owns user cleanup logic)
- ✅ Isolated failures (user cleanup failure doesn't affect order cleanup)
- ✅ Easier to test (cleanup logic in same service as domain logic)

**Disadvantages**:
- ❌ Duplicated cleanup patterns across services
- ❌ More services to monitor (need to check each service's cleanup job health)
- ❌ Harder to coordinate (e.g., "delete user → then delete all orders")

**Recommendation**: Pattern 1 (single cleanup service) for initial implementation, migrate to Pattern 2 if cleanup logic becomes too complex or resource-intensive.

---

### Batch Deletion Queries

#### Example 1: Users Table

```sql
-- Find users ready for hard delete (with safety checks)
WITH users_to_delete AS (
    SELECT id, email, tenant_id
    FROM users
    WHERE status = 'deleted'
      AND scheduled_deletion_at <= NOW()
      AND cleanup_attempted_at IS NULL
      -- Safety: Ensure grace period actually passed (double-check)
      AND deleted_at < NOW() - INTERVAL '90 days'
    LIMIT 100
)
UPDATE users
SET cleanup_attempted_at = NOW()
WHERE id IN (SELECT id FROM users_to_delete)
RETURNING id, email, tenant_id;
```

#### Example 2: Sessions Table (Simple Retention)

```sql
-- Delete sessions older than 90 days (no soft delete)
DELETE FROM sessions
WHERE id IN (
    SELECT id
    FROM sessions
    WHERE created_at < NOW() - INTERVAL '90 days'
    LIMIT 1000
);
```

#### Example 3: Notifications Table (Status-Based Retention)

```sql
-- Delete sent notifications older than 90 days, keep failed for investigation
DELETE FROM notifications
WHERE id IN (
    SELECT id
    FROM notifications
    WHERE created_at < NOW() - INTERVAL '90 days'
      AND status IN ('sent', 'cancelled')  -- Keep failed/pending for debugging
    LIMIT 1000
);
```

---

### Notification Triggers

#### Email Template: Deletion Warning

```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Account Deletion Warning</title>
</head>
<body>
    <h1>Your account will be permanently deleted soon</h1>
    <p>Hello {{ .Name }},</p>
    <p>Your account deletion was requested on {{ .DeletedAt }}. As per our retention policy, your account and all associated data will be permanently deleted on <strong>{{ .ScheduledDeletionAt }}</strong>.</p>
    <p><strong>Days remaining: {{ .DaysRemaining }}</strong></p>
    <h2>Want to keep your account?</h2>
    <p>If you changed your mind, you can restore your account before the deletion date by clicking the link below:</p>
    <p><a href="{{ .RestoreURL }}" style="background-color: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; border-radius: 4px;">Restore My Account</a></p>
    <p>If you did not request account deletion, please contact support immediately.</p>
    <p>After {{ .ScheduledDeletionAt }}, your data cannot be recovered.</p>
    <hr>
    <p style="font-size: 0.9em; color: #666;">This is an automated notification. You received this because your account is scheduled for deletion. If you have questions, please contact our support team.</p>
</body>
</html>
```

#### Notification Event Schema

```json
{
  "event_type": "user.deletion_warning",
  "user_id": "uuid",
  "tenant_id": "uuid",
  "timestamp": "2026-01-02T10:30:00Z",
  "data": {
    "email": "user@example.com",
    "name": "John Doe",
    "deleted_at": "2025-10-03T10:30:00Z",
    "scheduled_deletion_at": "2026-01-01T10:30:00Z",
    "days_remaining": 30,
    "restore_url": "https://pos.example.com/account/restore?token=xxx"
  }
}
```

---

### Monitoring Integration

#### Grafana Dashboard Panels

**Panel 1: Cleanup Job Health**
- Metric: `pos_cleanup_job_last_run_timestamp`
- Visualization: Status map (green if <1 hour ago, yellow if <24 hours, red if >24 hours)

**Panel 2: Records Processed Rate**
- Metric: `rate(pos_cleanup_records_processed_total[5m])`
- Visualization: Graph by table and result (success/failure)

**Panel 3: Cleanup Duration**
- Metric: `pos_cleanup_job_duration_seconds`
- Visualization: Heatmap (show p50, p95, p99 latencies)

**Panel 4: Pending Cleanup Backlog**
- Query: 
  ```sql
  SELECT 
      'users' AS table,
      COUNT(*) AS pending
  FROM users
  WHERE status = 'deleted'
    AND scheduled_deletion_at <= NOW()
    AND cleanup_attempted_at IS NULL
  ```
- Visualization: Stat panel (warn if >1000)

#### Alert Runbook

**Alert: CleanupJobFailed**
1. Check service logs: `kubectl logs -n pos deployment/cleanup-service --tail=100`
2. Check database connectivity: `kubectl exec -it deployment/cleanup-service -- nc -zv postgres-service 5432`
3. Check Redis connectivity: `kubectl exec -it deployment/cleanup-service -- redis-cli -h redis-service ping`
4. Check for lock contention: `redis-cli GET cleanup:soft_delete:lock` (should be nil or expired)
5. Manually trigger cleanup: `kubectl exec -it deployment/cleanup-service -- /app/cleanup-service --manual-trigger`

**Alert: CleanupJobStale**
1. Check if cleanup service is running: `kubectl get pods -n pos | grep cleanup-service`
2. If crashed, check crash loop: `kubectl describe pod -n pos <pod-name>`
3. Check resource limits: `kubectl top pod -n pos <pod-name>` (CPU/memory exhaustion?)
4. Restart service: `kubectl rollout restart deployment/cleanup-service -n pos`

---

## Summary

### Recommended Approach

1. **Job Scheduling**: `time.Ticker` goroutines with Redis distributed locking
2. **Retention Policies**: Database table (`retention_policies`) with code validation
3. **Soft Delete**: 90-day grace period with `deleted_at` + `scheduled_deletion_at` columns
4. **Hard Delete**: Batch deletion (100 records/iteration, 1-second pause)
5. **Notifications**: Queue-based with 30-day advance warning
6. **Idempotency**: Multi-layer (distributed lock + `cleanup_attempted_at` + transaction isolation + audit trail)
7. **Monitoring**: Prometheus metrics + Grafana dashboards + AlertManager rules

### Key Metrics

- **Job Frequency**: 
  - Soft delete marking: Every 1 hour
  - Notification scheduling: Every 1 hour
  - Hard delete execution: Daily (2am UTC)
  - Retention audit: Weekly (Sunday 3am UTC)

- **Performance Targets**:
  - Batch size: 100 records/iteration
  - Batch interval: 1 second between batches
  - Max job duration: 10 minutes (alert if exceeded)
  - Cleanup lag: <24 hours (alert if records pending >24 hours after scheduled_deletion_at)

- **Reliability**:
  - Distributed lock TTL: 5-10 minutes (longer than max job duration)
  - Lock acquisition failure: Log warning, retry next cycle
  - Hard delete failure: Log error, continue with next record, increment failure metric
  - Notification failure: Retry 3 times with exponential backoff (1m, 5m, 15m)

### Cost Efficiency

- **Database**: Batch deletion prevents long-running transactions, reduces replication lag
- **Storage**: Hard delete reclaims disk space (VACUUM reclaims after deletion)
- **Network**: Distributed locking uses Redis (already in stack, no new service)
- **Compute**: Single cleanup service handles all tables (no per-table microservices)

### Operational Simplicity

- **Single Service**: All cleanup logic in `cleanup-service` (easy to monitor, deploy, scale)
- **Standard Patterns**: Uses existing patterns from codebase (time.Ticker, Redis locking)
- **Observable**: Prometheus metrics + structured logs + audit trail
- **Testable**: Mock time.Ticker, test batch logic in unit tests
- **Recoverable**: Idempotency ensures safe retries after failures
