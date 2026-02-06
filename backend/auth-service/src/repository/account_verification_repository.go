package repository

// verify account verification repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pos/auth-service/src/utils"
)

type AccountVerificationRepository struct {
	db        *sql.DB
	encryptor utils.Encryptor
}

func NewVerifyAccountRepository(db *sql.DB) *AccountVerificationRepository {
	vaultClient, err := utils.NewVaultClient()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize Vault client for account verification: %v", err))
	}
	return &AccountVerificationRepository{
		db:        db,
		encryptor: vaultClient,
	}
}

// Find And Update User And TenantStatus By Token
func (r *AccountVerificationRepository) FindAndUpdateUserAndTenantStatusByToken(token string, now time.Time) error {
	// Encrypt token for database lookup (deterministic encryption)
	ctx := context.Background()
	encryptedToken, err := r.encryptor.EncryptWithContext(ctx, token, "verification_token:token")
	if err != nil {
		return fmt.Errorf("failed to encrypt verification token: %w", err)
	}

	var id string
	var tenantID string

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		SELECT id, tenant_id
		FROM users
		WHERE verification_token = $1 AND verification_token_expires_at > $2 AND email_verified = FALSE
		FOR UPDATE
	`
	row := tx.QueryRow(query, encryptedToken, now)
	if err := row.Scan(&id, &tenantID); err != nil {
		fmt.Printf("DEBUG: error check user by token, verification_token_expires_at, and email_verified %v\n", err)
		return fmt.Errorf("invalid or expired token")
	}

	updateUserStatus := `
		UPDATE users
		SET email_verified = true, status = $1, verification_token = NULL, verification_token_expires_at = NULL, updated_at = NOW()
		WHERE id = $2
	`
	if _, err := tx.Exec(updateUserStatus, "active", id); err != nil {
		return err
	}

	updateTenantStatus := `
		UPDATE tenants
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`
	if _, err := tx.Exec(updateTenantStatus, "active", tenantID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
