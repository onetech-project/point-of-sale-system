package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// RetentionPolicy represents a data retention policy (simplified version for order-service)
type RetentionPolicy struct {
	TableName            string
	RecordType           *string
	RetentionPeriodDays  int
	RetentionField       string
	GracePeriodDays      *int
	LegalMinimumDays     int
	CleanupMethod        string
	NotificationDaysBefore *int
	IsActive             bool
}

// CleanupOrchestrator provides cleanup functionality for order-service
type CleanupOrchestrator struct {
	db        *sql.DB
	batchSize int
}

// NewCleanupOrchestrator creates a new cleanup orchestrator
func NewCleanupOrchestrator(db *sql.DB) *CleanupOrchestrator {
	return &CleanupOrchestrator{
		db:        db,
		batchSize: 100,
	}
}

// RunCleanup executes cleanup for a specific retention policy
func (o *CleanupOrchestrator) RunCleanup(ctx context.Context, policy *RetentionPolicy) error {
	expiryDate := time.Now().AddDate(0, 0, -policy.RetentionPeriodDays)

	log.Printf("Starting cleanup for table=%s, method=%s, expiry_before=%s",
		policy.TableName, policy.CleanupMethod, expiryDate.Format(time.RFC3339))

	// Get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s 
		WHERE %s < $1
	`, policy.TableName, policy.RetentionField)

	var totalCount int
	if err := o.db.QueryRowContext(ctx, countQuery, expiryDate).Scan(&totalCount); err != nil {
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
		batchProcessed, err := o.executeCleanupBatch(ctx, policy, expiryDate)
		if err != nil {
			return fmt.Errorf("batch cleanup failed: %w", err)
		}

		processedCount += batchProcessed
		if batchProcessed == 0 {
			break
		}

		log.Printf("Processed %d/%d records for table=%s", processedCount, totalCount, policy.TableName)
	}

	log.Printf("Cleanup completed for table=%s, processed=%d records", policy.TableName, processedCount)
	return nil
}

// executeCleanupBatch executes cleanup for a single batch of records
func (o *CleanupOrchestrator) executeCleanupBatch(ctx context.Context, policy *RetentionPolicy, expiryDate time.Time) (int, error) {
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var query string
	if policy.CleanupMethod == "hard_delete" {
		query = fmt.Sprintf(`
			DELETE FROM %s 
			WHERE %s < $1
			LIMIT $2
		`, policy.TableName, policy.RetentionField)
	} else {
		return 0, fmt.Errorf("unsupported cleanup method: %s", policy.CleanupMethod)
	}

	result, err := tx.ExecContext(ctx, query, expiryDate, o.batchSize)
	if err != nil {
		return 0, fmt.Errorf("cleanup query failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(rowsAffected), nil
}

// CleanupGuestOrdersJob handles cleanup of expired guest orders (5 years)
type CleanupGuestOrdersJob struct {
	db           *sql.DB
	orchestrator *CleanupOrchestrator
}

// NewCleanupGuestOrdersJob creates a new guest orders cleanup job
func NewCleanupGuestOrdersJob(db *sql.DB, orchestrator *CleanupOrchestrator) *CleanupGuestOrdersJob {
	return &CleanupGuestOrdersJob{
		db:           db,
		orchestrator: orchestrator,
	}
}

// Run executes the cleanup job for expired guest orders
func (j *CleanupGuestOrdersJob) Run(ctx context.Context) error {
	log.Println("Starting cleanup job: guest_orders")

	// Hardcoded policy for guest orders (5 years per Indonesian tax law)
	policy := &RetentionPolicy{
		TableName:           "orders",
		RetentionPeriodDays: 1825, // 5 years
		RetentionField:      "created_at",
		LegalMinimumDays:    1825,
		CleanupMethod:       "hard_delete",
		IsActive:            true,
	}

	// Run cleanup using orchestrator
	startTime := time.Now()
	if err := j.orchestrator.RunCleanup(ctx, policy); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("Cleanup job completed: guest_orders (duration=%v)", duration)

	return nil
}

// GetExpiredOrdersCount returns the count of expired guest orders that will be cleaned up
func (j *CleanupGuestOrdersJob) GetExpiredOrdersCount(ctx context.Context) (int, error) {
	expiryDate := time.Now().AddDate(0, 0, -1825) // 5 years

	query := `
		SELECT COUNT(*) 
		FROM orders 
		WHERE created_at < $1
	`

	var count int
	err := j.db.QueryRowContext(ctx, query, expiryDate).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count expired orders: %w", err)
	}

	return count, nil
}
