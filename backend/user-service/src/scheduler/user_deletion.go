package scheduler

import (
	"context"
	"log"

	"github.com/pos/user-service/src/services"
	"github.com/robfig/cron/v3"
)

// UserDeletionScheduler handles the cron scheduling for user deletion cleanup
type UserDeletionScheduler struct {
	cron       *cron.Cron
	cleanupJob *services.CleanupJob
}

// NewUserDeletionScheduler creates a new scheduler for user deletion cleanup (T135-T138)
func NewUserDeletionScheduler(cleanupJob *services.CleanupJob) *UserDeletionScheduler {
	return &UserDeletionScheduler{
		cron:       cron.New(),
		cleanupJob: cleanupJob,
	}
}

// Start initializes and starts the cron scheduler
// Runs daily at 2 AM UTC to enforce 90-day retention policy per UU PDP Article 5
func (s *UserDeletionScheduler) Start() error {
	// Schedule cleanup job to run daily at 2 AM UTC
	// Cron format: "minute hour day month weekday"
	_, err := s.cron.AddFunc("0 2 * * *", func() {
		ctx := context.Background()
		s.cleanupJob.Run(ctx)
	})

	if err != nil {
		return err
	}

	s.cron.Start()
	log.Printf("User deletion cleanup scheduler started (runs daily at 2 AM UTC)")

	return nil
}

// Stop gracefully stops the cron scheduler
func (s *UserDeletionScheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
		log.Printf("User deletion cleanup scheduler stopped")
	}
}
