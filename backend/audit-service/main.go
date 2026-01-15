package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

	"github.com/pos/audit-service/src/config"
	"github.com/pos/audit-service/src/handlers/audit"
	"github.com/pos/audit-service/src/handlers/consent"
	"github.com/pos/audit-service/src/queue"
	"github.com/pos/audit-service/src/repository"
	"github.com/pos/audit-service/src/services"
	"github.com/pos/audit-service/src/utils"
)

func main() {
	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	// Load configuration from environment
	serviceName := utils.GetEnv("SERVICE_NAME")
	port := utils.GetEnv("PORT")
	dbHost := utils.GetEnv("DB_HOST")
	dbPort := utils.GetEnv("DB_PORT")
	dbName := utils.GetEnv("DB_NAME")
	dbUser := utils.GetEnv("DB_USER")
	dbPassword := utils.GetEnv("DB_PASSWORD")
	kafkaBrokers := utils.GetEnv("KAFKA_BROKERS")
	kafkaAuditTopic := utils.GetEnv("KAFKA_AUDIT_TOPIC")
	kafkaConsentTopic := utils.GetEnv("KAFKA_CONSENT_TOPIC")
	vaultAddr := utils.GetEnv("VAULT_ADDR")
	vaultToken := utils.GetEnv("VAULT_TOKEN")

	log.Info().Str("service", serviceName).Msg("Starting audit service")

	// Initialize Vault client
	if err := config.InitVaultClient(vaultAddr, vaultToken); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Vault client")
	}

	// Initialize database connection
	dbConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err := config.InitDatabase(dbConnStr)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize encryption client
	encryptor, err := utils.NewVaultClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize encryption client")
	}

	// Initialize repositories
	auditRepo := repository.NewAuditRepository(db)
	consentRepo := repository.NewConsentRepository(db, encryptor)

	// Initialize services
	consentService := services.NewConsentService(consentRepo)

	// Initialize partition manager service
	partitionService := services.NewPartitionService(db)

	// Start partition manager (monthly partition creation)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go partitionService.StartMonitor(ctx)

	// Initialize Kafka consumer for audit events
	consumerConfig := queue.KafkaConsumerConfig{
		Brokers:     kafkaBrokers,
		Topic:       kafkaAuditTopic,
		GroupID:     serviceName + "-consumer",
		StartOffset: -1, // Latest
	}
	auditConsumer := queue.NewAuditConsumer(consumerConfig, auditRepo)
	go auditConsumer.Start(ctx)

	// Initialize Kafka consumer for consent events
	consentConsumerConfig := queue.KafkaConsumerConfig{
		Brokers:     kafkaBrokers,
		Topic:       kafkaConsentTopic,
		GroupID:     serviceName + "-consent-consumer",
		StartOffset: -1, // Latest
	}
	consentConsumer := queue.NewConsentConsumer(consentConsumerConfig, consentRepo, encryptor)
	go consentConsumer.Start(ctx)
	log.Info().Str("consent_topic", kafkaConsentTopic).Msg("Consent consumer started")

	// Initialize Echo HTTP server
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middleware
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	// OpenTelemetry tracing
	e.Use(otelecho.Middleware(serviceName))

	// Prometheus metrics
	e.Use(echoprometheus.NewMiddleware(serviceName))
	e.GET("/metrics", echoprometheus.NewHandler())

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Audit query API handlers
	// Note: Authentication and RBAC are handled by API Gateway
	// This service should only be accessed through the gateway
	auditHandler := audit.NewQueryHandler(auditRepo, consentRepo)
	api := e.Group("/api/v1")
	api.GET("/audit-events", auditHandler.ListAuditEvents)
	api.GET("/audit-events/:event_id", auditHandler.GetAuditEvent)
	api.GET("/consent-records", auditHandler.ListConsentRecords)
	api.GET("/audit/tenant", auditHandler.ListTenantAuditEvents)

	// Consent management API handlers
	consentHandler := consent.NewHandler(consentService, consentRepo)
	api.GET("/consent/purposes", consentHandler.ListConsentPurposes)
	api.GET("/consent/purposes/:purpose_code", consentHandler.GetConsentPurposeByCode)
	api.POST("/consent/grant", consentHandler.GrantConsent)
	api.GET("/consent/status", consentHandler.GetConsentStatus)
	api.POST("/consent/revoke", consentHandler.RevokeConsent)
	api.GET("/consent/history", consentHandler.GetConsentHistory)
	api.GET("/privacy-policy", consentHandler.GetPrivacyPolicy)

	// Start HTTP server
	go func() {
		addr := ":" + port
		log.Info().Str("address", addr).Msg("HTTP server listening")
		if err := e.Start(addr); err != nil {
			log.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down audit service...")
	cancel() // Stop Kafka consumer and partition manager

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}

	log.Info().Msg("Audit service stopped")
}
