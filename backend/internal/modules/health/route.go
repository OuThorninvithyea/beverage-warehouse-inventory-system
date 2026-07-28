package health

import "github.com/gofiber/fiber/v3"

func RegisterRoutes(app *fiber.App, handler *Handler) {
	app.Get("/health", handler.Liveness)
	app.Get("/ready", handler.Readiness)
}
