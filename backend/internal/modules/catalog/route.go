package catalog

import (
	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

func RegisterRoutes(app *fiber.App, handler *Handler, tokens *auth.TokenManager) {
	api := app.Group("/api/v1", auth.Authenticate(tokens))
	mutators := []string{auth.RoleAdmin, auth.RoleWarehouseManager}

	categories := api.Group("/categories")
	categories.Get("", handler.ListCategories)
	categories.Post("", auth.RequireRoles(mutators...), handler.CreateCategory)
	categories.Get("/:category_id", handler.GetCategory)
	categories.Put("/:category_id", auth.RequireRoles(mutators...), handler.UpdateCategory)
	categories.Delete("/:category_id", auth.RequireRoles(mutators...), handler.DeactivateCategory)

	products := api.Group("/products")
	products.Get("", handler.ListProducts)
	products.Post("", auth.RequireRoles(mutators...), handler.CreateProduct)
	products.Get("/by-barcode/:barcode", handler.GetProductByBarcode)
	products.Get("/:product_id", handler.GetProduct)
	products.Put("/:product_id", auth.RequireRoles(mutators...), handler.UpdateProduct)
	products.Delete("/:product_id", auth.RequireRoles(mutators...), handler.DeactivateProduct)
}
