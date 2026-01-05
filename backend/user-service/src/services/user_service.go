package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pos/user-service/src/repository"
)

// UserService handles user-related business logic
type UserService struct {
	userRepo *repository.UserRepository
	db       *sql.DB
}

// NewUserService creates a new user service with a real VaultClient (production use)
func NewUserService(db *sql.DB) (*UserService, error) {
	userRepo, err := repository.NewUserRepositoryWithVault(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create user repository: %w", err)
	}
	return &UserService{
		userRepo: userRepo,
		db:       db,
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
			firstName, err = s.userRepo.DecryptField(ctx, encryptedFirstName.String)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt first_name for user %s: %w", id, err)
			}
		}

		if encryptedLastName.Valid && encryptedLastName.String != "" {
			lastName, err = s.userRepo.DecryptField(ctx, encryptedLastName.String)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt last_name for user %s: %w", id, err)
			}
		}

		email, err := s.userRepo.DecryptField(ctx, encryptedEmail)
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
