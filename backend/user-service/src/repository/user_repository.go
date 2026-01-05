package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pos/user-service/src/models"
	"github.com/pos/user-service/src/utils"
)

type UserRepository struct {
	db        *sql.DB
	encryptor utils.Encryptor
}

// NewUserRepository creates a new UserRepository with injected dependencies
// The encryptor parameter enables dependency injection for testing (mock Encryptor)
func NewUserRepository(db *sql.DB, encryptor utils.Encryptor) *UserRepository {
	return &UserRepository{
		db:        db,
		encryptor: encryptor,
	}
}

// NewUserRepositoryWithVault is a convenience constructor that creates a UserRepository
// with a real VaultClient. Use this in production code.
func NewUserRepositoryWithVault(db *sql.DB) (*UserRepository, error) {
	vaultClient, err := utils.NewVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize VaultClient: %w", err)
	}
	return NewUserRepository(db, vaultClient), nil
}

// Helper functions for pointer field encryption/decryption
func (r *UserRepository) encryptStringPtr(ctx context.Context, value *string) (string, error) {
	if value == nil || *value == "" {
		return "", nil
	}
	return r.encryptor.Encrypt(ctx, *value)
}

func (r *UserRepository) decryptToStringPtr(ctx context.Context, encrypted string) (*string, error) {
	if encrypted == "" {
		return nil, nil
	}
	decrypted, err := r.encryptor.Decrypt(ctx, encrypted)
	if err != nil {
		return nil, err
	}
	return &decrypted, nil
}

// DecryptField decrypts a single encrypted field value
func (r *UserRepository) DecryptField(ctx context.Context, encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	return r.encryptor.Decrypt(ctx, encrypted)
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, tenant_id, email, email_hash, password_hash, role, status, first_name, last_name, locale, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
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

	// Encrypt PII fields (T050 - UU PDP compliance)
	encryptedEmail, err := r.encryptor.Encrypt(ctx, user.Email)
	if err != nil {
		return err
	}

	// Generate searchable hash for email (T051 - efficient encrypted field search)
	emailHash := utils.HashForSearch(user.Email)

	encryptedFirstName, err := r.encryptStringPtr(ctx, user.FirstName)
	if err != nil {
		return err
	}
	encryptedLastName, err := r.encryptStringPtr(ctx, user.LastName)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query,
		user.ID,
		user.TenantID,
		encryptedEmail,
		emailHash,
		user.PasswordHash,
		user.Role,
		user.Status,
		encryptedFirstName,
		encryptedLastName,
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

	// Encrypt email for lookup (T051 - encrypted data comparison)
	encryptedEmail, err := r.encryptor.Encrypt(ctx, email)
	if err != nil {
		return nil, err
	}

	user := &models.User{}
	var encryptedEmailDB, encryptedFirstNameDB, encryptedLastNameDB string

	err = r.db.QueryRowContext(ctx, query, tenantID, encryptedEmail).Scan(
		&user.ID,
		&user.TenantID,
		&encryptedEmailDB,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&encryptedFirstNameDB,
		&encryptedLastNameDB,
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

	// Decrypt PII fields (T051 - transparent decryption)
	user.Email, err = r.encryptor.Decrypt(ctx, encryptedEmailDB)
	if err != nil {
		return nil, err
	}
	user.FirstName, err = r.decryptToStringPtr(ctx, encryptedFirstNameDB)
	if err != nil {
		return nil, err
	}
	user.LastName, err = r.decryptToStringPtr(ctx, encryptedLastNameDB)
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
	var encryptedEmailDB, encryptedFirstNameDB, encryptedLastNameDB string

	err := r.db.QueryRowContext(ctx, query, tenantID, id).Scan(
		&user.ID,
		&user.TenantID,
		&encryptedEmailDB,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&encryptedFirstNameDB,
		&encryptedLastNameDB,
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

	// Decrypt PII fields (T051 - transparent decryption)
	user.Email, err = r.encryptor.Decrypt(ctx, encryptedEmailDB)
	if err != nil {
		return nil, err
	}
	user.FirstName, err = r.decryptToStringPtr(ctx, encryptedFirstNameDB)
	if err != nil {
		return nil, err
	}
	user.LastName, err = r.decryptToStringPtr(ctx, encryptedLastNameDB)
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

	// Encrypt PII fields (T050 - UU PDP compliance)
	encryptedEmail, err := r.encryptor.Encrypt(ctx, user.Email)
	if err != nil {
		return err
	}
	encryptedFirstName, err := r.encryptStringPtr(ctx, user.FirstName)
	if err != nil {
		return err
	}
	encryptedLastName, err := r.encryptStringPtr(ctx, user.LastName)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query,
		encryptedEmail,
		user.Role,
		user.Status,
		encryptedFirstName,
		encryptedLastName,
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
		var encryptedEmailDB, encryptedFirstNameDB, encryptedLastNameDB string

		err := rows.Scan(
			&user.ID,
			&user.TenantID,
			&encryptedEmailDB,
			&user.PasswordHash,
			&user.Role,
			&user.Status,
			&encryptedFirstNameDB,
			&encryptedLastNameDB,
			&user.Locale,
			&user.LastLoginAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Decrypt PII fields (T051 - transparent decryption)
		user.Email, err = r.encryptor.Decrypt(ctx, encryptedEmailDB)
		if err != nil {
			return nil, err
		}
		user.FirstName, err = r.decryptToStringPtr(ctx, encryptedFirstNameDB)
		if err != nil {
			return nil, err
		}
		user.LastName, err = r.decryptToStringPtr(ctx, encryptedLastNameDB)
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
