package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/pos/tenant-service/src/services"
)

type TenantConfigHandler struct {
	configService *services.TenantConfigService
}

func NewTenantConfigHandler(configService *services.TenantConfigService) *TenantConfigHandler {
	return &TenantConfigHandler{
		configService: configService,
	}
}

// GetPublicTenantConfig handles GET /public/tenants/:tenant_slug/config
func (h *TenantConfigHandler) GetPublicTenantConfig(c echo.Context) error {
	tenantSlug := c.Param("tenant_slug")
	if tenantSlug == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_slug is required",
		})
	}

	config, err := h.configService.GetDeliveryConfig(c.Request().Context(), tenantSlug)

	if errors.Is(err, services.ErrTenantNotFound) {
		log.Warn().Str("tenant_slug", tenantSlug).Msg("Tenant not found")
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Tenant not found",
		})
	}

	var unavailable *services.TenantUnavailableError
	if errors.As(err, &unavailable) {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error":               "Tenant currently unavailable",
			"message":             "This tenant is currently not available at this moment.",
			"status":              unavailable.Status,
			"subscription_status": unavailable.SubscriptionStatus,
		})
	}

	var paymentMissing *services.PaymentNotConfiguredError
	if errors.As(err, &paymentMissing) {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error":                "Payment gateway is not configured",
			"message":              "Guest ordering is unavailable until a payment gateway is configured.",
			"reason":               paymentMissing.Reason(),
			"status":               paymentMissing.Status,
			"subscription_status":  paymentMissing.SubscriptionStatus,
			"payment_gateway":      paymentMissing.PaymentGateway,
			"payment_configured":   false,
			"midtrans_configured":  paymentMissing.PaymentGateway != "midtrans",
			"midtrans_environment": paymentMissing.MidtransEnvironment,
		})
	}

	if err != nil {
		log.Error().Err(err).Str("tenant_slug", tenantSlug).Msg("Failed to get tenant config")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve tenant configuration",
		})
	}

	return c.JSON(http.StatusOK, config)
}

// UpdateTenantConfig handles PATCH /admin/tenants/:tenant_id/config (for admin use)
func (h *TenantConfigHandler) UpdateTenantConfig(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	var req services.DeliveryConfig
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	req.TenantID = tenantID

	if err := h.configService.UpdateDeliveryConfig(c.Request().Context(), &req); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Tenant configuration updated successfully",
	})
}

// GetMidtransConfig handles GET /admin/tenants/:tenant_id/midtrans-config
func (h *TenantConfigHandler) GetMidtransConfig(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}
	if ok, err := ensureTenantPathMatchesHeader(c, tenantID); !ok || err != nil {
		return err
	}

	config, err := h.configService.GetMidtransConfig(c.Request().Context(), tenantID)
	if err != nil {
		c.Logger().Errorf("Failed to get Midtrans config for tenant %s: %v", tenantID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve Midtrans configuration",
		})
	}

	return c.JSON(http.StatusOK, config)
}

// UpdateMidtransConfig handles PATCH /admin/tenants/:tenant_id/midtrans-config
func (h *TenantConfigHandler) UpdateMidtransConfig(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}
	if ok, err := ensureTenantPathMatchesHeader(c, tenantID); !ok || err != nil {
		return err
	}

	var req services.MidtransConfig
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	req.TenantID = tenantID

	if err := h.configService.UpdateMidtransConfig(c.Request().Context(), &req); err != nil {
		return paymentConfigError(c, err, "Failed to update Midtrans configuration")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Midtrans configuration updated successfully",
	})
}

// GetManjoConfig handles GET /admin/tenants/:tenant_id/manjo-config.
// The client secret and private key are never returned.
func (h *TenantConfigHandler) GetManjoConfig(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	if ok, err := ensureTenantPathMatchesHeader(c, tenantID); !ok || err != nil {
		return err
	}

	config, err := h.configService.GetManjoConfig(c.Request().Context(), tenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get Manjo config")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve Manjo configuration",
		})
	}
	return c.JSON(http.StatusOK, config)
}

// UpdateManjoConfig handles PATCH /admin/tenants/:tenant_id/manjo-config
func (h *TenantConfigHandler) UpdateManjoConfig(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	if ok, err := ensureTenantPathMatchesHeader(c, tenantID); !ok || err != nil {
		return err
	}

	var req services.UpdateManjoConfigRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	req.TenantID = tenantID

	if err := h.configService.UpdateManjoConfig(c.Request().Context(), &req); err != nil {
		return paymentConfigError(c, err, "Failed to update Manjo configuration")
	}
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Manjo configuration updated successfully",
	})
}

// GetInternalManjoCredentials handles GET /internal/tenants/:tenant_id/manjo-credentials.
// It returns the decrypted secrets, so it is only reachable inside the service
// network: the API gateway does not route /internal paths.
func (h *TenantConfigHandler) GetInternalManjoCredentials(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	creds, err := h.configService.GetManjoCredentials(c.Request().Context(), tenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get Manjo credentials")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve Manjo credentials",
		})
	}
	return c.JSON(http.StatusOK, creds)
}

// GetPaymentGateway handles GET /admin/tenants/:tenant_id/payment-gateway
func (h *TenantConfigHandler) GetPaymentGateway(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	if ok, err := ensureTenantPathMatchesHeader(c, tenantID); !ok || err != nil {
		return err
	}

	settings, err := h.configService.GetPaymentGatewaySettings(c.Request().Context(), tenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get payment gateway settings")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve payment gateway settings",
		})
	}
	return c.JSON(http.StatusOK, settings)
}

// UpdatePaymentGateway handles PATCH /admin/tenants/:tenant_id/payment-gateway
func (h *TenantConfigHandler) UpdatePaymentGateway(c echo.Context) error {
	tenantID := c.Param("tenant_id")
	if ok, err := ensureTenantPathMatchesHeader(c, tenantID); !ok || err != nil {
		return err
	}

	var req struct {
		DefaultGateway string `json:"default_gateway"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if err := h.configService.SetDefaultPaymentGateway(c.Request().Context(), tenantID, req.DefaultGateway); err != nil {
		return paymentConfigError(c, err, "Failed to update default payment gateway")
	}
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Default payment gateway updated successfully",
	})
}

// paymentConfigError answers 400 for validation errors and 500 otherwise,
// without leaking internal error text.
func paymentConfigError(c echo.Context, err error, message string) error {
	var validationErr *services.ValidationError
	if errors.As(err, &validationErr) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": validationErr.Message})
	}
	log.Error().Err(err).Str("tenant_id", c.Param("tenant_id")).Msg(message)
	return c.JSON(http.StatusInternalServerError, map[string]string{"error": message})
}

func ensureTenantPathMatchesHeader(c echo.Context, tenantID string) (bool, error) {
	if tenantID == "" {
		return false, c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}
	requestTenantID := strings.TrimSpace(c.Request().Header.Get("X-Tenant-ID"))
	if requestTenantID == "" || requestTenantID == tenantID {
		return true, nil
	}
	return false, c.JSON(http.StatusForbidden, map[string]string{
		"error": "Cannot access payment configuration for another tenant",
	})
}
