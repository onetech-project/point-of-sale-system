package observability

import (
	"context"
	"log"
	"time"

	"github.com/pos/api-gateway/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace/noop"
)

func InitTracer() func(context.Context) error {
	// Create a context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(utils.GetEnv("OTEL_COLLECTOR_ENDPOINT")),
		otlptracegrpc.WithInsecure(),
	)

	if err != nil {
		log.Printf("Warning: failed to create OTLP trace exporter: %v. Continuing with noop tracer.", err)
		// Use noop tracer provider as fallback
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func(context.Context) error { return nil }
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(utils.GetEnv("SERVICE_NAME")),
			semconv.DeploymentEnvironment(utils.GetEnv("ENVIRONMENT")),
		)),
	)

	otel.SetTracerProvider(tp)
	log.Println("OTLP tracer initialized successfully")
	return tp.Shutdown
}
