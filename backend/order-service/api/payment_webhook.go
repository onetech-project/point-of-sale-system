package api

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/point-of-sale-system/order-service/src/services"
)

// PaymentWebhookHandler handles Midtrans payment webhook notifications
type PaymentWebhookHandler struct {
	paymentService *services.PaymentService
}

// NewPaymentWebhookHandler creates a new payment webhook handler
func NewPaymentWebhookHandler(paymentService *services.PaymentService) *PaymentWebhookHandler {
	return &PaymentWebhookHandler{
		paymentService: paymentService,
	}
}

// HandleMidtransNotification handles POST /payments/midtrans/notification
// Implements T063: Payment webhook handler with signature verification
// Implements T065: Full notification payload logging for audit trail
func (h *PaymentWebhookHandler) HandleMidtransNotification(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse notification payload
	var notification services.MidtransNotification
	if err := c.Bind(&notification); err != nil {
		log.Error().
			Err(err).
			Str("remote_addr", c.RealIP()).
			Msg("Failed to parse webhook notification")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid notification payload",
		})
	}

	// Log full notification payload for audit trail (T065)
	notificationJSON, _ := json.Marshal(notification)
	log.Info().
		RawJSON("notification", notificationJSON).
		Str("order_id", notification.OrderID).
		Str("transaction_id", notification.TransactionID).
		Str("transaction_status", notification.TransactionStatus).
		Str("payment_type", notification.PaymentType).
		Str("gross_amount", notification.GrossAmount).
		Str("signature_key", notification.SignatureKey).
		Str("remote_addr", c.RealIP()).
		Msg("Received Midtrans webhook notification")

	// Process notification (includes signature verification, idempotency check, status updates)
	err := h.paymentService.ProcessNotification(ctx, &notification)
	if err != nil {
		// Log error but return 200 to prevent Midtrans retries
		// Invalid signatures or duplicate notifications should not trigger retries
		log.Error().
			Err(err).
			Str("order_id", notification.OrderID).
			Str("transaction_id", notification.TransactionID).
			Msg("Failed to process webhook notification")

		// Check if this is a signature failure or other error
		if err.Error() == "invalid signature" {
			// Return 403 for invalid signatures
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Invalid signature",
			})
		}

		// For other errors (e.g., database failures), return 200 to acknowledge receipt
		// but log the error for manual intervention
		return c.JSON(http.StatusOK, map[string]string{
			"status": "acknowledged",
			"note":   "notification received but processing failed - manual intervention required",
		})
	}

	// Success response
	log.Info().
		Str("order_id", notification.OrderID).
		Str("transaction_id", notification.TransactionID).
		Str("transaction_status", notification.TransactionStatus).
		Msg("Webhook notification processed successfully")

	return c.JSON(http.StatusOK, map[string]string{
		"status": "success",
	})
}

// RegisterRoutes registers payment webhook routes
func (h *PaymentWebhookHandler) RegisterRoutes(e *echo.Echo) {
	// Public webhook endpoint (no auth required - Midtrans sends notifications here)
	// Signature verification is handled in the service layer
	// Route matches API gateway path: /api/v1/webhooks/payments/midtrans/notification
	e.POST("/api/v1/webhooks/payments/midtrans/notification", h.HandleMidtransNotification)
}
