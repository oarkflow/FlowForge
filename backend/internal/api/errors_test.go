package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestStatusText(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{400, "bad_request"},
		{401, "unauthorized"},
		{403, "forbidden"},
		{404, "not_found"},
		{409, "conflict"},
		{422, "validation_error"},
		{429, "rate_limited"},
		{500, "internal_error"},
		{503, "internal_error"},
		{0, "internal_error"},
	}
	for _, tt := range tests {
		got := statusText(tt.code)
		if got != tt.want {
			t.Errorf("statusText(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestErrorResponse_Fields(t *testing.T) {
	resp := ErrorResponse{
		Error:   "bad_request",
		Message: "invalid input",
		Code:    400,
	}
	if resp.Error != "bad_request" {
		t.Errorf("Error = %q", resp.Error)
	}
	if resp.Message != "invalid input" {
		t.Errorf("Message = %q", resp.Message)
	}
	if resp.Code != 400 {
		t.Errorf("Code = %d", resp.Code)
	}
}

func TestErrorHandler_FiberError(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})

	app.Get("/test-error", func(c fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "resource not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-error", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("body should not be empty")
	}
}

func TestErrorHandler_GenericError(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})

	app.Get("/test-generic", func(c fiber.Ctx) error {
		return errors.New("something went wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-generic", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
}

func TestErrorHandler_BadRequest(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})

	app.Get("/bad", func(c fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "bad input")
	})

	req := httptest.NewRequest(http.MethodGet, "/bad", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestErrorHandler_Unauthorized(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})

	app.Get("/auth", func(c fiber.Ctx) error {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
	})

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}
