package api

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/product-service/src/models"
	"github.com/pos/backend/product-service/src/services"
	"github.com/pos/backend/product-service/src/utils"
)

// PhotoHandler handles HTTP requests for product photos
type PhotoHandler struct {
	photoService *services.PhotoService
}

// NewPhotoHandler creates a new PhotoHandler
func NewPhotoHandler(photoService *services.PhotoService) *PhotoHandler {
	return &PhotoHandler{
		photoService: photoService,
	}
}

// UploadPhoto handles POST /api/v1/products/:product_id/photos
func (h *PhotoHandler) UploadPhoto(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse product ID from URL
	productID, err := uuid.Parse(c.Param("product_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "INVALID_PRODUCT_ID",
				"message": "Invalid product ID format",
			},
		})
	}

	// Get tenant ID from context (set by auth middleware)
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant ID not found in request context",
			},
		})
	}

	// Parse multipart form
	file, err := c.FormFile("photo")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "MISSING_FILE",
				"message": "Photo file is required",
			},
		})
	}

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "FILE_READ_ERROR",
				"message": "Failed to read uploaded file",
			},
		})
	}
	defer src.Close()

	// Parse optional parameters
	displayOrder := 0
	if orderStr := c.FormValue("display_order"); orderStr != "" {
		var order int
		if _, err := fmt.Sscanf(orderStr, "%d", &order); err == nil {
			displayOrder = order
		}
	}

	isPrimary := false
	if primaryStr := c.FormValue("is_primary"); primaryStr == "true" {
		isPrimary = true
	}

	// Upload photo
	photo, err := h.photoService.UploadPhoto(
		ctx,
		productID,
		tenantID,
		file.Filename,
		src,
		displayOrder,
		isPrimary,
	)

	if err != nil {
		return handlePhotoError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"status": "success",
		"data":   photo,
	})
}

// ListPhotos handles GET /api/v1/products/:product_id/photos
func (h *PhotoHandler) ListPhotos(c echo.Context) error {
	ctx := c.Request().Context()

	productID, err := uuid.Parse(c.Param("product_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "INVALID_PRODUCT_ID",
				"message": "Invalid product ID format",
			},
		})
	}

	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant ID not found in request context",
			},
		})
	}

	photos, err := h.photoService.ListPhotos(ctx, productID, tenantID)
	if err != nil {
		return handlePhotoError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"photos": photos,
			"count":  len(photos),
		},
	})
}

// GetPhoto handles GET /api/v1/products/:product_id/photos/:photo_id
func (h *PhotoHandler) GetPhoto(c echo.Context) error {
	ctx := c.Request().Context()

	photoID, err := uuid.Parse(c.Param("photo_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "INVALID_PHOTO_ID",
				"message": "Invalid photo ID format",
			},
		})
	}

	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant ID not found in request context",
			},
		})
	}

	photo, err := h.photoService.GetPhoto(ctx, photoID, tenantID)
	if err != nil {
		return handlePhotoError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   photo,
	})
}

// UpdatePhotoMetadata handles PATCH /api/v1/products/:product_id/photos/:photo_id
func (h *PhotoHandler) UpdatePhotoMetadata(c echo.Context) error {
	ctx := c.Request().Context()

	photoID, err := uuid.Parse(c.Param("photo_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "INVALID_PHOTO_ID",
				"message": "Invalid photo ID format",
			},
		})
	}

	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant ID not found in request context",
			},
		})
	}

	var req models.ProductPhotoUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	err = h.photoService.UpdatePhotoMetadata(ctx, photoID, tenantID, req.DisplayOrder, req.IsPrimary)
	if err != nil {
		return handlePhotoError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Photo metadata updated successfully",
	})
}

// ReplacePhoto handles PUT /api/v1/products/:product_id/photos/:photo_id
func (h *PhotoHandler) ReplacePhoto(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse photo ID from URL
	photoID, err := uuid.Parse(c.Param("photo_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "INVALID_PHOTO_ID",
				"message": "Invalid photo ID format",
			},
		})
	}

	// Get tenant ID from context
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant ID not found in request context",
			},
		})
	}

	// Parse multipart form
	file, err := c.FormFile("photo")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "MISSING_FILE",
				"message": "Photo file is required",
			},
		})
	}

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "FILE_READ_ERROR",
				"message": "Failed to read uploaded file",
			},
		})
	}
	defer src.Close()

	// Replace photo
	photo, err := h.photoService.ReplacePhoto(
		ctx,
		photoID,
		tenantID,
		file.Filename,
		src,
	)
	if err != nil {
		return handlePhotoError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   photo,
	})
}

// DeletePhoto handles DELETE /api/v1/products/:product_id/photos/:photo_id
func (h *PhotoHandler) DeletePhoto(c echo.Context) error {
	ctx := c.Request().Context()

	photoID, err := uuid.Parse(c.Param("photo_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "INVALID_PHOTO_ID",
				"message": "Invalid photo ID format",
			},
		})
	}

	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant ID not found in request context",
			},
		})
	}

	err = h.photoService.DeletePhoto(ctx, photoID, tenantID)
	if err != nil {
		return handlePhotoError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Photo deleted successfully",
	})
}

// ReorderPhotos handles PUT /api/v1/products/:product_id/photos/reorder
func (h *PhotoHandler) ReorderPhotos(c echo.Context) error {
	ctx := c.Request().Context()

	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant ID not found in request context",
			},
		})
	}

	var req models.ProductPhotoReorderRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	err = h.photoService.ReorderPhotos(ctx, tenantID, req.PhotoOrders)
	if err != nil {
		return handlePhotoError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Photos reordered successfully",
	})
}

// GetStorageQuota handles GET /api/v1/tenants/storage-quota
func (h *PhotoHandler) GetStorageQuota(c echo.Context) error {
	ctx := c.Request().Context()

	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status": "error",
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "Tenant ID not found in request context",
			},
		})
	}

	quota, err := h.photoService.GetStorageQuota(ctx, tenantID)
	if err != nil {
		return handlePhotoError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   quota,
	})
}

// handlePhotoError converts service errors to appropriate HTTP responses
func handlePhotoError(c echo.Context, err error) error {
	switch err {
	case models.ErrMaxPhotosReached:
		return utils.RespondConflict(c, err.Error())
	case models.ErrQuotaExceeded:
		return utils.RespondError(c, http.StatusForbidden, err.Error())
	case models.ErrPhotoNotFound:
		return utils.RespondNotFound(c, err.Error())
	case models.ErrUnauthorizedAccess:
		return utils.RespondError(c, http.StatusForbidden, err.Error())
	default:
		// Check for validation errors
		if validationErr, ok := err.(*models.ValidationError); ok {
			return utils.RespondBadRequest(c, validationErr.Error(), "Field: "+validationErr.Field)
		}

		// Generic server error
		return utils.RespondInternalError(c, "An internal error occurred")
	}
}
