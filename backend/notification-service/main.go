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
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/pos/notification-service/api"
	apimiddleware "github.com/pos/notification-service/api/middleware"
	"github.com/pos/notification-service/src/queue"
	"github.com/pos/notification-service/src/services"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Database connection
	dbURL := getEnv("DATABASE_URL")
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
	notificationService := services.NewNotificationService(db)

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
	apiV1.POST("/notifications/test", testNotificationHandler.SendTestNotification, apimiddleware.RateLimitForTestNotifications())

	// Notification config endpoints with normal rate limiting
	apiV1.GET("/notifications/config", notificationConfigHandler.GetNotificationConfig, apimiddleware.RateLimit())
	apiV1.PATCH("/notifications/config", notificationConfigHandler.PatchNotificationConfig, apimiddleware.RateLimit())

	// Notification history endpoints
	apiV1.GET("/notifications/history", notificationHistoryHandler.GetNotificationHistory, apimiddleware.RateLimit())
	apiV1.POST("/notifications/:notification_id/resend", resendNotificationHandler.ResendNotification, apimiddleware.RateLimit())

	// Kafka configuration
	kafkaBrokers := strings.Split(getEnv("KAFKA_BROKERS"), ",")
	kafkaTopic := getEnv("KAFKA_TOPIC")
	kafkaGroupID := getEnv("KAFKA_GROUP_ID")

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
	retryWorker := services.NewRetryWorker(db, notificationService)
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
	port := getEnv("PORT")
	log.Printf("Notification service starting on port %s", port)
	if err := e.Start(":" + port); err != nil {
		log.Printf("Server stopped: %v", err)
	}
}

func getEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	// throw error: missing environment variable
	panic("Environment variable " + key + " is not set")
}
