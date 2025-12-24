package main

import (
	"database/sql"
	"log"
	"strings"

	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/pos/user-service/api"
	"github.com/pos/user-service/middleware"
	"github.com/pos/user-service/src/observability"
	"github.com/pos/user-service/src/queue"
	"github.com/pos/user-service/src/services"
	"github.com/pos/user-service/src/utils"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func main() {
	observability.InitLogger()
	shutdown := observability.InitTracer()
	defer shutdown(nil)

	e := echo.New()

	e.Use(emw.Recover())

	// OTEL
	e.Use(otelecho.Middleware(utils.GetEnv("SERVICE_NAME")))

	// Trace → Log bridge
	e.Use(middleware.TraceLogger)

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

	// Kafka configuration
	kafkaBrokers := strings.Split(utils.GetEnv("KAFKA_BROKERS"), ",")
	kafkaTopic := utils.GetEnv("KAFKA_TOPIC")

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
	port := utils.GetEnv("PORT")
	log.Printf("User service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
