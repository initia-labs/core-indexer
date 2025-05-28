package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// RequestLogger returns a middleware that logs HTTP request information using zerolog
func RequestLogger(log *zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()

		// Process request
		err := c.Next()

		// Get status code
		status := c.Response().StatusCode()

		// Create log event
		event := log.Info().
			Str("method", method).
			Str("path", path).
			Int("status", status).
			Dur("latency", time.Since(start)).
			Str("ip", c.IP()).
			Str("user_agent", c.Get("User-Agent"))

		// Add error if any
		if err != nil {
			event.Err(err)
		}

		// Log the request
		event.Send()

		return err
	}
}
