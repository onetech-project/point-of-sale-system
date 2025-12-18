package repository

// verify account verification repository

import (
	"database/sql"
	"fmt"
	"time"
)

type AccountVerificationRepository struct {
	db *sql.DB
}

func NewVerifyAccountRepository(db *sql.DB) *AccountVerificationRepository {
	return &AccountVerificationRepository{db: db}
}

// Find And Update User And TenantStatus By Token
func (r *AccountVerificationRepository) FindAndUpdateUserAndTenantStatusByToken(token string, now time.Time) error {
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
	row := tx.QueryRow(query, token, now)
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
