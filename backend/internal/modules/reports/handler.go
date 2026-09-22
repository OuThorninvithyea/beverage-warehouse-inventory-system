package reports

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/httpx"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func actorFromContext(c fiber.Ctx) Actor {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok || claims == nil {
		return Actor{}
	}
	return Actor{ID: claims.Subject, Role: claims.Role, WarehouseID: claims.WarehouseID}
}

func reportsHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrValidation):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR",
			"report request data is invalid", err)
	case errors.Is(err, ErrForbidden):
		return httpx.NewError(fiber.StatusForbidden, "FORBIDDEN",
			"your role or warehouse assignment does not allow this report", err)
	default:
		return httpx.NewError(fiber.StatusInternalServerError, "REPORT_FAILED",
			"report could not be generated", err)
	}
}

func parseFilter(c fiber.Ctx) (Filter, error) {
	filter := Filter{}

	if raw := strings.TrimSpace(c.Query("days")); raw != "" {
		days, err := strconv.Atoi(raw)
		if err != nil || days < 1 || days > maxWindowDays {
			return Filter{}, httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST",
				"days must be an integer from 1 to 365", nil)
		}
		filter.Days = days
	}
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > maxRowLimit {
			return Filter{}, httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST",
				"limit must be an integer from 1 to 500", nil)
		}
		filter.Limit = limit
	}
	if raw := strings.TrimSpace(c.Query("warehouse_id")); raw != "" {
		filter.WarehouseID = &raw
	}
	return filter, nil
}

func (h *Handler) Dashboard(c fiber.Ctx) error {
	filter, err := parseFilter(c)
	if err != nil {
		return err
	}
	dashboard, err := h.service.Dashboard(c.Context(), actorFromContext(c), filter)
	if err != nil {
		return reportsHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(dashboard))
}

func (h *Handler) Valuation(c fiber.Ctx) error {
	filter, err := parseFilter(c)
	if err != nil {
		return err
	}
	valuation, err := h.service.Valuation(c.Context(), actorFromContext(c), filter)
	if err != nil {
		return reportsHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(valuation))
}

func (h *Handler) MovementSummary(c fiber.Ctx) error {
	filter, err := parseFilter(c)
	if err != nil {
		return err
	}
	summary, err := h.service.MovementSummary(c.Context(), actorFromContext(c), filter)
	if err != nil {
		return reportsHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(summary))
}

func (h *Handler) Velocity(c fiber.Ctx) error {
	filter, err := parseFilter(c)
	if err != nil {
		return err
	}
	velocity, err := h.service.Velocity(c.Context(), actorFromContext(c), filter)
	if err != nil {
		return reportsHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(velocity))
}
