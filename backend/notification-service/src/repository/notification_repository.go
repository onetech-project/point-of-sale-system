package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/pos/notification-service/src/models"
)

type NotificationRepository struct {
	db *sql.DB
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

	return r.db.QueryRowContext(
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
	).Scan(&notification.ID, &notification.CreatedAt, &notification.UpdatedAt)
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

// HasSentOrderNotification checks if a notification has already been sent for a given transaction_id
// This prevents duplicate notifications for the same order payment
func (r *NotificationRepository) HasSentOrderNotification(ctx context.Context, tenantID, transactionID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM notifications
			WHERE tenant_id = $1
			  AND event_type LIKE 'order.paid%'
			  AND metadata @> jsonb_build_object('transaction_id', $2)
			  AND status IN ('sent', 'pending')
		)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, tenantID, transactionID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// QueryRows executes a query and returns the result set
func (r *NotificationRepository) QueryRows(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, query, args...)
}
