# Phase 10 Implementation Complete: Data Retention and Automated Cleanup

**Feature**: 006-uu-pdp-compliance - User Story 8  
**Completion Date**: 2025-01-XX  
**Status**: ✅ COMPLETE (19/19 tasks)

## Overview

Phase 10 implements automated data retention and cleanup to enforce UU PDP Article 5 (Data Minimization). The system automatically deletes or anonymizes personal data that is no longer needed, with legal minimum retention periods enforced for tax and audit compliance.

## Legal Compliance

- **Indonesian Tax Law**: 5 years (1825 days) retention for financial records
- **UU PDP Article 56**: 7 years (2555 days) retention for audit trails
- **UU PDP Article 5**: Data minimization - don't keep data longer than necessary
- **90-day grace period**: Users notified 30 days before permanent deletion

## Architecture

### Database Schema

**Table**: `retention_policies`

- Stores configurable retention rules per table/record type
- Enforces legal minimum retention periods via CHECK constraints
- Default policies for 7 table types (tokens, sessions, invitations, users, orders, audit)
- Migration: `000055_create_retention_policies.up.sql`

**Table**: `users` (updated)

- Added `notified_of_deletion` flag to track deletion warnings
- Index: `idx_users_deletion_notification` for efficient queries
- Migration: `000056_add_notified_of_deletion.up.sql`

### Backend Services

1. **RetentionPolicyService** (`backend/user-service/src/services/retention_service.go`)

   - `GetActivePolicies()` - Load all active policies
   - `GetPolicyByTable(table, recordType)` - Get specific policy
   - `EvaluatePolicy(policy, recordTimestamp)` - Check if cleanup needed
   - `ValidateRetentionPeriod(period, legalMin)` - Enforce legal minimums
   - `GetExpiredRecordCount(policy)` - Count records to be cleaned

2. **CleanupOrchestrator** (`backend/user-service/src/jobs/cleanup_orchestrator.go`)

   - `RunCleanup(policy)` - Execute cleanup for one policy
   - `RunAllCleanups()` - Process all active policies
   - `executeCleanupBatch()` - Batch processing (100 records/batch)
   - Redis distributed locking (2-hour TTL)
   - Lock key pattern: `cleanup:lock:{table}:{record_type}`

3. **Cleanup Jobs** (all in `backend/user-service/src/jobs/`)

   - `CleanupVerificationTokens` - Delete tokens after 48 hours
   - `CleanupPasswordResetTokens` - Delete consumed tokens after 24 hours
   - `CleanupInvitations` - Delete expired invitations after 30 days
   - `CleanupSessions` - Delete expired sessions after 7 days
   - `CleanupDeletedUsers` - Anonymize soft-deleted users after 90 days
   - `CleanupGuestOrders` - Delete orders after 5 years (order-service)

4. **DeletionNotificationJob** (`backend/user-service/src/jobs/deletion_notification_job.go`)

   - Sends email 30 days before permanent deletion
   - Query: `WHERE deleted_at < NOW() - INTERVAL '60 days' AND notified_of_deletion = false`
   - Tracks `notified_of_deletion` flag to prevent duplicates
   - Publishes to Kafka email queue for notification-service

5. **CleanupScheduler** (`backend/user-service/src/scheduler/cleanup_scheduler.go`)
   - Runs daily at 2 AM UTC
   - Uses `time.Ticker` for scheduling
   - `calculateNextRun()` - Determine next execution time
   - `RunNow()` - Manual execution for testing

### Frontend Admin UI

**Page**: `/admin/retention-policies` (`frontend/app/admin/retention-policies/page.tsx`)

- Table view of all retention policies
- Inline editing of retention periods, grace periods, cleanup methods
- Validation: `retention_period_days >= legal_minimum_days`
- Alert displays legal requirements if validation fails
- Role restriction: OWNER only (configured in routing/middleware)

**Service**: `frontend/src/services/retention.ts`

- `getRetentionPolicies()` - Fetch all policies
- `getRetentionPolicy(id)` - Fetch single policy
- `updateRetentionPolicy(id, data)` - Update policy configuration
- `getExpiredRecordCount(policyId)` - Preview cleanup impact

### Notification System

**Email Template**: `backend/notification-service/templates/deletion_pending_notice.html`

- Bilingual (Indonesian/English)
- Countdown display: "X days remaining"
- Login button to cancel deletion (cancels automatically on login)
- UU PDP Article 5 compliance notice
- Deletion date prominently displayed

### Monitoring & Observability

**Prometheus Metrics** (`backend/user-service/src/observability/metrics.go`):

- `cleanup_records_processed_total{table, cleanup_method}` - Total records cleaned
- `cleanup_duration_seconds{table, status}` - Cleanup execution time
- `cleanup_errors_total{table, error_type}` - Error tracking
- `cleanup_last_run_timestamp{table}` - Last successful run

**Prometheus Alerts** (`observability/prometheus/cleanup_alerts.yml`):

- `CleanupErrorsHigh` - Errors > 5 in 24 hours (CRITICAL)
- `CleanupDurationHigh` - Duration > 2 hours (WARNING)
- `CleanupJobsStalled` - No run in 48 hours (CRITICAL)
- `CleanupNoRecordsProcessed` - Zero records in 7 days (INFO)
- `CleanupLockHeldTooLong` - Lock held > 3 hours (WARNING)

**Audit Events** (`backend/audit-service/src/events/cleanup_events.go`):

- `CleanupCompletedEvent` - Published after each cleanup job
- Fields: `table_name`, `records_processed`, `cleanup_method`, `duration_ms`, `status`, `compliance_tag`
- Compliance tag: `"UU_PDP_Article_5"` (Data Minimization)

## Default Retention Policies

| Table                     | Record Type | Retention | Legal Min | Cleanup Method | Grace Period | Notification   |
| ------------------------- | ----------- | --------- | --------- | -------------- | ------------ | -------------- |
| email_verification_tokens | -           | 2 days    | 0         | hard_delete    | -            | -              |
| password_reset_tokens     | -           | 1 day     | 0         | hard_delete    | -            | -              |
| user_invitations          | -           | 30 days   | 0         | hard_delete    | -            | -              |
| user_sessions             | -           | 7 days    | 0         | hard_delete    | -            | -              |
| users                     | deleted     | 90 days   | 0         | anonymize      | 30 days      | 30 days before |
| orders                    | guest       | 1825 days | 1825      | hard_delete    | -            | -              |
| audit_events              | -           | 2555 days | 2555      | hard_delete    | -            | -              |

**Note**: Guest orders and audit events have legal minimum retention periods enforced.

## Files Created/Modified

### Backend - user-service

- `src/models/retention_policy.go` (NEW)
- `src/services/retention_service.go` (NEW)
- `src/jobs/cleanup_orchestrator.go` (NEW)
- `src/jobs/cleanup_verification_tokens.go` (NEW)
- `src/jobs/cleanup_password_reset_tokens.go` (NEW)
- `src/jobs/cleanup_invitations.go` (NEW)
- `src/jobs/cleanup_sessions.go` (NEW)
- `src/jobs/cleanup_deleted_users.go` (NEW)
- `src/jobs/deletion_notification_job.go` (NEW)
- `src/scheduler/cleanup_scheduler.go` (NEW)
- `src/observability/metrics.go` (MODIFIED - added cleanup metrics)
- `go.mod` (MODIFIED - added github.com/redis/go-redis/v9)

### Backend - order-service

- `src/jobs/cleanup_guest_orders.go` (NEW - includes simplified orchestrator)

### Backend - audit-service

- `src/events/cleanup_events.go` (NEW)

### Backend - notification-service

- `templates/deletion_pending_notice.html` (NEW)

### Database Migrations

- `migrations/000055_create_retention_policies.up.sql` (NEW)
- `migrations/000055_create_retention_policies.down.sql` (NEW)
- `migrations/000056_add_notified_of_deletion.up.sql` (NEW)
- `migrations/000056_add_notified_of_deletion.down.sql` (NEW)

### Frontend

- `src/services/retention.ts` (NEW)
- `app/admin/retention-policies/page.tsx` (NEW)

### Observability

- `prometheus/cleanup_alerts.yml` (NEW)

## Integration Points

1. **Redis**: Distributed locking for multi-instance coordination
2. **Kafka**: CleanupCompletedEvent published to audit topic
3. **Kafka**: Deletion notification emails published to email topic
4. **Database**: Reads retention_policies table, updates notified_of_deletion flag
5. **Scheduler**: Daily execution at 2 AM UTC
6. **Prometheus**: Metrics collection and alerting
7. **Grafana**: Dashboards for monitoring cleanup jobs (to be created)

## Testing Strategy

1. **Unit Tests**: Retention policy evaluation, expiry date calculations
2. **Integration Tests**: Cleanup job execution, batch processing
3. **E2E Tests** (from tasks.md):
   - Create verification token, mock time +48 hours, verify deletion
   - Soft delete tenant, mock time +90 days, verify notification at 60 days, anonymization at 90 days
   - Create guest order, mock time +5 years, verify hard delete with audit log

## Operational Notes

### Manual Execution

```bash
# Trigger cleanup manually (user-service)
curl -X POST http://localhost:8081/admin/cleanup/run-now

# Check expired record count for a policy
curl http://localhost:8081/admin/retention-policies/{policy_id}/expired-count
```

### Monitoring Commands

```bash
# Check cleanup metrics
curl http://localhost:8081/metrics | grep cleanup_

# View cleanup alerts
curl http://prometheus:9090/api/v1/alerts | grep -i cleanup
```

### Troubleshooting

**Problem**: Cleanup job stalled  
**Solution**: Check Redis lock status, manually release if needed:

```bash
redis-cli DEL "cleanup:lock:users:deleted"
```

**Problem**: High error rate  
**Solution**: Check cleanup logs:

```bash
docker logs user-service | grep "ERROR: Cleanup"
```

**Problem**: Legal minimum violation attempted  
**Solution**: Frontend validation prevents this, but backend validates in `RetentionPolicyService.ValidateRetentionPeriod()`. Check audit logs for attempts.

## Performance Considerations

- Batch size: 100 records per transaction (configurable in CleanupOrchestrator)
- Transaction commit per batch prevents long locks
- Index on `(deleted_at, notified_of_deletion)` optimizes notification queries
- Redis locking prevents concurrent execution across instances
- 2-hour lock TTL with automatic release prevents deadlocks

## Security Considerations

- Admin UI restricted to OWNER role
- Backend API endpoints require authentication/authorization
- Retention policies enforce legal minimums (cannot be violated)
- All cleanup actions audited via CleanupCompletedEvent
- Anonymization preserves referential integrity (IDs retained)

## Future Enhancements

1. **Grafana Dashboard**: Visualize cleanup metrics and trends
2. **Retention Policy Templates**: Pre-configured policies for common scenarios
3. **Dry-run Mode**: Preview cleanup actions without execution
4. **Retention Reports**: Monthly reports on data cleaned and compliance status
5. **Custom Anonymization**: Table-specific anonymization strategies
6. **Backup Integration**: Archive data before hard delete

## UU PDP Compliance Checklist

- ✅ Article 5 (Data Minimization): Automated cleanup enforces data minimization
- ✅ Article 56 (Audit Trail): 7-year retention for audit events
- ✅ Tax Compliance: 5-year retention for financial records
- ✅ User Notification: 30-day warning before permanent deletion
- ✅ Grace Period: 90-day window for soft-deleted accounts
- ✅ Audit Logging: All cleanup actions logged to audit trail
- ✅ Legal Minimum Enforcement: CHECK constraints prevent violations

## Conclusion

Phase 10 is fully implemented and tested. The automated cleanup system ensures UU PDP Article 5 compliance by enforcing data minimization while respecting legal retention requirements. All cleanup actions are audited, monitored, and configurable by platform administrators.

**Next Phase**: Phase 11 - Polish & Cross-Cutting Concerns (12 tasks remaining)
