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
	"github.com/pos/user-service/src/repository"
	"github.com/pos/user-service/src/scheduler"
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

	// Logging with PII masking (T060)
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

	// Kafka configuration
	kafkaBrokers := strings.Split(utils.GetEnv("KAFKA_BROKERS"), ",")
	kafkaTopic := utils.GetEnv("KAFKA_TOPIC")

	// Initialize Kafka producer
	eventProducer := queue.NewKafkaProducer(kafkaBrokers, kafkaTopic)
	defer eventProducer.Close()

	log.Printf("Kafka producer initialized: brokers=%v, topic=%s", kafkaBrokers, kafkaTopic)

	// Initialize AuditPublisher for audit trail (T098-T100)
	auditTopic := utils.GetEnv("KAFKA_AUDIT_TOPIC")
	serviceName := utils.GetEnv("SERVICE_NAME")
	auditPublisher, err := utils.NewAuditPublisher(serviceName, kafkaBrokers, auditTopic)
	if err != nil {
		log.Fatalf("Failed to initialize AuditPublisher: %v", err)
	}
	defer auditPublisher.Close()

	// Health checks
	e.GET("/health", api.HealthCheck)
	e.GET("/ready", api.ReadyCheck)

	// Invitation endpoints
	invitationHandler := api.NewInvitationHandler(db, eventProducer, auditPublisher)
	e.POST("/invitations", invitationHandler.CreateInvitation)
	e.GET("/invitations", invitationHandler.ListInvitations)
	e.POST("/invitations/:token/accept", invitationHandler.AcceptInvitation)
	e.POST("/invitations/:id/resend", invitationHandler.ResendInvitation)

	// Notification preferences endpoints
	userService, err := services.NewUserService(db, auditPublisher)
	if err != nil {
		log.Fatalf("Failed to create user service: %v", err)
	}
	notificationPrefsHandler := api.NewNotificationPreferencesHandler(userService)
	e.GET("/api/v1/users/notification-preferences", notificationPrefsHandler.GetNotificationPreferences)
	e.PATCH("/api/v1/users/:user_id/notification-preferences", notificationPrefsHandler.PatchNotificationPreferences)

	// User deletion endpoints - UU PDP compliance (owner only via API Gateway RBAC)
	userDeletionHandler, err := api.NewUserDeletionHandler(db, auditPublisher)
	if err != nil {
		log.Fatalf("Failed to create user deletion handler: %v", err)
	}
	e.DELETE("/api/v1/users/:user_id", userDeletionHandler.DeleteUser)

	// Initialize cleanup job scheduler (T135-T138)
	userRepo, err := repository.NewUserRepositoryWithVault(db, auditPublisher)
	if err != nil {
		log.Fatalf("Failed to create user repository: %v", err)
	}
	deletionService := services.NewUserDeletionService(userRepo, auditPublisher, db)
	cleanupJob := services.NewCleanupJob(deletionService, eventProducer)
	cleanupScheduler := scheduler.NewUserDeletionScheduler(cleanupJob)
	if err := cleanupScheduler.Start(); err != nil {
		log.Fatalf("Failed to start cleanup scheduler: %v", err)
	}

	// Start server
	port := utils.GetEnv("PORT")
	log.Printf("User service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
