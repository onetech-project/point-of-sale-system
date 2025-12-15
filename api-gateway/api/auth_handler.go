package api

import (
	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/config"
	"github.com/pos/api-gateway/utils"
)

func InitAuthRoutes(public *echo.Group, protected *echo.Group) {
	authServiceURL := config.AuthServiceURL

	public.POST("/api/auth/login", utils.ProxyHandler(authServiceURL, "/login"))
	public.POST("/api/auth/refresh", utils.ProxyHandler(authServiceURL, "/refresh"))
	protected.POST("/api/auth/logout", utils.ProxyHandler(authServiceURL, "/logout"))
	protected.GET("/api/auth/session", utils.ProxyHandler(authServiceURL, "/session"))
}
