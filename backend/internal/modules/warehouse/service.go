package warehouse

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

var (
	ErrForbidden               = errors.New("warehouse access is forbidden")
	ErrValidation              = errors.New("warehouse data is invalid")
	ErrInvalidID               = errors.New("resource id is invalid")
	ErrWarehouseNotFound       = errors.New("warehouse not found")
	ErrLocationNotFound        = errors.New("location not found")
	ErrWarehouseCodeConflict   = errors.New("warehouse code already exists")
	ErrLocationCodeConflict    = errors.New("location code already exists in warehouse")
	ErrLocationBarcodeConflict = errors.New("location barcode already exists")
)

type Service interface {
	ListWarehouses(context.Context, Actor, ListFilter) (Page[Warehouse], error)
	GetWarehouse(context.Context, Actor, string) (Warehouse, error)
	CreateWarehouse(context.Context, Actor, WarehouseInput) (Warehouse, error)
	UpdateWarehouse(context.Context, Actor, string, WarehouseInput) (Warehouse, error)
	DeactivateWarehouse(context.Context, Actor, string) error
}

type service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &service{repository: repository}
}

func (s *service) ListWarehouses(
	ctx context.Context,
	actor Actor,
	filter ListFilter,
) (Page[Warehouse], error) {
	scope, err := warehouseScope(actor)
	if err != nil {
		return Page[Warehouse]{}, err
	}

	requestedLimit := normalizeLimit(filter.Limit)
	filter.Limit = requestedLimit + 1
	filter.Search = strings.TrimSpace(filter.Search)

	warehouses, err := s.repository.ListWarehouses(ctx, scope, filter)
	if err != nil {
		return Page[Warehouse]{}, err
	}
	return warehousePage(warehouses, requestedLimit), nil
}

func (s *service) GetWarehouse(ctx context.Context, actor Actor, id string) (Warehouse, error) {
	if !validUUID(id) {
		return Warehouse{}, ErrInvalidID
	}
	if err := requireWarehouseAccess(actor, id); err != nil {
		return Warehouse{}, err
	}
	return s.repository.GetWarehouse(ctx, id)
}

func (s *service) CreateWarehouse(
	ctx context.Context,
	actor Actor,
	input WarehouseInput,
) (Warehouse, error) {
	if actor.Role != auth.RoleAdmin {
		return Warehouse{}, ErrForbidden
	}

	normalized, err := normalizeWarehouseInput(input, true)
	if err != nil {
		return Warehouse{}, err
	}
	return s.repository.CreateWarehouse(ctx, normalized)
}

func (s *service) UpdateWarehouse(
	ctx context.Context,
	actor Actor,
	id string,
	input WarehouseInput,
) (Warehouse, error) {
	if actor.Role != auth.RoleAdmin {
		return Warehouse{}, ErrForbidden
	}
	if !validUUID(id) {
		return Warehouse{}, ErrInvalidID
	}

	normalized, err := normalizeWarehouseInput(input, false)
	if err != nil {
		return Warehouse{}, err
	}
	return s.repository.UpdateWarehouse(ctx, id, normalized)
}

func (s *service) DeactivateWarehouse(ctx context.Context, actor Actor, id string) error {
	if actor.Role != auth.RoleAdmin {
		return ErrForbidden
	}
	if !validUUID(id) {
		return ErrInvalidID
	}
	return s.repository.DeactivateWarehouse(ctx, id)
}

func normalizeWarehouseInput(input WarehouseInput, defaultActive bool) (WarehouseInput, error) {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	if input.Code == "" || input.Name == "" {
		return WarehouseInput{}, ErrValidation
	}

	input.Address = optionalText(input.Address)
	if defaultActive && input.IsActive == nil {
		input.IsActive = boolPointer(true)
	}
	return input, nil
}

func warehouseScope(actor Actor) (*string, error) {
	if actor.Role == auth.RoleAdmin {
		return nil, nil
	}
	if actor.WarehouseID == nil || strings.TrimSpace(*actor.WarehouseID) == "" {
		return nil, ErrForbidden
	}
	if !validUUID(*actor.WarehouseID) {
		return nil, ErrForbidden
	}
	return actor.WarehouseID, nil
}

func requireWarehouseAccess(actor Actor, warehouseID string) error {
	if actor.Role == auth.RoleAdmin {
		return nil
	}
	if actor.WarehouseID == nil || *actor.WarehouseID != warehouseID {
		return ErrForbidden
	}
	return nil
}

func warehousePage(items []Warehouse, requestedLimit int) Page[Warehouse] {
	hasMore := len(items) > requestedLimit
	if hasMore {
		items = items[:requestedLimit]
	}

	page := Page[Warehouse]{
		Items: items,
		Page:  PageInfo{HasMore: hasMore},
	}
	if hasMore && len(items) > 0 {
		cursor := EncodeCursor(items[len(items)-1].CreatedAt, items[len(items)-1].ID)
		page.Page.NextCursor = &cursor
	}
	return page
}

func normalizeLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > 100:
		return 100
	default:
		return limit
	}
}

func optionalText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func boolPointer(value bool) *bool {
	return &value
}

func validUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}
