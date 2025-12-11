package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
			  AND metadata @> jsonb_build_object('transaction_id', $2::text)
			  AND status IN ('sent', 'pending')
		)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, tenantID, transactionID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetByID retrieves a notification by ID
func (r *NotificationRepository) GetByID(id string) (*models.Notification, error) {
	query := `
		SELECT id, tenant_id, user_id, type, status, event_type, subject, body, recipient,
		       metadata, sent_at, failed_at, error_msg, retry_count, created_at, updated_at
		FROM notifications
		WHERE id = $1`

	notification := &models.Notification{}
	var metadataJSON []byte
	var eventType string

	err := r.db.QueryRow(query, id).Scan(
		&notification.ID,
		&notification.TenantID,
		&notification.UserID,
		&notification.Type,
		&notification.Status,
		&eventType, // Read event_type but don't store in struct
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
		return nil, err
	}

	// Parse metadata JSON
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &notification.Metadata); err != nil {
			return nil, err
		}
	}

	return notification, nil
}

// Update updates a notification
func (r *NotificationRepository) Update(notification *models.Notification) error {
	query := `
		UPDATE notifications
		SET status = $1, sent_at = $2, failed_at = $3, error_msg = $4, 
		    retry_count = $5, updated_at = NOW()
		WHERE id = $6`

	_, err := r.db.Exec(
		query,
		notification.Status,
		notification.SentAt,
		notification.FailedAt,
		notification.ErrorMsg,
		notification.RetryCount,
		notification.ID,
	)

	return err
}

// GetNotificationHistory retrieves notification history with filters
func (r *NotificationRepository) GetNotificationHistory(filters map[string]interface{}) ([]map[string]interface{}, error) {
	// Build query
	query := `
		SELECT id, event_type, type, recipient, subject, status,
		       sent_at, failed_at, error_msg, retry_count, created_at,
		       metadata->>'order_reference' as order_reference
		FROM notifications
		WHERE tenant_id = $1`

	args := []interface{}{filters["tenant_id"]}
	paramCount := 1

	// Add filters
	if orderRef, ok := filters["order_reference"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND metadata->>'order_reference' = $%d", paramCount)
		args = append(args, orderRef)
	}

	if status, ok := filters["status"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND status = $%d", paramCount)
		args = append(args, status)
	}

	if notifType, ok := filters["type"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND event_type LIKE $%d", paramCount)
		// Map type to event_type pattern
		typeMap := map[string]string{
			"order_staff":    "order.paid.staff%",
			"order_customer": "order.paid.customer%",
		}
		args = append(args, typeMap[notifType.(string)])
	}

	if startDate, ok := filters["start_date"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND created_at >= $%d", paramCount)
		args = append(args, startDate)
	}

	if endDate, ok := filters["end_date"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND created_at <= $%d", paramCount)
		args = append(args, endDate)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"

	if limit, ok := filters["limit"]; ok {
		paramCount++
		query += fmt.Sprintf(" LIMIT $%d", paramCount)
		args = append(args, limit)
	}

	if offset, ok := filters["offset"]; ok {
		paramCount++
		query += fmt.Sprintf(" OFFSET $%d", paramCount)
		args = append(args, offset)
	}

	// Execute query
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build results
	var notifications []map[string]interface{}

	for rows.Next() {
		var id, eventType, notifType, recipient, subject, status string
		var sentAt, failedAt sql.NullTime
		var errorMsg, orderReference sql.NullString
		var retryCount int
		var createdAt time.Time

		err := rows.Scan(
			&id,
			&eventType,
			&notifType,
			&recipient,
			&subject,
			&status,
			&sentAt,
			&failedAt,
			&errorMsg,
			&retryCount,
			&createdAt,
			&orderReference,
		)
		if err != nil {
			return nil, err
		}

		notification := map[string]interface{}{
			"id":          id,
			"event_type":  eventType,
			"type":        notifType,
			"recipient":   recipient,
			"subject":     subject,
			"status":      status,
			"retry_count": retryCount,
			"created_at":  createdAt.Format(time.RFC3339),
		}

		if sentAt.Valid {
			notification["sent_at"] = sentAt.Time.Format(time.RFC3339)
		}

		if failedAt.Valid {
			notification["failed_at"] = failedAt.Time.Format(time.RFC3339)
		}

		if errorMsg.Valid {
			notification["error_msg"] = errorMsg.String
		}

		if orderReference.Valid {
			notification["order_reference"] = orderReference.String
		}

		notifications = append(notifications, notification)
	}

	if notifications == nil {
		notifications = []map[string]interface{}{}
	}

	return notifications, nil
}

// CountNotifications counts notifications matching the filters
func (r *NotificationRepository) CountNotifications(filters map[string]interface{}) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE tenant_id = $1`

	args := []interface{}{filters["tenant_id"]}
	paramCount := 1

	// Add filters (same as GetNotificationHistory)
	if orderRef, ok := filters["order_reference"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND metadata->>'order_reference' = $%d", paramCount)
		args = append(args, orderRef)
	}

	if status, ok := filters["status"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND status = $%d", paramCount)
		args = append(args, status)
	}

	if notifType, ok := filters["type"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND event_type LIKE $%d", paramCount)
		typeMap := map[string]string{
			"order_staff":    "order.paid.staff%",
			"order_customer": "order.paid.customer%",
		}
		args = append(args, typeMap[notifType.(string)])
	}

	if startDate, ok := filters["start_date"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND created_at >= $%d", paramCount)
		args = append(args, startDate)
	}

	if endDate, ok := filters["end_date"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND created_at <= $%d", paramCount)
		args = append(args, endDate)
	}

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// QueryRows executes a query and returns the result set
func (r *NotificationRepository) QueryRows(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, query, args...)
}
