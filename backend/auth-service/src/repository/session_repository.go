package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pos/auth-service/src/models"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session record in PostgreSQL
func (r *SessionRepository) Create(ctx context.Context, session *models.Session) error {
	query := `
		INSERT INTO sessions (id, session_id, tenant_id, user_id, ip_address, user_agent, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, query,
		session.ID,
		session.SessionID,
		session.TenantID,
		session.UserID,
		session.IPAddress,
		session.UserAgent,
		session.ExpiresAt,
		session.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// FindByID finds a session by session ID
func (r *SessionRepository) FindByID(ctx context.Context, sessionID string) (*models.Session, error) {
	query := `
		SELECT id, session_id, tenant_id, user_id, ip_address, user_agent, 
		       expires_at, terminated_at, created_at
		FROM sessions
		WHERE session_id = $1
	`

	session := &models.Session{}
	var ipAddress, userAgent sql.NullString
	var terminatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID,
		&session.SessionID,
		&session.TenantID,
		&session.UserID,
		&ipAddress,
		&userAgent,
		&session.ExpiresAt,
		&terminatedAt,
		&session.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	if ipAddress.Valid {
		session.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		session.UserAgent = userAgent.String
	}
	if terminatedAt.Valid {
		session.TerminatedAt = &terminatedAt.Time
	}

	return session, nil
}

// Delete marks a session as terminated
func (r *SessionRepository) Delete(ctx context.Context, sessionID string) error {
	query := `
		UPDATE sessions
		SET terminated_at = $1
		WHERE session_id = $2 AND terminated_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("session not found or already terminated")
	}

	return nil
}

// DeleteExpired deletes expired sessions older than the retention period
func (r *SessionRepository) DeleteExpired(ctx context.Context, retentionDays int) (int64, error) {
	query := `
		DELETE FROM sessions
		WHERE created_at < $1
	`

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to check rows affected: %w", err)
	}

	return rows, nil
}

// FindByUserID finds all active sessions for a user
func (r *SessionRepository) FindByUserID(ctx context.Context, userID string) ([]*models.Session, error) {
	query := `
		SELECT id, session_id, tenant_id, user_id, ip_address, user_agent,
		       expires_at, terminated_at, created_at
		FROM sessions
		WHERE user_id = $1 AND terminated_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find sessions by user: %w", err)
	}
	defer rows.Close()

	var sessions []*models.Session

	for rows.Next() {
		session := &models.Session{}
		var ipAddress, userAgent sql.NullString
		var terminatedAt sql.NullTime

		err := rows.Scan(
			&session.ID,
			&session.SessionID,
			&session.TenantID,
			&session.UserID,
			&ipAddress,
			&userAgent,
			&session.ExpiresAt,
			&terminatedAt,
			&session.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		if ipAddress.Valid {
			session.IPAddress = ipAddress.String
		}
		if userAgent.Valid {
			session.UserAgent = userAgent.String
		}
		if terminatedAt.Valid {
			session.TerminatedAt = &terminatedAt.Time
		}

		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}
