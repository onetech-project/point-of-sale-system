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
	db             *sql.DB
	encryptor      utils.Encryptor
	auditPublisher *utils.AuditPublisher
}

// NewSessionRepository creates a new repository with dependency injection (for testing)
func NewSessionRepository(db *sql.DB, encryptor utils.Encryptor, auditPublisher *utils.AuditPublisher) *SessionRepository {
	return &SessionRepository{
		db:             db,
		encryptor:      encryptor,
		auditPublisher: auditPublisher,
	}
}

// NewSessionRepositoryWithVault creates a repository with real VaultClient (for production)
func NewSessionRepositoryWithVault(db *sql.DB, auditPublisher *utils.AuditPublisher) (*SessionRepository, error) {
	vaultClient, err := utils.NewVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize VaultClient: %w", err)
	}
	return NewSessionRepository(db, vaultClient, auditPublisher), nil
}

// encryptStringPtrWithContext encrypts a pointer to string with context (handles nil values)
func (r *SessionRepository) encryptStringPtrWithContext(ctx context.Context, value *string, encContext string) (string, error) {
	if value == nil || *value == "" {
		return "", nil
	}
	return r.encryptor.EncryptWithContext(ctx, *value, encContext)
}

// decryptToStringPtrWithContext decrypts to a pointer to string with context (handles empty values)
func (r *SessionRepository) decryptToStringPtrWithContext(ctx context.Context, encrypted string, encContext string) (*string, error) {
	if encrypted == "" {
		return nil, nil
	}
	decrypted, err := r.encryptor.DecryptWithContext(ctx, encrypted, encContext)
	if err != nil {
		return nil, err
	}
	return &decrypted, nil
}

// encryptStringPtr encrypts a pointer to string (handles nil values) - DEPRECATED: Use encryptStringPtrWithContext
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
	// Encrypt PII fields with context
	encryptedSessionID, err := r.encryptor.EncryptWithContext(ctx, session.SessionID, "session:session_id")
	if err != nil {
		return fmt.Errorf("failed to encrypt session_id: %w", err)
	}

	var encryptedIPAddress string
	if session.IPAddress != "" {
		encryptedIPAddress, err = r.encryptor.EncryptWithContext(ctx, session.IPAddress, "session:ip_address")
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

	// T104: Publish SessionCreatedEvent to audit trail
	if r.auditPublisher != nil {
		afterValue := map[string]interface{}{
			"session_id": encryptedSessionID,
			"tenant_id":  session.TenantID,
			"user_id":    session.UserID,
			"ip_address": encryptedIPAddress,
			"expires_at": session.ExpiresAt.Format(time.RFC3339),
		}

		userIDPtr := &session.UserID
		auditEvent := &utils.AuditEvent{
			TenantID:     session.TenantID,
			ActorType:    "user",
			ActorID:      userIDPtr,
			SessionID:    &session.SessionID,
			Action:       "CREATE",
			ResourceType: "session",
			ResourceID:   session.ID,
			AfterValue:   afterValue,
			IPAddress:    &session.IPAddress,
			UserAgent:    &session.UserAgent,
		}

		if err := r.auditPublisher.Publish(ctx, auditEvent); err != nil {
			fmt.Printf("Failed to publish session create audit event: %v\n", err)
		}
	}

	return nil
}

// FindByID finds a session by session ID with decrypted PII
func (r *SessionRepository) FindByID(ctx context.Context, sessionID string) (*models.Session, error) {
	// Encrypt the search session ID with context for deterministic encryption
	encryptedSessionID, err := r.encryptor.EncryptWithContext(ctx, sessionID, "session:session_id")
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

	// Decrypt PII fields with context
	session.SessionID, err = r.encryptor.DecryptWithContext(ctx, encryptedStoredSessionID, "session:session_id")
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt session_id: %w", err)
	}

	decryptedIP, err := r.decryptToStringPtrWithContext(ctx, encryptedIPAddress, "session:ip_address")
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
	// Encrypt session ID for search
	encryptedSessionID, err := r.encryptor.EncryptWithContext(ctx, sessionID, "session:session_id")
	if err != nil {
		return fmt.Errorf("failed to encrypt session_id for search: %w", err)
	}

	query := `
		UPDATE sessions
		SET terminated_at = $1
		WHERE session_id = $2 AND terminated_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), encryptedSessionID)
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

	// T104: Publish SessionExpiredEvent to audit trail
	if r.auditPublisher != nil {
		// Try to get session info for audit (best effort)
		session, _ := r.FindByID(ctx, sessionID)
		var tenantID, userID string
		if session != nil {
			tenantID = session.TenantID
			userID = session.UserID
		}

		userIDPtr := &userID
		auditEvent := &utils.AuditEvent{
			TenantID:     tenantID,
			ActorType:    "user",
			ActorID:      userIDPtr,
			Action:       "DELETE",
			ResourceType: "session",
			ResourceID:   sessionID,
			Metadata: map[string]interface{}{
				"termination_type": "manual",
			},
		}

		if err := r.auditPublisher.Publish(ctx, auditEvent); err != nil {
			fmt.Printf("Failed to publish session delete audit event: %v\n", err)
		}
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
		var encryptedSessionID, encryptedIPAddress string
		var userAgent sql.NullString
		var terminatedAt sql.NullTime

		err := rows.Scan(
			&session.ID,
			&encryptedSessionID,
			&session.TenantID,
			&session.UserID,
			&encryptedIPAddress,
			&userAgent,
			&session.ExpiresAt,
			&terminatedAt,
			&session.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		// Decrypt PII fields with context
		session.SessionID, err = r.encryptor.DecryptWithContext(ctx, encryptedSessionID, "session:session_id")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt session_id: %w", err)
		}

		decryptedIP, err := r.decryptToStringPtrWithContext(ctx, encryptedIPAddress, "session:ip_address")
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

		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return sessions, nil
}
