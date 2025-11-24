package repository

import (
	"context"
	"database/sql"
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
		INSERT INTO notifications (tenant_id, user_id, type, status, subject, body, recipient, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(
		ctx,
		query,
		notification.TenantID,
		notification.UserID,
		notification.Type,
		notification.Status,
		notification.Subject,
		notification.Body,
		notification.Recipient,
		notification.Metadata,
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
