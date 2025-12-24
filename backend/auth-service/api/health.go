package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pos/auth-service/src/utils"
)

func HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "ok",
		"service": utils.GetEnv("SERVICE_NAME"),
	})
}

func ReadyCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ready",
	})
}
