package api

import (
	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/config"
	"github.com/pos/api-gateway/utils"
)

func InitUserRoutes(
	public *echo.Group,
	protected *echo.Group,
	inviteGroup *echo.Group,
	userNotificationGroup *echo.Group,
) {
	userServiceURL := config.UserServiceURL

	public.POST("/api/invitations/:token/accept", utils.ProxyHandler(userServiceURL, "/invitations/:token/accept"))
	protected.GET("/api/invitations", utils.ProxyHandler(userServiceURL, "/invitations"))

	inviteGroup.POST("/api/invitations", utils.ProxyHandler(userServiceURL, "/invitations"))
	inviteGroup.POST("/api/invitations/:id/resend", utils.ProxyHandler(userServiceURL, "/invitations/:id/resend"))

	userNotificationGroup.GET("/notification-preferences", utils.ProxyHandler(userServiceURL, "/api/v1/users/notification-preferences"))
	userNotificationGroup.PATCH("/:user_id/notification-preferences", func(c echo.Context) error {
		userID := c.Param("user_id")
		return utils.ProxyHandler(userServiceURL, "/api/v1/users/"+userID+"/notification-preferences")(c)
	})
}
