package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pos/user-service/src/models"
	"github.com/pos/user-service/src/repository"
	"github.com/pos/user-service/src/utils"
)

var (
	ErrTeamMemberNotFound      = errors.New("team member not found")
	ErrTeamMemberForbidden     = errors.New("team member update forbidden")
	ErrInvalidTeamMemberUpdate = errors.New("invalid team member update")
)

// UserService handles user-related business logic
type UserService struct {
	userRepo       *repository.UserRepository
	db             *sql.DB
	auditPublisher utils.AuditPublisherInterface
}

// NewUserService creates a new user service with a real VaultClient (production use)
func NewUserService(db *sql.DB, auditPublisher utils.AuditPublisherInterface) (*UserService, error) {
	userRepo, err := repository.NewUserRepositoryWithVault(db, auditPublisher)
	if err != nil {
		return nil, fmt.Errorf("failed to create user repository: %w", err)
	}
	return &UserService{
		userRepo:       userRepo,
		db:             db,
		auditPublisher: auditPublisher,
	}, nil
}

// NewUserServiceWithRepository creates a user service with an injected repository (testing use)
// This allows you to inject a repository with a mock Encryptor for unit testing
func NewUserServiceWithRepository(db *sql.DB, userRepo *repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
		db:       db,
	}
}

func NewUserServiceWithRepositoryAndAuditPublisher(db *sql.DB, userRepo *repository.UserRepository, auditPublisher utils.AuditPublisherInterface) *UserService {
	return &UserService{
		userRepo:       userRepo,
		db:             db,
		auditPublisher: auditPublisher,
	}
}

// GetUsersWithNotificationPreferences returns all users in a tenant with their notification preferences
func (s *UserService) GetUsersWithNotificationPreferences(tenantID string) ([]map[string]interface{}, error) {
	ctx := context.Background()

	// Query to get all users with encrypted PII fields
	query := `
		SELECT 
			id,
			first_name,
			last_name,
			email,
			role,
			receive_order_notifications,
			created_at,
			updated_at
		FROM users
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var (
			id                        string
			encryptedFirstName        sql.NullString
			encryptedLastName         sql.NullString
			encryptedEmail            string
			role                      string
			receiveOrderNotifications bool
			createdAt                 string
			updatedAt                 string
		)

		if err := rows.Scan(&id, &encryptedFirstName, &encryptedLastName, &encryptedEmail, &role, &receiveOrderNotifications, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		// Decrypt PII fields
		var firstName, lastName string
		if encryptedFirstName.Valid && encryptedFirstName.String != "" {
			firstName, err = s.userRepo.DecryptFieldWithContext(ctx, encryptedFirstName.String, "user:first_name")
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt first_name for user %s: %w", id, err)
			}
		}

		if encryptedLastName.Valid && encryptedLastName.String != "" {
			lastName, err = s.userRepo.DecryptFieldWithContext(ctx, encryptedLastName.String, "user:last_name")
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt last_name for user %s: %w", id, err)
			}
		}

		email, err := s.userRepo.DecryptFieldWithContext(ctx, encryptedEmail, "user:email")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt email for user %s: %w", id, err)
		}

		// Combine first and last name
		name := fmt.Sprintf("%s %s", firstName, lastName)

		users = append(users, map[string]interface{}{
			"id":                          id,
			"name":                        name,
			"email":                       email,
			"role":                        role,
			"receive_order_notifications": receiveOrderNotifications,
			"created_at":                  createdAt,
			"updated_at":                  updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

func (s *UserService) ListTeamMembers(ctx context.Context, tenantID string) ([]*models.User, error) {
	users, err := s.userRepo.ListTeamMembers(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list team members: %w", err)
	}
	return users, nil
}

func (s *UserService) UpdateTeamMember(ctx context.Context, tenantID, actorID, actorRole, userID string, role, status *string, auditCtx AuditContext) (*models.User, error) {
	if role == nil && status == nil {
		return nil, ErrInvalidTeamMemberUpdate
	}
	if auditCtx.ActorID == "" {
		auditCtx.ActorID = actorID
	}
	if auditCtx.ActorRole == "" {
		auditCtx.ActorRole = actorRole
	}

	target, err := s.userRepo.FindByID(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find team member: %w", err)
	}
	if target == nil {
		return nil, ErrTeamMemberNotFound
	}

	if !canManageTeamMember(actorID, actorRole, target) {
		s.publishTeamMemberDeniedAudit(ctx, target, role, status, auditCtx)
		return nil, ErrTeamMemberForbidden
	}

	previousRole := target.Role
	previousStatus := target.Status

	if role != nil {
		if *role != string(models.RoleManager) && *role != string(models.RoleCashier) {
			return nil, ErrInvalidTeamMemberUpdate
		}
		target.Role = *role
	}

	if status != nil {
		if *status != string(models.UserStatusActive) && *status != string(models.UserStatusSuspended) {
			return nil, ErrInvalidTeamMemberUpdate
		}
		target.Status = *status
	}

	if err := s.userRepo.Update(ctx, target); err != nil {
		return nil, fmt.Errorf("failed to update team member: %w", err)
	}

	s.publishTeamMemberChangeAuditEvents(ctx, target, previousRole, previousStatus, auditCtx)

	return target, nil
}

func (s *UserService) publishTeamMemberDeniedAudit(ctx context.Context, target *models.User, role, status *string, auditCtx AuditContext) {
	metadata := map[string]interface{}{
		"reason":        "insufficient_permissions",
		"target_role":   target.Role,
		"target_status": target.Status,
	}
	if role != nil {
		metadata["requested_role"] = *role
	}
	if status != nil {
		metadata["requested_status"] = *status
	}

	publishTeamAuditEvent(
		ctx,
		s.auditPublisher,
		s.userRepo,
		auditCtx,
		target.TenantID,
		"ACCESS",
		"user",
		target.ID,
		"team.member.update_denied",
		nil,
		nil,
		metadata,
	)
}

func (s *UserService) publishTeamMemberChangeAuditEvents(ctx context.Context, target *models.User, previousRole, previousStatus string, auditCtx AuditContext) {
	if previousRole != target.Role {
		publishTeamAuditEvent(
			ctx,
			s.auditPublisher,
			s.userRepo,
			auditCtx,
			target.TenantID,
			"UPDATE",
			"user",
			target.ID,
			"team.member.role_changed",
			map[string]interface{}{"role": previousRole},
			map[string]interface{}{"role": target.Role},
			nil,
		)
	}

	if previousStatus != target.Status {
		publishTeamAuditEvent(
			ctx,
			s.auditPublisher,
			s.userRepo,
			auditCtx,
			target.TenantID,
			"UPDATE",
			"user",
			target.ID,
			"team.member.status_changed",
			map[string]interface{}{"status": previousStatus},
			map[string]interface{}{"status": target.Status},
			nil,
		)
	}
}

func canManageTeamMember(actorID, actorRole string, target *models.User) bool {
	if target == nil || actorID == "" || actorID == target.ID {
		return false
	}
	if target.Role == string(models.RoleOwner) {
		return false
	}

	switch actorRole {
	case string(models.RoleOwner):
		return target.Role == string(models.RoleManager) || target.Role == string(models.RoleCashier)
	case string(models.RoleManager):
		return target.Role == string(models.RoleCashier)
	default:
		return false
	}
}

// UpdateUserNotificationPreference updates a user's notification preference
func (s *UserService) UpdateUserNotificationPreference(tenantID, userID string, receive bool) error {
	ctx := context.Background()

	// First check if user exists and belongs to this tenant
	checkQuery := `
		SELECT id
		FROM users
		WHERE id = $1
		  AND tenant_id = $2
	`

	var existingUserID string
	err := s.db.QueryRowContext(ctx, checkQuery, userID, tenantID).Scan(&existingUserID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("user not found")
	}
	if err != nil {
		return fmt.Errorf("failed to check user: %w", err)
	}

	// Update the preference
	updateQuery := `
		UPDATE users
		SET receive_order_notifications = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
		  AND tenant_id = $3
	`

	result, err := s.db.ExecContext(ctx, updateQuery, receive, userID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update notification preference: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows updated")
	}

	return nil
}
