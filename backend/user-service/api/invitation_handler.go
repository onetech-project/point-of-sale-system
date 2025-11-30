package api

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/user-service/src/models"
	"github.com/pos/user-service/src/queue"
	"github.com/pos/user-service/src/services"
)

type InvitationHandler struct {
	invitationService *services.InvitationService
}

func NewInvitationHandler(db *sql.DB, eventProducer *queue.KafkaProducer) *InvitationHandler {
	return &InvitationHandler{
		invitationService: services.NewInvitationService(db, eventProducer),
	}
}

// CreateInvitation handles POST /invitations
func (h *InvitationHandler) CreateInvitation(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	var req models.InvitationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Email is required",
		})
	}

	if req.Role == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Role is required",
		})
	}

	// Validate role
	validRoles := map[string]bool{
		"admin":   true,
		"manager": true,
		"cashier": true,
	}
	if !validRoles[req.Role] {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid role. Must be one of: admin, manager, cashier",
		})
	}

	invitation, err := h.invitationService.Create(c.Request().Context(), tenantID, req.Email, req.Role, userID)
	if err != nil {
		if err == services.ErrEmailAlreadyExists {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "Email is already registered",
			})
		}
		if err == services.ErrEmailAlreadyInvited {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "Email already has a pending invitation",
			})
		}

		c.Logger().Errorf("Failed to create invitation: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create invitation",
		})
	}

	response := &models.InvitationResponse{
		ID:        invitation.ID,
		Email:     invitation.Email,
		Role:      invitation.Role,
		Status:    invitation.Status,
		ExpiresAt: invitation.ExpiresAt,
		InvitedBy: invitation.InvitedBy,
		CreatedAt: invitation.CreatedAt,
	}

	return c.JSON(http.StatusCreated, response)
}

// ListInvitations handles GET /invitations
func (h *InvitationHandler) ListInvitations(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	invitations, err := h.invitationService.List(c.Request().Context(), tenantID)
	if err != nil {
		c.Logger().Errorf("Failed to list invitations: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to list invitations",
		})
	}

	responses := make([]*models.InvitationResponse, len(invitations))
	for i, inv := range invitations {
		responses[i] = &models.InvitationResponse{
			ID:        inv.ID,
			Email:     inv.Email,
			Role:      inv.Role,
			Status:    inv.Status,
			ExpiresAt: inv.ExpiresAt,
			InvitedBy: inv.InvitedBy,
			CreatedAt: inv.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, responses)
}

// AcceptInvitation handles POST /invitations/:token/accept
func (h *InvitationHandler) AcceptInvitation(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Token is required",
		})
	}

	var req models.InvitationAcceptRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.FirstName == "" || req.LastName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "First name and last name are required",
		})
	}

	if req.Password == "" || len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Password must be at least 8 characters",
		})
	}

	user, err := h.invitationService.Accept(c.Request().Context(), token, req.FirstName, req.LastName, req.Password)
	if err != nil {
		if err == services.ErrInvitationNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Invitation not found",
			})
		}
		if err == services.ErrInvitationExpired {
			return c.JSON(http.StatusGone, map[string]string{
				"error": "Invitation has expired",
			})
		}
		if err == services.ErrInvitationInvalid {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invitation is no longer valid",
			})
		}
		if err == services.ErrEmailAlreadyExists {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "Email is already registered",
			})
		}

		c.Logger().Errorf("Failed to accept invitation: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to accept invitation",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Invitation accepted successfully. You can now log in.",
		"user": map[string]string{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// ResendInvitation handles POST /invitations/:id/resend
func (h *InvitationHandler) ResendInvitation(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	invitationID := c.Param("id")
	if invitationID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invitation ID is required",
		})
	}

	invitation, err := h.invitationService.Resend(c.Request().Context(), tenantID, invitationID, userID)
	if err != nil {
		if err == services.ErrInvitationNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Invitation not found",
			})
		}

		c.Logger().Errorf("Failed to resend invitation: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to resend invitation",
		})
	}

	response := &models.InvitationResponse{
		ID:        invitation.ID,
		Email:     invitation.Email,
		Role:      invitation.Role,
		Status:    invitation.Status,
		ExpiresAt: invitation.ExpiresAt,
		InvitedBy: invitation.InvitedBy,
		CreatedAt: invitation.CreatedAt,
	}

	return c.JSON(http.StatusOK, response)
}
