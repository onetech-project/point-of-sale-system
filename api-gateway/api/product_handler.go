package api

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/config"
	"github.com/pos/api-gateway/utils"
)

func InitProductRoutes(
	public *echo.Group,
	protected *echo.Group,
	productGroup *echo.Group,
) {
	productServiceURL := config.ProductServiceURL

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
	productGroup.Any("/api/v1/products*", utils.ProxyWildcard(productServiceURL))
	productGroup.Any("/api/v1/categories*", utils.ProxyWildcard(productServiceURL))
	productGroup.Any("/api/v1/inventory*", utils.ProxyWildcard(productServiceURL))
}
