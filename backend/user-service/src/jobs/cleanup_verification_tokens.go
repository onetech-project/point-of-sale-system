package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/pos/user-service/src/services"
)

// CleanupVerificationTokensJob handles cleanup of expired email verification tokens
type CleanupVerificationTokensJob struct {
	db               *sql.DB
	retentionService *services.RetentionPolicyService
	orchestrator     *CleanupOrchestrator
}

// NewCleanupVerificationTokensJob creates a new verification tokens cleanup job
func NewCleanupVerificationTokensJob(db *sql.DB, retentionService *services.RetentionPolicyService, orchestrator *CleanupOrchestrator) *CleanupVerificationTokensJob {
	return &CleanupVerificationTokensJob{
		db:               db,
		retentionService: retentionService,
		orchestrator:     orchestrator,
	}
}

// Run executes the cleanup job for expired verification tokens
func (j *CleanupVerificationTokensJob) Run(ctx context.Context) error {
	log.Println("Starting cleanup job: email_verification_tokens")

	// Get retention policy for email verification tokens
	policy, err := j.retentionService.GetPolicyByTable(ctx, "email_verification_tokens", nil)
	if err != nil {
		return fmt.Errorf("failed to get retention policy: %w", err)
	}

	// Check if policy is active
	if !policy.IsActive {
		log.Println("Retention policy for email_verification_tokens is not active, skipping cleanup")
		return nil
	}

	// Run cleanup using orchestrator
	startTime := time.Now()
	if err := j.orchestrator.RunCleanup(ctx, policy); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("Cleanup job completed: email_verification_tokens (duration=%v)", duration)

	return nil
}

// GetExpiredTokensCount returns the count of expired tokens that will be cleaned up
func (j *CleanupVerificationTokensJob) GetExpiredTokensCount(ctx context.Context) (int, error) {
	policy, err := j.retentionService.GetPolicyByTable(ctx, "email_verification_tokens", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get retention policy: %w", err)
	}

	return j.retentionService.GetExpiredRecordCount(ctx, policy)
}
