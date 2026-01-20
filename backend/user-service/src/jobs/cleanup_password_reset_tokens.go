package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/pos/user-service/src/services"
)

// CleanupPasswordResetTokensJob handles cleanup of consumed password reset tokens
type CleanupPasswordResetTokensJob struct {
	db               *sql.DB
	retentionService *services.RetentionPolicyService
	orchestrator     *CleanupOrchestrator
}

// NewCleanupPasswordResetTokensJob creates a new password reset tokens cleanup job
func NewCleanupPasswordResetTokensJob(db *sql.DB, retentionService *services.RetentionPolicyService, orchestrator *CleanupOrchestrator) *CleanupPasswordResetTokensJob {
	return &CleanupPasswordResetTokensJob{
		db:               db,
		retentionService: retentionService,
		orchestrator:     orchestrator,
	}
}

// Run executes the cleanup job for consumed password reset tokens
func (j *CleanupPasswordResetTokensJob) Run(ctx context.Context) error {
	log.Println("Starting cleanup job: password_reset_tokens")

	// Get retention policy for password reset tokens
	policy, err := j.retentionService.GetPolicyByTable(ctx, "password_reset_tokens", nil)
	if err != nil {
		return fmt.Errorf("failed to get retention policy: %w", err)
	}

	// Check if policy is active
	if !policy.IsActive {
		log.Println("Retention policy for password_reset_tokens is not active, skipping cleanup")
		return nil
	}

	// Run cleanup using orchestrator
	startTime := time.Now()
	if err := j.orchestrator.RunCleanup(ctx, policy); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("Cleanup job completed: password_reset_tokens (duration=%v)", duration)

	return nil
}

// GetExpiredTokensCount returns the count of consumed tokens that will be cleaned up
func (j *CleanupPasswordResetTokensJob) GetExpiredTokensCount(ctx context.Context) (int, error) {
	policy, err := j.retentionService.GetPolicyByTable(ctx, "password_reset_tokens", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get retention policy: %w", err)
	}

	return j.retentionService.GetExpiredRecordCount(ctx, policy)
}
