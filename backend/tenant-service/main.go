package main

import (
	"database/sql"
	"log"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"

	"github.com/pos/tenant-service/api"
	"github.com/pos/tenant-service/src/queue"
	"github.com/pos/tenant-service/src/repository"
	"github.com/pos/tenant-service/src/services"
	. "github.com/pos/tenant-service/src/utils"
)

func main() {
	e := echo.New()

	// Enable debug mode for detailed logging
	e.Debug = GetEnvBool("DEBUG")

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	// Note: CORS is handled by API Gateway, not by individual services

	dbURL := GetEnv("DATABASE_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize Kafka producer and event publisher
	kafkaBrokers := strings.Split(GetEnv("KAFKA_BROKERS"), ",")
	kafkaTopic := GetEnv("KAFKA_TOPIC")
	eventPublisher := queue.NewEventPublisher(kafkaBrokers, kafkaTopic)
	defer eventPublisher.Close()

	e.GET("/health", api.HealthCheck)
	e.GET("/ready", api.ReadyCheck)

	registerHandler := api.NewRegisterHandler(db, eventPublisher)
	e.POST("/register", registerHandler.Register)

	tenantHandler := api.NewTenantHandler(db)
	e.GET("/tenant", tenantHandler.GetTenant)

	// Tenant configuration routes
	configRepo := repository.NewTenantConfigRepository(db)
	configService := services.NewTenantConfigService(configRepo, db)
	configHandler := api.NewTenantConfigHandler(configService)

	// Public routes
	e.GET("/public/tenants/:tenant_id/config", configHandler.GetPublicTenantConfig)

	// Admin routes - match API Gateway pattern with /api/v1 prefix
	admin := e.Group("/api/v1/admin/tenants")
	admin.PATCH("/:tenant_id/config", configHandler.UpdateTenantConfig)
	admin.GET("/:tenant_id/midtrans-config", configHandler.GetMidtransConfig)
	admin.PATCH("/:tenant_id/midtrans-config", configHandler.UpdateMidtransConfig)

	port := GetEnv("PORT")

	log.Printf("Tenant service starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
