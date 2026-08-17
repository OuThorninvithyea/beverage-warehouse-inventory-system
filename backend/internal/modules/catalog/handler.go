package catalog

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/platform/httpx"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

type categoryWriteRequest struct {
	Name     string         `json:"name"`
	ParentID OptionalString `json:"parent_id"`
	IsActive *bool          `json:"is_active"`
}

type productWriteRequest struct {
	CategoryID   OptionalString `json:"category_id"`
	SKU          string         `json:"sku"`
	Barcode      OptionalString `json:"barcode"`
	Name         string         `json:"name"`
	Unit         string         `json:"unit"`
	IsLotTracked *bool          `json:"is_lot_tracked"`
	IsActive     *bool          `json:"is_active"`
}

func (h *Handler) ListCategories(c fiber.Ctx) error {
	filter, err := parseCatalogListFilter(c, true)
	if err != nil {
		return err
	}
	page, err := h.service.ListCategories(c.Context(), catalogActor(c), filter)
	if err != nil {
		return catalogHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(page))
}

func (h *Handler) GetCategory(c fiber.Ctx) error {
	category, err := h.service.GetCategory(c.Context(), catalogActor(c), c.Params("category_id"))
	if err != nil {
		return catalogHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(category))
}

func (h *Handler) CreateCategory(c fiber.Ctx) error {
	input, err := bindCategoryRequest(c)
	if err != nil {
		return err
	}
	category, err := h.service.CreateCategory(c.Context(), catalogActor(c), input)
	if err != nil {
		return catalogHTTPError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(category))
}

func (h *Handler) UpdateCategory(c fiber.Ctx) error {
	input, err := bindCategoryRequest(c)
	if err != nil {
		return err
	}
	category, err := h.service.UpdateCategory(c.Context(), catalogActor(c), c.Params("category_id"), input)
	if err != nil {
		return catalogHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(category))
}

func (h *Handler) DeactivateCategory(c fiber.Ctx) error {
	if err := h.service.DeactivateCategory(c.Context(), catalogActor(c), c.Params("category_id")); err != nil {
		return catalogHTTPError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) ListProducts(c fiber.Ctx) error {
	filter, err := parseCatalogListFilter(c, false)
	if err != nil {
		return err
	}
	page, err := h.service.ListProducts(c.Context(), catalogActor(c), filter)
	if err != nil {
		return catalogHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(page))
}

func (h *Handler) GetProduct(c fiber.Ctx) error {
	product, err := h.service.GetProduct(c.Context(), catalogActor(c), c.Params("product_id"))
	if err != nil {
		return catalogHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(product))
}

func (h *Handler) GetProductByBarcode(c fiber.Ctx) error {
	product, err := h.service.GetProductByBarcode(c.Context(), catalogActor(c), c.Params("barcode"))
	if err != nil {
		return catalogHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(product))
}

func (h *Handler) CreateProduct(c fiber.Ctx) error {
	input, err := bindProductRequest(c)
	if err != nil {
		return err
	}
	product, err := h.service.CreateProduct(c.Context(), catalogActor(c), input)
	if err != nil {
		return catalogHTTPError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(httpx.Success(product))
}

func (h *Handler) UpdateProduct(c fiber.Ctx) error {
	input, err := bindProductRequest(c)
	if err != nil {
		return err
	}
	product, err := h.service.UpdateProduct(c.Context(), catalogActor(c), c.Params("product_id"), input)
	if err != nil {
		return catalogHTTPError(err)
	}
	return c.Status(fiber.StatusOK).JSON(httpx.Success(product))
}

func (h *Handler) DeactivateProduct(c fiber.Ctx) error {
	if err := h.service.DeactivateProduct(c.Context(), catalogActor(c), c.Params("product_id")); err != nil {
		return catalogHTTPError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func bindCategoryRequest(c fiber.Ctx) (CategoryInput, error) {
	var request categoryWriteRequest
	if err := c.Bind().Body(&request); err != nil {
		return CategoryInput{}, invalidCatalogRequest("invalid JSON request", err)
	}
	return CategoryInput{
		Name: request.Name, ParentID: request.ParentID, IsActive: request.IsActive,
	}, nil
}

func bindProductRequest(c fiber.Ctx) (ProductInput, error) {
	var request productWriteRequest
	if err := c.Bind().Body(&request); err != nil {
		return ProductInput{}, invalidCatalogRequest("invalid JSON request", err)
	}
	return ProductInput{
		CategoryID:   request.CategoryID,
		SKU:          request.SKU,
		Barcode:      request.Barcode,
		Name:         request.Name,
		Unit:         request.Unit,
		IsLotTracked: request.IsLotTracked,
		IsActive:     request.IsActive,
	}, nil
}

func parseCatalogListFilter(c fiber.Ctx, categories bool) (ListFilter, error) {
	filter := ListFilter{Search: c.Query("search")}

	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return ListFilter{}, invalidCatalogRequest("limit must be an integer from 1 to 100", nil)
		}
		filter.Limit = limit
	}
	if raw := c.Query("after"); raw != "" {
		cursor, err := DecodeCursor(raw)
		if err != nil {
			return ListFilter{}, invalidCatalogRequest("after cursor is invalid", err)
		}
		filter.After = &cursor
	}
	active, err := catalogOptionalBool(c.Query("is_active"))
	if err != nil {
		return ListFilter{}, invalidCatalogRequest("is_active must be true or false", err)
	}
	filter.IsActive = active

	filterName := "category_id"
	if categories {
		filterName = "parent_id"
	}
	if raw := strings.TrimSpace(c.Query(filterName)); raw != "" {
		if _, err := uuid.Parse(raw); err != nil {
			return ListFilter{}, invalidCatalogRequest(filterName+" must be a UUID", err)
		}
		if categories {
			filter.ParentID = &raw
		} else {
			filter.CategoryID = &raw
		}
	}
	return filter, nil
}

func catalogOptionalBool(value string) (*bool, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func catalogActor(c fiber.Ctx) Actor {
	claims, _ := auth.ClaimsFromContext(c)
	if claims == nil {
		return Actor{}
	}
	return Actor{Role: claims.Role}
}

func invalidCatalogRequest(message string, err error) error {
	return httpx.NewError(fiber.StatusBadRequest, "INVALID_REQUEST", message, err)
}

func catalogHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidID), errors.Is(err, ErrInvalidCursor):
		return invalidCatalogRequest("a resource identifier, filter, or cursor is invalid", err)
	case errors.Is(err, ErrForbidden):
		return httpx.NewError(fiber.StatusForbidden, "FORBIDDEN", "your role does not allow this catalog action", err)
	case errors.Is(err, ErrCategoryNotFound):
		return httpx.NewError(fiber.StatusNotFound, "CATEGORY_NOT_FOUND", "category was not found", err)
	case errors.Is(err, ErrProductNotFound):
		return httpx.NewError(fiber.StatusNotFound, "PRODUCT_NOT_FOUND", "product was not found", err)
	case errors.Is(err, ErrCategoryNameConflict):
		return httpx.NewError(fiber.StatusConflict, "CATEGORY_NAME_CONFLICT", "category name already exists", err)
	case errors.Is(err, ErrCategoryInUse):
		return httpx.NewError(fiber.StatusConflict, "CATEGORY_IN_USE", "category is referenced by an active category or product", err)
	case errors.Is(err, ErrProductSKUConflict):
		return httpx.NewError(fiber.StatusConflict, "PRODUCT_SKU_CONFLICT", "product SKU already exists", err)
	case errors.Is(err, ErrProductBarcodeConflict):
		return httpx.NewError(fiber.StatusConflict, "PRODUCT_BARCODE_CONFLICT", "product barcode already exists", err)
	case errors.Is(err, ErrInvalidBarcode):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "INVALID_BARCODE", "barcode must be a valid UPC-A or EAN-13 value", err)
	case errors.Is(err, ErrCategoryCycle):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "CATEGORY_CYCLE", "category parent would create a hierarchy cycle", err)
	case errors.Is(err, ErrValidation):
		return httpx.NewError(fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "catalog data is invalid", err)
	default:
		return httpx.NewError(fiber.StatusInternalServerError, "CATALOG_OPERATION_FAILED", "catalog operation could not be completed", err)
	}
}
