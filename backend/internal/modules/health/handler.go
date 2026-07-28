package health

import (
	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/httpx"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Liveness(c fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(httpx.Success(h.service.Liveness()))
}

func (h *Handler) Readiness(c fiber.Ctx) error {
	status, err := h.service.Readiness(c.Context())
	if err != nil {
		return httpx.NewError(
			fiber.StatusServiceUnavailable,
			"SERVICE_NOT_READY",
			"database is unavailable",
			err,
		)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(status))
}
