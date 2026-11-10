package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/point-of-sale-system/order-service/src/manjo"
	"github.com/point-of-sale-system/order-service/src/services"
)

// ManjoBasePath is the base URL to register at Manjo, after the public API
// URL. Manjo appends its own paths: it asks /v1.0/access-token/b2b for a token,
// then posts payment notifications to /v1.0/qr/qr-mpm-notify.
const ManjoBasePath = "/api/v1/webhooks/payments/manjo"

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
	e.POST(ManjoBasePath+manjo.PathAccessToken, h.HandleManjoAccessToken)
	e.POST(ManjoBasePath+manjo.PathNotify, h.HandleManjoNotification)
}

// HandleManjoAccessToken handles Manjo's POST <base>/v1.0/access-token/b2b,
// the token request Manjo makes before sending a notification.
func (h *PaymentWebhookHandler) HandleManjoAccessToken(c echo.Context) error {
	body, err := io.ReadAll(io.LimitReader(c.Request().Body, 64<<10))
	if err != nil {
		return c.JSON(http.StatusBadRequest, manjo.TokenInvalidMandatory)
	}
	status, resp := h.paymentService.IssueManjoAccessToken(c.Request().Context(), c.Request().Header, body)
	if status != http.StatusOK {
		log.Warn().
			Int("status", status).
			Str("response_code", resp.ResponseCode).
			Str("client_key", c.Request().Header.Get("X-CLIENT-KEY")).
			Str("remote_addr", c.RealIP()).
			Msg("Rejected Manjo access token request")
	}
	c.Response().Header().Set("X-TIMESTAMP", c.Request().Header.Get("X-TIMESTAMP"))
	c.Response().Header().Set("X-CLIENT-KEY", c.Request().Header.Get("X-CLIENT-KEY"))
	return c.JSON(status, resp)
}

// HandleManjoNotification handles Manjo's POST <base>/v1.0/qr/qr-mpm-notify.
// The bearer token, X-EXTERNAL-ID and signature are checked, then the payment
// status is confirmed with Manjo's query API before the order is updated.
func (h *PaymentWebhookHandler) HandleManjoNotification(c echo.Context) error {
	ctx := c.Request().Context()

	body, err := io.ReadAll(io.LimitReader(c.Request().Body, 1<<20))
	if err != nil {
		return c.JSON(http.StatusBadRequest, manjo.AckInvalidBody)
	}

	log.Info().
		RawJSON("notification", compactJSONOrEmpty(body)).
		Str("external_id", c.Request().Header.Get("X-EXTERNAL-ID")).
		Str("remote_addr", c.RealIP()).
		Msg("Received Manjo payment notification")

	result, err := h.paymentService.ProcessManjoNotification(ctx, c.Request().Header, c.Request().URL.Path, body)
	switch result {
	case services.ManjoNotificationInvalidToken:
		log.Warn().Err(err).Msg("Rejected Manjo notification")
		return c.JSON(http.StatusUnauthorized, manjo.AckInvalidToken)
	case services.ManjoNotificationInvalidSignature:
		return c.JSON(http.StatusUnauthorized, manjo.AckUnauthorized)
	case services.ManjoNotificationDuplicate:
		log.Warn().Err(err).Msg("Rejected Manjo notification")
		return c.JSON(http.StatusConflict, manjo.AckConflict)
	case services.ManjoNotificationInvalidBody:
		log.Warn().Err(err).Msg("Rejected Manjo notification")
		return c.JSON(http.StatusBadRequest, manjo.AckInvalidBody)
	}
	if err != nil {
		// Acknowledge so Manjo does not retry forever; the guest order page
		// re-checks the payment at Manjo while it is pending.
		log.Error().Err(err).Msg("Failed to process Manjo notification - manual intervention may be required")
	}
	return c.JSON(http.StatusOK, manjo.AckSuccess)
}

func compactJSONOrEmpty(body []byte) []byte {
	if json.Valid(body) {
		return body
	}
	return []byte(`{}`)
}
