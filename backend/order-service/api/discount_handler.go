package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/services"
	"github.com/rs/zerolog/log"
)

type DiscountHandler struct {
	discountService *services.DiscountService
}

func NewDiscountHandler(discountService *services.DiscountService) *DiscountHandler {
	return &DiscountHandler{discountService: discountService}
}

func (h *DiscountHandler) CreateDiscountRule(c echo.Context) error {
	ctx := c.Request().Context()
	tenantID, ok := tenantIDFromHeader(c)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req models.CreateDiscountRuleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	rule, err := h.discountService.Create(ctx, tenantID, &req)
	if err != nil {
		log.Warn().Err(err).Str("tenant_id", tenantID).Msg("Failed to create discount rule")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, rule)
}

func (h *DiscountHandler) ListDiscountRules(c echo.Context) error {
	ctx := c.Request().Context()
	tenantID, ok := tenantIDFromHeader(c)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	filter := models.ListDiscountRulesFilter{
		TenantID: tenantID,
		Limit:    parseIntQuery(c, "limit", 50),
		Offset:   parseIntQuery(c, "offset", 0),
	}
	if targetType := c.QueryParam("target_type"); targetType != "" {
		typed := models.DiscountTargetType(targetType)
		filter.TargetType = &typed
	}
	if targetID := c.QueryParam("target_id"); targetID != "" {
		filter.TargetID = &targetID
	}
	if active := c.QueryParam("active"); active != "" {
		parsed, err := strconv.ParseBool(active)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "active must be true or false"})
		}
		filter.IsActive = &parsed
	}

	rules, err := h.discountService.List(ctx, filter)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to list discount rules")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list discount rules"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"discount_rules": rules})
}

func (h *DiscountHandler) GetDiscountRule(c echo.Context) error {
	ctx := c.Request().Context()
	tenantID, ok := tenantIDFromHeader(c)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	rule, err := h.discountService.GetByID(ctx, tenantID, c.Param("id"))
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Str("discount_rule_id", c.Param("id")).Msg("Failed to get discount rule")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get discount rule"})
	}
	if rule == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Discount rule not found"})
	}

	return c.JSON(http.StatusOK, rule)
}

func (h *DiscountHandler) UpdateDiscountRule(c echo.Context) error {
	ctx := c.Request().Context()
	tenantID, ok := tenantIDFromHeader(c)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req models.UpdateDiscountRuleRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	rule, err := h.discountService.Update(ctx, tenantID, c.Param("id"), &req)
	if err != nil {
		log.Warn().Err(err).Str("tenant_id", tenantID).Str("discount_rule_id", c.Param("id")).Msg("Failed to update discount rule")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if rule == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Discount rule not found"})
	}

	return c.JSON(http.StatusOK, rule)
}

func (h *DiscountHandler) DeleteDiscountRule(c echo.Context) error {
	ctx := c.Request().Context()
	tenantID, ok := tenantIDFromHeader(c)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	if err := h.discountService.Delete(ctx, tenantID, c.Param("id")); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Discount rule not found"})
		}
		log.Error().Err(err).Str("tenant_id", tenantID).Str("discount_rule_id", c.Param("id")).Msg("Failed to delete discount rule")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete discount rule"})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *DiscountHandler) PreviewPricing(c echo.Context) error {
	ctx := c.Request().Context()
	tenantID, ok := tenantIDFromHeader(c)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req models.PricingPreviewRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	req.TenantID = tenantID

	result, err := h.discountService.PreviewPricing(ctx, &req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

func RegisterDiscountRoutes(e *echo.Echo, handler *DiscountHandler, requireRoleMiddleware func(...string) echo.MiddlewareFunc) {
	discountRules := e.Group("/api/v1/admin/discount-rules", requireRoleMiddleware("owner", "manager"))

	discountRules.POST("/preview", handler.PreviewPricing)
	discountRules.GET("", handler.ListDiscountRules)
	discountRules.POST("", handler.CreateDiscountRule)
	discountRules.GET("/:id", handler.GetDiscountRule)
	discountRules.PATCH("/:id", handler.UpdateDiscountRule)
	discountRules.DELETE("/:id", handler.DeleteDiscountRule)

	log.Info().Msg("Discount rule routes registered successfully")
}

func tenantIDFromHeader(c echo.Context) (string, bool) {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	return tenantID, tenantID != ""
}

func parseIntQuery(c echo.Context, name string, fallback int) int {
	value := c.QueryParam(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
