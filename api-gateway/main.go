package main

import (
	stdlog "log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	emw "github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
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

	isDevelopment := utils.GetEnv("ENVIRONMENT") == "development"
	if isDevelopment {
		stdlog.Println("Running in development mode")
		e.Logger.SetLevel(log.DEBUG)
	}

	if !isDevelopment {
		// OTEL
		e.Use(otelecho.Middleware(utils.GetEnv("SERVICE_NAME")))

		// Trace to log bridge
		e.Use(middleware.TraceLogger)

		// Metrics
		middleware.MetricsMiddleware(e)
	}

	e.Use(middleware.Logging())
	e.Use(middleware.CORS())

	rateLimiter := middleware.NewRateLimiter()
	subscriptionEnforcer := middleware.NewSubscriptionEnforcer()

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
	inventoryServiceURL := utils.GetEnv("INVENTORY_SERVICE_URL")
	authServiceURL := utils.GetEnv("AUTH_SERVICE_URL")
	userServiceURL := utils.GetEnv("USER_SERVICE_URL")
	auditServiceURL := utils.GetEnv("AUDIT_SERVICE_URL")
	analyticsServiceURL := utils.GetEnv("ANALYTICS_SERVICE_URL")
	billingServiceURL := utils.GetEnv("BILLING_SERVICE_URL")
	platformServiceURL := utils.GetEnv("PLATFORM_SERVICE_URL")
	tenantStatusEnforcer := middleware.NewTenantStatusEnforcer(tenantServiceURL)

	registrationGroup := e.Group("")
	registrationGroup.Use(rateLimiter.RateLimit(5, 30*time.Minute))
	registrationGroup.POST("/api/tenants/register", proxyHandler(tenantServiceURL, "/register"))
	registrationGroup.POST("/api/auth/resend-verification", proxyHandler(authServiceURL, "/resend-verification"))
	public.GET("/api/public/tenants/:tenant_slug/config", func(c echo.Context) error {
		tenantSlug := c.Param("tenant_slug")
		return proxyHandler(tenantServiceURL, "/public/tenants/"+tenantSlug+"/config")(c)
	})
	registerPublicPlansRoute(public, billingServiceURL)
	public.POST("/api/v1/platform/auth/login", proxyHandler(platformServiceURL, "/api/v1/platform/auth/login"))

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
	}, tenantStatusEnforcer.RequirePublicTenantAvailableParam("tenant_id"))

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
	}, tenantStatusEnforcer.RequirePublicTenantAvailableParam("tenant_id"))

	public.POST("/api/auth/login", proxyHandler(authServiceURL, "/login"))
	public.POST("/api/auth/password-reset/request", proxyHandler(authServiceURL, "/password-reset/request"))
	public.POST("/api/auth/password-reset/reset", proxyHandler(authServiceURL, "/password-reset/reset"))
	public.POST("/api/auth/verify-account", proxyHandler(authServiceURL, "/verify-account"))

	public.POST("/api/invitations/:token/accept", proxyHandler(userServiceURL, "/invitations/:token/accept"))

	protected := e.Group("")
	protected.Use(middleware.JWTAuth())
	protected.Use(middleware.TenantScope())
	protected.Use(tenantStatusEnforcer.EnforceTenantAccount())
	protected.Use(subscriptionEnforcer.EnforceSubscription())

	// Refresh endpoint - outside protected group since it may not have valid JWT
	e.POST("/api/auth/refresh", proxyHandler(authServiceURL, "/refresh"))

	protected.GET("/api/auth/session", proxyHandler(authServiceURL, "/session"))
	protected.POST("/api/auth/logout", proxyHandler(authServiceURL, "/logout"))

	protected.GET("/api/tenant", proxyHandler(tenantServiceURL, "/tenant"))

	// Tenant Midtrans credentials can be managed only by owners for guest ordering.
	midtransConfig := protected.Group("/api/v1/admin/tenants")
	midtransConfig.Use(middleware.RBACMiddleware(middleware.RoleOwner))
	midtransConfig.GET("/:tenant_id/midtrans-config", proxyWildcard(tenantServiceURL))
	midtransConfig.PATCH("/:tenant_id/midtrans-config", proxyWildcard(tenantServiceURL))

	// Admin tenant configuration routes (owner only)
	adminTenantConfig := protected.Group("/api/v1/admin/tenants")
	adminTenantConfig.Use(middleware.RBACMiddleware(middleware.RoleOwner))
	adminTenantConfig.Any("/*", proxyWildcard(tenantServiceURL))

	// Invitation endpoints - only owner and manager can create/resend/revoke
	inviteGroup := protected.Group("")
	inviteGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	inviteGroup.POST("/api/invitations", proxyHandler(userServiceURL, "/invitations"))
	inviteGroup.POST("/api/invitations/:id/resend", proxyHandler(userServiceURL, "/invitations/:id/resend"))
	inviteGroup.POST("/api/invitations/:id/revoke", proxyHandler(userServiceURL, "/invitations/:id/revoke"))

	teamGroup := protected.Group("")
	teamGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	teamGroup.GET("/api/team/users", proxyHandler(userServiceURL, "/team/users"))
	teamGroup.PATCH("/api/team/users/:user_id", proxyHandler(userServiceURL, "/team/users/:user_id"))

	// All authenticated users can list invitations
	protected.GET("/api/invitations", proxyHandler(userServiceURL, "/invitations"))

	registerIngredientInventoryRoutes(protected, inventoryServiceURL)
	registerProductRoutes(protected, productServiceURL)

	// Order service routes
	orderServiceURL := utils.GetEnv("ORDER_SERVICE_URL")

	// Public guest ordering routes (no auth required)
	publicOrders := e.Group("/api/v1/public/:tenantId")
	publicOrders.Use(tenantStatusEnforcer.RequirePublicTenantAvailableParam("tenantId"))
	// publicOrders.Use(middleware.RateLimit()) // Rate limiting will be added later
	publicOrders.Any("/*", proxyWildcard(orderServiceURL))

	// Admin order management routes (requires auth + appropriate role)
	adminOrders := protected.Group("/api/v1/admin")
	adminOrders.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager, middleware.RoleCashier))
	adminOrders.Any("/orders*", proxyWildcard(orderServiceURL))
	adminOrders.Any("/offline-orders*", proxyWildcard(orderServiceURL))

	// Admin order settings routes (requires auth, owner/manager only)
	adminSettings := protected.Group("/api/v1/admin")
	adminSettings.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	adminSettings.Any("/settings*", proxyWildcard(orderServiceURL))

	// Discount rules are managed by owner/manager and evaluated by order-service pricing.
	adminDiscounts := protected.Group("/api/v1/admin")
	adminDiscounts.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	adminDiscounts.Any("/discount-rules*", proxyWildcard(orderServiceURL))

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

	registerUserOnboardingRoutes(protected, userServiceURL)

	// Audit service routes (owner only - compliance audit trail access)
	auditGroup := protected.Group("/api/v1")
	auditGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner))
	auditGroup.Any("/audit-events*", proxyWildcard(auditServiceURL))
	auditGroup.Any("/consent-records*", proxyWildcard(auditServiceURL))
	auditGroup.Any("/admin/compliance/report*", proxyWildcard(auditServiceURL)) // Compliance report (T201)

	// Read-only offboarding routes: active and deactivated tenants keep history/export access.
	offboardingProtected := e.Group("")
	offboardingProtected.Use(middleware.JWTAuth())
	offboardingProtected.Use(middleware.TenantScope())
	offboardingProtected.Use(tenantStatusEnforcer.AllowTenantStatuses("active", "inactive"))
	offboardingProtected.Use(middleware.RBACMiddleware(middleware.RoleOwner))
	offboardingProtected.Any("/api/v1/audit/tenant*", proxyWildcard(auditServiceURL)) // Tenant audit trail (T110)
	offboardingProtected.GET("/api/v1/tenant/data", proxyHandler(tenantServiceURL, "/api/v1/tenant/data"))
	offboardingProtected.POST("/api/v1/tenant/data/export", proxyHandler(tenantServiceURL, "/api/v1/tenant/data/export"))

	// User deletion routes (owner only - UU PDP compliance)
	userDeletionGroup := protected.Group("/api/v1/tenant/users")
	userDeletionGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner))
	userDeletionGroup.DELETE("/:user_id", func(c echo.Context) error {
		userID := c.Param("user_id")
		path := "/api/v1/users/" + userID
		// Forward force parameter if present
		if c.QueryParam("force") != "" {
			path += "?force=" + c.QueryParam("force")
		}
		return proxyHandler(userServiceURL, path)(c)
	})

	// Consent management routes (public for registration/checkout, authenticated for status/revoke)
	public.GET("/api/v1/consent/purposes", proxyHandler(auditServiceURL, "/api/v1/consent/purposes"))
	public.GET("/api/v1/privacy-policy", proxyHandler(auditServiceURL, "/api/v1/privacy-policy"))
	public.POST("/api/v1/consent/grant", proxyHandler(auditServiceURL, "/api/v1/consent/grant")) // Used during registration/checkout

	// Guest data rights routes (T144-T147) - public but require order verification
	// No authentication required, access controlled by order_reference + email/phone verification
	// public.GET("/api/v1/guest/order/:order_reference/data", proxyHandler(orderServiceURL, "/api/v1/guest/order/:order_reference/data"))
	// public.POST("/api/v1/guest/order/:order_reference/delete", proxyHandler(orderServiceURL, "/api/v1/guest/order/:order_reference/delete"))

	// Protected consent routes (require authentication)
	protected.GET("/api/v1/consent/status", proxyHandler(auditServiceURL, "/api/v1/consent/status"))
	protected.POST("/api/v1/consent/revoke", proxyHandler(auditServiceURL, "/api/v1/consent/revoke"))
	protected.GET("/api/v1/consent/history", proxyHandler(auditServiceURL, "/api/v1/consent/history"))

	// Cashiers can read operational task alerts; business analytics stay owner/manager only.
	analyticsTasks := protected.Group("/api/v1/analytics")
	analyticsTasks.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager, middleware.RoleCashier))
	analyticsTasks.GET("/tasks", proxyWildcard(analyticsServiceURL))

	// Analytics service routes (owner and manager only)
	analyticsGroup := protected.Group("/api/v1/analytics")
	analyticsGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	analyticsGroup.Any("/*", proxyWildcard(analyticsServiceURL))

	// Billing service routes use a separate group that skips the subscription enforcer
	// (expired tenants must be able to reach billing endpoints to fix their subscription)
	billingProtected := e.Group("")
	billingProtected.Use(middleware.JWTAuth())
	billingProtected.Use(middleware.TenantScope())
	billingReadGroup := billingProtected.Group("/api/v1/billing")
	billingReadGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager, middleware.RoleCashier))
	billingReadGroup.GET("/subscription", proxyHandler(billingServiceURL, "/api/v1/billing/subscription"))
	billingReadGroup.GET("/invoices", proxyHandler(billingServiceURL, "/api/v1/billing/invoices"))
	billingReadGroup.GET("/invoices/:id", proxyWildcard(billingServiceURL))
	billingReadGroup.GET("/invoices/:id/payments", proxyWildcard(billingServiceURL))

	billingManageGroup := billingProtected.Group("/api/v1/billing")
	billingManageGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	billingManageGroup.PUT("/subscription/cycle", proxyHandler(billingServiceURL, "/api/v1/billing/subscription/cycle"))
	billingManageGroup.POST("/subscription/cycle-switch", proxyHandler(billingServiceURL, "/api/v1/billing/subscription/cycle-switch"))
	billingManageGroup.POST("/subscription/upgrade", proxyHandler(billingServiceURL, "/api/v1/billing/subscription/upgrade"))
	billingManageGroup.POST("/invoices/:id/pay", proxyWildcard(billingServiceURL))

	// Billing webhook (no auth - signature verified by billing-service)
	e.POST("/api/v1/billing/webhook", proxyHandler(billingServiceURL, "/webhook/billing"))

	// Tenant support tickets bypass subscription/account-status enforcement so suspended tenants can reach support.
	tenantSupport := e.Group("/api/v1/platform")
	tenantSupport.Use(middleware.JWTAuth())
	tenantSupport.Use(middleware.TenantScope())
	tenantSupport.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	tenantSupport.POST("/tickets", proxyWildcard(platformServiceURL))

	// Platform-owner routes use a separate platform cookie and never enter tenant scope.
	platformProtected := e.Group("/api/v1/platform")
	platformProtected.Use(middleware.PlatformAdminAuth())
	platformProtected.GET("/auth/session", proxyHandler(platformServiceURL, "/api/v1/platform/auth/session"))
	platformProtected.POST("/auth/logout", proxyHandler(platformServiceURL, "/api/v1/platform/auth/logout"))
	platformProtected.Any("/analytics/*", proxyWildcard(platformServiceURL))
	platformProtected.Any("/tenants*", proxyWildcard(platformServiceURL))
	platformProtected.GET("/audit-events", proxyWildcard(platformServiceURL))
	platformProtected.GET("/tickets", proxyWildcard(platformServiceURL))
	platformProtected.PATCH("/tickets/:ticket_id", proxyWildcard(platformServiceURL))
	platformProtected.POST("/tickets/:ticket_id/notes", proxyWildcard(platformServiceURL))

	port := utils.GetEnv("PORT")
	stdlog.Printf("API Gateway starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}

func registerProductRoutes(protected *echo.Group, productServiceURL string) {
	// Product reads are needed by cashier offline-order creation; mutations remain owner/manager only.
	productReadGroup := protected.Group("")
	productReadGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager, middleware.RoleCashier))
	productReadGroup.GET("/api/v1/products*", proxyWildcard(productServiceURL))

	// Product imports are asynchronous create-only catalog mutations, restricted to owner/manager.
	productImportGroup := protected.Group("")
	productImportGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	productImportGroup.GET("/api/v1/product-imports/template", proxyWildcard(productServiceURL))
	productImportGroup.POST("/api/v1/product-imports", proxyWildcard(productServiceURL))
	productImportGroup.GET("/api/v1/product-imports/:id", proxyWildcard(productServiceURL))

	// Product service routes - only owner and manager can manage products.
	productGroup := protected.Group("")
	productGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	productGroup.POST("/api/v1/products*", proxyWildcard(productServiceURL))
	productGroup.PUT("/api/v1/products*", proxyWildcard(productServiceURL))
	productGroup.PATCH("/api/v1/products*", proxyWildcard(productServiceURL))
	productGroup.DELETE("/api/v1/products*", proxyWildcard(productServiceURL))
	productGroup.Any("/api/v1/categories*", proxyWildcard(productServiceURL))
	productGroup.Any("/api/v1/inventory*", proxyWildcard(productServiceURL))
}

func registerIngredientInventoryRoutes(protected *echo.Group, inventoryServiceURL string) {
	inventoryGroup := protected.Group("")
	inventoryGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))

	inventoryGroup.Any("/api/v1/uoms*", proxyWildcard(inventoryServiceURL))
	inventoryGroup.Any("/api/v1/ingredients*", proxyWildcard(inventoryServiceURL))
	inventoryGroup.GET("/api/v1/products/:productId/recipe", proxyWildcard(inventoryServiceURL))
	inventoryGroup.POST("/api/v1/products/:productId/recipes", proxyWildcard(inventoryServiceURL))
	inventoryGroup.GET("/api/v1/products/:productId/recipes", proxyWildcard(inventoryServiceURL))
	inventoryGroup.GET("/api/v1/products/:productId/recipes/:version", proxyWildcard(inventoryServiceURL))
	inventoryGroup.GET("/api/v1/products/:productId/recipe/cost", proxyWildcard(inventoryServiceURL))
	inventoryGroup.POST("/api/v1/inventory/initial-stock", proxyWildcard(inventoryServiceURL))
	inventoryGroup.POST("/api/v1/inventory/purchases", proxyWildcard(inventoryServiceURL))
	inventoryGroup.POST("/api/v1/inventory/adjustments", proxyWildcard(inventoryServiceURL))
	inventoryGroup.POST("/api/v1/inventory/waste", proxyWildcard(inventoryServiceURL))
	inventoryGroup.GET("/api/v1/inventory/ingredients/:ingredientId/movements", proxyWildcard(inventoryServiceURL))
	inventoryGroup.GET("/api/v1/inventory/low-stock", proxyWildcard(inventoryServiceURL))
	inventoryGroup.GET("/api/v1/inventory/valuation", proxyWildcard(inventoryServiceURL))
	inventoryGroup.GET("/api/v1/inventory/orders/:orderId/cost-snapshots", proxyWildcard(inventoryServiceURL))
	inventoryGroup.GET("/api/v1/inventory/orders/:orderId/profitability", proxyWildcard(inventoryServiceURL))
	inventoryGroup.Any("/api/v1/inventory/bundles*", proxyWildcard(inventoryServiceURL))
}

func registerPublicPlansRoute(public *echo.Group, billingServiceURL string) {
	public.GET("/api/v1/public/plans", proxyHandler(billingServiceURL, "/public/plans"))
}

func registerUserOnboardingRoutes(protected *echo.Group, userServiceURL string) {
	onboardingGroup := protected.Group("/api/v1/users/onboarding")
	onboardingGroup.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager, middleware.RoleCashier))
	onboardingGroup.GET("/progress", proxyHandler(userServiceURL, "/api/v1/users/onboarding/progress"))
	onboardingGroup.POST("/complete", proxyHandler(userServiceURL, "/api/v1/users/onboarding/complete"))
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

		resolvedPath, resolvedQuery := resolveProxyPath(c, path)
		originalPath := c.Request().URL.Path
		originalRawQuery := c.Request().URL.RawQuery
		c.Request().URL.Path = resolvedPath
		if resolvedQuery != "" {
			c.Request().URL.RawQuery = resolvedQuery
		}

		proxy.Director = func(req *http.Request) {
			req.Host = target.Host
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = resolvedPath
			if resolvedQuery != "" {
				req.URL.RawQuery = resolvedQuery
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
			if email := c.Get("email"); email != nil {
				req.Header.Set("X-User-Email", email.(string))
			}
			if platformSessionID := c.Get("platform_session_id"); platformSessionID != nil {
				req.Header.Set("X-Platform-Session-ID", platformSessionID.(string))
			}
			if platformAdminID := c.Get("platform_admin_id"); platformAdminID != nil {
				req.Header.Set("X-Platform-Admin-ID", platformAdminID.(string))
			}
			if platformEmail := c.Get("platform_email"); platformEmail != nil {
				req.Header.Set("X-Platform-Email", platformEmail.(string))
			}
			if platformRole := c.Get("platform_role"); platformRole != nil {
				req.Header.Set("X-Platform-Role", platformRole.(string))
			}
		}

		proxy.ServeHTTP(c.Response(), c.Request())

		c.Request().URL.Path = originalPath
		c.Request().URL.RawQuery = originalRawQuery

		return nil
	}
}

func resolveProxyPath(c echo.Context, path string) (string, string) {
	resolvedPath := path
	for _, name := range c.ParamNames() {
		resolvedPath = strings.ReplaceAll(resolvedPath, ":"+name, url.PathEscape(c.Param(name)))
	}

	if parts := strings.SplitN(resolvedPath, "?", 2); len(parts) == 2 {
		return parts[0], parts[1]
	}

	return resolvedPath, ""
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
			if email := c.Get("email"); email != nil {
				req.Header.Set("X-User-Email", email.(string))
			}
			if platformSessionID := c.Get("platform_session_id"); platformSessionID != nil {
				req.Header.Set("X-Platform-Session-ID", platformSessionID.(string))
			}
			if platformAdminID := c.Get("platform_admin_id"); platformAdminID != nil {
				req.Header.Set("X-Platform-Admin-ID", platformAdminID.(string))
			}
			if platformEmail := c.Get("platform_email"); platformEmail != nil {
				req.Header.Set("X-Platform-Email", platformEmail.(string))
			}
			if platformRole := c.Get("platform_role"); platformRole != nil {
				req.Header.Set("X-Platform-Role", platformRole.(string))
			}
		}

		proxy.ServeHTTP(c.Response(), c.Request())

		return nil
	}
}
