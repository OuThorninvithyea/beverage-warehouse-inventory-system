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
	ListLocations(context.Context, Actor, string, ListFilter) (Page[Location], error)
	GetLocation(context.Context, Actor, string, string) (Location, error)
	CreateLocation(context.Context, Actor, string, LocationInput) (Location, error)
	UpdateLocation(context.Context, Actor, string, string, LocationInput) (Location, error)
	DeactivateLocation(context.Context, Actor, string, string) error
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

func (s *service) ListLocations(
	ctx context.Context,
	actor Actor,
	warehouseID string,
	filter ListFilter,
) (Page[Location], error) {
	if !validUUID(warehouseID) {
		return Page[Location]{}, ErrInvalidID
	}
	if err := requireWarehouseAccess(actor, warehouseID); err != nil {
		return Page[Location]{}, err
	}

	requestedLimit := normalizeLimit(filter.Limit)
	filter.Limit = requestedLimit + 1
	filter.Search = strings.TrimSpace(filter.Search)

	locations, err := s.repository.ListLocations(ctx, warehouseID, filter)
	if err != nil {
		return Page[Location]{}, err
	}
	return locationPage(locations, requestedLimit), nil
}

func (s *service) GetLocation(
	ctx context.Context,
	actor Actor,
	warehouseID string,
	locationID string,
) (Location, error) {
	if !validUUID(warehouseID) || !validUUID(locationID) {
		return Location{}, ErrInvalidID
	}
	if err := requireWarehouseAccess(actor, warehouseID); err != nil {
		return Location{}, err
	}
	return s.repository.GetLocation(ctx, warehouseID, locationID)
}

func (s *service) CreateLocation(
	ctx context.Context,
	actor Actor,
	warehouseID string,
	input LocationInput,
) (Location, error) {
	if !canMutateLocations(actor.Role) {
		return Location{}, ErrForbidden
	}
	if !validUUID(warehouseID) {
		return Location{}, ErrInvalidID
	}
	if err := requireWarehouseAccess(actor, warehouseID); err != nil {
		return Location{}, err
	}

	normalized, err := normalizeLocationInput(input, true)
	if err != nil {
		return Location{}, err
	}
	return s.repository.CreateLocation(ctx, warehouseID, normalized)
}

func (s *service) UpdateLocation(
	ctx context.Context,
	actor Actor,
	warehouseID string,
	locationID string,
	input LocationInput,
) (Location, error) {
	if !canMutateLocations(actor.Role) {
		return Location{}, ErrForbidden
	}
	if !validUUID(warehouseID) || !validUUID(locationID) {
		return Location{}, ErrInvalidID
	}
	if err := requireWarehouseAccess(actor, warehouseID); err != nil {
		return Location{}, err
	}

	normalized, err := normalizeLocationInput(input, false)
	if err != nil {
		return Location{}, err
	}
	return s.repository.UpdateLocation(ctx, warehouseID, locationID, normalized)
}

func (s *service) DeactivateLocation(
	ctx context.Context,
	actor Actor,
	warehouseID string,
	locationID string,
) error {
	if !canMutateLocations(actor.Role) {
		return ErrForbidden
	}
	if !validUUID(warehouseID) || !validUUID(locationID) {
		return ErrInvalidID
	}
	if err := requireWarehouseAccess(actor, warehouseID); err != nil {
		return err
	}
	return s.repository.DeactivateLocation(ctx, warehouseID, locationID)
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

func normalizeLocationInput(input LocationInput, defaults bool) (LocationInput, error) {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	if input.Code == "" {
		return LocationInput{}, ErrValidation
	}

	input.Zone = optionalText(input.Zone)
	input.Aisle = optionalText(input.Aisle)
	input.Rack = optionalText(input.Rack)
	input.Shelf = optionalText(input.Shelf)
	input.Barcode = optionalText(input.Barcode)
	if defaults {
		if input.IsPickable == nil {
			input.IsPickable = boolPointer(true)
		}
		if input.IsActive == nil {
			input.IsActive = boolPointer(true)
		}
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
	if items == nil {
		items = make([]Warehouse, 0)
	}
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

func locationPage(items []Location, requestedLimit int) Page[Location] {
	if items == nil {
		items = make([]Location, 0)
	}
	hasMore := len(items) > requestedLimit
	if hasMore {
		items = items[:requestedLimit]
	}

	page := Page[Location]{
		Items: items,
		Page:  PageInfo{HasMore: hasMore},
	}
	if hasMore && len(items) > 0 {
		cursor := EncodeCursor(items[len(items)-1].CreatedAt, items[len(items)-1].ID)
		page.Page.NextCursor = &cursor
	}
	return page
}

func canMutateLocations(role string) bool {
	return role == auth.RoleAdmin || role == auth.RoleWarehouseManager
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
