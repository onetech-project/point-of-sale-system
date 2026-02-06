package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pos/notification-service/src/models"
	"github.com/pos/notification-service/src/utils"
)

type NotificationRepository struct {
	db        *sql.DB
	encryptor utils.Encryptor
}

// NewNotificationRepository creates a repository with custom encryptor (for testing)
func NewNotificationRepository(db *sql.DB, encryptor utils.Encryptor) *NotificationRepository {
	return &NotificationRepository{
		db:        db,
		encryptor: encryptor,
	}
}

// NewNotificationRepositoryWithVault creates a repository with Vault encryption (production)
func NewNotificationRepositoryWithVault(db *sql.DB) (*NotificationRepository, error) {
	vaultClient, err := utils.NewVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}
	return &NotificationRepository{
		db:        db,
		encryptor: vaultClient,
	}, nil
}

// encryptSensitiveMetadata encrypts sensitive PII fields within metadata
// Sensitive fields: email, name, inviter_name, token, invitation_token, ip_address, user_agent, customer_name, customer_email, customer_phone
func (r *NotificationRepository) encryptSensitiveMetadata(ctx context.Context, metadata map[string]interface{}) (map[string]interface{}, error) {
	if metadata == nil {
		return nil, nil
	}

	// Create a copy to avoid modifying the original
	encryptedMetadata := make(map[string]interface{})
	for k, v := range metadata {
		encryptedMetadata[k] = v
	}

	// List of sensitive string fields to encrypt with their contexts
	sensitiveFields := map[string]string{
		"email":            "notification_metadata:email",
		"name":             "notification_metadata:name",
		"inviter_name":     "notification_metadata:inviter_name",
		"token":            "notification_metadata:token",
		"invitation_token": "notification_metadata:invitation_token",
		"ip_address":       "notification_metadata:ip_address",
		"user_agent":       "notification_metadata:user_agent",
		"customer_name":    "notification_metadata:customer_name",
		"customer_email":   "notification_metadata:customer_email",
		"customer_phone":   "notification_metadata:customer_phone",
	}

	for field, context := range sensitiveFields {
		if val, ok := encryptedMetadata[field].(string); ok && val != "" {
			encrypted, err := r.encryptor.EncryptWithContext(ctx, val, context)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt %s: %w", field, err)
			}
			encryptedMetadata[field] = encrypted
		}
	}

	return encryptedMetadata, nil
}

// decryptSensitiveMetadata decrypts sensitive PII fields within metadata
func (r *NotificationRepository) decryptSensitiveMetadata(ctx context.Context, metadata map[string]interface{}) (map[string]interface{}, error) {
	if metadata == nil {
		return nil, nil
	}

	// Create a copy to avoid modifying the original
	decryptedMetadata := make(map[string]interface{})
	for k, v := range metadata {
		decryptedMetadata[k] = v
	}

	// List of sensitive string fields to decrypt with their contexts
	sensitiveFields := map[string]string{
		"email":            "notification_metadata:email",
		"name":             "notification_metadata:name",
		"inviter_name":     "notification_metadata:inviter_name",
		"token":            "notification_metadata:token",
		"invitation_token": "notification_metadata:invitation_token",
		"ip_address":       "notification_metadata:ip_address",
		"user_agent":       "notification_metadata:user_agent",
		"customer_name":    "notification_metadata:customer_name",
		"customer_email":   "notification_metadata:customer_email",
		"customer_phone":   "notification_metadata:customer_phone",
	}

	for field, context := range sensitiveFields {
		if val, ok := decryptedMetadata[field].(string); ok && val != "" {
			decrypted, err := r.encryptor.DecryptWithContext(ctx, val, context)
			if err != nil {
				// If decryption fails, it might be plaintext (old data), keep as is
				continue
			}
			decryptedMetadata[field] = decrypted
		}
	}

	return decryptedMetadata, nil
}

func (r *NotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	// Encrypt PII fields with context
	encryptedRecipient, err := r.encryptor.EncryptWithContext(ctx, notification.Recipient, "notification:recipient")
	if err != nil {
		return fmt.Errorf("failed to encrypt recipient: %w", err)
	}

	encryptedBody, err := r.encryptor.EncryptWithContext(ctx, notification.Body, "notification:body")
	if err != nil {
		return fmt.Errorf("failed to encrypt body: %w", err)
	}

	// Encrypt sensitive fields in metadata
	encryptedMetadata, err := r.encryptSensitiveMetadata(ctx, notification.Metadata)
	if err != nil {
		return fmt.Errorf("failed to encrypt metadata: %w", err)
	}

	query := `
		INSERT INTO notifications (tenant_id, user_id, type, status, event_type, subject, body, recipient, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	// Extract event_type from metadata if present
	eventType := "unknown"
	if encryptedMetadata != nil {
		if et, ok := encryptedMetadata["event_type"].(string); ok {
			eventType = et
		}
	}

	// Convert metadata map to JSON
	var metadataJSON []byte
	if encryptedMetadata != nil {
		metadataJSON, err = json.Marshal(encryptedMetadata)
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
		encryptedBody,
		encryptedRecipient,
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
	var encryptedBody, encryptedRecipient string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&notification.ID,
		&notification.TenantID,
		&notification.UserID,
		&notification.Type,
		&notification.Status,
		&notification.Subject,
		&encryptedBody,
		&encryptedRecipient,
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

	if err != nil {
		return nil, fmt.Errorf("failed to find notification: %w", err)
	}

	// Decrypt PII fields with context
	notification.Body, err = r.encryptor.DecryptWithContext(ctx, encryptedBody, "notification:body")
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt body: %w", err)
	}

	notification.Recipient, err = r.encryptor.DecryptWithContext(ctx, encryptedRecipient, "notification:recipient")
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt recipient: %w", err)
	}

	// Decrypt sensitive fields in metadata
	if notification.Metadata != nil {
		notification.Metadata, err = r.decryptSensitiveMetadata(ctx, notification.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt metadata: %w", err)
		}
	}

	return notification, nil
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
	var encryptedBody, encryptedRecipient string

	err := r.db.QueryRow(query, id).Scan(
		&notification.ID,
		&notification.TenantID,
		&notification.UserID,
		&notification.Type,
		&notification.Status,
		&eventType, // Read event_type but don't store in struct
		&notification.Subject,
		&encryptedBody,
		&encryptedRecipient,
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

	// Decrypt PII fields with context
	ctx := context.Background()
	notification.Body, err = r.encryptor.DecryptWithContext(ctx, encryptedBody, "notification:body")
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt body: %w", err)
	}

	notification.Recipient, err = r.encryptor.DecryptWithContext(ctx, encryptedRecipient, "notification:recipient")
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt recipient: %w", err)
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
		var id, eventType, notifType, encryptedRecipient, subject, status string
		var sentAt, failedAt sql.NullTime
		var errorMsg, orderReference sql.NullString
		var retryCount int
		var createdAt time.Time

		err := rows.Scan(
			&id,
			&eventType,
			&notifType,
			&encryptedRecipient,
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

		// Decrypt recipient
		ctx := context.Background()
		recipient, err := r.encryptor.DecryptWithContext(ctx, encryptedRecipient, "notification:recipient")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt recipient: %w", err)
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
