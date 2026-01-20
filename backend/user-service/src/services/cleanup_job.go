package services

import (
	"context"
	"log"
	"time"

	"github.com/pos/user-service/src/observability"
	"github.com/pos/user-service/src/queue"
)

// CleanupJob handles the automated user deletion cleanup process
// Implements UU PDP Article 5 - 90-day retention policy enforcement
type CleanupJob struct {
	deletionService      *UserDeletionService
	notificationProducer *queue.KafkaProducer
}

// NewCleanupJob creates a new cleanup job instance
func NewCleanupJob(deletionService *UserDeletionService, notificationProducer *queue.KafkaProducer) *CleanupJob {
	return &CleanupJob{
		deletionService:      deletionService,
		notificationProducer: notificationProducer,
	}
}

// Run executes the user deletion cleanup logic (T136-T137)
// Step 1: Send notifications to users at 60 days (30 days before hard delete)
// Step 2: Hard delete users at 90 days after soft delete
func (j *CleanupJob) Run(ctx context.Context) {
	startTime := time.Now()

	log.Printf("Starting user deletion cleanup job")

	// Step 1: Send deletion notifications (60 days after soft delete)
	notificationUsers, err := j.deletionService.GetUserDeletionNotificationEligible(ctx)
	if err != nil {
		log.Printf("Failed to get notification eligible users: %v", err)
		observability.CleanupJobErrorsTotal.Inc()
		return
	}

	log.Printf("Found %d users eligible for deletion notification", len(notificationUsers))

	for _, user := range notificationUsers {
		// Send notification via Kafka
		event := map[string]interface{}{
			"event_type":    "user_deletion_warning",
			"user_id":       user.ID,
			"tenant_id":     user.TenantID,
			"email":         user.Email,
			"deletion_date": time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
			"locale":        user.Locale,
		}

		if err := j.notificationProducer.Publish(ctx, user.ID, event); err != nil {
			log.Printf("Failed to publish deletion notification for user %s: %v", user.ID, err)
			observability.CleanupJobErrorsTotal.Inc()
			continue
		}

		// Record notification sent to prevent duplicate notifications
		if err := j.deletionService.RecordNotificationSent(ctx, user.ID); err != nil {
			log.Printf("Failed to record notification for user %s: %v", user.ID, err)
			observability.CleanupJobErrorsTotal.Inc()
			continue
		}

		observability.DeletedUsersNotifiedTotal.Inc()
		log.Printf("Sent deletion notification to user %s (%s)", user.ID, user.Email)
	}

	// Step 2: Hard delete users (90 days after soft delete)
	deletionUsers, err := j.deletionService.GetUserDeletionEligible(ctx)
	if err != nil {
		log.Printf("Failed to get deletion eligible users: %v", err)
		observability.CleanupJobErrorsTotal.Inc()
		return
	}

	log.Printf("Found %d users eligible for hard deletion", len(deletionUsers))

	for _, user := range deletionUsers {
		// Execute hard delete
		if err := j.deletionService.HardDelete(ctx, user.TenantID, user.ID, "system-cleanup-job"); err != nil {
			log.Printf("Failed to hard delete user %s: %v", user.ID, err)
			observability.CleanupJobErrorsTotal.Inc()
			continue
		}

		observability.DeletedUsersHardDeletedTotal.Inc()
		log.Printf("Hard deleted user %s (%s) - 90 day retention period complete", user.ID, user.Email)
	}

	// Record job duration
	duration := time.Since(startTime).Seconds()
	observability.CleanupJobDuration.Observe(duration)

	log.Printf("Completed user deletion cleanup job in %.2f seconds (notified: %d, deleted: %d)",
		duration, len(notificationUsers), len(deletionUsers))
}
