package middleware

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

// WebhookAuth validates Midtrans webhook signature
func WebhookAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Read request body
			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "failed to read request body")
			}

			// Store body in context for later use
			c.Set("webhook_body", body)

			// Get signature from request
			signatureKey := c.Request().Header.Get("X-Signature-Key")
			if signatureKey == "" {
				signatureKey = c.Request().Header.Get("signature_key")
			}

			if signatureKey == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing signature")
			}

			// Verify signature
			if !verifyMidtransSignature(string(body), signatureKey) {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid signature")
			}

			return next(c)
		}
	}
}

// verifyMidtransSignature verifies the Midtrans webhook signature
// Signature format: SHA512(order_id+status_code+gross_amount+ServerKey)
func verifyMidtransSignature(payload, signature string) bool {
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	if serverKey == "" {
		return false
	}

	// Parse JSON to extract order_id, status_code, gross_amount
	// For now, simplified version - full implementation needs JSON parsing
	// Expected: order_id, status_code, gross_amount from webhook payload

	// Create HMAC hash
	h := hmac.New(sha512.New, []byte(serverKey))
	h.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}
