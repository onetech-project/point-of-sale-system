package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/pos/user-service/src/services"
)

// CleanupInvitationsJob handles cleanup of expired user invitations
type CleanupInvitationsJob struct {
	db               *sql.DB
	retentionService *services.RetentionPolicyService
	orchestrator     *CleanupOrchestrator
}

// NewCleanupInvitationsJob creates a new invitations cleanup job
func NewCleanupInvitationsJob(db *sql.DB, retentionService *services.RetentionPolicyService, orchestrator *CleanupOrchestrator) *CleanupInvitationsJob {
	return &CleanupInvitationsJob{
		db:               db,
		retentionService: retentionService,
		orchestrator:     orchestrator,
	}
}

// Run executes the cleanup job for expired invitations
func (j *CleanupInvitationsJob) Run(ctx context.Context) error {
	log.Println("Starting cleanup job: user_invitations")

	// Get retention policy for user invitations
	policy, err := j.retentionService.GetPolicyByTable(ctx, "user_invitations", nil)
	if err != nil {
		return fmt.Errorf("failed to get retention policy: %w", err)
	}

	// Check if policy is active
	if !policy.IsActive {
		log.Println("Retention policy for user_invitations is not active, skipping cleanup")
		return nil
	}

	// Run cleanup using orchestrator
	startTime := time.Now()
	if err := j.orchestrator.RunCleanup(ctx, policy); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("Cleanup job completed: user_invitations (duration=%v)", duration)

	return nil
}

// GetExpiredInvitationsCount returns the count of expired invitations that will be cleaned up
func (j *CleanupInvitationsJob) GetExpiredInvitationsCount(ctx context.Context) (int, error) {
	policy, err := j.retentionService.GetPolicyByTable(ctx, "user_invitations", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get retention policy: %w", err)
	}

	return j.retentionService.GetExpiredRecordCount(ctx, policy)
}
