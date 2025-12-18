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

	// Initialize storage configuration (Feature 005)
	storageConfig := config.LoadStorageConfig()

	// Initialize storage service
	storageService, err := services.NewStorageService(storageConfig)
	if err != nil {
		log.Fatal("Failed to create storage service:", err)
	}

	// Initialize bucket (create if doesn't exist)
	ctx := context.Background()
	if err := storageService.InitializeBucket(ctx); err != nil {
		log.Fatal("Failed to initialize storage bucket:", err)
	}
	utils.Log.Info("Storage bucket '%s' initialized successfully", storageConfig.BucketName)

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

	// Initialize repositories
	productRepo := repository.NewProductRepository(config.DB)
	categoryRepo := repository.NewCategoryRepository(config.DB)
	stockRepo := repository.NewStockRepository(config.DB)
	photoRepo := repository.NewPhotoRepository(config.DB)

	// Initialize photo service and dependencies (needed for product handler)
	imageProcessor := services.NewImageProcessor(
		storageConfig.MaxPhotoSizeBytes,
		4096, // max width
		4096, // max height
	)

	// Initialize retry queue for background S3 deletion retries (Feature 005 - T074)
	retryQueue := services.NewRetryQueue(storageService, 30*time.Second) // Check every 30 seconds
	retryQueue.Start(ctx)
	utils.Log.Info("Retry queue started for background S3 deletion retries")

	photoService := services.NewPhotoService(
		photoRepo,
		storageService,
		imageProcessor,
		retryQueue,
		storageConfig.MaxPhotosPerProduct,
	)

	// Initialize product service and handler with photo service
	productService := services.NewProductService(productRepo)
	productHandler := api.NewProductHandler(productService, photoService)
	productHandler.RegisterRoutes(apiGroup)

	categoryService := services.NewCategoryService(categoryRepo)
	categoryHandler := api.NewCategoryHandler(categoryService)
	categoryHandler.RegisterRoutes(apiGroup)

	inventoryService := services.NewInventoryService(productRepo, stockRepo, config.DB)
	stockHandler := api.NewStockHandler(productService, inventoryService)
	stockHandler.RegisterRoutes(apiGroup)

	// Photo management endpoints (Feature 005)
	photoHandler := api.NewPhotoHandler(photoService)

	// Register photo routes
	apiGroup.POST("/products/:product_id/photos", photoHandler.UploadPhoto)
	apiGroup.GET("/products/:product_id/photos", photoHandler.ListPhotos)
	apiGroup.GET("/products/:product_id/photos/:photo_id", photoHandler.GetPhoto)
	apiGroup.PATCH("/products/:product_id/photos/:photo_id", photoHandler.UpdatePhotoMetadata)
	apiGroup.PUT("/products/:product_id/photos/:photo_id", photoHandler.ReplacePhoto)
	apiGroup.DELETE("/products/:product_id/photos/:photo_id", photoHandler.DeletePhoto)
	apiGroup.PUT("/products/:product_id/photos/reorder", photoHandler.ReorderPhotos)
	apiGroup.GET("/products/storage-quota", photoHandler.GetStorageQuota)

	// Public catalog endpoint (no authentication required)
	catalogService := services.NewCatalogService(config.DB)
	publicCatalogHandler := api.NewPublicCatalogHandler(catalogService, productService, photoService)
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

	// Stop retry queue first
	retryQueue.Stop()
	utils.Log.Info("Retry queue stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		utils.Log.Error("Server forced to shutdown: %v", err)
	}

	utils.Log.Info("Server exited")
}
