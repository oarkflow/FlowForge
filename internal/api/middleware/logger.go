package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func RequestLogger(log zerolog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		reqID := c.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		c.Locals("request_id", reqID)
		c.Set("X-Request-ID", reqID)

		err := c.Next()

		log.Info().
			Str("request_id", reqID).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("latency", time.Since(start)).
			Str("ip", c.IP()).
			Msg("request")

		return err
	}
}
