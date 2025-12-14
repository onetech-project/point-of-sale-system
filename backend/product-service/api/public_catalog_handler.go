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
	photoService   *services.PhotoService
}

func NewPublicCatalogHandler(catalogService *services.CatalogService, productService *services.ProductService, photoService *services.PhotoService) *PublicCatalogHandler {
	return &PublicCatalogHandler{
		catalogService: catalogService,
		productService: productService,
		photoService:   photoService,
	}
}

func (h *PublicCatalogHandler) GetPublicMenu(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	category := c.QueryParam("category")
	availableOnly := c.QueryParam("available_only") == "true"
	includePrimaryPhoto := c.QueryParam("include_primary_photo") == "true"

	products, err := h.catalogService.GetPublicCatalog(c.Request().Context(), tenantID, category, availableOnly)
	if err != nil {
		c.Logger().Error("Failed to get public catalog: ", err)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{
			"message": "failed to get product catalog",
			"error":   err.Error(),
		})
	}

	// Populate primary photos if requested (Feature 005)
	if includePrimaryPhoto && h.photoService != nil && len(products) > 0 {
		tenantUUID, err := uuid.Parse(tenantID)
		if err == nil {
			for i := range products {
				productUUID, err := uuid.Parse(products[i].ID)
				if err != nil {
					continue
				}
				photos, err := h.photoService.ListPhotos(c.Request().Context(), productUUID, tenantUUID)
				if err != nil {
					continue
				}
				// Find primary photo
				for _, photo := range photos {
					if photo.IsPrimary {
						products[i].ImageURL = &photo.PhotoURL
						break
					}
				}
				// If no primary, use first photo
				if products[i].ImageURL == nil && len(photos) > 0 {
					products[i].ImageURL = &photos[0].PhotoURL
				}
			}
		}
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
