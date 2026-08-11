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

func NewService(repository Repository) Service {
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

func (service *service) ListProducts(
	ctx context.Context,
	_ Actor,
	filter ListFilter,
) (Page[Product], error) {
	if filter.CategoryID != nil {
		categoryID := strings.TrimSpace(*filter.CategoryID)
		if !validUUID(categoryID) {
			return Page[Product]{}, ErrInvalidID
		}
		filter.CategoryID = &categoryID
	}
	requestedLimit := normalizeLimit(filter.Limit)
	filter.Limit = requestedLimit + 1
	filter.Search = strings.TrimSpace(filter.Search)

	items, err := service.repository.ListProducts(ctx, filter)
	if err != nil {
		return Page[Product]{}, err
	}
	return productPage(items, requestedLimit), nil
}

func (service *service) GetProduct(
	ctx context.Context,
	_ Actor,
	id string,
) (Product, error) {
	if !validUUID(id) {
		return Product{}, ErrInvalidID
	}
	return service.repository.GetProduct(ctx, id)
}

func (service *service) GetProductByBarcode(
	ctx context.Context,
	_ Actor,
	barcode string,
) (Product, error) {
	barcode = strings.TrimSpace(barcode)
	if !ValidateBarcode(barcode) {
		return Product{}, ErrInvalidBarcode
	}
	return service.repository.GetProductByBarcode(ctx, barcode)
}

func (service *service) CreateProduct(
	ctx context.Context,
	actor Actor,
	input ProductInput,
) (Product, error) {
	if !canMutateCatalog(actor.Role) {
		return Product{}, ErrForbidden
	}
	normalized, err := normalizeProductInput(input, true)
	if err != nil {
		return Product{}, err
	}
	return service.repository.CreateProduct(ctx, normalized)
}

func (service *service) UpdateProduct(
	ctx context.Context,
	actor Actor,
	id string,
	input ProductInput,
) (Product, error) {
	if !canMutateCatalog(actor.Role) {
		return Product{}, ErrForbidden
	}
	if !validUUID(id) {
		return Product{}, ErrInvalidID
	}
	normalized, err := normalizeProductInput(input, false)
	if err != nil {
		return Product{}, err
	}
	return service.repository.UpdateProduct(ctx, id, normalized)
}

func (service *service) DeactivateProduct(
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
	return service.repository.DeactivateProduct(ctx, id)
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

func normalizeProductInput(input ProductInput, defaults bool) (ProductInput, error) {
	input.SKU = strings.ToUpper(strings.TrimSpace(input.SKU))
	input.Name = strings.TrimSpace(input.Name)
	input.Unit = strings.ToLower(strings.TrimSpace(input.Unit))
	if input.SKU == "" || input.Name == "" || input.Unit == "" {
		return ProductInput{}, ErrValidation
	}

	if defaults && !input.CategoryID.Set {
		input.CategoryID = OptionalString{Set: true}
	}
	if input.CategoryID.Set && input.CategoryID.Value != nil {
		categoryID := strings.TrimSpace(*input.CategoryID.Value)
		if !validUUID(categoryID) {
			return ProductInput{}, ErrInvalidID
		}
		input.CategoryID.Value = &categoryID
	}

	if defaults && !input.Barcode.Set {
		input.Barcode = OptionalString{Set: true}
	}
	if input.Barcode.Set && input.Barcode.Value != nil {
		barcode := strings.TrimSpace(*input.Barcode.Value)
		if !ValidateBarcode(barcode) {
			return ProductInput{}, ErrInvalidBarcode
		}
		input.Barcode.Value = &barcode
	}

	if defaults {
		if input.IsLotTracked == nil {
			input.IsLotTracked = boolPointer(true)
		}
		if input.IsActive == nil {
			input.IsActive = boolPointer(true)
		}
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
