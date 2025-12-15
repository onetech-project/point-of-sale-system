package api

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/config"
	"github.com/pos/api-gateway/utils"
)

func InitTenantRoutes(
	public *echo.Group,
	protected *echo.Group,
	adminTenantConfig *echo.Group,
) {
	tenantServiceURL := config.TenantServiceURL

	public.POST("/api/tenants/register", utils.ProxyHandler(tenantServiceURL, "/register"))
	protected.GET("/api/tenant", utils.ProxyHandler(tenantServiceURL, "/tenant"))
	public.GET("/api/public/tenants/:tenant_id/config", func(c echo.Context) error {
		tenantID := c.Param("tenant_id")
		return utils.ProxyHandler(tenantServiceURL, fmt.Sprintf("/public/tenants/%s/config", tenantID))(c)
	})
	adminTenantConfig.Any("/*", utils.ProxyWildcard(tenantServiceURL))
}
