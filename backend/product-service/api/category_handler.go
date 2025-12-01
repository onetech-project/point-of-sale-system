package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/product-service/src/models"
	"github.com/pos/backend/product-service/src/services"
	"github.com/pos/backend/product-service/src/utils"
)

type CategoryHandler struct {
	service *services.CategoryService
}

func NewCategoryHandler(service *services.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

func (h *CategoryHandler) RegisterRoutes(e *echo.Group) {
	e.POST("/categories", h.CreateCategory)
	e.GET("/categories", h.ListCategories)
	e.GET("/categories/:id", h.GetCategory)
	e.PUT("/categories/:id", h.UpdateCategory)
	e.DELETE("/categories/:id", h.DeleteCategory)
}

type CreateCategoryRequest struct {
	Name         string `json:"name" validate:"required,min=1,max=100"`
	DisplayOrder int    `json:"display_order"`
}

func (h *CategoryHandler) CreateCategory(c echo.Context) error {
	var req CreateCategoryRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}

	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid tenant ID")
	}

	category := &models.Category{
		TenantID:     tenantUUID,
		Name:         req.Name,
		DisplayOrder: req.DisplayOrder,
	}

	if err := h.service.CreateCategory(c.Request().Context(), category); err != nil {
		utils.Log.Error("Failed to create category: %v", err)
		// Check for duplicate key constraint violation
		if strings.Contains(err.Error(), "idx_categories_tenant_name") || strings.Contains(err.Error(), "duplicate key") {
			return utils.RespondError(c, http.StatusConflict, "A category with this name already exists")
		}
		return utils.RespondInternalError(c, "Failed to create category")
	}

	return c.JSON(http.StatusCreated, category)
}

func (h *CategoryHandler) ListCategories(c echo.Context) error {
	categories, err := h.service.GetCategories(c.Request().Context())
	if err != nil {
		utils.Log.Error("Failed to list categories: %v", err)
		return utils.RespondInternalError(c, "Failed to list categories")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"categories": categories,
	})
}

func (h *CategoryHandler) GetCategory(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid category ID")
	}

	category, err := h.service.GetCategory(c.Request().Context(), id)
	if err != nil {
		utils.Log.Error("Failed to get category: %v", err)
		return utils.RespondInternalError(c, "Failed to get category")
	}

	if category == nil {
		return utils.RespondNotFound(c, "Category not found")
	}

	return c.JSON(http.StatusOK, category)
}

func (h *CategoryHandler) UpdateCategory(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid category ID")
	}

	var req CreateCategoryRequest
	if err := c.Bind(&req); err != nil {
		return utils.RespondBadRequest(c, "Invalid request body")
	}

	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid tenant ID")
	}

	category := &models.Category{
		ID:           id,
		TenantID:     tenantUUID,
		Name:         req.Name,
		DisplayOrder: req.DisplayOrder,
	}

	if err := h.service.UpdateCategory(c.Request().Context(), category); err != nil {
		utils.Log.Error("Failed to update category: %v", err)
		// Check for duplicate key constraint violation
		if strings.Contains(err.Error(), "idx_categories_tenant_name") || strings.Contains(err.Error(), "duplicate key") {
			return utils.RespondError(c, http.StatusConflict, "A category with this name already exists")
		}
		return utils.RespondInternalError(c, "Failed to update category")
	}

	return c.JSON(http.StatusOK, category)
}

func (h *CategoryHandler) DeleteCategory(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid category ID")
	}

	if err := h.service.DeleteCategory(c.Request().Context(), id); err != nil {
		if err.Error() == "cannot delete category with assigned products" {
			return utils.RespondError(c, http.StatusForbidden, "Cannot delete category with assigned products")
		}
		utils.Log.Error("Failed to delete category: %v", err)
		return utils.RespondInternalError(c, "Failed to delete category")
	}

	return c.NoContent(http.StatusNoContent)
}
