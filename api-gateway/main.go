package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logging())
	e.Use(middleware.CORS())

	// rateLimiter := middleware.NewRateLimiter()

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

	public := e.Group("")

	tenantServiceURL := getEnv("TENANT_SERVICE_URL", "http://localhost:8084")
	productServiceURL := getEnv("PRODUCT_SERVICE_URL", "http://localhost:8086")
	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8082")
	userServiceURL := getEnv("USER_SERVICE_URL", "http://localhost:8083")

	public.POST("/api/tenants/register", proxyHandler(tenantServiceURL, "/register"))
	public.GET("/api/public/tenants/:tenant_id/config", func(c echo.Context) error {
		tenantID := c.Param("tenant_id")
		return proxyHandler(tenantServiceURL, "/public/tenants/"+tenantID+"/config")(c)
	})

	// Public menu endpoint for guest ordering
	public.GET("/api/public/menu/:tenant_id/products", func(c echo.Context) error {
		tenantID := c.Param("tenant_id")
		targetURL := productServiceURL + "/public/menu/" + tenantID + "/products"

		// Forward query parameters
		if c.QueryString() != "" {
			targetURL += "?" + c.QueryString()
		}

		target, _ := url.Parse(targetURL)
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.Director = func(req *http.Request) {
			req.URL = target
			req.Host = target.Host
			// Forward tenant ID in header
			req.Header.Set("X-Tenant-ID", tenantID)
		}
		proxy.ServeHTTP(c.Response(), c.Request())
		return nil
	})

	// Public product photo endpoint
	public.GET("/api/public/products/:tenant_id/:id/photo", func(c echo.Context) error {
		tenantID := c.Param("tenant_id")
		productID := c.Param("id")
		targetURL := productServiceURL + "/public/products/" + tenantID + "/" + productID + "/photo"

		target, _ := url.Parse(targetURL)
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.Director = func(req *http.Request) {
			req.URL = target
			req.Host = target.Host
		}
		proxy.ServeHTTP(c.Response(), c.Request())
		return nil
	})

	public.POST("/api/auth/login", proxyHandler(authServiceURL, "/login"))
	public.POST("/api/auth/password-reset/request", proxyHandler(authServiceURL, "/password-reset/request"))
	public.POST("/api/auth/password-reset/reset", proxyHandler(authServiceURL, "/password-reset/reset"))

	public.POST("/api/invitations/:token/accept", proxyHandler(userServiceURL, "/invitations/:token/accept"))

	protected := e.Group("")
	protected.Use(middleware.JWTAuth())
	protected.Use(middleware.TenantScope())

	protected.GET("/api/auth/session", proxyHandler(authServiceURL, "/session"))
	protected.POST("/api/auth/logout", proxyHandler(authServiceURL, "/logout"))

	protected.GET("/api/tenant", proxyHandler(tenantServiceURL, "/tenant"))

	// Admin tenant configuration routes (owner only)
	adminTenantConfig := protected.Group("/api/v1/admin/tenants")
	adminTenantConfig.Use(middleware.RBACMiddleware(middleware.RoleOwner))
	adminTenantConfig.Any("/*", proxyWildcard(tenantServiceURL))

	// Invitation endpoints - only owner and manager can create/resend
	inviteGroup := protected.Group("")
	inviteGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	inviteGroup.POST("/api/invitations", proxyHandler(userServiceURL, "/invitations"))
	inviteGroup.POST("/api/invitations/:id/resend", proxyHandler(userServiceURL, "/invitations/:id/resend"))

	// All authenticated users can list invitations
	protected.GET("/api/invitations", proxyHandler(userServiceURL, "/invitations"))

	// Product service routes - only owner and manager can manage products
	productGroup := protected.Group("")
	productGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	productGroup.Any("/api/v1/products*", proxyWildcard(productServiceURL))
	productGroup.Any("/api/v1/categories*", proxyWildcard(productServiceURL))
	productGroup.Any("/api/v1/inventory*", proxyWildcard(productServiceURL))

	// Order service routes
	orderServiceURL := getEnv("ORDER_SERVICE_URL", "http://localhost:8087")
	notificationServiceURL := getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8085")

	// Public guest ordering routes (no auth required)
	publicOrders := e.Group("/api/v1/public/:tenantId")
	// publicOrders.Use(middleware.RateLimit()) // Rate limiting will be added later
	publicOrders.Any("/*", proxyWildcard(orderServiceURL))

	// Admin order management routes (requires auth + appropriate role)
	adminOrders := protected.Group("/api/v1/admin")
	adminOrders.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager, middleware.RoleCashier))
	adminOrders.Any("/orders*", proxyWildcard(orderServiceURL))

	// Admin order settings routes (requires auth, owner/manager only)
	adminSettings := protected.Group("/api/v1/admin")
	adminSettings.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	adminSettings.Any("/settings*", proxyWildcard(orderServiceURL))

	// Webhook routes (no auth, but signature verification in order-service)
	e.Any("/api/v1/webhooks/*", proxyWildcard(orderServiceURL))

	// SSE endpoint proxy to notification service (authenticated)
	// Clients will connect to /api/v1/sse on the gateway which will forward to the notification service.
	protected.GET("/api/v1/sse", proxyWildcard(notificationServiceURL))

	port := getEnv("PORT", "8080")
	log.Printf("API Gateway starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}

func proxyHandler(targetURL, path string) echo.HandlerFunc {
	return func(c echo.Context) error {
		target, err := url.Parse(targetURL)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Service configuration error",
			})
		}

		proxy := httputil.NewSingleHostReverseProxy(target)

		originalPath := c.Request().URL.Path
		c.Request().URL.Path = path

		proxy.Director = func(req *http.Request) {
			req.Host = target.Host
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = path

			if c.Param("token") != "" {
				req.URL.Path = "/invitations/" + c.Param("token") + "/accept"
			}
			if c.Param("id") != "" {
				req.URL.Path = "/invitations/" + c.Param("id") + "/resend"
			}
		}

		proxy.ServeHTTP(c.Response(), c.Request())

		c.Request().URL.Path = originalPath

		return nil
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func proxyWildcard(targetURL string) echo.HandlerFunc {
	return func(c echo.Context) error {
		target, err := url.Parse(targetURL)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Service configuration error",
			})
		}

		proxy := httputil.NewSingleHostReverseProxy(target)

		proxy.Director = func(req *http.Request) {
			req.Host = target.Host
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host

			// Preserve authentication headers from TenantScope middleware
			if tenantID := c.Request().Header.Get("X-Tenant-ID"); tenantID != "" {
				req.Header.Set("X-Tenant-ID", tenantID)
			}
			if userID := c.Request().Header.Get("X-User-ID"); userID != "" {
				req.Header.Set("X-User-ID", userID)
			}
			if role := c.Request().Header.Get("X-User-Role"); role != "" {
				req.Header.Set("X-User-Role", role)
			}
		}

		proxy.ServeHTTP(c.Response(), c.Request())

		return nil
	}
}
