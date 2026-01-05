package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"

	"github.com/pos/api-gateway/middleware"
	"github.com/pos/api-gateway/utils"

	"github.com/pos/api-gateway/observability"
)

func main() {
	observability.InitLogger()
	shutdown := observability.InitTracer()
	defer shutdown(nil)

	e := echo.New()

	e.Use(emw.Recover())

	// OTEL
	e.Use(otelecho.Middleware(utils.GetEnv("SERVICE_NAME")))

	// Trace → Log bridge
	e.Use(middleware.TraceLogger)

	// Metrics
	middleware.MetricsMiddleware(e)

	e.Use(middleware.Logging())
	e.Use(middleware.CORS())

	rateLimiter := middleware.NewRateLimiter()

	e.GET("/health", func(c echo.Context) error {
		tr := otel.Tracer(utils.GetEnv("SERVICE_NAME"))
		_, span := tr.Start(c.Request().Context(), "call-downstream-service")
		defer span.End()

		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"service": utils.GetEnv("SERVICE_NAME"),
		})
	})

	e.GET("/ready", func(c echo.Context) error {
		if !rateLimiter.IsRedisConnected() {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status": "down",
				"redis":  "unreachable",
			})
		}
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ready",
		})
	})

	public := e.Group("")

	tenantServiceURL := utils.GetEnv("TENANT_SERVICE_URL")
	productServiceURL := utils.GetEnv("PRODUCT_SERVICE_URL")
	authServiceURL := utils.GetEnv("AUTH_SERVICE_URL")
	userServiceURL := utils.GetEnv("USER_SERVICE_URL")
	auditServiceURL := utils.GetEnv("AUDIT_SERVICE_URL")

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
	public.POST("/api/auth/verify-account", proxyHandler(authServiceURL, "/verify-account"))

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
	orderServiceURL := utils.GetEnv("ORDER_SERVICE_URL")

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

	// Notification service routes (owner/manager only)
	notificationServiceURL := utils.GetEnv("NOTIFICATION_SERVICE_URL")
	notificationGroup := protected.Group("/api/v1")
	notificationGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	notificationGroup.Any("/notifications*", proxyWildcard(notificationServiceURL))

	// User notification preferences routes (owner/manager only)
	userNotificationGroup := protected.Group("/api/v1/users")
	userNotificationGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	userNotificationGroup.GET("/notification-preferences", proxyHandler(userServiceURL, "/api/v1/users/notification-preferences"))
	userNotificationGroup.PATCH("/:user_id/notification-preferences", func(c echo.Context) error {
		userID := c.Param("user_id")
		return proxyHandler(userServiceURL, "/api/v1/users/"+userID+"/notification-preferences")(c)
	})

	// Audit service routes (owner only - compliance audit trail access)
	auditGroup := protected.Group("/api/v1")
	auditGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner))
	auditGroup.Any("/audit-events*", proxyWildcard(auditServiceURL))
	auditGroup.Any("/consent-records*", proxyWildcard(auditServiceURL))

	port := utils.GetEnv("PORT")
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

			// Forward context values as headers
			if tenantID := c.Get("tenant_id"); tenantID != nil {
				req.Header.Set("X-Tenant-ID", tenantID.(string))
			}
			if userID := c.Get("user_id"); userID != nil {
				req.Header.Set("X-User-ID", userID.(string))
			}
			if role := c.Get("role"); role != nil {
				req.Header.Set("X-User-Role", role.(string))
			}
		}

		proxy.ServeHTTP(c.Response(), c.Request())

		c.Request().URL.Path = originalPath

		return nil
	}
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

			// Forward context values from auth middleware as headers
			if tenantID := c.Get("tenant_id"); tenantID != nil {
				req.Header.Set("X-Tenant-ID", tenantID.(string))
			}
			if userID := c.Get("user_id"); userID != nil {
				req.Header.Set("X-User-ID", userID.(string))
			}
			if role := c.Get("role"); role != nil {
				req.Header.Set("X-User-Role", role.(string))
			}
		}

		proxy.ServeHTTP(c.Response(), c.Request())

		return nil
	}
}
