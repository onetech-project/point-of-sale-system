package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/api"
	"github.com/pos/api-gateway/middleware"
	"github.com/pos/api-gateway/utils"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logging())
	e.Use(middleware.CORS())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "api-gateway",
		})
	})

	e.GET("/ready", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ready",
		})
	})

	rateLimiter := middleware.NewRateLimiter()
	public := e.Group("")
	publicOrders := e.Group("/api/v1/public/:tenantId")
	public.Use(rateLimiter.PublicRateLimit())
	publicOrders.Use(rateLimiter.PublicOrderRateLimit())

	protected := e.Group("")
	protected.Use(middleware.JWTAuth())
	protected.Use(middleware.TenantScope())

	adminTenantConfig := protected.Group("/api/v1/admin/tenants")
	adminTenantConfig.Use(middleware.RBACMiddleware(middleware.RoleOwner))

	adminOrders := protected.Group("/api/v1/admin")
	adminOrders.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager, middleware.RoleCashier))

	adminSettings := protected.Group("/api/v1/admin")
	adminSettings.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))

	productGroup := protected.Group("")
	productGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))

	inviteGroup := protected.Group("")
	inviteGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))

	notificationGroup := protected.Group("/api/v1")
	notificationGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))

	userNotificationGroup := protected.Group("/api/v1/users")
	userNotificationGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))

	// Initialize Auth routes
	api.InitAuthRoutes(public, protected)

	// Initialize Tenant routes
	api.InitTenantRoutes(public, protected, adminTenantConfig)

	// Initialize User routes
	api.InitUserRoutes(public, protected, inviteGroup, userNotificationGroup)

	// Initialize Product routes
	api.InitProductRoutes(public, protected, productGroup)

	// Initialize Order routes
	api.InitOrderRoutes(public, protected, adminOrders, publicOrders, adminSettings)

	// Initialize Notification routes
	api.InitNotificationRoutes(notificationGroup)

	port := utils.GetEnv("PORT", "8080")
	log.Printf("API Gateway starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
