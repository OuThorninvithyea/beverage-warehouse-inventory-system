package reports

import (
	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

func RegisterRoutes(app *fiber.App, handler *Handler, tokens *auth.TokenManager) {
	api := app.Group("/api/v1", auth.Authenticate(tokens))

	// Reports are a management view: plan.md scopes them to admin and
	// warehouse_manager. The service checks the role again, so the boundary
	// does not depend on route wiring alone.
	reports := api.Group("/reports", auth.RequireRoles(auth.RoleAdmin, auth.RoleWarehouseManager))
	reports.Get("/dashboard", handler.Dashboard)
	reports.Get("/valuation", handler.Valuation)
	reports.Get("/movement-summary", handler.MovementSummary)
	reports.Get("/velocity", handler.Velocity)
}
