package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"
	"github.com/pos/backend/inventory-service/api"
	"github.com/pos/backend/inventory-service/src/config"
	customMiddleware "github.com/pos/backend/inventory-service/src/middleware"
	"github.com/pos/backend/inventory-service/src/queue"
	"github.com/pos/backend/inventory-service/src/repository"
	"github.com/pos/backend/inventory-service/src/services"
	"github.com/pos/backend/inventory-service/src/utils"
)

func main() {
	utils.InitLogger()

	if err := config.InitDatabase(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer config.CloseDatabase()

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(emw.Logger())
	e.Use(emw.Recover())
	e.Use(emw.RequestID())

	healthHandler := api.NewHealthHandler(config.DB)
	e.GET("/health", healthHandler.HealthCheck)
	e.GET("/ready", healthHandler.ReadinessCheck)

	apiGroup := e.Group("/api/v1")
	apiGroup.Use(customMiddleware.TenantMiddleware)

	uomRepo := repository.NewUOMRepository(config.DB)
	ingredientRepo := repository.NewIngredientRepository(config.DB)
	conversionRepo := repository.NewConversionRepository(config.DB)
	stockRepo := repository.NewStockRepository(config.DB)
	recipeRepo := repository.NewRecipeRepository(config.DB)
	snapshotRepo := repository.NewSnapshotRepository(config.DB)
	bundleRepo := repository.NewBundleRepository(config.DB)

	uomService := services.NewUOMService(uomRepo)
	ingredientService := services.NewIngredientService(ingredientRepo, uomRepo)
	conversionService := services.NewConversionService(conversionRepo, uomRepo, ingredientRepo)
	stockService := services.NewStockService(stockRepo, conversionService)
	recipeService := services.NewRecipeService(recipeRepo, ingredientRepo, conversionService)
	snapshotService := services.NewSnapshotService(snapshotRepo)
	bundleService := services.NewBundleService(bundleRepo, recipeRepo)
	orderConsumptionService := services.NewOrderConsumptionService(config.DB, recipeRepo, bundleRepo, stockRepo, snapshotRepo)

	api.NewUOMHandler(uomService).RegisterRoutes(apiGroup)
	api.NewIngredientHandler(ingredientService).RegisterRoutes(apiGroup)
	api.NewConversionHandler(conversionService).RegisterRoutes(apiGroup)
	api.NewInventoryHandler(stockService).RegisterRoutes(apiGroup)
	api.NewRecipeHandler(recipeService).RegisterRoutes(apiGroup)
	api.NewSnapshotHandler(snapshotService).RegisterRoutes(apiGroup)
	api.NewBundleHandler(bundleService).RegisterRoutes(apiGroup)

	port := utils.GetEnvDefault("PORT", "8080")
	consumerCtx, stopConsumers := context.WithCancel(context.Background())
	defer stopConsumers()

	var orderConsumer *queue.OrderConsumer
	kafkaBrokers := splitCSV(utils.GetEnvDefault("KAFKA_BROKERS", ""))
	if len(kafkaBrokers) > 0 {
		orderConsumer = queue.NewOrderConsumer(queue.OrderConsumerConfig{
			Brokers: kafkaBrokers,
			Topic:   utils.GetEnvDefault("KAFKA_ORDER_EVENTS_TOPIC", "order-events"),
			GroupID: utils.GetEnvDefault("KAFKA_CONSUMER_GROUP", "inventory-service-order-consumer"),
		}, orderConsumptionService)
		go orderConsumer.Start(consumerCtx)
	}

	go func() {
		if err := e.Start(":" + port); err != nil {
			e.Logger.Infof("server shutdown: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	stopConsumers()
	if orderConsumer != nil {
		if err := orderConsumer.Close(); err != nil {
			e.Logger.Errorf("failed to close order consumer: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Errorf("server forced to shutdown: %v", err)
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return cleaned
}
