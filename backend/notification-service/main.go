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
	"github.com/pos/notification-service/src/queue"
	"github.com/pos/notification-service/src/services"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Database connection
	dbURL := getEnv("DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/pos_db?sslmode=disable")
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

	// Kafka configuration
	kafkaBrokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	kafkaTopic := getEnv("KAFKA_TOPIC", "notification-events")
	kafkaGroupID := getEnv("KAFKA_GROUP_ID", "notification-service-group")

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
	port := getEnv("PORT", "8085")
	log.Printf("Notification service starting on port %s", port)
	if err := e.Start(":" + port); err != nil {
		log.Printf("Server stopped: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
