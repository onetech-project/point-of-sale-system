package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/repository"
	"github.com/rs/zerolog/log"
)

// OrderSettingsHandler handles order settings operations
type OrderSettingsHandler struct {
	repo *repository.OrderSettingsRepository
}

// NewOrderSettingsHandler creates a new order settings handler
func NewOrderSettingsHandler(repo *repository.OrderSettingsRepository) *OrderSettingsHandler {
	return &OrderSettingsHandler{
		repo: repo,
	}
}

// GetOrderSettings handles GET /admin/settings/orders
func (h *OrderSettingsHandler) GetOrderSettings(c echo.Context) error {
	ctx := c.Request().Context()

	// Get tenant ID from query parameter
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Get or create settings
	settings, err := h.repo.GetOrCreate(ctx, tenantID)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID).
			Msg("Failed to get order settings")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve settings",
		})
	}

	log.Info().
		Str("tenant_id", tenantID).
		Msg("Order settings retrieved successfully")

	return c.JSON(http.StatusOK, settings)
}

// UpdateOrderSettings handles PUT /admin/settings/orders
func (h *OrderSettingsHandler) UpdateOrderSettings(c echo.Context) error {
	ctx := c.Request().Context()

	// Get tenant ID from query parameter
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Parse request body
	var req models.UpdateOrderSettingsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate settings
	if req.DefaultDeliveryFee != nil && *req.DefaultDeliveryFee < 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "default_delivery_fee must be non-negative",
		})
	}

	if req.MinOrderAmount != nil && *req.MinOrderAmount < 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "min_order_amount must be non-negative",
		})
	}

	if req.MaxDeliveryDistance != nil && *req.MaxDeliveryDistance < 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "max_delivery_distance must be non-negative",
		})
	}

	if req.EstimatedPrepTime != nil && *req.EstimatedPrepTime < 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "estimated_prep_time must be non-negative",
		})
	}

	// Update settings
	settings, err := h.repo.Update(ctx, tenantID, &req)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID).
			Msg("Failed to update order settings")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update settings",
		})
	}

	log.Info().
		Str("tenant_id", tenantID).
		Msg("Order settings updated successfully")

	return c.JSON(http.StatusOK, settings)
}

// RegisterRoutes registers order settings routes
func (h *OrderSettingsHandler) RegisterRoutes(e *echo.Echo) {
	// Admin routes for order settings
	admin := e.Group("/api/v1/admin/settings")
	// TODO: Add JWT middleware once auth integration is complete

	admin.GET("/orders", h.GetOrderSettings)
	admin.PUT("/orders", h.UpdateOrderSettings)
}
