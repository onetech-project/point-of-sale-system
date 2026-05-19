package api

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/pos/billing-service/src/models"
	"github.com/pos/billing-service/src/services"
	"github.com/pos/billing-service/src/utils"
)

// BillingHandler handles all billing HTTP endpoints.
type BillingHandler struct {
	svc *services.SubscriptionService
}

// NewBillingHandler creates a BillingHandler.
func NewBillingHandler(svc *services.SubscriptionService) *BillingHandler {
	return &BillingHandler{svc: svc}
}

// Health returns a basic liveness response.
func (h *BillingHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "ok",
		"service": utils.GetEnv("SERVICE_NAME"),
	})
}

// Ready returns a basic readiness response.
func (h *BillingHandler) Ready(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
}

// GetInternalSubscriptionStatus is the internal (no-auth) endpoint for API gateway enforcement.
// GET /internal/subscription/:tenant_id
func (h *BillingHandler) GetInternalSubscriptionStatus(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing tenant_id"})
	}
	resp, err := h.svc.GetSubscriptionStatus(c.Request().Context(), tenantID)
	if err != nil {
		return c.JSON(subscriptionStatusErrorCode(err), map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, resp)
}

// GetMySubscription returns the subscription status for the authenticated tenant.
// GET /api/v1/billing/subscription
func (h *BillingHandler) GetMySubscription(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing tenant ID"})
	}
	resp, err := h.svc.GetSubscriptionStatus(c.Request().Context(), tenantID)
	if err != nil {
		return c.JSON(subscriptionStatusErrorCode(err), map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, resp)
}

func subscriptionStatusErrorCode(err error) int {
	if err != nil && strings.Contains(err.Error(), "tenant not found") {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

// UpdateBillingCycle updates the tenant's preferred billing cycle.
// PUT /api/v1/billing/subscription/cycle
func (h *BillingHandler) UpdateBillingCycle(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing tenant ID"})
	}
	var body struct {
		BillingInterval string `json:"billing_interval"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if err := h.svc.UpdateBillingCycle(c.Request().Context(), tenantID, body.BillingInterval); err != nil {
		return c.JSON(billingActionErrorCode(err), map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

// UpgradeSubscription creates an invoice and initiates a Midtrans Snap payment.
// POST /api/v1/billing/subscription/upgrade
func (h *BillingHandler) UpgradeSubscription(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing tenant ID"})
	}
	var body struct {
		BillingInterval string `json:"billing_interval"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	result, err := h.svc.UpgradeSubscription(c.Request().Context(), tenantID, body.BillingInterval)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, result)
}

// SwitchBillingCycle creates a payment-backed invoice for changing billing cycle.
// POST /api/v1/billing/subscription/cycle-switch
func (h *BillingHandler) SwitchBillingCycle(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing tenant ID"})
	}
	var body struct {
		BillingInterval string `json:"billing_interval"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	result, err := h.svc.SwitchBillingCycle(c.Request().Context(), tenantID, body.BillingInterval)
	if err != nil {
		if locked, ok := services.AsCycleSwitchLockedError(err); ok {
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error":                err.Error(),
				"subscription_ends_at": locked.SubscriptionEndsAt,
			})
		}
		return c.JSON(billingActionErrorCode(err), map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, result)
}

func billingActionErrorCode(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}
	msg := err.Error()
	if strings.Contains(msg, "tenant not found") {
		return http.StatusNotFound
	}
	if strings.Contains(msg, "billing_interval") ||
		strings.Contains(msg, "already using") ||
		strings.Contains(msg, "require payment") {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

// ListInvoices returns all invoices for the tenant.
// GET /api/v1/billing/invoices
func (h *BillingHandler) ListInvoices(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing tenant ID"})
	}
	invoices, err := h.svc.GetInvoices(c.Request().Context(), tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if invoices == nil {
		invoices = []*models.BillingInvoice{}
	}
	return c.JSON(http.StatusOK, invoices)
}

// GetInvoice returns a specific invoice.
// GET /api/v1/billing/invoices/:id
func (h *BillingHandler) GetInvoice(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing tenant ID"})
	}
	invoiceID := c.Param("id")
	inv, err := h.svc.GetInvoice(c.Request().Context(), tenantID, invoiceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, inv)
}

// ListInvoicePaymentAttempts returns payment attempts for an invoice.
// GET /api/v1/billing/invoices/:id/payments
func (h *BillingHandler) ListInvoicePaymentAttempts(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing tenant ID"})
	}
	invoiceID := c.Param("id")
	attempts, err := h.svc.GetInvoicePaymentAttempts(c.Request().Context(), tenantID, invoiceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	if attempts == nil {
		attempts = []*models.BillingPaymentAttempt{}
	}
	return c.JSON(http.StatusOK, attempts)
}

// InitiatePayment creates (or re-creates) a Snap payment for an existing pending invoice.
// POST /api/v1/billing/invoices/:id/pay
func (h *BillingHandler) InitiatePayment(c echo.Context) error {
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing tenant ID"})
	}
	invoiceID := c.Param("id")
	result, err := h.svc.InitiatePayment(c.Request().Context(), tenantID, invoiceID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, result)
}

// HandleMidtransWebhook processes inbound Midtrans payment notifications.
// POST /webhook/billing
func (h *BillingHandler) HandleMidtransWebhook(c echo.Context) error {
	var payload services.MidtransWebhookPayload
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	if err := h.svc.HandlePaymentWebhook(c.Request().Context(), payload); err != nil {
		// Return 200 to Midtrans even on business logic errors to prevent retries for non-retryable errors.
		c.Logger().Errorf("webhook error: %v", err)
		return c.JSON(http.StatusOK, map[string]string{"status": "error", "message": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
