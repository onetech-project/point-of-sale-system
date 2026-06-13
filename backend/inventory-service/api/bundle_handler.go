package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/services"
	"github.com/pos/backend/inventory-service/src/utils"
)

type BundleHandler struct {
	service *services.BundleService
}

func NewBundleHandler(service *services.BundleService) *BundleHandler {
	return &BundleHandler{service: service}
}

func (h *BundleHandler) RegisterRoutes(e *echo.Group) {
	e.GET("/bundles", h.List)
	e.GET("/bundles/:id", h.Get)
	e.POST("/bundles", h.Create)
	e.PUT("/bundles/:id", h.Update)
	e.DELETE("/bundles/:id", h.Delete)
	e.GET("/bundles/:id/cost", h.Cost)
}

func (h *BundleHandler) List(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	limit, offset := parseLimitOffset(c)
	activeOnly := c.QueryParam("include_inactive") != "true"
	bundles, total, err := h.service.List(c.Request().Context(), tenantID, c.QueryParam("search"), activeOnly, limit, offset)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"bundles": bundles,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func (h *BundleHandler) Get(c echo.Context) error {
	tenantID, id, ok := bundleContext(c)
	if !ok {
		return nil
	}
	bundle, err := h.service.Get(c.Request().Context(), tenantID, id)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, bundle)
}

func (h *BundleHandler) Create(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	var req models.CreateBundleRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}
	bundle, err := h.service.Create(c.Request().Context(), tenantID, req)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusCreated, bundle)
}

func (h *BundleHandler) Update(c echo.Context) error {
	tenantID, id, ok := bundleContext(c)
	if !ok {
		return nil
	}
	var req models.UpdateBundleRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}
	bundle, err := h.service.Update(c.Request().Context(), tenantID, id, req)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, bundle)
}

func (h *BundleHandler) Delete(c echo.Context) error {
	tenantID, id, ok := bundleContext(c)
	if !ok {
		return nil
	}
	if err := h.service.Delete(c.Request().Context(), tenantID, id); err != nil {
		return respondServiceError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *BundleHandler) Cost(c echo.Context) error {
	tenantID, id, ok := bundleContext(c)
	if !ok {
		return nil
	}
	cost, err := h.service.Cost(c.Request().Context(), tenantID, id)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, cost)
}

func bundleContext(c echo.Context) (uuid.UUID, uuid.UUID, bool) {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		_ = utils.RespondUnauthorized(c, "Tenant ID not found")
		return uuid.Nil, uuid.Nil, false
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		_ = utils.RespondBadRequest(c, "Invalid bundle ID")
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, id, true
}
