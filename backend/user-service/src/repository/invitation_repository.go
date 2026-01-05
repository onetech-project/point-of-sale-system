package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pos/user-service/src/models"
	"github.com/pos/user-service/src/utils"
)

type InvitationRepository struct {
	db        *sql.DB
	encryptor utils.Encryptor
}

// NewInvitationRepository creates a new repository with dependency injection (for testing)
func NewInvitationRepository(db *sql.DB, encryptor utils.Encryptor) *InvitationRepository {
	return &InvitationRepository{
		db:        db,
		encryptor: encryptor,
	}
}

// NewInvitationRepositoryWithVault creates a repository with real VaultClient (for production)
func NewInvitationRepositoryWithVault(db *sql.DB) (*InvitationRepository, error) {
	vaultClient, err := utils.NewVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize VaultClient: %w", err)
	}
	return NewInvitationRepository(db, vaultClient), nil
}

func (r *InvitationRepository) Create(ctx context.Context, invitation *models.Invitation) error {
	// Encrypt PII fields
	encryptedEmail, err := r.encryptor.Encrypt(ctx, invitation.Email)
	if err != nil {
		return fmt.Errorf("failed to encrypt email: %w", err)
	}

	encryptedToken, err := r.encryptor.Encrypt(ctx, invitation.Token)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Generate searchable hashes
	emailHash := utils.HashForSearch(invitation.Email)
	tokenHash := utils.HashForSearch(invitation.Token)

	query := `
		INSERT INTO invitations (
			id, tenant_id, email, email_hash, role, token, token_hash, status, invited_by, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		invitation.ID,
		invitation.TenantID,
		encryptedEmail,
		emailHash,
		invitation.Role,
		encryptedToken,
		tokenHash,
		invitation.Status,
		invitation.InvitedBy,
		invitation.ExpiresAt,
		invitation.CreatedAt,
		invitation.UpdatedAt,
	)

	return err
}

func (r *InvitationRepository) FindByToken(ctx context.Context, token string) (*models.Invitation, error) {
	// Use hash for efficient lookup instead of decrypting all records
	tokenHash := utils.HashForSearch(token)

	query := `
		SELECT id, tenant_id, email, role, token, status, invited_by, expires_at, accepted_at, created_at, updated_at
		FROM invitations
		WHERE token_hash = $1 AND status = $2
		LIMIT 1
	`

	invitation := &models.Invitation{}
	var acceptedAt sql.NullTime
	var encryptedEmail, encryptedToken string

	err := r.db.QueryRowContext(ctx, query, tokenHash, models.InvitationPending).Scan(
		&invitation.ID,
		&invitation.TenantID,
		&encryptedEmail,
		&invitation.Role,
		&encryptedToken,
		&invitation.Status,
		&invitation.InvitedBy,
		&invitation.ExpiresAt,
		&acceptedAt,
		&invitation.CreatedAt,
		&invitation.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	// Decrypt PII fields
	invitation.Email, err = r.encryptor.Decrypt(ctx, encryptedEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt email: %w", err)
	}

	invitation.Token, err = r.encryptor.Decrypt(ctx, encryptedToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	if acceptedAt.Valid {
		invitation.AcceptedAt = &acceptedAt.Time
	}

	return invitation, nil
}

func (r *InvitationRepository) FindByEmail(ctx context.Context, tenantID, email string) (*models.Invitation, error) {
	// Use hash for efficient lookup instead of decrypting all records
	emailHash := utils.HashForSearch(email)

	query := `
		SELECT id, tenant_id, email, role, token, status, invited_by, expires_at, accepted_at, created_at, updated_at
		FROM invitations
		WHERE tenant_id = $1 AND email_hash = $2 AND status = $3
		ORDER BY created_at DESC
		LIMIT 1
	`

	invitation := &models.Invitation{}
	var acceptedAt sql.NullTime
	var encryptedEmail, encryptedToken string

	err := r.db.QueryRowContext(ctx, query, tenantID, emailHash, models.InvitationPending).Scan(
		&invitation.ID,
		&invitation.TenantID,
		&encryptedEmail,
		&invitation.Role,
		&encryptedToken,
		&invitation.Status,
		&invitation.InvitedBy,
		&invitation.ExpiresAt,
		&acceptedAt,
		&invitation.CreatedAt,
		&invitation.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	// Decrypt PII fields
	invitation.Email, err = r.encryptor.Decrypt(ctx, encryptedEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt email: %w", err)
	}

	invitation.Token, err = r.encryptor.Decrypt(ctx, encryptedToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	if acceptedAt.Valid {
		invitation.AcceptedAt = &acceptedAt.Time
	}

	return invitation, nil
}

func (r *InvitationRepository) ListByTenant(ctx context.Context, tenantID string) ([]*models.Invitation, error) {
	query := `
		SELECT id, tenant_id, email, role, token, status, invited_by, expires_at, accepted_at, created_at, updated_at
		FROM invitations
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invitations := []*models.Invitation{}
	for rows.Next() {
		invitation := &models.Invitation{}
		var acceptedAt sql.NullTime
		var encryptedEmail, encryptedToken string

		err := rows.Scan(
			&invitation.ID,
			&invitation.TenantID,
			&encryptedEmail,
			&invitation.Role,
			&encryptedToken,
			&invitation.Status,
			&invitation.InvitedBy,
			&invitation.ExpiresAt,
			&acceptedAt,
			&invitation.CreatedAt,
			&invitation.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		// Decrypt PII fields
		invitation.Email, err = r.encryptor.Decrypt(ctx, encryptedEmail)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt email: %w", err)
		}

		invitation.Token, err = r.encryptor.Decrypt(ctx, encryptedToken)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt token: %w", err)
		}

		if acceptedAt.Valid {
			invitation.AcceptedAt = &acceptedAt.Time
		}

		invitations = append(invitations, invitation)
	}

	return invitations, nil
}

func (r *InvitationRepository) UpdateStatus(ctx context.Context, id string, status models.InvitationStatus) error {
	query := `
		UPDATE invitations
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	return err
}

func (r *InvitationRepository) MarkAccepted(ctx context.Context, id string) error {
	query := `
		UPDATE invitations
		SET status = $1, accepted_at = $2, updated_at = $3
		WHERE id = $4
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, models.InvitationAccepted, now, now, id)
	return err
}

func (r *InvitationRepository) FindByID(ctx context.Context, id string) (*models.Invitation, error) {
	query := `
		SELECT id, tenant_id, email, role, token, status, invited_by, expires_at, accepted_at, created_at, updated_at
		FROM invitations
		WHERE id = $1
	`

	invitation := &models.Invitation{}
	var acceptedAt sql.NullTime
	var encryptedEmail, encryptedToken string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&invitation.ID,
		&invitation.TenantID,
		&encryptedEmail,
		&invitation.Role,
		&encryptedToken,
		&invitation.Status,
		&invitation.InvitedBy,
		&invitation.ExpiresAt,
		&acceptedAt,
		&invitation.CreatedAt,
		&invitation.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	// Decrypt PII fields
	invitation.Email, err = r.encryptor.Decrypt(ctx, encryptedEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt email: %w", err)
	}

	invitation.Token, err = r.encryptor.Decrypt(ctx, encryptedToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	if acceptedAt.Valid {
		invitation.AcceptedAt = &acceptedAt.Time
	}

	return invitation, nil
}

func (r *InvitationRepository) UpdateToken(ctx context.Context, id, token string, expiresAt time.Time) error {
	// Encrypt the new token
	encryptedToken, err := r.encryptor.Encrypt(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Generate new hash
	tokenHash := utils.HashForSearch(token)

	query := `
		UPDATE invitations
		SET token = $1, token_hash = $2, expires_at = $3, updated_at = $4
		WHERE id = $5
	`

	_, err = r.db.ExecContext(ctx, query, encryptedToken, tokenHash, expiresAt, time.Now(), id)
	return err
}
