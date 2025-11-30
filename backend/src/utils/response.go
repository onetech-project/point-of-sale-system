package utils

import "github.com/labstack/echo/v4"

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

type SuccessResponse struct {
	Data    any    `json:"data"`
	Message string `json:"message,omitempty"`
}

// RespondError sends a JSON error response
func RespondError(c echo.Context, statusCode int, message string) error {
	return c.JSON(statusCode, ErrorResponse{Error: message})
}

// RespondSuccess sends a JSON success response
func RespondSuccess(c echo.Context, statusCode int, data any, message string) error {
	return c.JSON(statusCode, SuccessResponse{
		Data:    data,
		Message: message,
	})
}
