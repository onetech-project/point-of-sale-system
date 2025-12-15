package api

import (
	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/config"
	"github.com/pos/api-gateway/utils"
)

func InitOrderRoutes(public *echo.Group, protected *echo.Group, adminOrders *echo.Group, publicOrders *echo.Group, adminSettings *echo.Group) {
	orderServiceURL := config.OrderServiceURL

	adminOrders.Any("/orders*", utils.ProxyWildcard(orderServiceURL))
	publicOrders.Any("/*", utils.ProxyWildcard(orderServiceURL))
	adminSettings.Any("/settings*", utils.ProxyWildcard(orderServiceURL))
	public.Any("/api/v1/webhooks/*", utils.ProxyWildcard(orderServiceURL))
}
