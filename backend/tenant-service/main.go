package main

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"

	"context"

	"github.com/pos/tenant-service/api"
	"github.com/pos/tenant-service/src/config"
	"github.com/pos/tenant-service/src/queue"
	"github.com/pos/tenant-service/src/queue/handlers"
	"github.com/pos/tenant-service/src/services"
)

func main() {
	e := echo.New()

	// Enable debug mode for detailed logging
	e.Debug = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	// Note: CORS is handled by API Gateway, not by individual services

	db, err := sql.Open("postgres", config.DB_URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	e.GET("/health", api.HealthCheck)
	e.GET("/ready", api.ReadyCheck)

	registerHandler := api.NewRegisterHandler(db)
	e.POST("/register", registerHandler.Register)

	tenantHandler := api.NewTenantHandler(db)
	e.GET("/tenant", tenantHandler.GetTenant)

	// Tenant configuration routes
	configService := services.NewTenantConfigService(db)
	configHandler := api.NewTenantConfigHandler(configService)

	// Public routes
	e.GET("/public/tenants/:tenant_id/config", configHandler.GetPublicTenantConfig)

	// Admin routes - match API Gateway pattern with /api/v1 prefix
	admin := e.Group("/api/v1/admin/tenants")
	admin.PATCH("/:tenant_id/config", configHandler.UpdateTenantConfig)
	admin.GET("/:tenant_id/midtrans-config", configHandler.GetMidtransConfig)
	admin.PATCH("/:tenant_id/midtrans-config", configHandler.UpdateMidtransConfig)

	// Start Kafka producer and event publisher
	eventPublisher := queue.NewEventPublisher(config.KAFKA_BROKERS)
	defer eventPublisher.Close()

	// Start Kafka consumers
	tenantService := services.NewTenantService(db)
	authConsumer := handlers.NewAuthConsumer(tenantService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go authConsumer.StartAuthConsumer(ctx)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down notification service...")
		cancel()
		authConsumer.Stop()
		e.Close()
	}()

	log.Printf("Tenant service starting on port %s", config.PORT)
	e.Logger.Fatal(e.Start(":" + config.PORT))
}
