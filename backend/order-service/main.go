package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/point-of-sale-system/order-service/api"
	"github.com/point-of-sale-system/order-service/src/config"
	customMiddleware "github.com/point-of-sale-system/order-service/src/middleware"
	"github.com/point-of-sale-system/order-service/src/queue"
	"github.com/point-of-sale-system/order-service/src/repository"
	"github.com/point-of-sale-system/order-service/src/services"
	"github.com/point-of-sale-system/order-service/src/utils"
	"github.com/rs/zerolog/log"
)

func main() {
	// Initialize logger
	utils.InitLogger()

	// Initialize configurations
	if err := config.InitDatabase(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize database")
	}
	defer config.CloseDatabase()

	if err := config.InitRedis(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Redis")
	}
	defer config.CloseRedis()

	if err := config.InitMidtrans(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Midtrans")
	}

	if err := config.InitGoogleMaps(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Google Maps")
	}
	defer config.CloseGoogleMaps()

	// Initialize Echo
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	// Note: CORS is handled by API Gateway, not by individual services

	// Rate limiting for public endpoints
	customMiddleware.InitRateLimiter()

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "healthy",
			"service": "order-service",
		})
	})

	// Initialize handlers
	// TODO: Get product service URL from environment
	inventoryService := services.NewInventoryService(config.GetDB(), config.GetRedis(), "http://product-service:8082")

	// Initialize repositories
	paymentRepo := repository.NewPaymentRepository(config.GetDB())
	orderRepo := repository.NewOrderRepository(config.GetDB())
	orderSettingsRepo := repository.NewOrderSettingsRepository(config.GetDB())

	// Initialize cart service (shared between cart handler and checkout handler)
	ttl := time.Duration(config.GetEnvAsInt("CART_SESSION_TTL", 86400)) * time.Second
	cartRepo := repository.NewCartRepository(config.GetRedis(), ttl)
	reservationRepo := repository.NewReservationRepository(config.GetDB())
	cartService := services.NewCartService(cartRepo, reservationRepo, config.GetDB())

	// Initialize order service
	orderService := services.NewOrderService(config.GetDB(), orderRepo)

	// Initialize payment service (needs orderService for adding notes)
	paymentService := services.NewPaymentService(config.GetDB(), paymentRepo, orderRepo, inventoryService, orderService)

	// Initialize geocoding and delivery fee services
	// TODO: Initialize Google Maps client properly
	geocodingService := services.NewGeocodingService(nil, config.GetRedis())
	deliveryFeeService := services.NewDeliveryFeeService()
	addressRepo := repository.NewAddressRepository(config.GetDB())

	// Initialize Kafka producer for notifications
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}
	brokerList := []string{kafkaBrokers}
	kafkaProducer := queue.NewKafkaProducer(brokerList, "notification-events")
	log.Info().Strs("brokers", brokerList).Msg("Kafka producer initialized")

	// Initialize handlers
	webhookHandler := api.NewPaymentWebhookHandler(paymentService)
	adminOrderHandler := api.NewAdminOrderHandler(orderService)
	orderSettingsHandler := api.NewOrderSettingsHandler(orderSettingsRepo)
	cartHandler := api.NewCartHandlerWithService(cartService)
	checkoutHandler := api.NewCheckoutHandler(config.GetDB(), config.GetRedis(), cartService, inventoryService, paymentService, geocodingService, deliveryFeeService, addressRepo, orderSettingsRepo, kafkaProducer)

	// Start reservation cleanup job in background
	cleanupJob := services.NewReservationCleanupJob(inventoryService)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cleanupJob.Start(ctx)

	// Public cart routes (guest shopping)
	publicCart := e.Group("/api/v1/public/:tenantId")
	publicCart.Use(customMiddleware.RateLimit())
	publicCart.GET("/cart", cartHandler.GetCart)
	publicCart.POST("/cart/items", cartHandler.AddItem)
	publicCart.PATCH("/cart/items/:productId", cartHandler.UpdateItem)
	publicCart.DELETE("/cart/items/:productId", cartHandler.RemoveItem)
	publicCart.DELETE("/cart", cartHandler.ClearCart)

	// Public checkout routes
	publicCart.POST("/checkout", checkoutHandler.CreateOrder)

	// Public order lookup route (no tenantId needed for order reference)
	e.GET("/api/v1/public/orders/:orderReference", checkoutHandler.GetPublicOrder)

	// Webhook routes (public - signature verified in service layer)
	webhookHandler.RegisterRoutes(e)

	// Admin routes (JWT auth will be added in future)
	adminOrderHandler.RegisterRoutes(e)
	orderSettingsHandler.RegisterRoutes(e)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8087"
	}

	log.Info().Str("port", port).Msg("Starting order-service")

	// Graceful shutdown
	go func() {
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Stop cleanup job
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}
