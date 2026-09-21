package inventory

import (
	"errors"
	"strconv"
	"time"

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

func invalidBodyError(err error) error {
	return httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST", "request body is not valid JSON", err)
}

func invalidQueryError(message string) error {
	return httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST", message, nil)
}

func inventoryHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrValidation):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "inventory request data is invalid", err)
	case errors.Is(err, ErrForbidden):
		return httpx.NewError(fiber.StatusForbidden, "FORBIDDEN", "your role or warehouse assignment does not allow this action", err)
	case errors.Is(err, ErrProductNotFound):
		return httpx.NewError(fiber.StatusNotFound, "PRODUCT_NOT_FOUND", "product was not found or is not active", err)
	case errors.Is(err, ErrLocationNotFound):
		return httpx.NewError(fiber.StatusNotFound, "LOCATION_NOT_FOUND", "location was not found", err)
	case errors.Is(err, ErrLotNotFound):
		return httpx.NewError(fiber.StatusNotFound, "LOT_NOT_FOUND", "lot was not found for this product", err)
	case errors.Is(err, ErrLotExpiryRequired):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "LOT_EXPIRY_REQUIRED", "lot number and expiration date are required for lot-tracked products", err)
	case errors.Is(err, ErrMovementNotFound):
		return httpx.NewError(fiber.StatusNotFound, "MOVEMENT_NOT_FOUND", "movement was not found", err)
	case errors.Is(err, ErrInsufficientStock):
		return httpx.NewError(fiber.StatusConflict, "INSUFFICIENT_STOCK", "not enough available stock for this operation", err)
	case errors.Is(err, ErrSameLocationTransfer):
		return httpx.NewError(fiber.StatusConflict, "SAME_LOCATION_TRANSFER", "transfer source and destination locations must differ", err)
	case errors.Is(err, ErrWarehouseMismatch):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "WAREHOUSE_MISMATCH", "location does not belong to your assigned warehouse", err)
	default:
		return httpx.NewError(fiber.StatusInternalServerError, "INVENTORY_OPERATION_FAILED", "inventory operation could not be completed", err)
	}
}

func (h *Handler) ListBalances(c fiber.Ctx) error {
	filter, err := parseBalanceFilter(c)
	if err != nil {
		return err
	}
	page, err := h.service.ListBalances(c.Context(), actorFromContext(c), filter)
	if err != nil {
		return inventoryHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(page))
}

func (h *Handler) ListLots(c fiber.Ctx) error {
	lots, err := h.service.ListLots(c.Context(), actorFromContext(c), c.Params("product_id"))
	if err != nil {
		return inventoryHTTPError(err)
	}
	if lots == nil {
		lots = []Lot{}
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(lots))
}

type receiveRequest struct {
	LocationID     string         `json:"location_id"`
	ProductID      string         `json:"product_id"`
	Quantity       string         `json:"quantity"`
	UnitCost       string         `json:"unit_cost"`
	LotNumber      OptionalString `json:"lot_number"`
	ExpirationDate OptionalString `json:"expiration_date"`
	Reference      OptionalString `json:"reference"`
	Notes          OptionalString `json:"notes"`
}

func (h *Handler) Receive(c fiber.Ctx) error {
	var request receiveRequest
	if err := c.Bind().Body(&request); err != nil {
		return invalidBodyError(err)
	}
	movement, balance, err := h.service.Receive(c.Context(), actorFromContext(c), ReceiveInput{
		LocationID: request.LocationID, ProductID: request.ProductID,
		Quantity: request.Quantity, UnitCost: request.UnitCost,
		LotNumber: request.LotNumber, ExpirationDate: request.ExpirationDate,
		Reference: request.Reference, Notes: request.Notes,
	})
	if err != nil {
		return inventoryHTTPError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(fiber.Map{
		"movement": movement, "balance": balance,
	}))
}

type pickRequest struct {
	LocationID string         `json:"location_id"`
	ProductID  string         `json:"product_id"`
	Quantity   string         `json:"quantity"`
	LotID      OptionalString `json:"lot_id"`
	Reference  OptionalString `json:"reference"`
	Notes      OptionalString `json:"notes"`
}

func (h *Handler) Pick(c fiber.Ctx) error {
	var request pickRequest
	if err := c.Bind().Body(&request); err != nil {
		return invalidBodyError(err)
	}
	movements, err := h.service.Pick(c.Context(), actorFromContext(c), PickInput{
		LocationID: request.LocationID, ProductID: request.ProductID, Quantity: request.Quantity,
		LotID: request.LotID, Reference: request.Reference, Notes: request.Notes,
	})
	if err != nil {
		return inventoryHTTPError(err)
	}
	total := "0"
	if len(movements) > 0 {
		total = request.Quantity
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(fiber.Map{
		"movements": movements, "total_quantity": total,
	}))
}

type transferRequest struct {
	ProductID      string         `json:"product_id"`
	LotID          OptionalString `json:"lot_id"`
	Quantity       string         `json:"quantity"`
	FromLocationID string         `json:"from_location_id"`
	ToLocationID   string         `json:"to_location_id"`
	Reference      OptionalString `json:"reference"`
	Notes          OptionalString `json:"notes"`
}

func (h *Handler) Transfer(c fiber.Ctx) error {
	var request transferRequest
	if err := c.Bind().Body(&request); err != nil {
		return invalidBodyError(err)
	}
	movement, source, destination, err := h.service.Transfer(c.Context(), actorFromContext(c), TransferInput{
		ProductID: request.ProductID, LotID: request.LotID, Quantity: request.Quantity,
		FromLocationID: request.FromLocationID, ToLocationID: request.ToLocationID,
		Reference: request.Reference, Notes: request.Notes,
	})
	if err != nil {
		return inventoryHTTPError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(fiber.Map{
		"movement": movement, "source_balance": source, "destination_balance": destination,
	}))
}

type adjustRequest struct {
	LocationID string         `json:"location_id"`
	ProductID  string         `json:"product_id"`
	LotID      OptionalString `json:"lot_id"`
	Direction  string         `json:"direction"`
	Quantity   string         `json:"quantity"`
	Notes      OptionalString `json:"notes"`
}

func (h *Handler) Adjust(c fiber.Ctx) error {
	var request adjustRequest
	if err := c.Bind().Body(&request); err != nil {
		return invalidBodyError(err)
	}
	movement, balance, err := h.service.Adjust(c.Context(), actorFromContext(c), AdjustInput{
		LocationID: request.LocationID, ProductID: request.ProductID, LotID: request.LotID,
		Direction: request.Direction, Quantity: request.Quantity, Notes: request.Notes,
	})
	if err != nil {
		return inventoryHTTPError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(fiber.Map{
		"movement": movement, "balance": balance,
	}))
}

func (h *Handler) ListMovements(c fiber.Ctx) error {
	filter, err := parseMovementFilter(c)
	if err != nil {
		return err
	}
	page, err := h.service.ListMovements(c.Context(), actorFromContext(c), filter)
	if err != nil {
		return inventoryHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(page))
}

func (h *Handler) GetMovement(c fiber.Ctx) error {
	movement, err := h.service.GetMovement(c.Context(), actorFromContext(c), c.Params("movement_id"))
	if err != nil {
		return inventoryHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(movement))
}

func parseExpiryAlertFilter(c fiber.Ctx) (ExpiryAlertFilter, error) {
	filter := ExpiryAlertFilter{}
	if raw := c.Query("within_days"); raw != "" {
		days, err := strconv.Atoi(raw)
		if err != nil || days < 0 || days > maxExpiryWindowDays {
			return ExpiryAlertFilter{}, invalidQueryError("within_days must be an integer from 0 to 365")
		}
		filter.WithinDays = &days
	}
	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > maxExpiryAlerts {
			return ExpiryAlertFilter{}, invalidQueryError("limit must be an integer from 1 to 500")
		}
		filter.Limit = limit
	}
	filter.WarehouseID = optionalQueryUUID(c.Query("warehouse_id"))
	return filter, nil
}

func (h *Handler) ListExpiryAlerts(c fiber.Ctx) error {
	filter, err := parseExpiryAlertFilter(c)
	if err != nil {
		return err
	}
	alerts, err := h.service.ListExpiryAlerts(c.Context(), actorFromContext(c), filter)
	if err != nil {
		return inventoryHTTPError(err)
	}
	if alerts == nil {
		alerts = []ExpiryAlert{}
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(alerts))
}

func parseBalanceFilter(c fiber.Ctx) (BalanceListFilter, error) {
	filter := BalanceListFilter{}
	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return BalanceListFilter{}, invalidQueryError("limit must be an integer from 1 to 100")
		}
		filter.Limit = limit
	}
	if raw := c.Query("after"); raw != "" {
		cursor, err := DecodeCursor(raw)
		if err != nil {
			return BalanceListFilter{}, invalidQueryError("after cursor is invalid")
		}
		filter.After = &cursor
	}
	filter.LocationID = optionalQueryUUID(c.Query("location_id"))
	filter.ProductID = optionalQueryUUID(c.Query("product_id"))
	filter.WarehouseID = optionalQueryUUID(c.Query("warehouse_id"))
	filter.LotID = optionalQueryUUID(c.Query("lot_id"))
	return filter, nil
}

func parseMovementFilter(c fiber.Ctx) (MovementListFilter, error) {
	filter := MovementListFilter{}
	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return MovementListFilter{}, invalidQueryError("limit must be an integer from 1 to 100")
		}
		filter.Limit = limit
	}
	if raw := c.Query("after"); raw != "" {
		cursor, err := DecodeCursor(raw)
		if err != nil {
			return MovementListFilter{}, invalidQueryError("after cursor is invalid")
		}
		filter.After = &cursor
	}
	filter.ProductID = optionalQueryUUID(c.Query("product_id"))
	filter.LocationID = optionalQueryUUID(c.Query("location_id"))
	if raw := c.Query("movement_type"); raw != "" {
		filter.MovementType = &raw
	}
	if raw := c.Query("from"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return MovementListFilter{}, invalidQueryError("from must be an RFC3339 timestamp")
		}
		filter.From = &parsed
	}
	if raw := c.Query("to"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return MovementListFilter{}, invalidQueryError("to must be an RFC3339 timestamp")
		}
		filter.To = &parsed
	}
	return filter, nil
}

func optionalQueryUUID(raw string) *string {
	if raw == "" {
		return nil
	}
	return &raw
}
