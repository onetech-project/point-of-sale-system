package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const RequestIDHeader = "X-Request-ID"

// RequestIDMiddleware adds a unique request ID to each request
// If the client provides an X-Request-ID header, it will be used
// Otherwise, a new UUID will be generated
func RequestIDMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		res := c.Response()

		// Get request ID from header or generate new one
		requestID := req.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add request ID to response header
		res.Header().Set(RequestIDHeader, requestID)

		// Store in context for logging
		c.Set("request_id", requestID)

		return next(c)
	}
}
