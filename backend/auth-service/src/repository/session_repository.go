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

type SessionRepository struct {
	db        *sql.DB
	encryptor utils.Encryptor
}

// NewSessionRepository creates a new repository with dependency injection (for testing)
func NewSessionRepository(db *sql.DB, encryptor utils.Encryptor) *SessionRepository {
	return &SessionRepository{
		db:        db,
		encryptor: encryptor,
	}
}

// NewSessionRepositoryWithVault creates a repository with real VaultClient (for production)
func NewSessionRepositoryWithVault(db *sql.DB) (*SessionRepository, error) {
	vaultClient, err := utils.NewVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize VaultClient: %w", err)
	}
	return NewSessionRepository(db, vaultClient), nil
}

// encryptStringPtr encrypts a pointer to string (handles nil values)
func (r *SessionRepository) encryptStringPtr(ctx context.Context, value *string) (string, error) {
	if value == nil || *value == "" {
		return "", nil
	}
	return r.encryptor.Encrypt(ctx, *value)
}

// decryptToStringPtr decrypts to a pointer to string (handles empty values)
func (r *SessionRepository) decryptToStringPtr(ctx context.Context, encrypted string) (*string, error) {
	if encrypted == "" {
		return nil, nil
	}
	decrypted, err := r.encryptor.Decrypt(ctx, encrypted)
	if err != nil {
		return nil, err
	}
	return &decrypted, nil
}

// Create creates a new session record in PostgreSQL with encrypted PII
func (r *SessionRepository) Create(ctx context.Context, session *models.Session) error {
	// Encrypt PII fields
	encryptedSessionID, err := r.encryptor.Encrypt(ctx, session.SessionID)
	if err != nil {
		return fmt.Errorf("failed to encrypt session_id: %w", err)
	}

	var encryptedIPAddress string
	if session.IPAddress != "" {
		encryptedIPAddress, err = r.encryptor.Encrypt(ctx, session.IPAddress)
		if err != nil {
			return fmt.Errorf("failed to encrypt ip_address: %w", err)
		}
	}

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

	_, err = r.db.ExecContext(ctx, query,
		session.ID,
		encryptedSessionID,
		session.TenantID,
		session.UserID,
		encryptedIPAddress,
		session.UserAgent,
		session.ExpiresAt,
		session.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// FindByID finds a session by session ID with decrypted PII
func (r *SessionRepository) FindByID(ctx context.Context, sessionID string) (*models.Session, error) {
	// Encrypt the search session ID to match against encrypted database values
	encryptedSessionID, err := r.encryptor.Encrypt(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt search session_id: %w", err)
	}

	query := `
		SELECT id, session_id, tenant_id, user_id, ip_address, user_agent, 
		       expires_at, terminated_at, created_at
		FROM sessions
		WHERE session_id = $1
	`

	session := &models.Session{}
	var encryptedStoredSessionID, encryptedIPAddress string
	var userAgent sql.NullString
	var terminatedAt sql.NullTime

	err = r.db.QueryRowContext(ctx, query, encryptedSessionID).Scan(
		&session.ID,
		&encryptedStoredSessionID,
		&session.TenantID,
		&session.UserID,
		&encryptedIPAddress,
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

	// Decrypt PII fields
	session.SessionID, err = r.encryptor.Decrypt(ctx, encryptedStoredSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt session_id: %w", err)
	}

	decryptedIP, err := r.decryptToStringPtr(ctx, encryptedIPAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt ip_address: %w", err)
	}
	if decryptedIP != nil {
		session.IPAddress = *decryptedIP
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
