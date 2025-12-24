package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

func TraceLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		span := trace.SpanFromContext(c.Request().Context())
		traceID := span.SpanContext().TraceID().String()

		log.Info().
			Str("trace_id", traceID).
			Str("method", c.Request().Method).
			Str("path", c.Path()).
			Msg("incoming request")

		return next(c)
	}
}
