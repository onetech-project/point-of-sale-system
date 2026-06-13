package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/services"
	"github.com/pos/backend/inventory-service/src/utils"
)

type ConversionHandler struct {
	service *services.ConversionService
}

func NewConversionHandler(service *services.ConversionService) *ConversionHandler {
	return &ConversionHandler{service: service}
}

func (h *ConversionHandler) RegisterRoutes(e *echo.Group) {
	e.GET("/ingredients/:id/conversions", h.List)
	e.POST("/ingredients/:id/conversions", h.Create)
	e.PUT("/ingredients/:id/conversions/:conversionId", h.Update)
	e.DELETE("/ingredients/:id/conversions/:conversionId", h.Delete)
}

func (h *ConversionHandler) List(c echo.Context) error {
	tenantID, ingredientID, ok := h.contextIDs(c)
	if !ok {
		return nil
	}
	conversions, err := h.service.List(c.Request().Context(), tenantID, ingredientID)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"conversions": conversions})
}

func (h *ConversionHandler) Create(c echo.Context) error {
	tenantID, ingredientID, ok := h.contextIDs(c)
	if !ok {
		return nil
	}
	var req models.CreateConversionRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}
	conversion, err := h.service.Create(c.Request().Context(), tenantID, ingredientID, req)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusCreated, conversion)
}

func (h *ConversionHandler) Update(c echo.Context) error {
	tenantID, ingredientID, ok := h.contextIDs(c)
	if !ok {
		return nil
	}
	conversionID, err := uuid.Parse(c.Param("conversionId"))
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid conversion ID")
	}
	var req models.UpdateConversionRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}
	conversion, err := h.service.Update(c.Request().Context(), tenantID, ingredientID, conversionID, req)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, conversion)
}

func (h *ConversionHandler) Delete(c echo.Context) error {
	tenantID, ingredientID, ok := h.contextIDs(c)
	if !ok {
		return nil
	}
	conversionID, err := uuid.Parse(c.Param("conversionId"))
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid conversion ID")
	}
	if err := h.service.Delete(c.Request().Context(), tenantID, ingredientID, conversionID); err != nil {
		return respondServiceError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ConversionHandler) contextIDs(c echo.Context) (uuid.UUID, uuid.UUID, bool) {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		_ = utils.RespondUnauthorized(c, "Tenant ID not found")
		return uuid.Nil, uuid.Nil, false
	}
	ingredientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		_ = utils.RespondBadRequest(c, "Invalid ingredient ID")
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, ingredientID, true
}
