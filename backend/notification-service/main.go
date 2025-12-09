package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/pos/notification-service/api"
	"github.com/pos/notification-service/src/providers"
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

	// Redis provider setup
	// Prefer REDIS_URL if provided, otherwise use REDIS_HOST and REDIS_DB
	redisURL := os.Getenv("REDIS_URL")
	var redisProvider *providers.RedisProvider
	if redisURL != "" {
		// parse redis URL via redis.ParseURL
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Fatalf("Failed to parse REDIS_URL: %v", err)
		}
		retention := int64(86400)
		maxlen := int64(10000)
		if v := os.Getenv("REDIS_STREAM_RETENTION_SECONDS"); v != "" {
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				retention = parsed
			}
		}
		if v := os.Getenv("REDIS_MAX_STREAM_LEN"); v != "" {
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				maxlen = parsed
			}
		}
		redisProvider = providers.NewRedisProvider(opts, retention, maxlen)
	} else {
		// build options from host/port
		host := getEnv("REDIS_HOST", "localhost:6379")
		// go-redis expects address and DB separately
		opts := &redis.Options{
			Addr: host,
		}
		retention := int64(86400)
		maxlen := int64(10000)
		if v := os.Getenv("REDIS_STREAM_RETENTION_SECONDS"); v != "" {
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				retention = parsed
			}
		}
		if v := os.Getenv("REDIS_MAX_STREAM_LEN"); v != "" {
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				maxlen = parsed
			}
		}
		redisProvider = providers.NewRedisProvider(opts, retention, maxlen)
	}

	// Notification service
	notificationService := services.NewNotificationService(db, redisProvider)

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
