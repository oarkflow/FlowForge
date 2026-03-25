package middleware

import (
	"github.com/gofiber/fiber/v3"
)

var roleHierarchy = map[string]int{
	"owner":     4,
	"admin":     3,
	"developer": 2,
	"viewer":    1,
}

func RequireRole(minRole string) fiber.Handler {
	return func(c fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok || role == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
		}

		userLevel := roleHierarchy[role]
		requiredLevel := roleHierarchy[minRole]

		if userLevel < requiredLevel {
			return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
		}

		return c.Next()
	}
}

func RequireOwner() fiber.Handler  { return RequireRole("owner") }
func RequireAdmin() fiber.Handler  { return RequireRole("admin") }
func RequireDev() fiber.Handler    { return RequireRole("developer") }
func RequireViewer() fiber.Handler { return RequireRole("viewer") }
