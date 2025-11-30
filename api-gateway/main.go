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
	public.POST("/api/tenants/register", proxyHandler(tenantServiceURL, "/register"))

	authServiceURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8082")
	public.POST("/api/auth/login", proxyHandler(authServiceURL, "/login"))

	userServiceURL := getEnv("USER_SERVICE_URL", "http://localhost:8083")
	public.POST("/api/invitations/:token/accept", proxyHandler(userServiceURL, "/invitations/:token/accept"))

	protected := e.Group("")
	protected.Use(middleware.JWTAuth())
	protected.Use(middleware.TenantScope())

	protected.GET("/api/auth/session", proxyHandler(authServiceURL, "/session"))
	protected.POST("/api/auth/logout", proxyHandler(authServiceURL, "/logout"))

	protected.GET("/api/tenant", proxyHandler(tenantServiceURL, "/tenant"))

	protected.POST("/api/invitations", proxyHandler(userServiceURL, "/invitations"))
	protected.GET("/api/invitations", proxyHandler(userServiceURL, "/invitations"))
	protected.POST("/api/invitations/:id/resend", proxyHandler(userServiceURL, "/invitations/:id/resend"))

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
