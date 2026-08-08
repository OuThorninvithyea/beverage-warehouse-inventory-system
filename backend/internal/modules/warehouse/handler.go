package warehouse

import (
	"errors"
	"strconv"

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

type warehouseWriteRequest struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	Address  *string `json:"address"`
	IsActive *bool   `json:"is_active"`
}

type locationWriteRequest struct {
	Code       string  `json:"code"`
	Zone       *string `json:"zone"`
	Aisle      *string `json:"aisle"`
	Rack       *string `json:"rack"`
	Shelf      *string `json:"shelf"`
	Barcode    *string `json:"barcode"`
	IsPickable *bool   `json:"is_pickable"`
	IsActive   *bool   `json:"is_active"`
}

func (h *Handler) ListWarehouses(c fiber.Ctx) error {
	filter, err := parseListFilter(c, false)
	if err != nil {
		return err
	}
	page, err := h.service.ListWarehouses(c.Context(), actorFromContext(c), filter)
	if err != nil {
		return warehouseHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(page))
}

func (h *Handler) GetWarehouse(c fiber.Ctx) error {
	warehouse, err := h.service.GetWarehouse(c.Context(), actorFromContext(c), c.Params("warehouse_id"))
	if err != nil {
		return warehouseHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(warehouse))
}

func (h *Handler) CreateWarehouse(c fiber.Ctx) error {
	request, err := bindWarehouseRequest(c)
	if err != nil {
		return err
	}
	warehouse, err := h.service.CreateWarehouse(c.Context(), actorFromContext(c), request)
	if err != nil {
		return warehouseHTTPError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(warehouse))
}

func (h *Handler) UpdateWarehouse(c fiber.Ctx) error {
	request, err := bindWarehouseRequest(c)
	if err != nil {
		return err
	}
	warehouse, err := h.service.UpdateWarehouse(
		c.Context(),
		actorFromContext(c),
		c.Params("warehouse_id"),
		request,
	)
	if err != nil {
		return warehouseHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(warehouse))
}

func (h *Handler) DeactivateWarehouse(c fiber.Ctx) error {
	if err := h.service.DeactivateWarehouse(
		c.Context(),
		actorFromContext(c),
		c.Params("warehouse_id"),
	); err != nil {
		return warehouseHTTPError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) ListLocations(c fiber.Ctx) error {
	filter, err := parseListFilter(c, true)
	if err != nil {
		return err
	}
	page, err := h.service.ListLocations(
		c.Context(),
		actorFromContext(c),
		c.Params("warehouse_id"),
		filter,
	)
	if err != nil {
		return warehouseHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(page))
}

func (h *Handler) GetLocation(c fiber.Ctx) error {
	location, err := h.service.GetLocation(
		c.Context(),
		actorFromContext(c),
		c.Params("warehouse_id"),
		c.Params("location_id"),
	)
	if err != nil {
		return warehouseHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(location))
}

func (h *Handler) CreateLocation(c fiber.Ctx) error {
	request, err := bindLocationRequest(c)
	if err != nil {
		return err
	}
	location, err := h.service.CreateLocation(
		c.Context(),
		actorFromContext(c),
		c.Params("warehouse_id"),
		request,
	)
	if err != nil {
		return warehouseHTTPError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(location))
}

func (h *Handler) UpdateLocation(c fiber.Ctx) error {
	request, err := bindLocationRequest(c)
	if err != nil {
		return err
	}
	location, err := h.service.UpdateLocation(
		c.Context(),
		actorFromContext(c),
		c.Params("warehouse_id"),
		c.Params("location_id"),
		request,
	)
	if err != nil {
		return warehouseHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(location))
}

func (h *Handler) DeactivateLocation(c fiber.Ctx) error {
	if err := h.service.DeactivateLocation(
		c.Context(),
		actorFromContext(c),
		c.Params("warehouse_id"),
		c.Params("location_id"),
	); err != nil {
		return warehouseHTTPError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func bindWarehouseRequest(c fiber.Ctx) (WarehouseInput, error) {
	var request warehouseWriteRequest
	if err := c.Bind().Body(&request); err != nil {
		return WarehouseInput{}, httpx.NewError(
			fiber.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid JSON request",
			err,
		)
	}
	return WarehouseInput{
		Code: request.Code, Name: request.Name, Address: request.Address, IsActive: request.IsActive,
	}, nil
}

func bindLocationRequest(c fiber.Ctx) (LocationInput, error) {
	var request locationWriteRequest
	if err := c.Bind().Body(&request); err != nil {
		return LocationInput{}, httpx.NewError(
			fiber.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid JSON request",
			err,
		)
	}
	return LocationInput{
		Code: request.Code, Zone: request.Zone, Aisle: request.Aisle, Rack: request.Rack,
		Shelf: request.Shelf, Barcode: request.Barcode,
		IsPickable: request.IsPickable, IsActive: request.IsActive,
	}, nil
}

func parseListFilter(c fiber.Ctx, includePickable bool) (ListFilter, error) {
	filter := ListFilter{Search: c.Query("search")}

	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return ListFilter{}, invalidQueryError("limit must be an integer from 1 to 100")
		}
		filter.Limit = limit
	}
	if raw := c.Query("after"); raw != "" {
		cursor, err := DecodeCursor(raw)
		if err != nil {
			return ListFilter{}, invalidQueryError("after cursor is invalid")
		}
		filter.After = &cursor
	}

	active, err := optionalQueryBool(c.Query("is_active"))
	if err != nil {
		return ListFilter{}, invalidQueryError("is_active must be true or false")
	}
	filter.IsActive = active

	if includePickable {
		pickable, err := optionalQueryBool(c.Query("is_pickable"))
		if err != nil {
			return ListFilter{}, invalidQueryError("is_pickable must be true or false")
		}
		filter.IsPickable = pickable
	}
	return filter, nil
}

func optionalQueryBool(value string) (*bool, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func invalidQueryError(message string) error {
	return httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST", message, nil)
}

func actorFromContext(c fiber.Ctx) Actor {
	claims, _ := auth.ClaimsFromContext(c)
	if claims == nil {
		return Actor{}
	}
	return Actor{Role: claims.Role, WarehouseID: claims.WarehouseID}
}

func warehouseHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrValidation):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "warehouse or location data is invalid", err)
	case errors.Is(err, ErrForbidden):
		return httpx.NewError(fiber.StatusForbidden, "FORBIDDEN", "your role or warehouse assignment does not allow this action", err)
	case errors.Is(err, ErrInvalidID), errors.Is(err, ErrInvalidCursor):
		return httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST", "a resource identifier or cursor is invalid", err)
	case errors.Is(err, ErrWarehouseNotFound):
		return httpx.NewError(fiber.StatusNotFound, "WAREHOUSE_NOT_FOUND", "warehouse was not found", err)
	case errors.Is(err, ErrLocationNotFound):
		return httpx.NewError(fiber.StatusNotFound, "LOCATION_NOT_FOUND", "location was not found", err)
	case errors.Is(err, ErrWarehouseCodeConflict):
		return httpx.NewError(fiber.StatusConflict, "WAREHOUSE_CODE_CONFLICT", "warehouse code already exists", err)
	case errors.Is(err, ErrLocationCodeConflict):
		return httpx.NewError(fiber.StatusConflict, "LOCATION_CODE_CONFLICT", "location code already exists in this warehouse", err)
	case errors.Is(err, ErrLocationBarcodeConflict):
		return httpx.NewError(fiber.StatusConflict, "LOCATION_BARCODE_CONFLICT", "location barcode already exists", err)
	default:
		return httpx.NewError(fiber.StatusInternalServerError, "WAREHOUSE_OPERATION_FAILED", "warehouse operation could not be completed", err)
	}
}
