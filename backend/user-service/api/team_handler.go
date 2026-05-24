package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/user-service/src/models"
	"github.com/pos/user-service/src/services"
)

type TeamHandler struct {
	userService *services.UserService
}

func NewTeamHandler(userService *services.UserService) *TeamHandler {
	return &TeamHandler{userService: userService}
}

func (h *TeamHandler) ListTeamMembers(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	users, err := h.userService.ListTeamMembers(c.Request().Context(), tenantID)
	if err != nil {
		c.Logger().Errorf("Failed to list team members: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list team members",
		})
	}

	responses := make([]*models.TeamMemberResponse, len(users))
	for i, user := range users {
		responses[i] = user.ToTeamMemberResponse()
	}

	return c.JSON(http.StatusOK, responses)
}

func (h *TeamHandler) UpdateTeamMember(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	actorID := c.Request().Header.Get("X-User-ID")
	actorRole := c.Request().Header.Get("X-User-Role")
	if tenantID == "" || actorID == "" || actorRole == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	userID := c.Param("user_id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID is required",
		})
	}

	var req models.TeamMemberUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	user, err := h.userService.UpdateTeamMember(c.Request().Context(), tenantID, actorID, actorRole, userID, req.Role, req.Status, auditContextFromRequest(c))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrTeamMemberNotFound):
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Team member not found",
			})
		case errors.Is(err, services.ErrTeamMemberForbidden):
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "You cannot manage this team member",
			})
		case errors.Is(err, services.ErrInvalidTeamMemberUpdate):
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid team member update",
			})
		default:
			c.Logger().Errorf("Failed to update team member: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update team member",
			})
		}
	}

	return c.JSON(http.StatusOK, user.ToTeamMemberResponse())
}
