package api

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/services"
	"github.com/pos/backend/inventory-service/src/utils"
)

type RecipeHandler struct {
	service *services.RecipeService
}

func NewRecipeHandler(service *services.RecipeService) *RecipeHandler {
	return &RecipeHandler{service: service}
}

func (h *RecipeHandler) RegisterRoutes(e *echo.Group) {
	e.GET("/products/:productId/recipe", h.GetActive)
	e.POST("/products/:productId/recipes", h.Create)
	e.GET("/products/:productId/recipes", h.ListVersions)
	e.GET("/products/:productId/recipes/:version", h.GetVersion)
	e.GET("/products/:productId/recipe/cost", h.GetCost)
}

func (h *RecipeHandler) Create(c echo.Context) error {
	tenantID, productID, ok := recipeContext(c)
	if !ok {
		return nil
	}
	var req models.CreateRecipeRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}
	recipe, err := h.service.Create(c.Request().Context(), tenantID, productID, req)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusCreated, recipe)
}

func (h *RecipeHandler) GetActive(c echo.Context) error {
	tenantID, productID, ok := recipeContext(c)
	if !ok {
		return nil
	}
	recipe, err := h.service.GetActive(c.Request().Context(), tenantID, productID)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, recipe)
}

func (h *RecipeHandler) GetCost(c echo.Context) error {
	tenantID, productID, ok := recipeContext(c)
	if !ok {
		return nil
	}
	cost, err := h.service.GetCost(c.Request().Context(), tenantID, productID)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, cost)
}

func (h *RecipeHandler) GetVersion(c echo.Context) error {
	tenantID, productID, ok := recipeContext(c)
	if !ok {
		return nil
	}
	version, err := strconv.Atoi(c.Param("version"))
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid recipe version")
	}
	recipe, err := h.service.GetVersion(c.Request().Context(), tenantID, productID, version)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, recipe)
}

func (h *RecipeHandler) ListVersions(c echo.Context) error {
	tenantID, productID, ok := recipeContext(c)
	if !ok {
		return nil
	}
	versions, err := h.service.ListVersions(c.Request().Context(), tenantID, productID)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"recipes": versions})
}

func recipeContext(c echo.Context) (uuid.UUID, uuid.UUID, bool) {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		_ = utils.RespondUnauthorized(c, "Tenant ID not found")
		return uuid.Nil, uuid.Nil, false
	}
	productID, err := uuid.Parse(c.Param("productId"))
	if err != nil {
		_ = utils.RespondBadRequest(c, "Invalid product ID")
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, productID, true
}
