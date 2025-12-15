package utils

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/labstack/echo/v4"
)

func ProxyHandler(targetURL, path string) echo.HandlerFunc {
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

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func ProxyWildcard(targetURL string) echo.HandlerFunc {
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
