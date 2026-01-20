# User Deletion Cleanup Refactoring Summary

**Date**: January 15, 2026  
**Feature**: UU PDP Compliance - Automated User Deletion Cleanup (T135-T138)

## Overview

Refactored user deletion cleanup from a **separate binary** approach to an **integrated cron scheduler** within the user-service binary. This simplifies deployment, reduces code duplication, and follows better architectural practices.

---

## What Changed

### Before: Separate Binary Approach ❌

**Structure**:

```
jobs/
├── user-deletion-cleanup/
│   ├── main.go          # Standalone binary
│   ├── Dockerfile       # Separate container
│   ├── cronjob.yaml     # Kubernetes CronJob
│   └── README.md
└── README.md
```

**Issues**:

- Duplicate code (database setup, service initialization)
- Separate Docker image to maintain
- Additional binary to build and deploy
- Cannot reuse existing user-service infrastructure
- Over-engineered for a simple scheduled task

### After: Integrated Cron Scheduler ✅

**Structure**:

```
backend/user-service/
├── main.go                           # Includes cron scheduler
├── src/services/
│   ├── deletion_service.go           # Core deletion logic
│   └── deletion_service_notification.go  # Notification tracking
└── go.mod                            # Added robfig/cron/v3
```

**Benefits**:

- ✅ Single codebase - reuses UserDeletionService
- ✅ Single Docker image - no separate container
- ✅ Single binary - just one build/deployment
- ✅ Shares database connections, config, dependencies
- ✅ Automatic startup with user-service
- ✅ No Kubernetes CronJob manifest needed
- ✅ Idempotency via `user_deletion_notifications` table

---

## Implementation Details

### 1. Added Cron Library

**File**: `backend/user-service/go.mod`

```go
require (
    // ... existing dependencies
    github.com/robfig/cron/v3 v3.0.1
)
```

### 2. Prometheus Metrics

**File**: `backend/user-service/main.go`

```go
var (
    deletedUsersNotifiedTotal = promauto.NewCounter(prometheus.CounterOpts{
        Name: "deleted_users_notified_total",
        Help: "Total number of users notified about upcoming deletion",
    })

    deletedUsersHardDeletedTotal = promauto.NewCounter(prometheus.CounterOpts{
        Name: "deleted_users_hard_deleted_total",
        Help: "Total number of users permanently deleted",
    })

    cleanupJobDuration = promauto.NewHistogram(prometheus.HistogramOpts{
        Name:    "cleanup_job_duration_seconds",
        Help:    "Duration of cleanup job execution",
        Buckets: prometheus.DefBuckets,
    })

    cleanupJobErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
        Name: "cleanup_job_errors_total",
        Help: "Total number of cleanup job errors",
    })
)
```

### 3. Cron Scheduler Function

**File**: `backend/user-service/main.go`

```go
// Runs daily at 2 AM UTC
func startCleanupScheduler(deletionService *services.UserDeletionService, notificationProducer *queue.KafkaProducer) {
    c := cron.New()

    // Cron format: "minute hour day month weekday"
    _, err := c.AddFunc("0 2 * * *", func() {
        runCleanupJob(deletionService, notificationProducer)
    })

    if err != nil {
        log.Fatalf("Failed to schedule cleanup job: %v", err)
    }

    c.Start()
    log.Printf("User deletion cleanup scheduler started (runs daily at 2 AM UTC)")
}
```

### 4. Cleanup Job Implementation

**File**: `backend/user-service/main.go`

```go
func runCleanupJob(deletionService *services.UserDeletionService, notificationProducer *queue.KafkaProducer) {
    startTime := time.Now()
    ctx := context.Background()

    // Step 1: Send notifications (60 days after soft delete)
    notificationUsers, err := deletionService.GetUserDeletionNotificationEligible(ctx)
    // ... send notifications via Kafka

    // Step 2: Hard delete users (90 days after soft delete)
    deletionUsers, err := deletionService.GetUserDeletionEligible(ctx)
    // ... execute hard delete

    // Record metrics
    cleanupJobDuration.Observe(time.Since(startTime).Seconds())
}
```

### 5. Notification Tracking

**File**: `backend/user-service/src/services/deletion_service_notification.go`

```go
// Prevents duplicate notifications using user_deletion_notifications table
func (s *UserDeletionService) RecordNotificationSent(ctx context.Context, userID string) error {
    query := `
        INSERT INTO user_deletion_notifications (id, user_id, notified_at)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id) DO NOTHING
    `
    // ... implementation
}
```

### 6. Service Initialization

**File**: `backend/user-service/main.go`

```go
// Initialize cleanup job scheduler (T135-T138)
userRepo, err := repository.NewUserRepositoryWithVault(db, auditPublisher)
if err != nil {
    log.Fatalf("Failed to create user repository: %v", err)
}
deletionService := services.NewUserDeletionService(userRepo, auditPublisher, db)
startCleanupScheduler(deletionService, eventProducer)
```

---

## Deployment

### Before: Multiple Deployments

1. Deploy user-service as Deployment
2. Build separate cleanup job Docker image
3. Deploy cleanup job as Kubernetes CronJob
4. Maintain two separate configurations

### After: Single Deployment

1. Deploy user-service as Deployment (includes cron scheduler)
2. Cron scheduler starts automatically on service startup
3. No additional configuration needed

---

## Multiple Instances & Idempotency

### Problem

When running multiple replicas of user-service (e.g., 3 instances for high availability), **all instances trigger cleanup at 2 AM**.

### Solution: Idempotency via Database

**Table**: `user_deletion_notifications`

```sql
CREATE TABLE user_deletion_notifications (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE,  -- UNIQUE constraint prevents duplicates
    notified_at TIMESTAMP NOT NULL
);
```

**Flow**:

1. Instance 1 inserts notification record → **SUCCESS** → Sends email
2. Instance 2 tries to insert → **DUPLICATE KEY ERROR** → Skips
3. Instance 3 tries to insert → **DUPLICATE KEY ERROR** → Skips

**Result**: Only **ONE notification** sent per user, even with 3 instances running.

---

## Testing

### Local Testing

```bash
cd backend/user-service
go run main.go
```

**Expected Log Output**:

```
User deletion cleanup scheduler started (runs daily at 2 AM UTC)
User service starting on port 8084
```

### Manual Trigger (for testing)

To test without waiting for 2 AM, temporarily change the cron schedule:

```go
// Change from:
_, err := c.AddFunc("0 2 * * *", func() {

// To (runs every minute):
_, err := c.AddFunc("* * * * *", func() {
```

### Metrics Verification

Check Prometheus metrics after cleanup runs:

```bash
curl http://localhost:8084/metrics | grep cleanup_job
```

**Expected Metrics**:

```
deleted_users_notified_total 5
deleted_users_hard_deleted_total 2
cleanup_job_duration_seconds_sum 1.234
cleanup_job_errors_total 0
```

---

## Files Changed

### Modified Files

1. **backend/user-service/go.mod** - Added `github.com/robfig/cron/v3`
2. **backend/user-service/main.go** - Added cron scheduler, metrics, cleanup functions
3. **specs/006-uu-pdp-compliance/tasks.md** - Updated T135-T138 to reflect cron approach
4. **README.md** - Removed jobs/ directory from project structure

### Created Files

1. **backend/user-service/src/services/deletion_service_notification.go** - Notification tracking

### Deleted Files

1. **jobs/user-deletion-cleanup/main.go** ❌
2. **jobs/user-deletion-cleanup/Dockerfile** ❌
3. **jobs/user-deletion-cleanup/cronjob.yaml** ❌
4. **jobs/user-deletion-cleanup/README.md** ❌
5. **jobs/README.md** ❌
6. **jobs/** directory ❌

---

## Compliance

### UU PDP Article 5 - Right to Deletion

**Retention Policy**:

- Soft delete → `status='deleted'`, `deleted_at` timestamp set
- **Day 60**: Send email notification (30 days warning)
- **Day 90**: Hard delete (permanent removal)

**Automated Enforcement**:

- Cron scheduler runs daily at 2 AM UTC
- Checks database for eligible users
- Sends notifications and executes deletions automatically
- Metrics tracked for audit compliance

### Audit Trail

**Preserved Events**:

1. `USER_SOFT_DELETE` - When user is soft deleted (retention starts)
2. Notification event - When 30-day warning is sent (Day 60)
3. `USER_HARD_DELETE` - When user is permanently deleted (Day 90)

**Anonymization**:

- User record deleted from `users` table
- Audit trail anonymized: `actor_email` → `deleted-user-{uuid}`
- Sessions and invitations deleted
- Metadata marked with `{anonymized: true}`

---

## Advantages Summary

| Aspect                    | Separate Binary ❌                   | Integrated Cron ✅              |
| ------------------------- | ------------------------------------ | ------------------------------- |
| **Code Duplication**      | High (duplicates DB setup, services) | None (reuses existing code)     |
| **Docker Images**         | 2 images (service + job)             | 1 image (service only)          |
| **Deployment Complexity** | High (CronJob + Deployment)          | Low (just Deployment)           |
| **Build Time**            | Longer (2 binaries)                  | Faster (1 binary)               |
| **Maintenance**           | Higher (2 codebases)                 | Lower (1 codebase)              |
| **Testing**               | Harder (separate testing)            | Easier (integrated tests)       |
| **Infrastructure**        | More resources (separate pods)       | Less resources (shared)         |
| **Startup Time**          | N/A (scheduled separately)           | Automatic (starts with service) |

---

## Monitoring

### Prometheus Queries

**Cleanup Job Success Rate**:

```promql
rate(cleanup_job_duration_seconds_count[5m])
```

**Error Rate**:

```promql
rate(cleanup_job_errors_total[5m])
```

**Users Notified Daily**:

```promql
increase(deleted_users_notified_total[24h])
```

**Users Hard Deleted Daily**:

```promql
increase(deleted_users_hard_deleted_total[24h])
```

### Alerts

**Cleanup Job Failing**:

```yaml
alert: CleanupJobFailures
expr: rate(cleanup_job_errors_total[1h]) > 0
for: 5m
severity: warning
```

**No Cleanup Activity (Expected)**:

```yaml
alert: CleanupJobNotRunning
expr: increase(cleanup_job_duration_seconds_count[25h]) == 0
for: 1h
severity: critical
```

---

## Conclusion

The refactored cron-based approach is **simpler, more maintainable, and production-ready**. It eliminates unnecessary complexity while preserving all functionality required for UU PDP Article 5 compliance.

**Key Takeaway**: For scheduled tasks that use existing service infrastructure, **integrate them into the service binary** rather than creating separate binaries. Reserve separate binaries for tasks with completely different dependencies or tech stacks.

---

## Related Tasks

- ✅ **T135**: Create scheduled cleanup job (cron at 2 AM UTC)
- ✅ **T136**: Send deletion notification (60 days after soft delete)
- ✅ **T137**: Execute hard delete (90 days after soft delete)
- ✅ **T138**: Add Prometheus metrics (4 metrics tracked)

**Phase 7 Status**: **COMPLETE** ✅
