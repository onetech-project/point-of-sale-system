package api

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/product-service/src/models"
	"github.com/pos/backend/product-service/src/services"
	"github.com/pos/backend/product-service/src/utils"
)

type ProductHandler struct {
	service *services.ProductService
}

func NewProductHandler(service *services.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

func (h *ProductHandler) RegisterRoutes(e *echo.Group) {
	e.POST("/products", h.CreateProduct)
	e.GET("/products", h.ListProducts)
	e.GET("/products/:id", h.GetProduct)
	e.PUT("/products/:id", h.UpdateProduct)
	e.DELETE("/products/:id", h.DeleteProduct)
	e.PATCH("/products/:id/archive", h.ArchiveProduct)
	e.PATCH("/products/:id/restore", h.RestoreProduct)
	e.POST("/products/:id/photo", h.UploadPhoto)
	e.GET("/products/:id/photo", h.GetPhoto)
	e.DELETE("/products/:id/photo", h.DeletePhoto)
}

type CreateProductRequest struct {
	SKU           string     `json:"sku" validate:"required,min=1,max=50"`
	Name          string     `json:"name" validate:"required,min=1,max=255"`
	Description   *string    `json:"description"`
	CategoryID    *uuid.UUID `json:"category_id"`
	SellingPrice  float64    `json:"selling_price" validate:"required,gte=0"`
	CostPrice     float64    `json:"cost_price" validate:"required,gte=0"`
	TaxRate       float64    `json:"tax_rate" validate:"gte=0,lte=100"`
	StockQuantity int        `json:"stock_quantity"`
}

func (h *ProductHandler) CreateProduct(c echo.Context) error {
	var req CreateProductRequest
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

	product := &models.Product{
		TenantID:      tenantUUID,
		SKU:           req.SKU,
		Name:          req.Name,
		Description:   req.Description,
		CategoryID:    req.CategoryID,
		SellingPrice:  req.SellingPrice,
		CostPrice:     req.CostPrice,
		TaxRate:       req.TaxRate,
		StockQuantity: req.StockQuantity,
	}

	if err := h.service.CreateProduct(c.Request().Context(), product); err != nil {
		if err.Error() == "SKU already exists" {
			return utils.RespondConflict(c, "SKU already exists", "A product with this SKU already exists in your catalog")
		}
		utils.Log.Error("Failed to create product: %v", err)
		return utils.RespondInternalError(c, "Failed to create product")
	}

	return c.JSON(http.StatusCreated, product)
}

func (h *ProductHandler) ListProducts(c echo.Context) error {
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid tenant ID")
	}

	limitStr := c.QueryParam("limit")
	offsetStr := c.QueryParam("offset")
	search := c.QueryParam("search")
	categoryIDStr := c.QueryParam("category_id")
	lowStockStr := c.QueryParam("low_stock")
	archivedStr := c.QueryParam("archived")

	limit := 50
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	filters := make(map[string]interface{})
	if search != "" {
		filters["search"] = search
	}

	if categoryIDStr != "" {
		if categoryID, err := uuid.Parse(categoryIDStr); err == nil {
			filters["category_id"] = categoryID
		}
	}

	if lowStockStr != "" {
		if lowStock, err := strconv.Atoi(lowStockStr); err == nil {
			filters["low_stock"] = lowStock
		}
	}

	if archivedStr != "" {
		filters["archived"] = archivedStr == "true"
	}

	products, total, err := h.service.GetProducts(c.Request().Context(), tenantUUID, filters, limit, offset)
	if err != nil {
		utils.Log.Error("Failed to list products: %v", err)
		return utils.RespondInternalError(c, "Failed to list products")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"products": products,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *ProductHandler) GetProduct(c echo.Context) error {
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid tenant ID")
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid product ID")
	}

	product, err := h.service.GetProduct(c.Request().Context(), tenantUUID, id)
	if err != nil {
		utils.Log.Error("Failed to get product: %v", err)
		return utils.RespondInternalError(c, "Failed to get product")
	}

	if product == nil {
		return utils.RespondNotFound(c, "Product not found")
	}

	return c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) UpdateProduct(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid product ID")
	}

	var req CreateProductRequest
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

	// Get existing product to preserve stock quantity and photo
	existingProduct, err := h.service.GetProduct(c.Request().Context(), tenantUUID, id)
	if err != nil {
		return utils.RespondNotFound(c, "Product not found")
	}

	product := &models.Product{
		ID:            id,
		TenantID:      tenantUUID,
		SKU:           req.SKU,
		Name:          req.Name,
		Description:   req.Description,
		CategoryID:    req.CategoryID,
		SellingPrice:  req.SellingPrice,
		CostPrice:     req.CostPrice,
		TaxRate:       req.TaxRate,
		StockQuantity: existingProduct.StockQuantity, // Preserve existing stock
		PhotoPath:     existingProduct.PhotoPath,     // Preserve existing photo
		PhotoSize:     existingProduct.PhotoSize,     // Preserve existing photo size
	}

	if err := h.service.UpdateProduct(c.Request().Context(), product); err != nil {
		if err.Error() == "product not found" {
			return utils.RespondNotFound(c, "Product not found")
		}
		if err.Error() == "SKU already exists" {
			return utils.RespondConflict(c, "SKU already exists", "A product with this SKU already exists in your catalog")
		}
		utils.Log.Error("Failed to update product: %v", err)
		return utils.RespondInternalError(c, "Failed to update product")
	}

	// Fetch updated product with category information
	updatedProduct, err := h.service.GetProduct(c.Request().Context(), tenantUUID, id)
	if err != nil {
		utils.Log.Error("Failed to get updated product: %v", err)
		return utils.RespondInternalError(c, "Failed to get updated product")
	}

	return c.JSON(http.StatusOK, updatedProduct)
}

func (h *ProductHandler) DeleteProduct(c echo.Context) error {
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid tenant ID")
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid product ID")
	}

	if err := h.service.DeleteProduct(c.Request().Context(), tenantUUID, id); err != nil {
		if err.Error() == "cannot delete product with sales history" {
			return utils.RespondError(c, http.StatusForbidden, "Cannot delete product with sales history. Consider archiving instead.")
		}
		utils.Log.Error("Failed to delete product: %v", err)
		return utils.RespondInternalError(c, "Failed to delete product")
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *ProductHandler) ArchiveProduct(c echo.Context) error {
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid tenant ID")
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid product ID")
	}

	if err := h.service.ArchiveProduct(c.Request().Context(), tenantUUID, id); err != nil {
		utils.Log.Error("Failed to archive product: %v", err)
		return utils.RespondInternalError(c, "Failed to archive product")
	}

	// Return updated product
	product, err := h.service.GetProduct(c.Request().Context(), tenantUUID, id)
	if err != nil {
		utils.Log.Error("Failed to get archived product: %v", err)
		return utils.RespondInternalError(c, "Product archived but failed to retrieve")
	}

	return c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) RestoreProduct(c echo.Context) error {
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid tenant ID")
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid product ID")
	}

	if err := h.service.RestoreProduct(c.Request().Context(), tenantUUID, id); err != nil {
		utils.Log.Error("Failed to restore product: %v", err)
		return utils.RespondInternalError(c, "Failed to restore product")
	}

	// Return updated product
	product, err := h.service.GetProduct(c.Request().Context(), tenantUUID, id)
	if err != nil {
		utils.Log.Error("Failed to get restored product: %v", err)
		return utils.RespondInternalError(c, "Product restored but failed to retrieve")
	}

	return c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) UploadPhoto(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid product ID")
	}

	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid tenant ID")
	}

	file, header, err := c.Request().FormFile("photo")
	if err != nil {
		return utils.RespondBadRequest(c, "Photo file is required")
	}
	defer file.Close()

	if err := h.service.UploadPhoto(c.Request().Context(), id, tenantUUID, file, header); err != nil {
		utils.Log.Error("Failed to upload photo: %v", err)
		return utils.RespondBadRequest(c, err.Error())
	}

	// Return updated product
	product, err := h.service.GetProduct(c.Request().Context(), tenantUUID, id)
	if err != nil {
		utils.Log.Error("Failed to get updated product: %v", err)
		return utils.RespondInternalError(c, "Photo uploaded but failed to retrieve product")
	}

	return c.JSON(http.StatusOK, product)
}

func (h *ProductHandler) GetPhoto(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid product ID")
	}

	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid tenant ID")
	}

	photoPath, err := h.service.GetPhotoPath(c.Request().Context(), id, tenantUUID)
	if err != nil {
		return utils.RespondNotFound(c, "Photo not found")
	}

	return c.File(photoPath)
}

func (h *ProductHandler) DeletePhoto(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid product ID")
	}

	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	tenantUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Invalid tenant ID")
	}

	if err := h.service.DeletePhoto(c.Request().Context(), id, tenantUUID); err != nil {
		utils.Log.Error("Failed to delete photo: %v", err)
		return utils.RespondInternalError(c, "Failed to delete photo")
	}

	return c.NoContent(http.StatusNoContent)
}
