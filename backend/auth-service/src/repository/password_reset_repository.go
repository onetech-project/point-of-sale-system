package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/pos/auth-service/src/models"
)

type PasswordResetRepository struct {
	db *sql.DB
}

func NewPasswordResetRepository(db *sql.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

func (r *PasswordResetRepository) Create(token *models.PasswordResetToken) error {
	query := `INSERT INTO password_reset_tokens (user_id, tenant_id, token, expires_at) 
	          VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	return r.db.QueryRow(query, token.UserID, token.TenantID, token.Token, token.ExpiresAt).
		Scan(&token.ID, &token.CreatedAt)
}

func (r *PasswordResetRepository) FindByToken(token string) (*models.PasswordResetToken, error) {
	var resetToken models.PasswordResetToken
	query := `SELECT id, user_id, tenant_id, token, expires_at, used_at, created_at 
	          FROM password_reset_tokens WHERE token = $1`
	err := r.db.QueryRow(query, token).Scan(
		&resetToken.ID,
		&resetToken.UserID,
		&resetToken.TenantID,
		&resetToken.Token,
		&resetToken.ExpiresAt,
		&resetToken.UsedAt,
		&resetToken.CreatedAt,
	)
	if err != nil {
		return nil, err
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
