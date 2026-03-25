package middleware

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

var validate = validator.New()

func ValidateBody(out interface{}) fiber.Handler {
	return func(c fiber.Ctx) error {
		if err := c.Bind().JSON(out); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body: "+err.Error())
		}
		if err := validate.Struct(out); err != nil {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "validation error: "+err.Error())
		}
		c.Locals("body", out)
		return c.Next()
	}
}

func Validate(s interface{}) error {
	return validate.Struct(s)
}
