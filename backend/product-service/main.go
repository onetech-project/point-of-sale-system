package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pos/backend/product-service/api"
	"github.com/pos/backend/product-service/src/config"
	customMiddleware "github.com/pos/backend/product-service/src/middleware"
	"github.com/pos/backend/product-service/src/repository"
	"github.com/pos/backend/product-service/src/services"
	"github.com/pos/backend/product-service/src/utils"
)

func main() {
	utils.InitLogger()

	if err := config.InitDatabase(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer config.CloseDatabase()

	if err := config.InitRedis(); err != nil {
		log.Fatal("Failed to initialize Redis:", err)
	}
	defer config.CloseRedis()

	e := echo.New()

	// CORS configuration
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000,http://localhost:3001"
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{allowedOrigins},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-Tenant-ID"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(customMiddleware.RequestIDMiddleware)
	e.Use(customMiddleware.MetricsMiddleware)

	// Rate limiting: 100 requests per minute per IP
	rateLimiter := customMiddleware.NewRateLimiter(100, time.Minute)
	e.Use(customMiddleware.RateLimitMiddleware(rateLimiter))

	// Health check endpoints (no authentication required)
	healthHandler := api.NewHealthHandler(config.DB)
	e.GET("/health", healthHandler.HealthCheck)
	e.GET("/ready", healthHandler.ReadinessCheck)

	apiGroup := e.Group("/api/v1")
	apiGroup.Use(customMiddleware.TenantMiddleware)

	productRepo := repository.NewProductRepository(config.DB)
	productService := services.NewProductService(productRepo)
	productHandler := api.NewProductHandler(productService)
	productHandler.RegisterRoutes(apiGroup)

	categoryRepo := repository.NewCategoryRepository(config.DB)
	categoryService := services.NewCategoryService(categoryRepo)
	categoryHandler := api.NewCategoryHandler(categoryService)
	categoryHandler.RegisterRoutes(apiGroup)

	stockRepo := repository.NewStockRepository(config.DB)
	inventoryService := services.NewInventoryService(productRepo, stockRepo, config.DB)
	stockHandler := api.NewStockHandler(productService, inventoryService)
	stockHandler.RegisterRoutes(apiGroup)

	// Public catalog endpoint (no authentication required)
	catalogService := services.NewCatalogService(config.DB)
	publicCatalogHandler := api.NewPublicCatalogHandler(catalogService, productService)
	e.GET("/public/menu/:tenant_id/products", publicCatalogHandler.GetPublicMenu)
	e.GET("/public/products/:tenant_id/:id/photo", publicCatalogHandler.GetPublicPhoto)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	utils.Log.Info("Product service starting on port %s", port)

	// Start server in a goroutine
	go func() {
		if err := e.Start(":" + port); err != nil {
			utils.Log.Error("Server shutdown: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	utils.Log.Info("Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		utils.Log.Error("Server forced to shutdown: %v", err)
	}

	utils.Log.Info("Server exited")
}
