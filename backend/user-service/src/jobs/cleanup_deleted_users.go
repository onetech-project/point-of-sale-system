package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/pos/user-service/src/services"
)

// CleanupDeletedUsersJob handles anonymization of soft-deleted users after grace period
type CleanupDeletedUsersJob struct {
	db               *sql.DB
	retentionService *services.RetentionPolicyService
	orchestrator     *CleanupOrchestrator
}

// NewCleanupDeletedUsersJob creates a new deleted users cleanup job
func NewCleanupDeletedUsersJob(db *sql.DB, retentionService *services.RetentionPolicyService, orchestrator *CleanupOrchestrator) *CleanupDeletedUsersJob {
	return &CleanupDeletedUsersJob{
		db:               db,
		retentionService: retentionService,
		orchestrator:     orchestrator,
	}
}

// Run executes the cleanup job for soft-deleted users
func (j *CleanupDeletedUsersJob) Run(ctx context.Context) error {
	log.Println("Starting cleanup job: users (soft-deleted)")

	// Get retention policy for deleted users
	recordType := "deleted"
	policy, err := j.retentionService.GetPolicyByTable(ctx, "users", &recordType)
	if err != nil {
		return fmt.Errorf("failed to get retention policy: %w", err)
	}

	// Check if policy is active
	if !policy.IsActive {
		log.Println("Retention policy for deleted users is not active, skipping cleanup")
		return nil
	}

	// Run cleanup using orchestrator (anonymize method)
	startTime := time.Now()
	if err := j.orchestrator.RunCleanup(ctx, policy); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("Cleanup job completed: users (soft-deleted) (duration=%v)", duration)

	return nil
}

// GetDeletedUsersCount returns the count of soft-deleted users beyond grace period
func (j *CleanupDeletedUsersJob) GetDeletedUsersCount(ctx context.Context) (int, error) {
	recordType := "deleted"
	policy, err := j.retentionService.GetPolicyByTable(ctx, "users", &recordType)
	if err != nil {
		return 0, fmt.Errorf("failed to get retention policy: %w", err)
	}

	return j.retentionService.GetExpiredRecordCount(ctx, policy)
}
