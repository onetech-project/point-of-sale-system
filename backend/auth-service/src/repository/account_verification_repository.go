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

type PendingVerificationAccount struct {
	UserID            string
	TenantID          string
	FirstName         string
	LastName          string
	VerificationToken string
}

func NewAccountVerificationRepository(db *sql.DB, encryptor utils.Encryptor) *AccountVerificationRepository {
	return &AccountVerificationRepository{
		db:        db,
		encryptor: encryptor,
	}
}

func NewVerifyAccountRepository(db *sql.DB) *AccountVerificationRepository {
	vaultClient, err := utils.NewVaultClient()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize Vault client for account verification: %v", err))
	}
	return NewAccountVerificationRepository(db, vaultClient)
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

func (r *AccountVerificationRepository) PrepareVerificationResend(ctx context.Context, email string, now time.Time, newToken string, newExpiresAt time.Time) (*PendingVerificationAccount, error) {
	encryptedEmail, err := r.encryptor.EncryptWithContext(ctx, email, "user:email")
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt email for verification lookup: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		SELECT u.id, u.tenant_id, u.first_name, u.last_name, u.verification_token, u.verification_token_expires_at
		FROM users u
		JOIN tenants t ON t.id = u.tenant_id
		WHERE u.email = $1
			AND u.role = 'owner'
			AND u.email_verified = FALSE
			AND u.status = 'inactive'
			AND t.status = 'inactive'
		ORDER BY u.created_at DESC
		LIMIT 1
		FOR UPDATE OF u
	`

	account := &PendingVerificationAccount{}
	var encryptedFirstName, encryptedLastName, encryptedExistingToken sql.NullString
	var existingTokenExpiresAt sql.NullTime
	err = tx.QueryRowContext(ctx, query, encryptedEmail).Scan(
		&account.UserID,
		&account.TenantID,
		&encryptedFirstName,
		&encryptedLastName,
		&encryptedExistingToken,
		&existingTokenExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	tokenIsReusable := encryptedExistingToken.Valid &&
		encryptedExistingToken.String != "" &&
		existingTokenExpiresAt.Valid &&
		existingTokenExpiresAt.Time.After(now)

	if tokenIsReusable {
		account.VerificationToken, err = r.encryptor.DecryptWithContext(ctx, encryptedExistingToken.String, "verification_token:token")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt verification token: %w", err)
		}
	} else {
		encryptedNewToken, err := r.encryptor.EncryptWithContext(ctx, newToken, "verification_token:token")
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt verification token: %w", err)
		}
		updateTokenQuery := `
			UPDATE users
			SET verification_token = $1, verification_token_expires_at = $2, updated_at = NOW()
			WHERE id = $3 AND email_verified = FALSE
		`
		if _, err := tx.ExecContext(ctx, updateTokenQuery, encryptedNewToken, newExpiresAt, account.UserID); err != nil {
			return nil, err
		}
		account.VerificationToken = newToken
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if encryptedFirstName.Valid && encryptedFirstName.String != "" {
		account.FirstName, err = r.encryptor.DecryptWithContext(ctx, encryptedFirstName.String, "user:first_name")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt first name: %w", err)
		}
	}
	if encryptedLastName.Valid && encryptedLastName.String != "" {
		account.LastName, err = r.encryptor.DecryptWithContext(ctx, encryptedLastName.String, "user:last_name")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt last name: %w", err)
		}
	}

	return account, nil
}
