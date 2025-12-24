package middleware

import (
	"net/http"
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/point-of-sale-system/order-service/src/config"
	. "github.com/point-of-sale-system/order-service/src/observability"
)

func MetricsMiddleware(e *echo.Echo) {
	e.Use(echoprometheus.NewMiddleware(config.GetEnvAsString("SERVICE_NAME")))

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			status := c.Response().Status

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
