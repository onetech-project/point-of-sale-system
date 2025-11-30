package api

import (
	"encoding/json"
	"net/http"

	"github.com/pos/auth-service/src/services"

	"github.com/labstack/echo/v4"
)

type PasswordResetHandler struct {
	passwordResetService *services.PasswordResetService
}

func NewPasswordResetHandler(passwordResetService *services.PasswordResetService) *PasswordResetHandler {
	return &PasswordResetHandler{
		passwordResetService: passwordResetService,
	}
}

type RequestResetRequest struct {
	Email    string `json:"email" validate:"required,email"`
	TenantID string `json:"tenant_id" validate:"required,uuid"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

func (h *PasswordResetHandler) RequestReset(c echo.Context) error {
	var req RequestResetRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	token, err := h.passwordResetService.RequestReset(req.Email, req.TenantID)
	if err != nil {
		c.Logger().Error("Failed to request password reset: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to process request"})
	}

	if token != "" {
		event := map[string]interface{}{
			"type":  "password.reset_requested",
			"email": req.Email,
			"token": token,
		}
		eventJSON, _ := json.Marshal(event)
		c.Logger().Info("Password reset requested: ", string(eventJSON))
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "If the email exists, a password reset link has been sent",
	})
}

func (h *PasswordResetHandler) ResetPassword(c echo.Context) error {
	var req ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	err := h.passwordResetService.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	event := map[string]interface{}{
		"type":  "password.changed",
		"token": req.Token,
	}
	eventJSON, _ := json.Marshal(event)
	c.Logger().Info("Password reset completed: ", string(eventJSON))

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Password has been reset successfully",
	})
}
