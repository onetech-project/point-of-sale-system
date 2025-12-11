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

// NewUserService creates a new user service
func NewUserService(db *sql.DB) *UserService {
	return &UserService{
		userRepo: repository.NewUserRepository(db),
		db:       db,
	}
}

// GetUsersWithNotificationPreferences returns all users in a tenant with their notification preferences
func (s *UserService) GetUsersWithNotificationPreferences(tenantID string) ([]map[string]interface{}, error) {
	ctx := context.Background()

	// Query to get all users with notification preferences
	query := `
		SELECT 
			id,
			(first_name) || ' ' || (last_name) AS name,
			email,
			role,
			receive_order_notifications,
			created_at,
			updated_at
		FROM users
		WHERE tenant_id = $1
		ORDER BY name ASC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var (
			userID                    string
			name                      string
			email                     string
			role                      string
			receiveOrderNotifications bool
			createdAt                 string
			updatedAt                 string
		)

		if err := rows.Scan(&userID, &name, &email, &role, &receiveOrderNotifications, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		users = append(users, map[string]interface{}{
			"user_id":                     userID,
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
		SELECT user_id
		FROM users
		WHERE user_id = $1
		  AND tenant_id = $2
		  AND deleted_at IS NULL
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
		WHERE user_id = $2
		  AND tenant_id = $3
		  AND deleted_at IS NULL
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
