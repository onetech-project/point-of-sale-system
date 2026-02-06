package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pos/analytics-service/api"
	"github.com/pos/analytics-service/src/config"
	customMiddleware "github.com/pos/analytics-service/src/middleware"
	"github.com/pos/analytics-service/src/repository"
	"github.com/pos/analytics-service/src/services"
	"github.com/pos/analytics-service/src/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Initialize logger
	initLogger()

	log.Info().Msg("Starting Analytics Service...")

	// Initialize database
	if err := config.InitDatabase(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer config.CloseDatabase()

	// Initialize Redis
	if err := config.InitRedis(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Redis")
	}
	defer config.CloseRedis()

	// Initialize Vault encryptor
	encryptor, err := utils.NewVaultClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Vault client")
	}

	// Initialize Echo
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())

	// Initialize handlers
	healthHandler := api.NewHealthHandler()

	// Initialize services
	currentTTL := time.Duration(utils.GetEnvInt("CACHE_TTL_CURRENT_MONTH")) * time.Second
	historicalTTL := time.Duration(utils.GetEnvInt("CACHE_TTL_HISTORICAL")) * time.Second
	timezone := utils.GetEnv("TZ") // Get timezone from environment
	analyticsService := services.NewAnalyticsService(config.GetDB(), config.GetRedis(), encryptor, currentTTL, historicalTTL, timezone)
	analyticsHandler := api.NewAnalyticsHandler(analyticsService)

	// Initialize task repository and handler
	taskRepo := repository.NewTaskRepository(config.GetDB(), encryptor, timezone)
	tasksHandler := api.NewTasksHandler(taskRepo)

	// Routes
	e.GET("/health", healthHandler.Health)

	// API v1 routes (authenticated by API Gateway)
	v1 := e.Group("/api/v1")
	v1.Use(customMiddleware.TenantAuth())

	// Analytics routes
	v1.GET("/analytics/overview", analyticsHandler.GetSalesOverview)
	v1.GET("/analytics/top-products", analyticsHandler.GetTopProducts)
	v1.GET("/analytics/top-customers", analyticsHandler.GetTopCustomers)
	v1.GET("/analytics/sales-trend", analyticsHandler.GetSalesTrend)
	v1.GET("/analytics/tasks", tasksHandler.GetOperationalTasks)

	// Start server
	port := utils.GetEnv("PORT")
	serverAddr := fmt.Sprintf(":%s", port)

	loc, err := time.LoadLocation(utils.GetEnv("TZ"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load timezone")
	}

	time.Local = loc

	go func() {
		log.Info().
			Str("port", port).
			Str("env", utils.GetEnv("ENV")).
			Msg("Analytics Service is running")

		if err := e.Start(serverAddr); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Info().Msg("Shutting down Analytics Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Analytics Service stopped")
}

func initLogger() {
	// Set log level from environment
	level := zerolog.InfoLevel
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		switch logLevel {
		case "debug":
			level = zerolog.DebugLevel
		case "info":
			level = zerolog.InfoLevel
		case "warn":
			level = zerolog.WarnLevel
		case "error":
			level = zerolog.ErrorLevel
		}
	}
	zerolog.SetGlobalLevel(level)

	// Pretty logging for development
	if utils.GetEnv("ENV") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		})
	}
}
