package utils

import (
	"github.com/labstack/echo/v4"
)

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func NewErrorResponse(code int, message string, details ...string) *ErrorResponse {
	er := &ErrorResponse{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		er.Details = details[0]
	}
	return er
}

func RespondError(c echo.Context, code int, message string, details ...string) error {
	return c.JSON(code, NewErrorResponse(code, message, details...))
}

func RespondBadRequest(c echo.Context, message string, details ...string) error {
	return RespondError(c, 400, message, details...)
}

func RespondNotFound(c echo.Context, message string) error {
	return RespondError(c, 404, message)
}

func RespondConflict(c echo.Context, message string, details ...string) error {
	return RespondError(c, 409, message, details...)
}

func RespondInternalError(c echo.Context, message string) error {
	return RespondError(c, 500, message)
}
