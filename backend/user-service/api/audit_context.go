package api

import (
	"github.com/labstack/echo/v4"
	"github.com/pos/user-service/src/services"
)

func auditContextFromRequest(c echo.Context) services.AuditContext {
	return services.AuditContext{
		ActorID:    c.Request().Header.Get("X-User-ID"),
		ActorEmail: c.Request().Header.Get("X-User-Email"),
		ActorRole:  c.Request().Header.Get("X-User-Role"),
		IPAddress:  c.RealIP(),
		UserAgent:  c.Request().UserAgent(),
		RequestID:  requestIDFromHeaders(c),
	}
}

func requestIDFromHeaders(c echo.Context) string {
	for _, header := range []string{"X-Request-ID", "X-Correlation-ID", "X-Amzn-Trace-Id"} {
		if value := c.Request().Header.Get(header); value != "" {
			return value
		}
	}
	return ""
}
