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
	db             *sql.DB
	encryptor      utils.Encryptor
	auditPublisher utils.AuditPublisherInterface
}

// NewUserRepository creates a new UserRepository with injected dependencies
// The encryptor parameter enables dependency injection for testing (mock Encryptor)
func NewUserRepository(db *sql.DB, encryptor utils.Encryptor, auditPublisher utils.AuditPublisherInterface) *UserRepository {
	return &UserRepository{
		db:             db,
		encryptor:      encryptor,
		auditPublisher: auditPublisher,
	}
}

// NewUserRepositoryWithVault is a convenience constructor that creates a UserRepository
// with a real VaultClient. Use this in production code.
func NewUserRepositoryWithVault(db *sql.DB, auditPublisher utils.AuditPublisherInterface) (*UserRepository, error) {
	vaultClient, err := utils.NewVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize VaultClient: %w", err)
	}
	return NewUserRepository(db, vaultClient, auditPublisher), nil
}

// Helper functions for pointer field encryption/decryption
func (r *UserRepository) encryptStringPtr(ctx context.Context, value *string) (string, error) {
	if value == nil || *value == "" {
		return "", nil
	}
	return r.encryptor.Encrypt(ctx, *value)
}

func (r *UserRepository) encryptStringPtrWithContext(ctx context.Context, value *string, encryptionContext string) (string, error) {
	if value == nil || *value == "" {
		return "", nil
	}
	return r.encryptor.EncryptWithContext(ctx, *value, encryptionContext)
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

func (r *UserRepository) decryptToStringPtrWithContext(ctx context.Context, encrypted string, encryptionContext string) (*string, error) {
	if encrypted == "" {
		return nil, nil
	}
	decrypted, err := r.encryptor.DecryptWithContext(ctx, encrypted, encryptionContext)
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

// DecryptFieldWithContext decrypts a field with the specified context
func (r *UserRepository) DecryptFieldWithContext(ctx context.Context, encrypted string, encryptionContext string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	return r.encryptor.DecryptWithContext(ctx, encrypted, encryptionContext)
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

	// Encrypt PII fields with context for deterministic encryption (Phase 2)
	encryptedEmail, err := r.encryptor.EncryptWithContext(ctx, user.Email, "user:email")
	if err != nil {
		return err
	}

	encryptedFirstName, err := r.encryptStringPtrWithContext(ctx, user.FirstName, "user:first_name")
	if err != nil {
		return err
	}
	encryptedLastName, err := r.encryptStringPtrWithContext(ctx, user.LastName, "user:last_name")
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query,
		user.ID,
		user.TenantID,
		encryptedEmail,
		user.PasswordHash,
		user.Role,
		user.Status,
		encryptedFirstName,
		encryptedLastName,
		user.Locale,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return err
	}

	// T098: Publish UserCreatedEvent to audit trail
	if r.auditPublisher != nil {
		afterValue := map[string]interface{}{
			"email":      encryptedEmail,
			"first_name": encryptedFirstName,
			"last_name":  encryptedLastName,
			"role":       user.Role,
			"status":     user.Status,
		}

		auditEvent := &utils.AuditEvent{
			TenantID:     user.TenantID,
			ActorType:    "system",
			Action:       "CREATE",
			ResourceType: "user",
			ResourceID:   user.ID,
			AfterValue:   afterValue,
			Metadata: map[string]interface{}{
				"locale": user.Locale,
			},
		}

		if err := r.auditPublisher.Publish(ctx, auditEvent); err != nil {
			// Log error but don't fail the operation
			// Audit publishing is non-blocking
			fmt.Printf("Failed to publish user create audit event: %v\n", err)
		}
	}

	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, tenantID, email string) (*models.User, error) {
	query := `
		SELECT id, tenant_id, email, password_hash, role, status, first_name, last_name, locale, last_login_at, created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND email = $2 AND status != 'deleted'
	`

	// Encrypt email with context for deterministic lookup (Phase 2)
	encryptedEmail, err := r.encryptor.EncryptWithContext(ctx, email, "user:email")
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

	// Decrypt PII fields with context (Phase 2)
	user.Email, err = r.encryptor.DecryptWithContext(ctx, encryptedEmailDB, "user:email")
	if err != nil {
		return nil, err
	}
	user.FirstName, err = r.decryptToStringPtrWithContext(ctx, encryptedFirstNameDB, "user:first_name")
	if err != nil {
		return nil, err
	}
	user.LastName, err = r.decryptToStringPtrWithContext(ctx, encryptedLastNameDB, "user:last_name")
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

	// Decrypt PII fields with context (Phase 2)
	user.Email, err = r.encryptor.DecryptWithContext(ctx, encryptedEmailDB, "user:email")
	if err != nil {
		return nil, err
	}
	user.FirstName, err = r.decryptToStringPtrWithContext(ctx, encryptedFirstNameDB, "user:first_name")
	if err != nil {
		return nil, err
	}
	user.LastName, err = r.decryptToStringPtrWithContext(ctx, encryptedLastNameDB, "user:last_name")
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	// T099: Fetch before value for audit trail
	var beforeValue map[string]interface{}
	if r.auditPublisher != nil {
		existingUser, err := r.FindByID(ctx, user.TenantID, user.ID)
		if err == nil && existingUser != nil {
			// Store encrypted values for audit
			encEmail, _ := r.encryptor.EncryptWithContext(ctx, existingUser.Email, "user:email")
			encFirstName, _ := r.encryptStringPtrWithContext(ctx, existingUser.FirstName, "user:first_name")
			encLastName, _ := r.encryptStringPtrWithContext(ctx, existingUser.LastName, "user:last_name")

			beforeValue = map[string]interface{}{
				"email":      encEmail,
				"first_name": encFirstName,
				"last_name":  encLastName,
				"role":       existingUser.Role,
				"status":     existingUser.Status,
			}
		}
	}

	query := `
		UPDATE users
		SET email = $1, role = $2, status = $3, first_name = $4, last_name = $5, locale = $6, last_login_at = $7, updated_at = $8
		WHERE tenant_id = $9 AND id = $10
	`

	user.UpdatedAt = time.Now()

	// Encrypt PII fields with context (Phase 2)
	encryptedEmail, err := r.encryptor.EncryptWithContext(ctx, user.Email, "user:email")
	if err != nil {
		return err
	}
	encryptedFirstName, err := r.encryptStringPtrWithContext(ctx, user.FirstName, "user:first_name")
	if err != nil {
		return err
	}
	encryptedLastName, err := r.encryptStringPtrWithContext(ctx, user.LastName, "user:last_name")
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

	if err != nil {
		return err
	}

	// T099: Publish UserUpdatedEvent to audit trail
	if r.auditPublisher != nil {
		afterValue := map[string]interface{}{
			"email":      encryptedEmail,
			"first_name": encryptedFirstName,
			"last_name":  encryptedLastName,
			"role":       user.Role,
			"status":     user.Status,
		}

		auditEvent := &utils.AuditEvent{
			TenantID:     user.TenantID,
			ActorType:    "system",
			Action:       "UPDATE",
			ResourceType: "user",
			ResourceID:   user.ID,
			BeforeValue:  beforeValue,
			AfterValue:   afterValue,
			Metadata: map[string]interface{}{
				"locale": user.Locale,
			},
		}

		if err := r.auditPublisher.Publish(ctx, auditEvent); err != nil {
			fmt.Printf("Failed to publish user update audit event: %v\n", err)
		}
	}

	return nil
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

		// Decrypt PII fields with context (Phase 2)
		user.Email, err = r.encryptor.DecryptWithContext(ctx, encryptedEmailDB, "user:email")
		if err != nil {
			return nil, err
		}
		user.FirstName, err = r.decryptToStringPtrWithContext(ctx, encryptedFirstNameDB, "user:first_name")
		if err != nil {
			return nil, err
		}
		user.LastName, err = r.decryptToStringPtrWithContext(ctx, encryptedLastNameDB, "user:last_name")
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

// Delete soft deletes a user by setting status to 'deleted' (T100)
func (r *UserRepository) Delete(ctx context.Context, tenantID, userID string, deleteType string) error {
	// Fetch before value for audit trail
	var beforeValue map[string]interface{}
	if r.auditPublisher != nil {
		existingUser, err := r.FindByID(ctx, tenantID, userID)
		if err == nil && existingUser != nil {
			encEmail, _ := r.encryptor.EncryptWithContext(ctx, existingUser.Email, "user:email")
			encFirstName, _ := r.encryptStringPtrWithContext(ctx, existingUser.FirstName, "user:first_name")
			encLastName, _ := r.encryptStringPtrWithContext(ctx, existingUser.LastName, "user:last_name")

			beforeValue = map[string]interface{}{
				"email":      encEmail,
				"first_name": encFirstName,
				"last_name":  encLastName,
				"role":       existingUser.Role,
				"status":     existingUser.Status,
			}
		}
	}

	query := `
		UPDATE users
		SET status = 'deleted', updated_at = $1
		WHERE tenant_id = $2 AND id = $3 AND status != 'deleted'
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, tenantID, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found or already deleted")
	}

	// T100: Publish UserDeletedEvent to audit trail
	if r.auditPublisher != nil {
		auditEvent := &utils.AuditEvent{
			TenantID:     tenantID,
			ActorType:    "system",
			Action:       "DELETE",
			ResourceType: "user",
			ResourceID:   userID,
			BeforeValue:  beforeValue,
			Metadata: map[string]interface{}{
				"delete_type": deleteType, // "soft" or "hard"
			},
		}

		if err := r.auditPublisher.Publish(ctx, auditEvent); err != nil {
			fmt.Printf("Failed to publish user delete audit event: %v\n", err)
		}
	}

	return nil
}
