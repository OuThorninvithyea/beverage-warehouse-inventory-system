package catalog

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

var (
	ErrForbidden              = errors.New("catalog access is forbidden")
	ErrValidation             = errors.New("catalog data is invalid")
	ErrInvalidID              = errors.New("resource id is invalid")
	ErrInvalidBarcode         = errors.New("barcode is invalid")
	ErrCategoryNotFound       = errors.New("category not found")
	ErrProductNotFound        = errors.New("product not found")
	ErrCategoryNameConflict   = errors.New("category name already exists")
	ErrCategoryInUse          = errors.New("category is in use")
	ErrCategoryCycle          = errors.New("category hierarchy cycle")
	ErrProductSKUConflict     = errors.New("product sku already exists")
	ErrProductBarcodeConflict = errors.New("product barcode already exists")
)

type Repository interface {
	ListCategories(context.Context, ListFilter) ([]Category, error)
	GetCategory(context.Context, string) (Category, error)
	CreateCategory(context.Context, CategoryInput) (Category, error)
	UpdateCategory(context.Context, string, CategoryInput) (Category, error)
	DeactivateCategory(context.Context, string) error
	ListProducts(context.Context, ListFilter) ([]Product, error)
	GetProduct(context.Context, string) (Product, error)
	GetProductByBarcode(context.Context, string) (Product, error)
	CreateProduct(context.Context, ProductInput) (Product, error)
	UpdateProduct(context.Context, string, ProductInput) (Product, error)
	DeactivateProduct(context.Context, string) error
}

type Service interface {
	ListCategories(context.Context, Actor, ListFilter) (Page[Category], error)
	GetCategory(context.Context, Actor, string) (Category, error)
	CreateCategory(context.Context, Actor, CategoryInput) (Category, error)
	UpdateCategory(context.Context, Actor, string, CategoryInput) (Category, error)
	DeactivateCategory(context.Context, Actor, string) error
	ListProducts(context.Context, Actor, ListFilter) (Page[Product], error)
	GetProduct(context.Context, Actor, string) (Product, error)
	GetProductByBarcode(context.Context, Actor, string) (Product, error)
	CreateProduct(context.Context, Actor, ProductInput) (Product, error)
	UpdateProduct(context.Context, Actor, string, ProductInput) (Product, error)
	DeactivateProduct(context.Context, Actor, string) error
}

type service struct {
	repository Repository
}

func NewService(repository Repository) *service {
	return &service{repository: repository}
}

func (service *service) ListCategories(
	ctx context.Context,
	_ Actor,
	filter ListFilter,
) (Page[Category], error) {
	if filter.ParentID != nil {
		parentID := strings.TrimSpace(*filter.ParentID)
		if !validUUID(parentID) {
			return Page[Category]{}, ErrInvalidID
		}
		filter.ParentID = &parentID
	}
	requestedLimit := normalizeLimit(filter.Limit)
	filter.Limit = requestedLimit + 1
	filter.Search = strings.TrimSpace(filter.Search)

	items, err := service.repository.ListCategories(ctx, filter)
	if err != nil {
		return Page[Category]{}, err
	}
	return categoryPage(items, requestedLimit), nil
}

func (service *service) GetCategory(
	ctx context.Context,
	_ Actor,
	id string,
) (Category, error) {
	if !validUUID(id) {
		return Category{}, ErrInvalidID
	}
	return service.repository.GetCategory(ctx, id)
}

func (service *service) CreateCategory(
	ctx context.Context,
	actor Actor,
	input CategoryInput,
) (Category, error) {
	if !canMutateCatalog(actor.Role) {
		return Category{}, ErrForbidden
	}
	normalized, err := normalizeCategoryInput(input, true)
	if err != nil {
		return Category{}, err
	}
	return service.repository.CreateCategory(ctx, normalized)
}

func (service *service) UpdateCategory(
	ctx context.Context,
	actor Actor,
	id string,
	input CategoryInput,
) (Category, error) {
	if !canMutateCatalog(actor.Role) {
		return Category{}, ErrForbidden
	}
	if !validUUID(id) {
		return Category{}, ErrInvalidID
	}
	normalized, err := normalizeCategoryInput(input, false)
	if err != nil {
		return Category{}, err
	}
	return service.repository.UpdateCategory(ctx, id, normalized)
}

func (service *service) DeactivateCategory(
	ctx context.Context,
	actor Actor,
	id string,
) error {
	if !canMutateCatalog(actor.Role) {
		return ErrForbidden
	}
	if !validUUID(id) {
		return ErrInvalidID
	}
	return service.repository.DeactivateCategory(ctx, id)
}

func normalizeCategoryInput(input CategoryInput, defaults bool) (CategoryInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return CategoryInput{}, ErrValidation
	}
	if defaults && !input.ParentID.Set {
		input.ParentID = OptionalString{Set: true}
	}
	if input.ParentID.Set && input.ParentID.Value != nil {
		parentID := strings.TrimSpace(*input.ParentID.Value)
		if !validUUID(parentID) {
			return CategoryInput{}, ErrInvalidID
		}
		input.ParentID.Value = &parentID
	}
	if defaults && input.IsActive == nil {
		input.IsActive = boolPointer(true)
	}
	return input, nil
}

func canMutateCatalog(role string) bool {
	return role == auth.RoleAdmin || role == auth.RoleWarehouseManager
}

func validUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

func boolPointer(value bool) *bool {
	return &value
}
