package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pos/analytics-service/src/middleware"
	"github.com/pos/analytics-service/src/models"
	"github.com/pos/analytics-service/src/services"
	"github.com/rs/zerolog/log"
)

type InventoryAnalyticsHandler struct {
	service *services.InventoryAnalyticsService
}

func NewInventoryAnalyticsHandler(service *services.InventoryAnalyticsService) *InventoryAnalyticsHandler {
	return &InventoryAnalyticsHandler{
		service: service,
	}
}

func (h *InventoryAnalyticsHandler) GetProductProfitability(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Tenant ID not found"})
	}

	startStr := c.QueryParam("start_date")
	endStr := c.QueryParam("end_date")

	if startStr == "" || endStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "start_date and end_date required"})
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid start_date format"})
	}

	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid end_date format"})
	}

	limit := 20
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	response, err := h.service.GetProductProfitability(c.Request().Context(), tenantID, start, end, limit)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get product profitability")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve product profitability"})
	}

	return c.JSON(http.StatusOK, response)
}

func (h *InventoryAnalyticsHandler) GetIngredientForecast(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Tenant ID not found"})
	}

	startStr := c.QueryParam("start_date")
	endStr := c.QueryParam("end_date")

	if startStr == "" || endStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "start_date and end_date required"})
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid start_date format"})
	}

	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid end_date format"})
	}

	response, err := h.service.GetIngredientForecast(c.Request().Context(), tenantID, start, end)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get ingredient forecast")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve ingredient forecast"})
	}

	return c.JSON(http.StatusOK, response)
}

func (h *InventoryAnalyticsHandler) SimulateRevenueTarget(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Tenant ID not found"})
	}

	var req models.RevenueTargetRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.TargetRevenue == "" || req.StartDate == "" || req.EndDate == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "target_revenue, start_date, and end_date required"})
	}

	response, err := h.service.SimulateRevenueTarget(c.Request().Context(), tenantID, &req)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to simulate revenue target")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to simulate revenue target"})
	}

	return c.JSON(http.StatusOK, response)
}

func (h *InventoryAnalyticsHandler) SimulateBudget(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Tenant ID not found"})
	}

	var req models.BudgetSimulatorRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.BudgetAmount == "" || req.StartDate == "" || req.EndDate == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "budget_amount, start_date, and end_date required"})
	}

	response, err := h.service.SimulateBudget(c.Request().Context(), tenantID, &req)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to simulate budget")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to simulate budget"})
	}

	return c.JSON(http.StatusOK, response)
}

func (h *InventoryAnalyticsHandler) GetProductRecommendations(c echo.Context) error {
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Tenant ID not found"})
	}

	response, err := h.service.GetProductRecommendations(c.Request().Context(), tenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get product recommendations")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve product recommendations"})
	}

	return c.JSON(http.StatusOK, response)
}
