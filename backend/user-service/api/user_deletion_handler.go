package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/user-service/src/repository"
	"github.com/pos/user-service/src/services"
	"github.com/pos/user-service/src/utils"
)

type UserDeletionHandler struct {
	userDeletionService *services.UserDeletionService
}

func NewUserDeletionHandler(db *sql.DB, auditPublisher utils.AuditPublisherInterface) (*UserDeletionHandler, error) {
	userRepo, err := repository.NewUserRepositoryWithVault(db, auditPublisher)
	if err != nil {
		return nil, fmt.Errorf("failed to create user repository: %w", err)
	}

	userDeletionService := services.NewUserDeletionService(userRepo, auditPublisher, db)

	return &UserDeletionHandler{
		userDeletionService: userDeletionService,
	}, nil
}

// DeleteUser handles user deletion for UU PDP compliance (Article 5 - right to deletion)
// DELETE /api/v1/users/:user_id?force=true
// Supports both soft delete (default) and hard delete (force=true)
func (h *UserDeletionHandler) DeleteUser(c echo.Context) error {
	userID := c.Param("user_id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing user ID",
		})
	}

	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Missing tenant ID",
		})
	}

	// Verify the requesting user is the tenant owner (set by RBAC middleware in API Gateway)
	userRole := c.Request().Header.Get("X-User-Role")
	if userRole != "owner" {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Only tenant owners can delete users",
		})
	}

	// Get the deleter's user ID
	deletedBy := c.Request().Header.Get("X-User-ID")
	if deletedBy == "" {
		deletedBy = "system"
	}

	// Check if force=true for hard delete
	force := c.QueryParam("force") == "true"

	var err error
	if force {
		// Hard delete - permanent removal with audit trail anonymization
		err = h.userDeletionService.HardDelete(c.Request().Context(), tenantID, userID, deletedBy)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to hard delete user: %v", err),
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"message":     "User permanently deleted",
			"user_id":     userID,
			"delete_type": "hard",
		})
	} else {
		// Soft delete - 90-day retention period
		err = h.userDeletionService.SoftDelete(c.Request().Context(), tenantID, userID, deletedBy)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to soft delete user: %v", err),
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"message":      "User marked for deletion",
			"user_id":      userID,
			"delete_type":  "soft",
			"retention_days": 90,
			"permanent_deletion_after": "90 days",
		})
	}
}
