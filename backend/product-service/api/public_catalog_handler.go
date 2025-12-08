package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/product-service/src/services"
)

type PublicCatalogHandler struct {
	catalogService *services.CatalogService
	productService *services.ProductService
}

func NewPublicCatalogHandler(catalogService *services.CatalogService, productService *services.ProductService) *PublicCatalogHandler {
	return &PublicCatalogHandler{
		catalogService: catalogService,
		productService: productService,
	}
}

func (h *PublicCatalogHandler) GetPublicMenu(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	category := c.QueryParam("category")
	availableOnly := c.QueryParam("available_only") == "true"

	products, err := h.catalogService.GetPublicCatalog(c.Request().Context(), tenantID, category, availableOnly)
	if err != nil {
		c.Logger().Error("Failed to get public catalog: ", err)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"message": "failed to get product catalog",
			"error":   err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"products": products,
	})
}

// GetPublicPhoto serves product photos without authentication
func (h *PublicCatalogHandler) GetPublicPhoto(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid product ID")
	}

	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid tenant ID")
	}

	photoPath, err := h.productService.GetPhotoPath(c.Request().Context(), id, tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Photo not found")
	}

	return c.File(photoPath)
}
