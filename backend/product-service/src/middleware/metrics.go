package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	. "github.com/pos/backend/product-service/src/observability"
	"github.com/pos/backend/product-service/src/utils"
)

// MetricsMiddleware logs API response times and status codes
func MetricsMiddleware(e *echo.Echo) {
	e.Use(echoprometheus.NewMiddleware(utils.GetEnv("SERVICE_NAME")))

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Process request
			err := next(c)

			// Calculate duration
			duration := time.Since(start)

			// Get response status
			status := c.Response().Status

			// Get request info
			method := c.Request().Method
			path := c.Request().URL.Path

			// Get request ID if available
			requestID := ""
			if id := c.Get("request_id"); id != nil {
				requestID = id.(string)
			}

			// Log metrics
			utils.Log.Info("API Request: method=%s, path=%s, status=%d, duration=%s, request_id=%s",
				method, path, status, duration, requestID)

			// Add metrics to response header
			c.Response().Header().Set("X-Response-Time", strconv.FormatInt(duration.Milliseconds(), 10)+"ms")

			HttpRequestsTotal.WithLabelValues(
				c.Request().Method,
				c.Path(),
				http.StatusText(status),
			).Inc()

			HttpRequestDuration.WithLabelValues(
				c.Request().Method,
				c.Path(),
			).Observe(time.Since(start).Seconds())

			return err
		}
	})

	e.GET("/metrics", echoprometheus.NewHandler())
}
