package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// SetupRoutes wires the handlers into the Fiber app
func SetupRoutes(app *fiber.App, h *Handler) {
	// Standard Middleware
	app.Use(logger.New())

	// Health Check for Kubernetes/Monitoring
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// V1 API Group
	v1 := app.Group("/api/v1")

	// --- Tenant Routes ---
	tenants := v1.Group("/tenants")
	tenants.Post("/", h.CreateTenant)

	// Use exactly :slug here (all lowercase)
	tenants.Post("/:slug/keys", h.CreateKey)

	/// Auth Routes (Public)
	auth := v1.Group("/auth")
	auth.Post("/register", h.Register)
	auth.Post("/login", h.Login)
}
