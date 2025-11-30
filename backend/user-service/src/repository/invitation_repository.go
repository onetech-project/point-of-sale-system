package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/pos/user-service/src/models"
)

type InvitationRepository struct {
	db *sql.DB
}

func NewInvitationRepository(db *sql.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

func (r *InvitationRepository) Create(ctx context.Context, invitation *models.Invitation) error {
	query := `
		INSERT INTO invitations (
			id, tenant_id, email, role, token, status, invited_by, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		invitation.ID,
		invitation.TenantID,
		invitation.Email,
		invitation.Role,
		invitation.Token,
		invitation.Status,
		invitation.InvitedBy,
		invitation.ExpiresAt,
		invitation.CreatedAt,
		invitation.UpdatedAt,
	)

	return err
}

func (r *InvitationRepository) FindByToken(ctx context.Context, token string) (*models.Invitation, error) {
	query := `
		SELECT id, tenant_id, email, role, token, status, invited_by, expires_at, accepted_at, created_at, updated_at
		FROM invitations
		WHERE token = $1
	`

	invitation := &models.Invitation{}
	var acceptedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&invitation.ID,
		&invitation.TenantID,
		&invitation.Email,
		&invitation.Role,
		&invitation.Token,
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

	if acceptedAt.Valid {
		invitation.AcceptedAt = &acceptedAt.Time
	}

	return invitation, nil
}

func (r *InvitationRepository) FindByEmail(ctx context.Context, tenantID, email string) (*models.Invitation, error) {
	query := `
		SELECT id, tenant_id, email, role, token, status, invited_by, expires_at, accepted_at, created_at, updated_at
		FROM invitations
		WHERE tenant_id = $1 AND email = $2 AND status = $3
		ORDER BY created_at DESC
		LIMIT 1
	`

	invitation := &models.Invitation{}
	var acceptedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, tenantID, email, models.InvitationPending).Scan(
		&invitation.ID,
		&invitation.TenantID,
		&invitation.Email,
		&invitation.Role,
		&invitation.Token,
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

		err := rows.Scan(
			&invitation.ID,
			&invitation.TenantID,
			&invitation.Email,
			&invitation.Role,
			&invitation.Token,
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
