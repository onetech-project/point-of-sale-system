package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/pos/user-service/src/services"
)

// CleanupSessionsJob handles cleanup of expired user sessions
type CleanupSessionsJob struct {
	db               *sql.DB
	retentionService *services.RetentionPolicyService
	orchestrator     *CleanupOrchestrator
}

// NewCleanupSessionsJob creates a new sessions cleanup job
func NewCleanupSessionsJob(db *sql.DB, retentionService *services.RetentionPolicyService, orchestrator *CleanupOrchestrator) *CleanupSessionsJob {
	return &CleanupSessionsJob{
		db:               db,
		retentionService: retentionService,
		orchestrator:     orchestrator,
	}
}

// Run executes the cleanup job for expired sessions
func (j *CleanupSessionsJob) Run(ctx context.Context) error {
	log.Println("Starting cleanup job: user_sessions")

	// Get retention policy for user sessions
	policy, err := j.retentionService.GetPolicyByTable(ctx, "user_sessions", nil)
	if err != nil {
		return fmt.Errorf("failed to get retention policy: %w", err)
	}

	// Check if policy is active
	if !policy.IsActive {
		log.Println("Retention policy for user_sessions is not active, skipping cleanup")
		return nil
	}

	// Run cleanup using orchestrator
	startTime := time.Now()
	if err := j.orchestrator.RunCleanup(ctx, policy); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("Cleanup job completed: user_sessions (duration=%v)", duration)

	return nil
}

// GetExpiredSessionsCount returns the count of expired sessions that will be cleaned up
func (j *CleanupSessionsJob) GetExpiredSessionsCount(ctx context.Context) (int, error) {
	policy, err := j.retentionService.GetPolicyByTable(ctx, "user_sessions", nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get retention policy: %w", err)
	}

	return j.retentionService.GetExpiredRecordCount(ctx, policy)
}
