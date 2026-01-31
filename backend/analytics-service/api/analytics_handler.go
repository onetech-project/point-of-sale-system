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

// AnalyticsHandler handles analytics API requests
type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(analyticsService *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// GetSalesOverview handles GET /analytics/overview
// Returns sales metrics, daily sales chart, and category breakdown
func (h *AnalyticsHandler) GetSalesOverview(c echo.Context) error {
	startTime := time.Now()

	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Parse query parameters
	timeRangeStr := c.QueryParam("time_range")
	if timeRangeStr == "" {
		timeRangeStr = "this_month" // Default to current month
	}

	timeRange := models.TimeRange(timeRangeStr)
	if !timeRange.IsValid() {
		log.Warn().
			Str("tenant_id", tenantID).
			Str("time_range", timeRangeStr).
			Msg("Invalid time_range parameter")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid time_range parameter",
		})
	}

	// Parse custom date range if provided
	var startDate, endDate *time.Time
	if timeRange == models.TimeRangeCustom {
		startStr := c.QueryParam("start_date")
		endStr := c.QueryParam("end_date")

		if startStr == "" || endStr == "" {
			log.Warn().
				Str("tenant_id", tenantID).
				Str("start_date", startStr).
				Str("end_date", endStr).
				Msg("Missing start_date or end_date for custom time range")
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "start_date and end_date required for custom time range",
			})
		}

		start, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			log.Warn().
				Str("tenant_id", tenantID).
				Str("start_date", startStr).
				Err(err).
				Msg("Invalid start_date format")
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid start_date format (use YYYY-MM-DD)",
			})
		}

		end, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			log.Warn().
				Str("tenant_id", tenantID).
				Str("end_date", endStr).
				Err(err).
				Msg("Invalid end_date format")
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid end_date format (use YYYY-MM-DD)",
			})
		}

		startDate = &start
		endDate = &end
	}

	// Get sales overview from service
	response, err := h.analyticsService.GetSalesOverview(c.Request().Context(), tenantID, timeRange, startDate, endDate)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get sales overview")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve sales overview",
		})
	}

	// Log query performance
	queryTime := time.Since(startTime).Milliseconds()
	log.Info().
		Str("tenant_id", tenantID).
		Str("time_range", string(timeRange)).
		Int64("query_time_ms", queryTime).
		Msg("Sales overview retrieved successfully")

	return c.JSON(http.StatusOK, response)
}

// GetTopProducts handles GET /analytics/top-products
// Returns top and bottom products by revenue and quantity
func (h *AnalyticsHandler) GetTopProducts(c echo.Context) error {
	startTime := time.Now()

	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Parse query parameters
	timeRangeStr := c.QueryParam("time_range")
	if timeRangeStr == "" {
		timeRangeStr = "this_month"
	}

	timeRange := models.TimeRange(timeRangeStr)
	if !timeRange.IsValid() {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid time_range parameter",
		})
	}

	// Parse limit parameter
	limit := 5 // Default limit
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit < 1 || parsedLimit > 20 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid limit parameter (must be between 1 and 20)",
			})
		}
		limit = parsedLimit
	}

	// Parse custom date range if provided
	var startDate, endDate *time.Time
	if timeRange == models.TimeRangeCustom {
		startStr := c.QueryParam("start_date")
		endStr := c.QueryParam("end_date")

		if startStr == "" || endStr == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "start_date and end_date required for custom time range",
			})
		}

		start, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid start_date format (use YYYY-MM-DD)",
			})
		}

		end, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid end_date format (use YYYY-MM-DD)",
			})
		}

		startDate = &start
		endDate = &end
	}

	// Get top products from service
	response, err := h.analyticsService.GetTopProducts(c.Request().Context(), tenantID, timeRange, startDate, endDate, limit)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get top products")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve top products",
		})
	}

	// Log query performance
	queryTime := time.Since(startTime).Milliseconds()
	log.Info().
		Str("tenant_id", tenantID).
		Str("time_range", string(timeRange)).
		Int("limit", limit).
		Int64("query_time_ms", queryTime).
		Msg("Top products retrieved successfully")

	return c.JSON(http.StatusOK, response)
}

// GetTopCustomers handles GET /analytics/top-customers
// Returns top customers by spending and order count (with masked PII)
func (h *AnalyticsHandler) GetTopCustomers(c echo.Context) error {
	startTime := time.Now()

	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Parse query parameters
	timeRangeStr := c.QueryParam("time_range")
	if timeRangeStr == "" {
		timeRangeStr = "this_month"
	}

	timeRange := models.TimeRange(timeRangeStr)
	if !timeRange.IsValid() {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid time_range parameter",
		})
	}

	// Parse limit parameter
	limit := 5 // Default limit
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit < 1 || parsedLimit > 20 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid limit parameter (must be between 1 and 20)",
			})
		}
		limit = parsedLimit
	}

	// Parse custom date range if provided
	var startDate, endDate *time.Time
	if timeRange == models.TimeRangeCustom {
		startStr := c.QueryParam("start_date")
		endStr := c.QueryParam("end_date")

		if startStr == "" || endStr == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "start_date and end_date required for custom time range",
			})
		}

		start, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid start_date format (use YYYY-MM-DD)",
			})
		}

		end, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid end_date format (use YYYY-MM-DD)",
			})
		}

		startDate = &start
		endDate = &end
	}

	// Get top customers from service
	response, err := h.analyticsService.GetTopCustomers(c.Request().Context(), tenantID, timeRange, startDate, endDate, limit)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get top customers")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve top customers",
		})
	}

	// Log query performance (PII already masked by service layer)
	queryTime := time.Since(startTime).Milliseconds()
	log.Info().
		Str("tenant_id", tenantID).
		Str("time_range", string(timeRange)).
		Int("limit", limit).
		Int64("query_time_ms", queryTime).
		Msg("Top customers retrieved successfully")

	return c.JSON(http.StatusOK, response)
}

// GetSalesTrend handles GET /analytics/sales-trend
// Returns time series data for sales revenue and order count with configurable granularity
func (h *AnalyticsHandler) GetSalesTrend(c echo.Context) error {
	startTime := time.Now()

	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Parse query parameters
	granularity := c.QueryParam("granularity")
	if granularity == "" {
		granularity = "daily" // Default to daily
	}

	// Validate granularity
	validGranularities := map[string]bool{
		"daily":     true,
		"weekly":    true,
		"monthly":   true,
		"quarterly": true,
		"yearly":    true,
	}
	if !validGranularities[granularity] {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid granularity. Must be one of: daily, weekly, monthly, quarterly, yearly",
		})
	}

	startDateStr := c.QueryParam("start_date")
	endDateStr := c.QueryParam("end_date")

	if startDateStr == "" || endDateStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "start_date and end_date are required (YYYY-MM-DD format)",
		})
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid start_date format. Use YYYY-MM-DD",
		})
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid end_date format. Use YYYY-MM-DD",
		})
	}

	// Validate date range
	if endDate.Before(startDate) {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "end_date must be after start_date",
		})
	}

	// Check if dates are in the future
	// now := time.Now()
	// if startDate.After(now) || endDate.After(now) {
	// 	return c.JSON(http.StatusBadRequest, map[string]string{
	// 		"error": "Dates in the future are not allowed",
	// 	})
	// }

	// Get sales trend from service
	response, err := h.analyticsService.GetSalesTrend(c.Request().Context(), tenantID, startDate, endDate, granularity)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Str("granularity", granularity).Msg("Failed to get sales trend")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve sales trend",
		})
	}

	// Log query performance
	queryTime := time.Since(startTime).Milliseconds()
	log.Info().
		Str("tenant_id", tenantID).
		Str("granularity", granularity).
		Str("start_date", startDateStr).
		Str("end_date", endDateStr).
		Int64("query_time_ms", queryTime).
		Msg("Sales trend retrieved successfully")

	return c.JSON(http.StatusOK, response)
}
