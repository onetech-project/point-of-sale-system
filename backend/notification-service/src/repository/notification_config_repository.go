package repository

import (
	"context"
	"database/sql"
	"time"
)

// NotificationConfig represents tenant-level notification settings
type NotificationConfig struct {
	ID                        string    `db:"id"`
	TenantID                  string    `db:"tenant_id"`
	OrderNotificationsEnabled bool      `db:"order_notifications_enabled"`
	TestMode                  bool      `db:"test_mode"`
	TestEmail                 *string   `db:"test_email"`
	CreatedAt                 time.Time `db:"created_at"`
	UpdatedAt                 time.Time `db:"updated_at"`
}

// NotificationConfigRepository manages notification configuration data
type NotificationConfigRepository struct {
	db *sql.DB
}

// NewNotificationConfigRepository creates a new NotificationConfigRepository
func NewNotificationConfigRepository(db *sql.DB) *NotificationConfigRepository {
	return &NotificationConfigRepository{db: db}
}

// GetByTenantID retrieves notification config for a tenant
func (r *NotificationConfigRepository) GetByTenantID(ctx context.Context, tenantID string) (*NotificationConfig, error) {
	query := `
		SELECT id, tenant_id, order_notifications_enabled, test_mode, test_email, created_at, updated_at
		FROM notification_configs
		WHERE tenant_id = $1
	`

	var config NotificationConfig
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(
		&config.ID,
		&config.TenantID,
		&config.OrderNotificationsEnabled,
		&config.TestMode,
		&config.TestEmail,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return default config if none exists
		return &NotificationConfig{
			TenantID:                  tenantID,
			OrderNotificationsEnabled: true,
			TestMode:                  false,
			TestEmail:                 nil,
		}, nil
	}

	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Create creates a new notification config
func (r *NotificationConfigRepository) Create(ctx context.Context, config *NotificationConfig) error {
	query := `
		INSERT INTO notification_configs (tenant_id, order_notifications_enabled, test_mode, test_email)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx,
		query,
		config.TenantID,
		config.OrderNotificationsEnabled,
		config.TestMode,
		config.TestEmail,
	).Scan(&config.ID, &config.CreatedAt, &config.UpdatedAt)
}

// Update updates an existing notification config
func (r *NotificationConfigRepository) Update(ctx context.Context, config *NotificationConfig) error {
	query := `
		UPDATE notification_configs
		SET order_notifications_enabled = $1, test_mode = $2, test_email = $3, updated_at = NOW()
		WHERE tenant_id = $4
		RETURNING updated_at
	`

	return r.db.QueryRowContext(
		ctx,
		query,
		config.OrderNotificationsEnabled,
		config.TestMode,
		config.TestEmail,
		config.TenantID,
	).Scan(&config.UpdatedAt)
}
