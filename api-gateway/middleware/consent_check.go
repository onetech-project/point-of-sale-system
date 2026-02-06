package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/utils"
)

// ConsentStatus represents the active consent status for a subject
type ConsentStatus struct {
	PurposeCode string    `json:"purpose_code"`
	Granted     bool      `json:"granted"`
	RevokedAt   *string   `json:"revoked_at"`
	GrantedAt   time.Time `json:"granted_at"`
}

// ConsentCheckResponse represents the API response from audit-service
type ConsentCheckResponse struct {
	Data struct {
		SubjectType string          `json:"subject_type"`
		Consents    []ConsentStatus `json:"consents"`
	} `json:"data"`
}

// RequireConsentMiddleware verifies that the user has granted required consents before allowing data operations
// Usage: e.POST("/api/orders", handler, middleware.RequireConsentMiddleware([]string{"operational", "third_party_midtrans"}))
func RequireConsentMiddleware(requiredPurposes []string) echo.MiddlewareFunc {
	auditServiceURL := utils.GetEnv("AUDIT_SERVICE_URL")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract subject information from context
			tenantID := c.Get("tenant_id")
			if tenantID == nil {
				// Skip consent check for unauthenticated requests (will be caught by auth middleware)
				return next(c)
			}

			// For guest users, check guest_order_id from query param
			guestOrderID := c.QueryParam("guest_order_id")
			var checkURL string

			if guestOrderID != "" {
				// Guest consent check
				checkURL = fmt.Sprintf("%s/api/v1/consent/status?guest_order_id=%s", auditServiceURL, guestOrderID)
			} else {
				// Tenant user consent check (authenticated)
				userID := c.Get("user_id")
				if userID == nil {
					return echo.NewHTTPError(http.StatusUnauthorized, "User ID not found in context")
				}
				checkURL = fmt.Sprintf("%s/api/v1/consent/status", auditServiceURL)
			}

			// Call audit-service to check consent status
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create consent check request")
			}

			// Forward authentication headers
			if authHeader := c.Request().Header.Get("Authorization"); authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}
			req.Header.Set("X-Tenant-ID", fmt.Sprintf("%v", tenantID))

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				// Log error but allow request to proceed (fail-open for availability)
				c.Logger().Warnf("Consent check failed - allowing request: %v", err)
				return next(c)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("Consent check failed: %s", string(body)))
			}

			// Parse consent status response
			var consentResp ConsentCheckResponse
			if err := json.NewDecoder(resp.Body).Decode(&consentResp); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse consent response")
			}

			// Verify all required purposes are granted and active
			grantedPurposes := make(map[string]bool)
			for _, consent := range consentResp.Data.Consents {
				if consent.Granted && consent.RevokedAt == nil {
					grantedPurposes[consent.PurposeCode] = true
				}
			}

			missingPurposes := []string{}
			for _, purpose := range requiredPurposes {
				if !grantedPurposes[purpose] {
					missingPurposes = append(missingPurposes, purpose)
				}
			}

			if len(missingPurposes) > 0 {
				return echo.NewHTTPError(http.StatusForbidden, map[string]interface{}{
					"error": map[string]interface{}{
						"code":             "CONSENT_REQUIRED",
						"message":          "Required consent not granted",
						"missing_purposes": missingPurposes,
					},
				})
			}

			// All required consents are granted - proceed with request
			return next(c)
		}
	}
}

// OptionalConsentCheck adds consent status to context without blocking the request
// Useful for analytics/marketing features that should gracefully degrade
func OptionalConsentCheck(purposeCode string) echo.MiddlewareFunc {
	auditServiceURL := utils.GetEnv("AUDIT_SERVICE_URL")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := c.Get("tenant_id")
			if tenantID == nil {
				// No authentication, skip check
				c.Set(fmt.Sprintf("consent_%s", purposeCode), false)
				return next(c)
			}

			checkURL := fmt.Sprintf("%s/api/v1/consent/status", auditServiceURL)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
			if err != nil {
				c.Set(fmt.Sprintf("consent_%s", purposeCode), false)
				return next(c)
			}

			if authHeader := c.Request().Header.Get("Authorization"); authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}
			req.Header.Set("X-Tenant-ID", fmt.Sprintf("%v", tenantID))

			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				c.Set(fmt.Sprintf("consent_%s", purposeCode), false)
				return next(c)
			}
			defer resp.Body.Close()

			var consentResp ConsentCheckResponse
			if resp.StatusCode == http.StatusOK && json.NewDecoder(resp.Body).Decode(&consentResp) == nil {
				for _, consent := range consentResp.Data.Consents {
					if consent.PurposeCode == purposeCode && consent.Granted && consent.RevokedAt == nil {
						c.Set(fmt.Sprintf("consent_%s", purposeCode), true)
						return next(c)
					}
				}
			}

			c.Set(fmt.Sprintf("consent_%s", purposeCode), false)
			return next(c)
		}
	}
}
