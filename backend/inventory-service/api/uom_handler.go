package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/services"
	"github.com/pos/backend/inventory-service/src/utils"
)

type UOMHandler struct {
	service *services.UOMService
}

func NewUOMHandler(service *services.UOMService) *UOMHandler {
	return &UOMHandler{service: service}
}

func (h *UOMHandler) RegisterRoutes(e *echo.Group) {
	e.GET("/uoms", h.List)
	e.POST("/uoms", h.Create)
	e.PUT("/uoms/:id", h.Update)
	e.DELETE("/uoms/:id", h.Delete)
}

func (h *UOMHandler) List(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	includeInactive := c.QueryParam("include_inactive") == "true"
	uoms, err := h.service.List(c.Request().Context(), tenantID, includeInactive)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"uoms": uoms})
}

func (h *UOMHandler) Create(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	var req models.CreateUOMRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}
	uom, err := h.service.Create(c.Request().Context(), tenantID, req)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusCreated, uom)
}

func (h *UOMHandler) Update(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid UoM ID")
	}
	var req models.UpdateUOMRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}
	uom, err := h.service.Update(c.Request().Context(), tenantID, id, req)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, uom)
}

func (h *UOMHandler) Delete(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid UoM ID")
	}
	if err := h.service.Delete(c.Request().Context(), tenantID, id); err != nil {
		return respondServiceError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
