package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/pos/user-service/src/queue"
)

// DeletionNotificationJob handles notifying users before permanent deletion
type DeletionNotificationJob struct {
	db            *sql.DB
	emailProducer *queue.KafkaProducer
	batchSize     int
}

// NewDeletionNotificationJob creates a new deletion notification job
func NewDeletionNotificationJob(db *sql.DB, emailProducer *queue.KafkaProducer) *DeletionNotificationJob {
	return &DeletionNotificationJob{
		db:            db,
		emailProducer: emailProducer,
		batchSize:     100,
	}
}

// Run executes the deletion notification job
// Notifies users 30 days before permanent deletion (90-day grace - 60 days elapsed)
func (j *DeletionNotificationJob) Run(ctx context.Context) error {
	log.Println("Starting deletion notification job")

	// Calculate cutoff: deleted_at < NOW() - 60 days (leaving 30 days before hard delete)
	cutoffDate := time.Now().AddDate(0, 0, -60)

	// Query for users who should be notified
	query := `
		SELECT id, email, full_name, tenant_id, deleted_at
		FROM users
		WHERE deleted_at IS NOT NULL
		  AND notified_of_deletion = false
		  AND deleted_at < $1
		LIMIT $2
	`

	rows, err := j.db.QueryContext(ctx, query, cutoffDate, j.batchSize)
	if err != nil {
		return fmt.Errorf("failed to query users for notification: %w", err)
	}
	defer rows.Close()

	var notifiedCount int
	for rows.Next() {
		var (
			id        string
			email     string
			fullName  string
			tenantID  string
			deletedAt time.Time
		)

		if err := rows.Scan(&id, &email, &fullName, &tenantID, &deletedAt); err != nil {
			log.Printf("ERROR: Failed to scan user row: %v", err)
			continue
		}

		// Calculate days until permanent deletion
		daysSinceDeletion := int(time.Since(deletedAt).Hours() / 24)
		daysRemaining := 90 - daysSinceDeletion

		// Send notification email
		if err := j.sendDeletionNotification(ctx, id, email, fullName, tenantID, daysRemaining); err != nil {
			log.Printf("ERROR: Failed to send notification to user %s: %v", id, err)
			continue
		}

		// Mark user as notified
		if err := j.markAsNotified(ctx, id); err != nil {
			log.Printf("ERROR: Failed to mark user %s as notified: %v", id, err)
			continue
		}

		notifiedCount++
		log.Printf("Notified user %s (%s) about pending deletion in %d days", id, email, daysRemaining)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating user rows: %w", err)
	}

	log.Printf("Deletion notification job completed: notified %d users", notifiedCount)
	return nil
}

// sendDeletionNotification sends a deletion pending email
func (j *DeletionNotificationJob) sendDeletionNotification(ctx context.Context, userID, email, fullName, tenantID string, daysRemaining int) error {
	// Create email message for Kafka
	message := map[string]interface{}{
		"type":            "deletion_pending_notice",
		"recipient_email": email,
		"recipient_name":  fullName,
		"user_id":         userID,
		"tenant_id":       tenantID,
		"template_data": map[string]interface{}{
			"full_name":      fullName,
			"days_remaining": daysRemaining,
			"deletion_date":  time.Now().AddDate(0, 0, daysRemaining).Format("2006-01-02"),
		},
		"compliance_tag": "UU_PDP_Article_5",
	}

	// Publish to email queue (notification-service will consume)
	return j.emailProducer.Publish(ctx, "email", message)
}

// markAsNotified updates the notified_of_deletion flag
func (j *DeletionNotificationJob) markAsNotified(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET notified_of_deletion = true
		WHERE id = $1
	`

	_, err := j.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to update notified_of_deletion flag: %w", err)
	}

	return nil
}

// GetPendingNotificationCount returns the count of users pending notification
func (j *DeletionNotificationJob) GetPendingNotificationCount(ctx context.Context) (int, error) {
	cutoffDate := time.Now().AddDate(0, 0, -60)

	query := `
		SELECT COUNT(*) 
		FROM users
		WHERE deleted_at IS NOT NULL
		  AND notified_of_deletion = false
		  AND deleted_at < $1
	`

	var count int
	err := j.db.QueryRowContext(ctx, query, cutoffDate).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count pending notifications: %w", err)
	}

	return count, nil
}
