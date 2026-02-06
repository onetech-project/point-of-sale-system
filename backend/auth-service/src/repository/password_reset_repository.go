package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pos/auth-service/src/models"
	"github.com/pos/auth-service/src/utils"
)

type PasswordResetRepository struct {
	db        *sql.DB
	encryptor utils.Encryptor
}

// NewPasswordResetRepository creates a new repository with dependency injection (for testing)
func NewPasswordResetRepository(db *sql.DB, encryptor utils.Encryptor) *PasswordResetRepository {
	return &PasswordResetRepository{
		db:        db,
		encryptor: encryptor,
	}
}

// NewPasswordResetRepositoryWithVault creates a repository with real VaultClient (for production)
func NewPasswordResetRepositoryWithVault(db *sql.DB) (*PasswordResetRepository, error) {
	vaultClient, err := utils.NewVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize VaultClient: %w", err)
	}
	return NewPasswordResetRepository(db, vaultClient), nil
}

func (r *PasswordResetRepository) Create(token *models.PasswordResetToken) error {
	// Encrypt token with context before storing
	ctx := context.Background()
	encryptedToken, err := r.encryptor.EncryptWithContext(ctx, token.Token, "reset_token:token")
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	query := `INSERT INTO password_reset_tokens (user_id, tenant_id, token, expires_at) 
	          VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	return r.db.QueryRow(query, token.UserID, token.TenantID, encryptedToken, token.ExpiresAt).
		Scan(&token.ID, &token.CreatedAt)
}

func (r *PasswordResetRepository) FindByToken(token string) (*models.PasswordResetToken, error) {
	// Encrypt the search token with context for deterministic encryption
	ctx := context.Background()
	encryptedToken, err := r.encryptor.EncryptWithContext(ctx, token, "reset_token:token")
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt search token: %w", err)
	}

	var resetToken models.PasswordResetToken
	var storedEncryptedToken string
	query := `SELECT id, user_id, tenant_id, token, expires_at, used_at, created_at 
	          FROM password_reset_tokens WHERE token = $1`
	err = r.db.QueryRow(query, encryptedToken).Scan(
		&resetToken.ID,
		&resetToken.UserID,
		&resetToken.TenantID,
		&storedEncryptedToken,
		&resetToken.ExpiresAt,
		&resetToken.UsedAt,
		&resetToken.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Decrypt the token with context for use in the application
	resetToken.Token, err = r.encryptor.DecryptWithContext(ctx, storedEncryptedToken, "reset_token:token")
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	return &resetToken, nil
}

func (r *PasswordResetRepository) MarkAsUsed(tokenID uuid.UUID) error {
	query := `UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(query, tokenID)
	return err
}

func (r *PasswordResetRepository) DeleteExpired() error {
	query := `DELETE FROM password_reset_tokens WHERE expires_at < NOW() OR used_at IS NOT NULL`
	_, err := r.db.Exec(query)
	return err
}

func (r *PasswordResetRepository) CountRecentRequests(userID uuid.UUID, tenantID uuid.UUID, since time.Time) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM password_reset_tokens 
	          WHERE user_id = $1 AND tenant_id = $2 AND created_at > $3`
	err := r.db.QueryRow(query, userID, tenantID, since).Scan(&count)
	return count, err
}
