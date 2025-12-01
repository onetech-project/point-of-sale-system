package main

import (
	"log"
	"os"

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

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "healthy"})
	})

	e.GET("/ready", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ready"})
	})

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	utils.Log.Info("Product service starting on port %s", port)
	if err := e.Start(":" + port); err != nil {
		log.Fatal(err)
	}
}
