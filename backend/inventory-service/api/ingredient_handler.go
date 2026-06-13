package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/services"
	"github.com/pos/backend/inventory-service/src/utils"
)

type IngredientHandler struct {
	service *services.IngredientService
}

func NewIngredientHandler(service *services.IngredientService) *IngredientHandler {
	return &IngredientHandler{service: service}
}

func (h *IngredientHandler) RegisterRoutes(e *echo.Group) {
	e.GET("/ingredients", h.List)
	e.GET("/ingredients/:id", h.Get)
	e.POST("/ingredients", h.Create)
	e.PUT("/ingredients/:id", h.Update)
	e.DELETE("/ingredients/:id", h.Delete)
}

func (h *IngredientHandler) List(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	limit, offset := parseLimitOffset(c)
	activeOnly := c.QueryParam("include_inactive") != "true"
	ingredients, total, err := h.service.List(c.Request().Context(), tenantID, c.QueryParam("search"), activeOnly, limit, offset)
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

func (h *IngredientHandler) Get(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid ingredient ID")
	}
	ingredient, err := h.service.Get(c.Request().Context(), tenantID, id)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, ingredient)
}

func (h *IngredientHandler) Create(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	var req models.CreateIngredientRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}
	ingredient, err := h.service.Create(c.Request().Context(), tenantID, req)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusCreated, ingredient)
}

func (h *IngredientHandler) Update(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid ingredient ID")
	}
	var req models.UpdateIngredientRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}
	ingredient, err := h.service.Update(c.Request().Context(), tenantID, id, req)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, ingredient)
}

func (h *IngredientHandler) Delete(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondUnauthorized(c, "Tenant ID not found")
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid ingredient ID")
	}
	if err := h.service.Delete(c.Request().Context(), tenantID, id); err != nil {
		return respondServiceError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
