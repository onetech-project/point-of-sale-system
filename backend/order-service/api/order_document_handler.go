package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/point-of-sale-system/order-service/src/services"
)

type OrderDocumentHandler struct {
	documentService *services.OrderDocumentService
}

type batchOrderDocumentsRequest struct {
	DocumentType string   `json:"document_type"`
	OrderIDs     []string `json:"order_ids"`
}

func NewOrderDocumentHandler(documentService *services.OrderDocumentService) *OrderDocumentHandler {
	return &OrderDocumentHandler{documentService: documentService}
}

func (h *OrderDocumentHandler) RegisterRoutes(e *echo.Echo) {
	adminOrders := e.Group("/api/v1/admin/orders")
	adminOrders.GET("/:id/documents/:document_type", h.downloadGuestOrderDocument)
	adminOrders.POST("/:id/documents/:document_type/resend", h.resendGuestOrderDocument)
	adminOrders.POST("/documents/batch", h.batchGuestOrderDocuments)

	offlineOrders := e.Group("/api/v1/admin/offline-orders")
	offlineOrders.GET("/:id/documents/:document_type", h.downloadOfflineOrderDocument)
	offlineOrders.POST("/:id/documents/:document_type/resend", h.resendOfflineOrderDocument)
	offlineOrders.POST("/documents/batch", h.batchOfflineOrderDocuments)
}

func (h *OrderDocumentHandler) downloadGuestOrderDocument(c echo.Context) error {
	return h.downloadDocument(c, services.OrderDocumentSourceAny)
}

func (h *OrderDocumentHandler) resendGuestOrderDocument(c echo.Context) error {
	return h.resendDocument(c, services.OrderDocumentSourceAny)
}

func (h *OrderDocumentHandler) batchGuestOrderDocuments(c echo.Context) error {
	return h.batchDocuments(c, services.OrderDocumentSourceAny)
}

func (h *OrderDocumentHandler) downloadOfflineOrderDocument(c echo.Context) error {
	return h.downloadDocument(c, services.OrderDocumentSourceOffline)
}

func (h *OrderDocumentHandler) resendOfflineOrderDocument(c echo.Context) error {
	return h.resendDocument(c, services.OrderDocumentSourceOffline)
}

func (h *OrderDocumentHandler) batchOfflineOrderDocuments(c echo.Context) error {
	return h.batchDocuments(c, services.OrderDocumentSourceOffline)
}

func (h *OrderDocumentHandler) downloadDocument(c echo.Context, source services.OrderDocumentSource) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	document, err := h.documentService.GeneratePDF(
		c.Request().Context(),
		source,
		tenantID,
		c.Param("id"),
		c.Param("document_type"),
	)
	if err != nil {
		return writeOrderDocumentError(c, err)
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+document.Filename+`"`)
	return c.Blob(http.StatusOK, "application/pdf", document.Content)
}

func (h *OrderDocumentHandler) resendDocument(c echo.Context, source services.OrderDocumentSource) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	userID := c.Request().Header.Get("X-User-ID")
	if err := h.documentService.Resend(
		c.Request().Context(),
		source,
		tenantID,
		c.Param("id"),
		c.Param("document_type"),
		userID,
	); err != nil {
		return writeOrderDocumentError(c, err)
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"message":       "document resend queued",
		"order_id":      c.Param("id"),
		"document_type": c.Param("document_type"),
	})
}

func (h *OrderDocumentHandler) batchDocuments(c echo.Context, source services.OrderDocumentSource) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	var req batchOrderDocumentsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	document, err := h.documentService.GenerateBatchZIP(
		c.Request().Context(),
		source,
		tenantID,
		req.OrderIDs,
		req.DocumentType,
	)
	if err != nil {
		return writeOrderDocumentError(c, err)
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+document.Filename+`"`)
	return c.Blob(http.StatusOK, "application/zip", document.Content)
}

func writeOrderDocumentError(c echo.Context, err error) error {
	var batchErr *services.BatchOrderDocumentError
	if errors.As(err, &batchErr) {
		return c.JSON(http.StatusConflict, map[string]interface{}{
			"error":  batchErr.Error(),
			"errors": batchErr.Errors,
		})
	}

	var docErr *services.OrderDocumentError
	if errors.As(err, &docErr) {
		status := http.StatusBadRequest
		switch docErr.Code {
		case services.OrderDocumentErrorNotFound:
			status = http.StatusNotFound
		case services.OrderDocumentErrorForbidden:
			status = http.StatusForbidden
		case services.OrderDocumentErrorIneligible:
			status = http.StatusConflict
		case services.OrderDocumentErrorMissingEmail, services.OrderDocumentErrorInvalidEmail:
			status = http.StatusUnprocessableEntity
		case services.OrderDocumentErrorInvalidBatch, services.OrderDocumentErrorInvalidType:
			status = http.StatusBadRequest
		}

		return c.JSON(status, map[string]interface{}{
			"error":        docErr.Message,
			"code":         docErr.Code,
			"order_id":     docErr.OrderID,
			"documentable": false,
		})
	}

	return c.JSON(http.StatusInternalServerError, map[string]string{
		"error": "Failed to process order document",
	})
}
