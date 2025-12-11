package main

import (
	"database/sql"
	"log"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/pos/user-service/api"
	"github.com/pos/user-service/src/queue"
	"github.com/pos/user-service/src/services"
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

	// Kafka configuration
	kafkaBrokers := strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	kafkaTopic := getEnv("KAFKA_TOPIC", "notification-events")

	// Initialize Kafka producer
	eventProducer := queue.NewKafkaProducer(kafkaBrokers, kafkaTopic)
	defer eventProducer.Close()

	log.Printf("Kafka producer initialized: brokers=%v, topic=%s", kafkaBrokers, kafkaTopic)

	// Health checks
	e.GET("/health", api.HealthCheck)
	e.GET("/ready", api.ReadyCheck)

	// Invitation endpoints
	invitationHandler := api.NewInvitationHandler(db, eventProducer)
	e.POST("/invitations", invitationHandler.CreateInvitation)
	e.GET("/invitations", invitationHandler.ListInvitations)
	e.POST("/invitations/:token/accept", invitationHandler.AcceptInvitation)
	e.POST("/invitations/:id/resend", invitationHandler.ResendInvitation)

	// Notification preferences endpoints
	userService := services.NewUserService(db)
	notificationPrefsHandler := api.NewNotificationPreferencesHandler(userService)
	e.GET("/api/v1/users/notification-preferences", notificationPrefsHandler.GetNotificationPreferences)
	e.PATCH("/api/v1/users/:user_id/notification-preferences", notificationPrefsHandler.PatchNotificationPreferences)

	// Start server
	port := getEnv("PORT", "8083")
	log.Printf("User service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
