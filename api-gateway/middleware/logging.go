package middleware

import (
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func Logging() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}
			c.Set("request_id", requestID)
			c.Response().Header().Set("X-Request-ID", requestID)

			err := next(c)

			duration := time.Since(start)

			tenantID := c.Get("tenant_id")
			userID := c.Get("user_id")

			logFields := map[string]interface{}{
				"timestamp":   start.Format(time.RFC3339),
				"request_id":  requestID,
				"method":      c.Request().Method,
				"path":        c.Request().URL.Path,
				"status":      c.Response().Status,
				"duration_ms": duration.Milliseconds(),
				"ip":          c.RealIP(),
				"user_agent":  c.Request().UserAgent(),
			}

			if tenantID != nil {
				logFields["tenant_id"] = tenantID
			}
			if userID != nil {
				logFields["user_id"] = userID
			}

			if err != nil {
				logFields["error"] = err.Error()
				c.Logger().Errorj(logFields)
			} else {
				c.Logger().Infoj(logFields)
			}

			return err
		}
	}
}
