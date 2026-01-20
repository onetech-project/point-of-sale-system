package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/point-of-sale-system/order-service/src/utils"
	"github.com/rs/zerolog/log"
)

// LoggingMiddleware logs HTTP requests and responses with PII masking
// Integrates LogMasker from src/utils/masker.go per T062
// Implements FR-013 to FR-017: Masks email, phone, token, IP, name in logs
func LoggingMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	masker := utils.NewLogMasker()

	return func(c echo.Context) error {
		start := time.Now()

		// Capture request details
		req := c.Request()
		method := req.Method
		path := c.Path()
		clientIP := c.RealIP()

		// Mask client IP before logging
		maskedIP := masker.MaskIP(clientIP)

		// Read and mask request body if present
		var requestBody string
		if req.Body != nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err == nil {
				requestBody = string(bodyBytes)
				// Restore body for handler
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Mask sensitive data in request body
		maskedRequestBody := masker.MaskAll(requestBody)
		maskedRequestBody = masker.MaskSensitiveFields(maskedRequestBody)

		// Log incoming request with masked PII
		log.Info().
			Str("method", method).
			Str("path", path).
			Str("client_ip", maskedIP).
			Str("user_agent", req.UserAgent()).
			Msg("incoming request")

		// Log request body only for non-GET requests and if not empty
		if method != "GET" && requestBody != "" {
			log.Debug().
				Str("method", method).
				Str("path", path).
				Str("body", maskedRequestBody).
				Msg("request body")
		}

		// Call next handler
		err := next(c)

		// Log response details
		duration := time.Since(start)
		statusCode := c.Response().Status

		logEvent := log.Info().
			Str("method", method).
			Str("path", path).
			Str("client_ip", maskedIP).
			Int("status", statusCode).
			Dur("duration_ms", duration)

		if err != nil {
			// Mask error messages that might contain PII
			maskedError := masker.MaskAll(err.Error())
			logEvent.Str("error", maskedError)
		}

		logEvent.Msg("request completed")

		return err
	}
}
