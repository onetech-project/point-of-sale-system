package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/pos/notification-service/src/models"
	"github.com/pos/notification-service/src/repository"
)

// RetryWorker handles retrying failed notifications with exponential backoff
type RetryWorker struct {
	repo     *repository.NotificationRepository
	service  *NotificationService
	interval time.Duration
}

// NewRetryWorker creates a new retry worker
func NewRetryWorker(db *sql.DB, service *NotificationService) (*RetryWorker, error) {
	repo, err := repository.NewNotificationRepositoryWithVault(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification repository: %w", err)
	}

	return &RetryWorker{
		repo:     repo,
		service:  service,
		interval: 1 * time.Minute, // Check every minute
	}, nil
}

// Start begins the retry worker loop
func (w *RetryWorker) Start(ctx context.Context) {
	log.Println("Starting retry worker...")
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping retry worker...")
			return
		case <-ticker.C:
			w.processFailedNotifications(ctx)
		}
	}
}

// processFailedNotifications finds and retries failed notifications using exponential backoff
func (w *RetryWorker) processFailedNotifications(ctx context.Context) {
	// Retry strategy:
	// - 1st retry: after 1 minute
	// - 2nd retry: after 5 minutes (total 6 minutes)
	// - 3rd retry: after 15 minutes (total 21 minutes)
	// - Max retries: 3

	now := time.Now()

	// Query failed notifications that are eligible for retry
	query := `
		SELECT id, tenant_id, user_id, type, status, event_type, subject, body, recipient, 
		       metadata, sent_at, failed_at, error_msg, retry_count, created_at, updated_at
		FROM notifications
		WHERE status = 'failed'
		  AND retry_count < 3
		  AND (
		    (retry_count = 0 AND failed_at < $1) OR  -- 1st retry after 1 minute
		    (retry_count = 1 AND failed_at < $2) OR  -- 2nd retry after 5 minutes
		    (retry_count = 2 AND failed_at < $3)     -- 3rd retry after 15 minutes
		  )
		LIMIT 100`

	// Calculate retry thresholds
	oneMinuteAgo := now.Add(-1 * time.Minute)
	fiveMinutesAgo := now.Add(-5 * time.Minute)
	fifteenMinutesAgo := now.Add(-15 * time.Minute)

	rows, err := w.repo.QueryRows(ctx, query, oneMinuteAgo, fiveMinutesAgo, fifteenMinutesAgo)
	if err != nil {
		log.Printf("Failed to query failed notifications: %v", err)
		return
	}
	defer rows.Close()

	retryCount := 0
	for rows.Next() {
		var notification models.Notification
		var metadataJSON []byte
		var eventType string

		err := rows.Scan(
			&notification.ID,
			&notification.TenantID,
			&notification.UserID,
			&notification.Type,
			&notification.Status,
			&eventType, // event_type column - not stored in model
			&notification.Subject,
			&notification.Body,
			&notification.Recipient,
			&metadataJSON,
			&notification.SentAt,
			&notification.FailedAt,
			&notification.ErrorMsg,
			&notification.RetryCount,
			&notification.CreatedAt,
			&notification.UpdatedAt,
		)
		if err != nil {
			log.Printf("Failed to scan notification: %v", err)
			continue
		}

		// Deserialize metadata
		if err := json.Unmarshal(metadataJSON, &notification.Metadata); err != nil {
			log.Printf("Failed to unmarshal metadata for notification %s: %v", notification.ID, err)
			continue
		}

		log.Printf("Retrying notification %s (attempt %d)", notification.ID, notification.RetryCount+1)

		// Retry based on type
		var retryErr error
		switch notification.Type {
		case models.NotificationTypeEmail:
			retryErr = w.service.sendEmail(ctx, &notification)
		case models.NotificationTypePush:
			// TODO: Implement push retry
			log.Printf("Push notification retry not yet implemented")
		default:
			log.Printf("Unknown notification type: %s", notification.Type)
			continue
		}

		if retryErr != nil {
			log.Printf("Retry failed for notification %s: %v", notification.ID, retryErr)
		} else {
			log.Printf("Successfully retried notification %s", notification.ID)
			retryCount++
		}
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating failed notifications: %v", err)
	}

	if retryCount > 0 {
		log.Printf("Retry worker processed %d notifications", retryCount)
	}
}
