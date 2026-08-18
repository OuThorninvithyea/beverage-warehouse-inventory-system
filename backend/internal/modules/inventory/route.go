package inventory

import (
	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

func RegisterRoutes(app *fiber.App, handler *Handler, tokens *auth.TokenManager) {
	api := app.Group("/api/v1", auth.Authenticate(tokens))
	receiveOrPick := auth.RequireRoles(auth.RoleAdmin, auth.RoleWarehouseManager, auth.RolePicker)
	transferOrAdjust := auth.RequireRoles(auth.RoleAdmin, auth.RoleWarehouseManager)

	inventory := api.Group("/inventory")
	inventory.Get("", handler.ListBalances)
	inventory.Get("/products/:product_id/lots", handler.ListLots)

	movements := inventory.Group("/movements")
	movements.Post("/receive", receiveOrPick, handler.Receive)
	movements.Post("/pick", receiveOrPick, handler.Pick)
	movements.Post("/transfer", transferOrAdjust, handler.Transfer)
	movements.Post("/adjust", transferOrAdjust, handler.Adjust)
	movements.Get("", handler.ListMovements)
	movements.Get("/:movement_id", handler.GetMovement)
}
