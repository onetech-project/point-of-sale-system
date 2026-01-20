package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pos/user-service/src/models"
	"github.com/pos/user-service/src/repository"
	"github.com/pos/user-service/src/utils"
)

// UserDeletionService handles user deletion operations for UU PDP compliance
// Supports soft delete (90-day retention) and hard delete (permanent removal with anonymization)
type UserDeletionService struct {
	userRepo       *repository.UserRepository
	auditPublisher utils.AuditPublisherInterface
	db             *sql.DB
}

func NewUserDeletionService(
	userRepo *repository.UserRepository,
	auditPublisher utils.AuditPublisherInterface,
	db *sql.DB,
) *UserDeletionService {
	return &UserDeletionService{
		userRepo:       userRepo,
		auditPublisher: auditPublisher,
		db:             db,
	}
}

// SoftDelete marks user as deleted with status='deleted' and sets deleted_at timestamp
// User data is retained for 90 days per UU PDP grace period before hard deletion
// Implements UU PDP Article 5 - Right to Deletion (with retention period)
func (s *UserDeletionService) SoftDelete(ctx context.Context, tenantID string, userID string, deletedBy string) error {
	// Get user before deletion for audit trail
	user, err := s.userRepo.FindByID(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	if user.Status == string(models.UserStatusDeleted) {
		return fmt.Errorf("user already deleted")
	}

	// Update user status to deleted and set deleted_at timestamp
	query := `
		UPDATE users
		SET status = 'deleted',
		    deleted_at = $1,
		    updated_at = $2
		WHERE id = $3
	`

	now := time.Now()
	_, err = s.db.ExecContext(ctx, query, now, now, userID)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	// Publish audit event for soft deletion
	deletedByPtr := &deletedBy
	auditEvent := &utils.AuditEvent{
		EventID:      uuid.New().String(),
		TenantID:     user.TenantID,
		Action:       "USER_SOFT_DELETE",
		ActorType:    "user",
		ActorID:      deletedByPtr,
		ActorEmail:   deletedByPtr, // Should be email of deleter
		ResourceType: "user",
		ResourceID:   userID,
		Timestamp:    now,
		Metadata: map[string]interface{}{
			"user_email":  user.Email,
			"user_role":   user.Role,
			"deleted_at":  now.Format(time.RFC3339),
			"retention_days": 90,
		},
	}

	if err := s.auditPublisher.Publish(ctx, auditEvent); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to publish audit event for soft delete: %v\n", err)
	}

	return nil
}

// HardDelete permanently removes user data and anonymizes audit trail entries
// Implements UU PDP Article 5 - Right to Deletion (permanent removal after retention period)
func (s *UserDeletionService) HardDelete(ctx context.Context, tenantID string, userID string, deletedBy string) error {
	// Get user before deletion for audit trail
	user, err := s.userRepo.FindByID(ctx, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Start transaction for atomic deletion + anonymization
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Permanently delete user record
	deleteUserQuery := `DELETE FROM users WHERE id = $1`
	_, err = tx.ExecContext(ctx, deleteUserQuery, userID)
	if err != nil {
		return fmt.Errorf("failed to hard delete user: %w", err)
	}

	// 2. Anonymize audit trail entries for this user
	// Replace actor_email with "deleted-user-{uuid}" to preserve audit integrity
	anonymizedEmail := fmt.Sprintf("deleted-user-%s", uuid.New().String())
	anonymizeAuditQuery := `
		UPDATE audit_events
		SET actor_email = $1,
		    metadata = jsonb_set(
		        COALESCE(metadata, '{}'::jsonb),
		        '{anonymized}',
		        'true'::jsonb
		    )
		WHERE actor_id = $2
	`
	_, err = tx.ExecContext(ctx, anonymizeAuditQuery, anonymizedEmail, userID)
	if err != nil {
		return fmt.Errorf("failed to anonymize audit trail: %w", err)
	}

	// 3. Delete user sessions
	deleteSessionsQuery := `DELETE FROM sessions WHERE user_id = $1`
	_, err = tx.ExecContext(ctx, deleteSessionsQuery, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	// 4. Delete user invitations (if any)
	deleteInvitationsQuery := `DELETE FROM user_invitations WHERE email = $1`
	_, err = tx.ExecContext(ctx, deleteInvitationsQuery, user.Email)
	if err != nil {
		// Non-critical error, log and continue
		fmt.Printf("Failed to delete user invitations: %v\n", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit hard delete transaction: %w", err)
	}

	// Publish audit event for hard deletion (after commit)
	deletedByPtr := &deletedBy
	auditEvent := &utils.AuditEvent{
		EventID:      uuid.New().String(),
		TenantID:     user.TenantID,
		Action:       "USER_HARD_DELETE",
		ActorType:    "user",
		ActorID:      deletedByPtr,
		ActorEmail:   deletedByPtr, // Should be email of deleter
		ResourceType: "user",
		ResourceID:   userID,
		Timestamp:    time.Now(),
		Metadata: map[string]interface{}{
			"user_email":        user.Email,
			"user_role":         user.Role,
			"anonymized_email":  anonymizedEmail,
			"audit_trail_anonymized": true,
		},
	}

	if err := s.auditPublisher.Publish(ctx, auditEvent); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to publish audit event for hard delete: %v\n", err)
	}

	return nil
}

// GetUserDeletionEligible returns users eligible for hard deletion (deleted > 90 days ago)
// Used by cleanup job to enforce retention policy
func (s *UserDeletionService) GetUserDeletionEligible(ctx context.Context) ([]models.User, error) {
	query := `
		SELECT id, tenant_id, email, password_hash, role, status, first_name, last_name, locale, created_at, updated_at
		FROM users
		WHERE status = 'deleted'
		  AND deleted_at < NOW() - INTERVAL '90 days'
		ORDER BY deleted_at ASC
		LIMIT 100
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query eligible users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var firstName, lastName sql.NullString

		err := rows.Scan(
			&user.ID,
			&user.TenantID,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.Status,
			&firstName,
			&lastName,
			&user.Locale,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if firstName.Valid {
			user.FirstName = &firstName.String
		}
		if lastName.Valid {
			user.LastName = &lastName.String
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// GetUserDeletionNotificationEligible returns users who should be notified about upcoming deletion (60 days after soft delete)
func (s *UserDeletionService) GetUserDeletionNotificationEligible(ctx context.Context) ([]models.User, error) {
	query := `
		SELECT id, tenant_id, email, password_hash, role, status, first_name, last_name, locale, created_at, updated_at
		FROM users
		WHERE status = 'deleted'
		  AND deleted_at < NOW() - INTERVAL '60 days'
		  AND deleted_at >= NOW() - INTERVAL '61 days'
		  AND NOT EXISTS (
		      SELECT 1 FROM user_deletion_notifications
		      WHERE user_id = users.id
		  )
		ORDER BY deleted_at ASC
		LIMIT 100
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query notification eligible users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var firstName, lastName sql.NullString

		err := rows.Scan(
			&user.ID,
			&user.TenantID,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.Status,
			&firstName,
			&lastName,
			&user.Locale,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if firstName.Valid {
			user.FirstName = &firstName.String
		}
		if lastName.Valid {
			user.LastName = &lastName.String
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
