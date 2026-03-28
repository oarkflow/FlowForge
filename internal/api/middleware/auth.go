package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func Auth(secret string) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing authorization header")
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid authorization format")
		}

		claims := &Claims{}
		t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !t.Valid {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("email", claims.Email)
		c.Locals("username", claims.Username)
		c.Locals("role", claims.Role)
		c.Locals("claims", claims)

		return c.Next()
	}
}

func OptionalAuth(secret string) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Next()
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth {
			return c.Next()
		}

		claims := &Claims{}
		t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err == nil && t.Valid {
			c.Locals("user_id", claims.UserID)
			c.Locals("email", claims.Email)
			c.Locals("username", claims.Username)
			c.Locals("role", claims.Role)
			c.Locals("claims", claims)
		}

		return c.Next()
	}
}
