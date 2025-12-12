package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/product-service/src/services"
	"github.com/pos/backend/product-service/src/utils"
)

type StockHandler struct {
	productService   *services.ProductService
	inventoryService *services.InventoryService
}

func NewStockHandler(productService *services.ProductService, inventoryService *services.InventoryService) *StockHandler {
	return &StockHandler{
		productService:   productService,
		inventoryService: inventoryService,
	}
}

// RegisterRoutes registers stock and inventory related routes
func (h *StockHandler) RegisterRoutes(e *echo.Group) {
	e.GET("/inventory/summary", h.GetInventorySummary)
	e.GET("/inventory/adjustments", h.GetAllAdjustments)
	e.GET("/products/:id/adjustments", h.GetProductAdjustments)
	e.POST("/products/:id/stock", h.AdjustStock)
}

// GetInventorySummary returns overall inventory statistics
func (h *StockHandler) GetInventorySummary(c echo.Context) error {
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid tenant ID")
	}

	summary, err := h.productService.GetInventorySummary(c.Request().Context(), tenantUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]interface{}{
			"message": "Failed to fetch inventory summary",
			"error":   err.Error(),
		})
	}

	return c.JSON(http.StatusOK, summary)
}

// AdjustStock handles manual stock adjustments
func (h *StockHandler) AdjustStock(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid product ID")
	}

	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid tenant ID")
	}

	userID := c.Get("user_id")
	if userID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "User ID not found")
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid user ID")
	}

	type AdjustStockRequest struct {
		NewQuantity int    `json:"new_quantity" validate:"required"`
		Reason      string `json:"reason" validate:"required,oneof=supplier_delivery physical_count shrinkage damage return correction"`
		Notes       string `json:"notes"`
	}

	var req AdjustStockRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}

	// Validate reason
	validReasons := map[string]bool{
		"supplier_delivery": true,
		"physical_count":    true,
		"shrinkage":         true,
		"damage":            true,
		"return":            true,
		"correction":        true,
	}

	if !validReasons[req.Reason] {
		return utils.RespondBadRequest(c, "Invalid reason code. Must be one of: supplier_delivery, physical_count, shrinkage, damage, return, correction")
	}

	product, err := h.inventoryService.AdjustStock(c.Request().Context(), id, tenantUUID, userUUID, req.NewQuantity, req.Reason, req.Notes)
	if err != nil {
		utils.Log.Error("Failed to adjust stock: %v", err)
		return utils.RespondInternalError(c, "Failed to adjust stock")
	}

	return c.JSON(http.StatusOK, product)
}

// GetProductAdjustments returns stock adjustment history for a specific product
func (h *StockHandler) GetProductAdjustments(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid product ID")
	}

	limitStr := c.QueryParam("limit")
	offsetStr := c.QueryParam("offset")

	limit := 50
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	adjustments, total, err := h.inventoryService.GetAdjustmentHistory(c.Request().Context(), id, limit, offset)
	if err != nil {
		utils.Log.Error("Failed to get adjustment history: %v", err)
		return utils.RespondInternalError(c, "Failed to get adjustment history")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"adjustments": adjustments,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// GetAllAdjustments returns all stock adjustments for the tenant with filtering
func (h *StockHandler) GetAllAdjustments(c echo.Context) error {
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid tenant ID")
	}

	limitStr := c.QueryParam("limit")
	offsetStr := c.QueryParam("offset")
	reason := c.QueryParam("reason")
	startDateStr := c.QueryParam("start_date")
	endDateStr := c.QueryParam("end_date")
	productIDStr := c.QueryParam("product_id")
	userIDStr := c.QueryParam("user_id")

	limit := 50
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	filters := make(map[string]interface{})

	if reason != "" {
		filters["reason"] = reason
	}

	if startDateStr != "" {
		if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			filters["start_date"] = startDate
		}
	}

	if endDateStr != "" {
		if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			filters["end_date"] = endDate
		}
	}

	if productIDStr != "" {
		if productID, err := uuid.Parse(productIDStr); err == nil {
			filters["product_id"] = productID
		}
	}

	if userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			filters["user_id"] = userID
		}
	}

	adjustments, total, err := h.inventoryService.GetAdjustmentsByFilters(c.Request().Context(), tenantUUID, filters, limit, offset)
	if err != nil {
		utils.Log.Error("Failed to get adjustments: %v", err)
		return utils.RespondInternalError(c, "Failed to get adjustments")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"adjustments": adjustments,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}
