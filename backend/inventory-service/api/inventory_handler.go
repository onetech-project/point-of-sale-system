package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/services"
	"github.com/pos/backend/inventory-service/src/utils"
)

type InventoryHandler struct {
	service *services.StockService
}

func NewInventoryHandler(service *services.StockService) *InventoryHandler {
	return &InventoryHandler{service: service}
}

func (h *InventoryHandler) RegisterRoutes(e *echo.Group) {
	e.POST("/inventory/initial-stock", h.InitialStock)
	e.POST("/inventory/purchases", h.PurchaseIn)
	e.POST("/inventory/adjustments", h.ManualAdjustment)
	e.POST("/inventory/waste", h.Waste)
	e.GET("/inventory/ingredients/:ingredientId/movements", h.Movements)
	e.GET("/inventory/low-stock", h.LowStock)
	e.GET("/inventory/valuation", h.Valuation)
}

func (h *InventoryHandler) InitialStock(c echo.Context) error {
	return h.stockMutation(c, h.service.InitialStock)
}

func (h *InventoryHandler) PurchaseIn(c echo.Context) error {
	return h.stockMutation(c, h.service.PurchaseIn)
}

func (h *InventoryHandler) Waste(c echo.Context) error {
	return h.stockMutation(c, h.service.Waste)
}

func (h *InventoryHandler) ManualAdjustment(c echo.Context) error {
	tenantID, userID, ok := requestContext(c)
	if !ok {
		return nil
	}
	var req models.ManualAdjustmentRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}
	movement, err := h.service.ManualAdjustment(c.Request().Context(), tenantID, userID, req)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusCreated, movement)
}

func (h *InventoryHandler) stockMutation(c echo.Context, fn func(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req models.StockMutationRequest) (*models.StockMovementResponse, error)) error {
	tenantID, userID, ok := requestContext(c)
	if !ok {
		return nil
	}
	var req models.StockMutationRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}
	movement, err := fn(c.Request().Context(), tenantID, userID, req)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusCreated, movement)
}

func (h *InventoryHandler) Movements(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	ingredientID, err := uuid.Parse(c.Param("ingredientId"))
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid ingredient ID")
	}
	limit, offset := parseLimitOffset(c)
	movements, total, err := h.service.Movements(c.Request().Context(), tenantID, ingredientID, limit, offset)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"movements": movements,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

func (h *InventoryHandler) LowStock(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	limit, offset := parseLimitOffset(c)
	ingredients, total, err := h.service.LowStock(c.Request().Context(), tenantID, limit, offset)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"ingredients": ingredients,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

func (h *InventoryHandler) Valuation(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	total, err := h.service.Valuation(c.Request().Context(), tenantID)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"valuation": total})
}

func requestContext(c echo.Context) (uuid.UUID, *uuid.UUID, bool) {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		_ = utils.RespondUnauthorized(c, "Tenant ID not found")
		return uuid.Nil, nil, false
	}
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		_ = utils.RespondUnauthorized(c, "Invalid user ID")
		return uuid.Nil, nil, false
	}
	return tenantID, userID, true
}
