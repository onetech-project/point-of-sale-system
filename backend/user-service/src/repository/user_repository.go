package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/pos/user-service/src/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, tenant_id, email, password_hash, role, status, first_name, last_name, locale, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	if user.Status == "" {
		user.Status = string(models.UserStatusActive)
	}

	if user.Locale == "" {
		user.Locale = "en"
	}

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.TenantID,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.Status,
		user.FirstName,
		user.LastName,
		user.Locale,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

func (r *UserRepository) FindByEmail(ctx context.Context, tenantID, email string) (*models.User, error) {
	query := `
		SELECT id, tenant_id, email, password_hash, role, status, first_name, last_name, locale, last_login_at, created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND email = $2 AND status != 'deleted'
	`

	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, tenantID, email).Scan(
		&user.ID,
		&user.TenantID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.FirstName,
		&user.LastName,
		&user.Locale,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, tenantID, id string) (*models.User, error) {
	query := `
		SELECT id, tenant_id, email, password_hash, role, status, first_name, last_name, locale, last_login_at, created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND id = $2 AND status != 'deleted'
	`

	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, tenantID, id).Scan(
		&user.ID,
		&user.TenantID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.FirstName,
		&user.LastName,
		&user.Locale,
		&user.LastLoginAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET email = $1, role = $2, status = $3, first_name = $4, last_name = $5, locale = $6, last_login_at = $7, updated_at = $8
		WHERE tenant_id = $9 AND id = $10
	`

	user.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		user.Email,
		user.Role,
		user.Status,
		user.FirstName,
		user.LastName,
		user.Locale,
		user.LastLoginAt,
		user.UpdatedAt,
		user.TenantID,
		user.ID,
	)

	return err
}

// FindStaffWithOrderNotifications retrieves all active staff users who have opted in to receive order notifications
func (r *UserRepository) FindStaffWithOrderNotifications(ctx context.Context, tenantID string) ([]*models.User, error) {
	query := `
		SELECT id, tenant_id, email, password_hash, role, status, first_name, last_name, locale, last_login_at, created_at, updated_at
		FROM users
		WHERE tenant_id = $1 
		  AND status = 'active'
		  AND role IN ('admin', 'staff')
		  AND receive_order_notifications = true
		ORDER BY email
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*models.User{}
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.TenantID,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.Status,
			&user.FirstName,
			&user.LastName,
			&user.Locale,
			&user.LastLoginAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
