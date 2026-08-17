package users

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

var (
	ErrForbidden                 = errors.New("user management access is forbidden")
	ErrValidation                = errors.New("user data is invalid")
	ErrInvalidID                 = errors.New("resource id is invalid")
	ErrSelfDeactivationForbidden = errors.New("an administrator cannot deactivate or demote themselves")
)

type Service interface {
	CreateUser(ctx context.Context, actor Actor, input UserCreateInput) (User, error)
	ListUsers(ctx context.Context, actor Actor, filter ListFilter) (Page[User], error)
	GetUser(ctx context.Context, actor Actor, id string) (User, error)
	UpdateUser(ctx context.Context, actor Actor, id string, input UserUpdateInput) (User, error)
	DeactivateUser(ctx context.Context, actor Actor, id string) error
	ResetPassword(ctx context.Context, actor Actor, id string, input PasswordResetInput) error
}

type service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &service{repository: repository}
}

func validUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

var knownRoles = map[string]struct{}{
	auth.RoleAdmin:            {},
	auth.RoleWarehouseManager: {},
	auth.RolePicker:           {},
	auth.RoleViewer:           {},
}

func isKnownRole(role string) bool {
	_, ok := knownRoles[role]
	return ok
}

func canManageUsers(role string) bool {
	return role == auth.RoleAdmin
}

func (service *service) CreateUser(ctx context.Context, actor Actor, input UserCreateInput) (User, error) {
	if !canManageUsers(actor.Role) {
		return User{}, ErrForbidden
	}

	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.FullName = strings.TrimSpace(input.FullName)
	if input.Email == "" || input.FullName == "" {
		return User{}, ErrValidation
	}
	if !isKnownRole(input.Role) {
		return User{}, ErrInvalidRole
	}
	if input.WarehouseID.Set && input.WarehouseID.Value != nil && !validUUID(*input.WarehouseID.Value) {
		return User{}, ErrInvalidID
	}
	if len(input.Password) < 12 {
		return User{}, ErrValidation
	}
	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return User{}, ErrValidation
	}

	return service.repository.CreateUser(ctx, input, passwordHash)
}

func (service *service) ListUsers(ctx context.Context, actor Actor, filter ListFilter) (Page[User], error) {
	if !canManageUsers(actor.Role) {
		return Page[User]{}, ErrForbidden
	}
	if filter.Role != "" && !isKnownRole(filter.Role) {
		return Page[User]{}, ErrInvalidRole
	}
	if filter.WarehouseID != nil {
		warehouseID := strings.TrimSpace(*filter.WarehouseID)
		if !validUUID(warehouseID) {
			return Page[User]{}, ErrInvalidID
		}
		filter.WarehouseID = &warehouseID
	}
	requestedLimit := normalizeLimit(filter.Limit)
	filter.Limit = requestedLimit + 1
	filter.Search = strings.TrimSpace(filter.Search)

	items, err := service.repository.ListUsers(ctx, filter)
	if err != nil {
		return Page[User]{}, err
	}
	return userPage(items, requestedLimit), nil
}

func (service *service) GetUser(ctx context.Context, actor Actor, id string) (User, error) {
	if !canManageUsers(actor.Role) {
		return User{}, ErrForbidden
	}
	if !validUUID(id) {
		return User{}, ErrInvalidID
	}
	return service.repository.GetUser(ctx, id)
}

func (service *service) UpdateUser(ctx context.Context, actor Actor, id string, input UserUpdateInput) (User, error) {
	panic("not implemented until Task 11")
}

func (service *service) DeactivateUser(ctx context.Context, actor Actor, id string) error {
	panic("not implemented until Task 11")
}

func (service *service) ResetPassword(ctx context.Context, actor Actor, id string, input PasswordResetInput) error {
	panic("not implemented until Task 11")
}
