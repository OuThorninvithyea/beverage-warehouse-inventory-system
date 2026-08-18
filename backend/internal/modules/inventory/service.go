package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

var (
	ErrForbidden            = errors.New("actor is not permitted to perform this action")
	ErrValidation           = errors.New("inventory request data is invalid")
	ErrSameLocationTransfer = errors.New("transfer source and destination locations must differ")
)

type Service interface {
	ListBalances(ctx context.Context, actor Actor, filter BalanceListFilter) (Page[Balance], error)
	ListLots(ctx context.Context, actor Actor, productID string) ([]Lot, error)
	Receive(ctx context.Context, actor Actor, input ReceiveInput) (Movement, Balance, error)
	Pick(ctx context.Context, actor Actor, input PickInput) ([]Movement, error)
	Transfer(ctx context.Context, actor Actor, input TransferInput) (Movement, Balance, Balance, error)
	Adjust(ctx context.Context, actor Actor, input AdjustInput) (Movement, Balance, error)
	ListMovements(ctx context.Context, actor Actor, filter MovementListFilter) (Page[Movement], error)
	GetMovement(ctx context.Context, actor Actor, id string) (Movement, error)
}

type service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &service{repository: repository}
}

func canReceiveOrPick(role string) bool {
	switch role {
	case auth.RoleAdmin, auth.RoleWarehouseManager, auth.RolePicker:
		return true
	}
	return false
}

func canTransferOrAdjust(role string) bool {
	switch role {
	case auth.RoleAdmin, auth.RoleWarehouseManager:
		return true
	}
	return false
}

// requireWarehouseScope returns "" for admin (unrestricted) or the actor's
// assigned warehouse for every other role. A non-admin with no assignment
// is rejected outright.
func requireWarehouseScope(actor Actor) (string, error) {
	if actor.Role == auth.RoleAdmin {
		return "", nil
	}
	if actor.WarehouseID == nil || strings.TrimSpace(*actor.WarehouseID) == "" {
		return "", ErrForbidden
	}
	return *actor.WarehouseID, nil
}

func isInvalidUUID(value string) bool {
	_, err := uuid.Parse(strings.TrimSpace(value))
	return err != nil
}

func isPositiveDecimal(value string) bool {
	amount, err := decimal.NewFromString(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return amount.GreaterThan(decimal.Zero)
}

func isNonNegativeDecimal(value string) bool {
	amount, err := decimal.NewFromString(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return amount.GreaterThanOrEqual(decimal.Zero)
}

func (s *service) ListBalances(ctx context.Context, actor Actor, filter BalanceListFilter) (Page[Balance], error) {
	warehouseID, err := requireWarehouseScope(actor)
	if err != nil {
		return Page[Balance]{}, err
	}
	if warehouseID != "" {
		if filter.WarehouseID != nil && *filter.WarehouseID != warehouseID {
			return Page[Balance]{}, ErrForbidden
		}
		filter.WarehouseID = &warehouseID
	}
	limit := normalizeLimit(filter.Limit)
	queryFilter := filter
	queryFilter.Limit = limit + 1
	items, err := s.repository.ListBalances(ctx, queryFilter)
	if err != nil {
		return Page[Balance]{}, fmt.Errorf("list balances: %w", err)
	}
	return balancePage(items, limit), nil
}

func (s *service) ListLots(ctx context.Context, actor Actor, productID string) ([]Lot, error) {
	warehouseID, err := requireWarehouseScope(actor)
	if err != nil {
		return nil, err
	}
	if isInvalidUUID(productID) {
		return nil, ErrValidation
	}
	var warehouseFilter *string
	if warehouseID != "" {
		warehouseFilter = &warehouseID
	}
	lots, err := s.repository.ListLots(ctx, productID, warehouseFilter)
	if err != nil {
		return nil, fmt.Errorf("list lots: %w", err)
	}
	return lots, nil
}

func (s *service) Receive(ctx context.Context, actor Actor, input ReceiveInput) (Movement, Balance, error) {
	if !canReceiveOrPick(actor.Role) {
		return Movement{}, Balance{}, ErrForbidden
	}
	warehouseID, err := requireWarehouseScope(actor)
	if err != nil {
		return Movement{}, Balance{}, err
	}
	if isInvalidUUID(input.LocationID) || isInvalidUUID(input.ProductID) {
		return Movement{}, Balance{}, ErrValidation
	}
	if !isPositiveDecimal(input.Quantity) {
		return Movement{}, Balance{}, ErrValidation
	}
	if !isNonNegativeDecimal(input.UnitCost) {
		return Movement{}, Balance{}, ErrValidation
	}

	var actorWarehouseID *string
	if warehouseID != "" {
		actorWarehouseID = &warehouseID
	}
	movement, balance, err := s.repository.Receive(ctx, actor.ID, actorWarehouseID, input)
	if err != nil {
		return Movement{}, Balance{}, err
	}
	return movement, balance, nil
}

func (s *service) Pick(ctx context.Context, actor Actor, input PickInput) ([]Movement, error) {
	if !canReceiveOrPick(actor.Role) {
		return nil, ErrForbidden
	}
	warehouseID, err := requireWarehouseScope(actor)
	if err != nil {
		return nil, err
	}
	if isInvalidUUID(input.LocationID) || isInvalidUUID(input.ProductID) {
		return nil, ErrValidation
	}
	if !isPositiveDecimal(input.Quantity) {
		return nil, ErrValidation
	}
	if input.LotID.Set && input.LotID.Value != nil && isInvalidUUID(*input.LotID.Value) {
		return nil, ErrValidation
	}

	var actorWarehouseID *string
	if warehouseID != "" {
		actorWarehouseID = &warehouseID
	}
	movements, err := s.repository.Pick(ctx, actor.ID, actorWarehouseID, input)
	if err != nil {
		return nil, err
	}
	return movements, nil
}

func (s *service) Transfer(ctx context.Context, actor Actor, input TransferInput) (Movement, Balance, Balance, error) {
	if !canTransferOrAdjust(actor.Role) {
		return Movement{}, Balance{}, Balance{}, ErrForbidden
	}
	warehouseID, err := requireWarehouseScope(actor)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}
	if isInvalidUUID(input.ProductID) || isInvalidUUID(input.FromLocationID) || isInvalidUUID(input.ToLocationID) {
		return Movement{}, Balance{}, Balance{}, ErrValidation
	}
	if input.FromLocationID == input.ToLocationID {
		return Movement{}, Balance{}, Balance{}, ErrSameLocationTransfer
	}
	if !isPositiveDecimal(input.Quantity) {
		return Movement{}, Balance{}, Balance{}, ErrValidation
	}
	if input.LotID.Set && input.LotID.Value != nil && isInvalidUUID(*input.LotID.Value) {
		return Movement{}, Balance{}, Balance{}, ErrValidation
	}

	var actorWarehouseID *string
	if warehouseID != "" {
		actorWarehouseID = &warehouseID
	}
	movement, source, destination, err := s.repository.Transfer(ctx, actor.ID, actorWarehouseID, input)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}
	return movement, source, destination, nil
}

func (s *service) Adjust(ctx context.Context, actor Actor, input AdjustInput) (Movement, Balance, error) {
	if !canTransferOrAdjust(actor.Role) {
		return Movement{}, Balance{}, ErrForbidden
	}
	warehouseID, err := requireWarehouseScope(actor)
	if err != nil {
		return Movement{}, Balance{}, err
	}
	if isInvalidUUID(input.LocationID) || isInvalidUUID(input.ProductID) {
		return Movement{}, Balance{}, ErrValidation
	}
	if input.Direction != "increase" && input.Direction != "decrease" {
		return Movement{}, Balance{}, ErrValidation
	}
	if !isPositiveDecimal(input.Quantity) {
		return Movement{}, Balance{}, ErrValidation
	}
	if input.LotID.Set && input.LotID.Value != nil && isInvalidUUID(*input.LotID.Value) {
		return Movement{}, Balance{}, ErrValidation
	}

	var actorWarehouseID *string
	if warehouseID != "" {
		actorWarehouseID = &warehouseID
	}
	movement, balance, err := s.repository.Adjust(ctx, actor.ID, actorWarehouseID, input)
	if err != nil {
		return Movement{}, Balance{}, err
	}
	return movement, balance, nil
}

func (s *service) ListMovements(ctx context.Context, actor Actor, filter MovementListFilter) (Page[Movement], error) {
	if _, err := requireWarehouseScope(actor); err != nil {
		return Page[Movement]{}, err
	}
	limit := normalizeLimit(filter.Limit)
	queryFilter := filter
	queryFilter.Limit = limit + 1
	items, err := s.repository.ListMovements(ctx, queryFilter)
	if err != nil {
		return Page[Movement]{}, fmt.Errorf("list movements: %w", err)
	}
	return movementPage(items, limit), nil
}

func (s *service) GetMovement(ctx context.Context, actor Actor, id string) (Movement, error) {
	if _, err := requireWarehouseScope(actor); err != nil {
		return Movement{}, err
	}
	if isInvalidUUID(id) {
		return Movement{}, ErrValidation
	}
	movement, err := s.repository.GetMovement(ctx, id)
	if err != nil {
		return Movement{}, err
	}
	return movement, nil
}
