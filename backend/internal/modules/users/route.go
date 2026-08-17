package users

import (
	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

func RegisterRoutes(app *fiber.App, handler *Handler, tokens *auth.TokenManager) {
	api := app.Group("/api/v1", auth.Authenticate(tokens))
	adminOnly := auth.RequireRoles(auth.RoleAdmin)

	usersGroup := api.Group("/users", adminOnly)
	usersGroup.Get("", handler.ListUsers)
	usersGroup.Post("", handler.CreateUser)
	usersGroup.Get("/:user_id", handler.GetUser)
	usersGroup.Put("/:user_id", handler.UpdateUser)
	usersGroup.Delete("/:user_id", handler.DeactivateUser)
	usersGroup.Post("/:user_id/password-reset", handler.ResetPassword)
}
