package warehouse

import (
	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

func RegisterRoutes(app *fiber.App, handler *Handler, tokens *auth.TokenManager) {
	api := app.Group("/api/v1", auth.Authenticate(tokens))
	warehouses := api.Group("/warehouses")

	warehouses.Get("", handler.ListWarehouses)
	warehouses.Post("", auth.RequireRoles(auth.RoleAdmin), handler.CreateWarehouse)
	warehouses.Get("/:warehouse_id", handler.GetWarehouse)
	warehouses.Put("/:warehouse_id", auth.RequireRoles(auth.RoleAdmin), handler.UpdateWarehouse)
	warehouses.Delete("/:warehouse_id", auth.RequireRoles(auth.RoleAdmin), handler.DeactivateWarehouse)

	locations := warehouses.Group("/:warehouse_id/locations")
	locations.Get("", handler.ListLocations)
	locations.Post("", auth.RequireRoles(auth.RoleAdmin, auth.RoleWarehouseManager), handler.CreateLocation)
	locations.Get("/:location_id", handler.GetLocation)
	locations.Put("/:location_id", auth.RequireRoles(auth.RoleAdmin, auth.RoleWarehouseManager), handler.UpdateLocation)
	locations.Delete("/:location_id", auth.RequireRoles(auth.RoleAdmin, auth.RoleWarehouseManager), handler.DeactivateLocation)
}
