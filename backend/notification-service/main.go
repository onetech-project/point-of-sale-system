package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/pos/notification-service/api"
	"github.com/pos/notification-service/middleware"
	"github.com/pos/notification-service/src/observability"
	"github.com/pos/notification-service/src/queue"
	"github.com/pos/notification-service/src/services"
	"github.com/pos/notification-service/src/utils"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func main() {
	observability.InitLogger()
	shutdown := observability.InitTracer()
	defer shutdown(nil)

	e := echo.New()

	e.Use(emw.Logger())
	e.Use(emw.Recover())

	// OTEL
	e.Use(otelecho.Middleware(utils.GetEnv("SERVICE_NAME")))

	// Trace → Log bridge
	e.Use(middleware.TraceLogger)

	// Logging with PII masking (T064)
	e.Use(middleware.LoggingMiddleware)

	middleware.MetricsMiddleware(e)

	// Database connection
	dbURL := utils.GetEnv("DATABASE_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Health endpoints
	e.GET("/health", api.HealthCheck)
	e.GET("/ready", api.ReadyCheck)

	// Notification service
	notificationService, err := services.NewNotificationService(db)
	if err != nil {
		log.Fatalf("Failed to create notification service: %v", err)
	}

	// Notification config service
	notificationConfigService := services.NewNotificationConfigService(db)

	// API handlers
	testNotificationHandler := api.NewTestNotificationHandler(notificationService)
	notificationConfigHandler := api.NewNotificationConfigHandler(notificationConfigService)
	notificationHistoryHandler := api.NewNotificationHistoryHandler(notificationService)
	resendNotificationHandler := api.NewResendNotificationHandler(notificationService)

	// API routes with rate limiting
	apiV1 := e.Group("/api/v1")

	// Test notification endpoint with stricter rate limiting (5 requests/min)
	apiV1.POST("/notifications/test", testNotificationHandler.SendTestNotification, middleware.RateLimitForTestNotifications())

	// Notification config endpoints with normal rate limiting
	apiV1.GET("/notifications/config", notificationConfigHandler.GetNotificationConfig, middleware.RateLimit())
	apiV1.PATCH("/notifications/config", notificationConfigHandler.PatchNotificationConfig, middleware.RateLimit())

	// Notification history endpoints
	apiV1.GET("/notifications/history", notificationHistoryHandler.GetNotificationHistory, middleware.RateLimit())
	apiV1.POST("/notifications/:notification_id/resend", resendNotificationHandler.ResendNotification, middleware.RateLimit())

	// Kafka configuration
	kafkaBrokers := strings.Split(utils.GetEnv("KAFKA_BROKERS"), ",")
	kafkaTopic := utils.GetEnv("KAFKA_TOPIC")
	kafkaGroupID := utils.GetEnv("KAFKA_GROUP_ID")

	// Start Kafka consumer
	consumer := queue.NewKafkaConsumer(
		kafkaBrokers,
		kafkaTopic,
		kafkaGroupID,
		notificationService.HandleEvent,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumer in background
	go consumer.Start(ctx)

	// Start retry worker in background
	retryWorker, err := services.NewRetryWorker(db, notificationService)
	if err != nil {
		log.Fatalf("Failed to create retry worker: %v", err)
	}
	go retryWorker.Start(ctx)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down notification service...")
		cancel()
		consumer.Close()
		e.Close()
	}()

	// Start HTTP server
	port := utils.GetEnv("PORT")
	log.Printf("Notification service starting on port %s", port)
	if err := e.Start(":" + port); err != nil {
		log.Printf("Server stopped: %v", err)
	}
}
