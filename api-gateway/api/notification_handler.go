package api

import (
	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/config"
	"github.com/pos/api-gateway/utils"
)

func InitNotificationRoutes(notificationGroup *echo.Group) {
	notificationServiceURL := config.NotificationServiceURL

	notificationGroup.Any("/notifications*", utils.ProxyWildcard(notificationServiceURL))
}
