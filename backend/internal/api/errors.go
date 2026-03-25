package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func ErrorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	var e *fiber.Error
	if ok := errors.As(err, &e); ok {
		code = e.Code
	}

	return c.Status(code).JSON(ErrorResponse{
		Error:   statusText(code),
		Message: err.Error(),
		Code:    code,
	})
}

func statusText(code int) string {
	switch code {
	case 400:
		return "bad_request"
	case 401:
		return "unauthorized"
	case 403:
		return "forbidden"
	case 404:
		return "not_found"
	case 409:
		return "conflict"
	case 422:
		return "validation_error"
	case 429:
		return "rate_limited"
	default:
		return "internal_error"
	}
}
