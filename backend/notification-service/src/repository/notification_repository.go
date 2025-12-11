package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/pos/notification-service/src/models"
)

type NotificationRepository struct {
	db *sql.DB
}

// NotificationRepositoryInterface defines the methods used by services to interact with notifications.
type NotificationRepositoryInterface interface {
	Create(ctx context.Context, notification *models.Notification) error
	UpdateStatus(ctx context.Context, id string, status models.NotificationStatus, sentAt, failedAt *time.Time, errorMsg *string) error
	FindByID(ctx context.Context, id string) (*models.Notification, error)
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	query := `
		INSERT INTO notifications (tenant_id, user_id, type, status, event_type, subject, body, recipient, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	// Extract event_type from metadata if present
	eventType := "unknown"
	if notification.Metadata != nil {
		if et, ok := notification.Metadata["event_type"].(string); ok {
			eventType = et
		}
	}

	// Convert metadata map to JSON
	var metadataJSON []byte
	var err error
	if notification.Metadata != nil {
		metadataJSON, err = json.Marshal(notification.Metadata)
		if err != nil {
			return err
		}
	}

	// Ensure metadata JSON is valid for insertion; default to empty object if nil
	if metadataJSON == nil {
		metadataJSON = []byte("{}")
	}

	// Notification.Type may contain in_app but DB schema's CHECK currently expects ('email','sms','push').
	// Map in_app -> push for DB storage until schema is expanded.
	row := r.db.QueryRowContext(
		ctx,
		query,
		notification.TenantID,
		notification.UserID,
		notification.Type,
		notification.Status,
		eventType,
		notification.Subject,
		notification.Body,
		notification.Recipient,
		metadataJSON,
	)

	err = row.Scan(&notification.ID, &notification.CreatedAt, &notification.UpdatedAt)
	if err != nil {
		log.Printf("notification insert error: %v", err)
	}
	return err
}

func (r *NotificationRepository) UpdateStatus(ctx context.Context, id string, status models.NotificationStatus, sentAt, failedAt *time.Time, errorMsg *string) error {
	query := `
		UPDATE notifications
		SET status = $1, sent_at = $2, failed_at = $3, error_msg = $4, updated_at = NOW()
		WHERE id = $5`

	_, err := r.db.ExecContext(ctx, query, status, sentAt, failedAt, errorMsg, id)
	return err
}

func (r *NotificationRepository) FindByID(ctx context.Context, id string) (*models.Notification, error) {
	query := `
		SELECT id, tenant_id, user_id, type, status, subject, body, recipient, 
		       metadata, sent_at, failed_at, error_msg, retry_count, created_at, updated_at
		FROM notifications
		WHERE id = $1`

	notification := &models.Notification{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&notification.ID,
		&notification.TenantID,
		&notification.UserID,
		&notification.Type,
		&notification.Status,
		&notification.Subject,
		&notification.Body,
		&notification.Recipient,
		&notification.Metadata,
		&notification.SentAt,
		&notification.FailedAt,
		&notification.ErrorMsg,
		&notification.RetryCount,
		&notification.CreatedAt,
		&notification.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return notification, err
}
