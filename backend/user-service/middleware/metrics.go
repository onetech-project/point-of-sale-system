package middleware

import (
	"net/http"
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	. "github.com/pos/user-service/src/observability"
	"github.com/pos/user-service/src/utils"
)

func MetricsMiddleware(e *echo.Echo) {
	e.Use(echoprometheus.NewMiddleware(utils.GetEnv("SERVICE_NAME")))

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
