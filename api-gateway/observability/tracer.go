package observability

import (
	"context"
	"log"

	"github.com/pos/api-gateway/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func InitTracer() func(context.Context) error {
	exporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(utils.GetEnv("OTEL_COLLECTOR_ENDPOINT")),
		otlptracegrpc.WithInsecure(),
	)

	if err != nil {
		log.Fatalf("failed to create OTLP trace exporter: %v", err)
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
	return tp.Shutdown
}
