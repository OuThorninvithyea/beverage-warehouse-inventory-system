package auth

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/httpx"
)

func RegisterRoutes(app *fiber.App, handler *Handler, tokens *TokenManager) {
	api := app.Group("/api/v1")
	authGroup := api.Group("/auth")

	authGroup.Post("/login", limiter.New(limiter.Config{
		Max:        5,
		Expiration: time.Minute,
		LimitReached: func(c fiber.Ctx) error {
			return httpx.NewError(
				fiber.StatusTooManyRequests,
				"RATE_LIMITED",
				"too many login attempts; try again later",
				nil,
			)
		},
	}), handler.Login)
	authGroup.Post("/refresh", handler.Refresh)
	authGroup.Post("/logout", handler.Logout)
	authGroup.Get("/me", Authenticate(tokens), handler.Me)

	api.Get(
		"/admin/ping",
		Authenticate(tokens),
		RequireRoles(RoleAdmin),
		handler.AdminPing,
	)
}
