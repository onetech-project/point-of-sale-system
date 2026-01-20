package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/pos/user-service/src/models"
	"github.com/pos/user-service/src/services"
	"github.com/redis/go-redis/v9"
)

// CleanupOrchestrator coordinates automated cleanup jobs based on retention policies
type CleanupOrchestrator struct {
	db               *sql.DB
	redis            *redis.Client
	retentionService *services.RetentionPolicyService
	batchSize        int
	lockTTL          time.Duration
}

// NewCleanupOrchestrator creates a new cleanup orchestrator
func NewCleanupOrchestrator(db *sql.DB, redisClient *redis.Client, retentionService *services.RetentionPolicyService) *CleanupOrchestrator {
	return &CleanupOrchestrator{
		db:               db,
		redis:            redisClient,
		retentionService: retentionService,
		batchSize:        100, // Process 100 records per batch
		lockTTL:          2 * time.Hour,
	}
}

// AcquireLock attempts to acquire a distributed lock for a specific cleanup job
func (o *CleanupOrchestrator) AcquireLock(ctx context.Context, tableName string, recordType *string) (bool, error) {
	lockKey := fmt.Sprintf("cleanup:lock:%s", tableName)
	if recordType != nil {
		lockKey = fmt.Sprintf("%s:%s", lockKey, *recordType)
	}

	// Try to set the lock with SETNX (SET if Not eXists)
	result, err := o.redis.SetNX(ctx, lockKey, time.Now().Unix(), o.lockTTL).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return result, nil
}

// ReleaseLock releases a distributed lock after cleanup completion
func (o *CleanupOrchestrator) ReleaseLock(ctx context.Context, tableName string, recordType *string) error {
	lockKey := fmt.Sprintf("cleanup:lock:%s", tableName)
	if recordType != nil {
		lockKey = fmt.Sprintf("%s:%s", lockKey, *recordType)
	}

	_, err := o.redis.Del(ctx, lockKey).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	return nil
}

// RunCleanup executes cleanup for a specific retention policy
func (o *CleanupOrchestrator) RunCleanup(ctx context.Context, policy *models.RetentionPolicy) error {
	// Acquire distributed lock to prevent concurrent execution
	locked, err := o.AcquireLock(ctx, policy.TableName, policy.RecordType)
	if err != nil {
		return fmt.Errorf("lock acquisition failed: %w", err)
	}
	if !locked {
		log.Printf("Cleanup already running for table=%s, skipping", policy.TableName)
		return nil
	}
	defer o.ReleaseLock(ctx, policy.TableName, policy.RecordType)

	// Calculate expiry date
	expiryDate := time.Now().AddDate(0, 0, -policy.RetentionPeriodDays)

	log.Printf("Starting cleanup for table=%s, method=%s, expiry_before=%s",
		policy.TableName, policy.CleanupMethod, expiryDate.Format(time.RFC3339))

	// Get total count of records to cleanup
	totalCount, err := o.retentionService.GetExpiredRecordCount(ctx, policy)
	if err != nil {
		return fmt.Errorf("failed to count expired records: %w", err)
	}

	if totalCount == 0 {
		log.Printf("No expired records found for table=%s", policy.TableName)
		return nil
	}

	log.Printf("Found %d expired records for table=%s", totalCount, policy.TableName)

	// Process in batches
	var processedCount int
	for processedCount < totalCount {
		// Execute cleanup based on method
		batchProcessed, err := o.executeCleanupBatch(ctx, policy, expiryDate)
		if err != nil {
			return fmt.Errorf("batch cleanup failed: %w", err)
		}

		processedCount += batchProcessed

		// Break if no more records were processed (avoid infinite loop)
		if batchProcessed == 0 {
			break
		}

		log.Printf("Processed %d/%d records for table=%s", processedCount, totalCount, policy.TableName)
	}

	log.Printf("Cleanup completed for table=%s, processed=%d records", policy.TableName, processedCount)
	return nil
}

// executeCleanupBatch executes cleanup for a single batch of records
func (o *CleanupOrchestrator) executeCleanupBatch(ctx context.Context, policy *models.RetentionPolicy, expiryDate time.Time) (int, error) {
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var query string
	var args []interface{}

	switch policy.CleanupMethod {
	case "soft_delete":
		// Set deleted_at timestamp
		query = fmt.Sprintf(`
			UPDATE %s 
			SET deleted_at = NOW()
			WHERE %s < $1 
			  AND deleted_at IS NULL
			LIMIT $2
		`, policy.TableName, policy.RetentionField)
		args = []interface{}{expiryDate, o.batchSize}

	case "hard_delete":
		// Permanently delete records
		query = fmt.Sprintf(`
			DELETE FROM %s 
			WHERE %s < $1
			LIMIT $2
		`, policy.TableName, policy.RetentionField)
		args = []interface{}{expiryDate, o.batchSize}

	case "anonymize":
		// Anonymize sensitive data (implementation depends on table structure)
		// For now, we'll use a generic approach
		query = fmt.Sprintf(`
			UPDATE %s 
			SET email = CONCAT('anonymized_', id, '@deleted.local'),
			    full_name = 'Deleted User',
			    phone_number = NULL
			WHERE %s < $1 
			  AND email NOT LIKE 'anonymized_%%'
			LIMIT $2
		`, policy.TableName, policy.RetentionField)
		args = []interface{}{expiryDate, o.batchSize}

	default:
		return 0, fmt.Errorf("unsupported cleanup method: %s", policy.CleanupMethod)
	}

	// Execute cleanup query
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("cleanup query failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(rowsAffected), nil
}

// RunAllCleanups executes cleanup for all active retention policies
func (o *CleanupOrchestrator) RunAllCleanups(ctx context.Context) error {
	policies, err := o.retentionService.GetActivePolicies(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active policies: %w", err)
	}

	log.Printf("Starting cleanup for %d active retention policies", len(policies))

	var errorCount int
	for _, policy := range policies {
		if err := o.RunCleanup(ctx, policy); err != nil {
			log.Printf("ERROR: Cleanup failed for table=%s: %v", policy.TableName, err)
			errorCount++
			continue
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("cleanup completed with %d errors", errorCount)
	}

	log.Printf("All cleanups completed successfully")
	return nil
}
