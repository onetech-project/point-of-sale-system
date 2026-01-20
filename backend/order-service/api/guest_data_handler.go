package api

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/point-of-sale-system/order-service/src/queue"
	"github.com/point-of-sale-system/order-service/src/services"
	"github.com/point-of-sale-system/order-service/src/utils"
	"github.com/rs/zerolog/log"
)

// GuestDataHandler handles guest customer data access and deletion requests (T144-T145)
// Implements UU PDP Article 4 (Right to Access) and Article 5 (Right to Deletion)
type GuestDataHandler struct {
	guestDataService     *services.GuestDataService
	guestDeletionService *services.GuestDeletionService
	notificationProducer *queue.KafkaProducer
}

// NewGuestDataHandler creates a new guest data handler
func NewGuestDataHandler(
	db *sql.DB,
	encryptor utils.Encryptor,
	auditPublisher *utils.AuditPublisher,
	notificationProducer *queue.KafkaProducer,
) *GuestDataHandler {
	guestDataService := services.NewGuestDataService(db, encryptor)
	guestDeletionService := services.NewGuestDeletionService(db, encryptor, auditPublisher)

	return &GuestDataHandler{
		guestDataService:     guestDataService,
		guestDeletionService: guestDeletionService,
		notificationProducer: notificationProducer,
	}
}

// GetGuestData handles GET /api/v1/public/orders/:order_reference/data (T144)
// Requires order_reference + email OR phone for verification
func (h *GuestDataHandler) GetGuestData(c echo.Context) error {
	ctx := c.Request().Context()
	orderReference := c.Param("order_reference")

	if orderReference == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "order_reference is required",
		})
	}

	// Get verification credentials from query params or body
	email := c.QueryParam("email")
	phone := c.QueryParam("phone")

	if email == "" && phone == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Either email or phone must be provided for verification",
		})
	}

	// Verify access (T146)
	var emailPtr, phonePtr *string
	if email != "" {
		emailPtr = &email
	}
	if phone != "" {
		phonePtr = &phone
	}

	verified, err := h.guestDataService.VerifyGuestAccess(ctx, orderReference, emailPtr, phonePtr)
	if err != nil {
		// log the error internally but return a generic message
		log.Error().Err(err).Str("order_reference", orderReference).Msg("Failed to verify guest access")
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to verify access",
		})
	}

	if !verified {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "Verification failed - email or phone does not match order",
		})
	}

	// Get guest data
	data, err := h.guestDataService.GetGuestOrderData(ctx, orderReference)
	if err != nil {
		log.Error().Err(err).Str("order_reference", orderReference).Msg("Failed to get guest order data")
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Order not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve order data",
		})
	}

	return c.JSON(http.StatusOK, data)
}

// DeleteGuestData handles POST /api/v1/public/orders/:order_reference/delete (T145)
// Anonymizes guest personal data while preserving order record
func (h *GuestDataHandler) DeleteGuestData(c echo.Context) error {
	ctx := c.Request().Context()
	orderReference := c.Param("order_reference")

	if orderReference == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "order_reference is required",
		})
	}

	// Parse request body for verification
	var req struct {
		Email *string `json:"email"`
		Phone *string `json:"phone"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	if req.Email == nil && req.Phone == nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Either email or phone must be provided for verification",
		})
	}

	// Verify access (T146)
	verified, err := h.guestDataService.VerifyGuestAccess(ctx, orderReference, req.Email, req.Phone)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to verify access",
		})
	}

	if !verified {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "Verification failed - email or phone does not match order",
		})
	}

	// Check if can anonymize
	canAnonymize, err := h.guestDeletionService.CanAnonymizeOrder(ctx, orderReference)
	if err != nil {
		if err.Error() == "order not found" {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Order not found",
			})
		}
		if err.Error() == "order not completed or cancelled" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Only completed or cancelled orders can be anonymized",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to check order status",
		})
	}

	if !canAnonymize {
		return c.JSON(http.StatusConflict, map[string]interface{}{
			"error": "Order data has already been anonymized",
		})
	}

	// Get customer data before anonymization (for notification)
	guestData, err := h.guestDataService.GetGuestOrderData(ctx, orderReference)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve guest data",
		})
	}

	// Anonymize guest data
	if err := h.guestDeletionService.AnonymizeGuestData(ctx, orderReference); err != nil {
		log.Error().Err(err).Str("order_reference", orderReference).Msg("Failed to anonymize guest data")
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to anonymize guest data",
		})
	}

	// Send confirmation notification (T155) - non-blocking
	go func() {
		if req.Email != nil && *req.Email != "" {
			// Use background context with timeout - don't let request cancellation affect notification
			notifCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			event := map[string]interface{}{
				"event_type": "guest_data_deleted",
				"tenant_id":  guestData.TenantID, // Include tenant_id for notification record
				"data": map[string]interface{}{
					"email":           *req.Email,
					"order_reference": orderReference,
					"customer_name":   guestData.CustomerInfo.Name,
					"anonymized_at":   time.Now().Format(time.RFC3339),
					"language":        "id", // Default to Indonesian, can be enhanced with language detection
				},
			}
			_ = h.notificationProducer.Publish(notifCtx, orderReference, event)
		}
	}()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":         "Guest data successfully anonymized",
		"order_reference": orderReference,
		"anonymized_at":   "now",
	})
}
