package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/product-service/src/services"
	"github.com/pos/backend/product-service/src/utils"
)

type ProductImportHandler struct {
	service *services.ProductImportService
}

func NewProductImportHandler(service *services.ProductImportService) *ProductImportHandler {
	return &ProductImportHandler{service: service}
}

func (h *ProductImportHandler) RegisterRoutes(e *echo.Group) {
	e.GET("/product-imports/template", h.GetTemplate)
	e.POST("/product-imports", h.CreateImport)
	e.GET("/product-imports/:id", h.GetImport)
}

func (h *ProductImportHandler) GetTemplate(c echo.Context) error {
	format := c.QueryParam("format")
	if format == "" {
		format = "csv"
	}

	filename, contentType, data, err := services.BuildProductImportTemplate(format)
	if err != nil {
		return utils.RespondBadRequest(c, "Unsupported template format", "format must be csv or xlsx")
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", filename))
	return c.Blob(http.StatusOK, contentType, data)
}

func (h *ProductImportHandler) CreateImport(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "User ID not found")
	}

	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, services.ProductImportMaxFileSizeBytes+1024*1024)
	file, err := c.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			return utils.RespondError(c, http.StatusRequestEntityTooLarge, "Import file is too large")
		}
		return utils.RespondBadRequest(c, "Import file is required")
	}
	if file.Size == 0 {
		return utils.RespondBadRequest(c, "Import file cannot be empty")
	}
	if file.Size > services.ProductImportMaxFileSizeBytes {
		return utils.RespondError(c, http.StatusRequestEntityTooLarge, "Import file is too large")
	}

	format, err := services.NormalizeProductImportFormat(file.Filename, file.Header.Get(echo.HeaderContentType), c.FormValue("format"))
	if err != nil {
		return utils.RespondBadRequest(c, "Unsupported import format", "format must be csv or xlsx")
	}

	src, err := file.Open()
	if err != nil {
		return utils.RespondInternalError(c, "Failed to read import file")
	}
	defer src.Close()

	fileBytes, err := io.ReadAll(io.LimitReader(src, services.ProductImportMaxFileSizeBytes+1))
	if err != nil {
		return utils.RespondInternalError(c, "Failed to read import file")
	}
	if int64(len(fileBytes)) > services.ProductImportMaxFileSizeBytes {
		return utils.RespondError(c, http.StatusRequestEntityTooLarge, "Import file is too large")
	}

	job, err := h.service.StartImport(c.Request().Context(), tenantID, userID, file.Filename, format, fileBytes)
	if err != nil {
		utils.Log.Error("Failed to start product import: %v", err)
		return utils.RespondInternalError(c, "Failed to start product import")
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"import_id": job.ID,
		"status":    job.Status,
	})
}

func (h *ProductImportHandler) GetImport(c echo.Context) error {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		return utils.RespondError(c, http.StatusUnauthorized, "Tenant ID not found")
	}

	importID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.RespondBadRequest(c, "Invalid import ID")
	}

	job, err := h.service.GetImport(c.Request().Context(), tenantID, importID)
	if err != nil {
		utils.Log.Error("Failed to get product import: %v", err)
		return utils.RespondInternalError(c, "Failed to get product import")
	}
	if job == nil {
		return utils.RespondNotFound(c, "Import not found")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"import_id":    job.ID,
		"status":       job.Status,
		"failure_code": job.FailureCode,
		"summary": map[string]int{
			"total_rows":       job.RowCount,
			"rows":             job.RowCount,
			"created_count":    job.CreatedCount,
			"products_created": job.CreatedCount,
			"error_count":      job.ErrorCount,
			"warning_count":    job.WarningCount,
		},
		"errors":       job.Errors,
		"warnings":     job.Warnings,
		"created_at":   job.CreatedAt,
		"updated_at":   job.UpdatedAt,
		"completed_at": job.CompletedAt,
	})
}

func getUserIDFromContext(c echo.Context) (uuid.UUID, error) {
	userValue := c.Get("user_id")
	if userValue == nil {
		if header := c.Request().Header.Get("X-User-ID"); header != "" {
			return uuid.Parse(header)
		}
		return uuid.Nil, fmt.Errorf("user ID not found")
	}

	switch value := userValue.(type) {
	case uuid.UUID:
		return value, nil
	case string:
		return uuid.Parse(value)
	default:
		return uuid.Nil, fmt.Errorf("invalid user ID")
	}
}
