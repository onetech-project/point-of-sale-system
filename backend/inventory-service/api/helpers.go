package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/pos/backend/inventory-service/src/services"
	"github.com/pos/backend/inventory-service/src/utils"
)

func parseLimitOffset(c echo.Context) (int, int) {
	limit := 50
	offset := 0
	if raw := c.QueryParam("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	if raw := c.QueryParam("offset"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			offset = parsed
		}
	}
	return limit, offset
}

func respondServiceError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, services.ErrNotFound):
		return utils.RespondNotFound(c, "Resource not found")
	case errors.Is(err, services.ErrProductNotFound):
		return utils.RespondNotFound(c, "Product not found")
	case errors.Is(err, services.ErrInvalidInput), errors.Is(err, services.ErrMissingConversion), errors.Is(err, services.ErrInactiveIngredient):
		return utils.RespondBadRequest(c, "Invalid request", err.Error())
	case errors.Is(err, services.ErrConflict):
		return utils.RespondConflict(c, "Conflict", err.Error())
	case errors.Is(err, services.ErrInsufficientStock):
		return utils.RespondError(c, http.StatusUnprocessableEntity, "Insufficient stock")
	case errors.Is(err, services.ErrReadOnlyGlobalUOM):
		return utils.RespondError(c, http.StatusForbidden, "Global UoMs are read-only")
	default:
		c.Logger().Errorf("service error: %v", err)
		return utils.RespondInternalError(c, "Internal server error")
	}
}
