package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/pos/user-service/src/jobs"
)

// CleanupScheduler handles scheduling for all retention-based cleanup jobs
type CleanupScheduler struct {
	orchestrator *jobs.CleanupOrchestrator
	stopChan     chan struct{}
	running      bool
}

// NewCleanupScheduler creates a new scheduler for retention-based cleanup
func NewCleanupScheduler(orchestrator *jobs.CleanupOrchestrator) *CleanupScheduler {
	return &CleanupScheduler{
		orchestrator: orchestrator,
		stopChan:     make(chan struct{}),
		running:      false,
	}
}

// Start initializes and starts the cleanup scheduler
// Runs daily at 2 AM UTC to enforce retention policies
func (s *CleanupScheduler) Start() error {
	if s.running {
		log.Println("Cleanup scheduler is already running")
		return nil
	}

	s.running = true
	log.Println("Cleanup scheduler started (runs daily at 2 AM UTC)")

	// Start goroutine to run daily at 2 AM UTC
	go s.scheduleLoop()

	return nil
}

// Stop gracefully stops the cleanup scheduler
func (s *CleanupScheduler) Stop() {
	if !s.running {
		return
	}

	log.Println("Stopping cleanup scheduler...")
	close(s.stopChan)
	s.running = false
	log.Println("Cleanup scheduler stopped")
}

// scheduleLoop runs the cleanup job daily at 2 AM UTC
func (s *CleanupScheduler) scheduleLoop() {
	// Calculate time until next 2 AM UTC
	nextRun := s.calculateNextRun()
	log.Printf("Next cleanup run scheduled for: %s", nextRun.Format(time.RFC3339))

	// Initial wait until 2 AM
	timer := time.NewTimer(time.Until(nextRun))
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// Run cleanup
			ctx := context.Background()
			if err := s.orchestrator.RunAllCleanups(ctx); err != nil {
				log.Printf("ERROR: Cleanup scheduler run failed: %v", err)
			} else {
				log.Println("Cleanup scheduler run completed successfully")
			}

			// Schedule next run (24 hours later)
			nextRun = s.calculateNextRun()
			log.Printf("Next cleanup run scheduled for: %s", nextRun.Format(time.RFC3339))
			timer.Reset(time.Until(nextRun))

		case <-s.stopChan:
			log.Println("Cleanup scheduler loop stopped")
			return
		}
	}
}

// calculateNextRun calculates the next 2 AM UTC time
func (s *CleanupScheduler) calculateNextRun() time.Time {
	now := time.Now().UTC()

	// Calculate next 2 AM UTC
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, time.UTC)

	// If we've already passed 2 AM today, schedule for tomorrow
	if now.After(nextRun) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	return nextRun
}

// RunNow executes cleanup immediately (for testing or manual execution)
func (s *CleanupScheduler) RunNow(ctx context.Context) error {
	log.Println("Running cleanup manually...")
	return s.orchestrator.RunAllCleanups(ctx)
}
