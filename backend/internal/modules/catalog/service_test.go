package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

const (
	testCategoryID = "7e5d55b1-6356-492b-8296-2b981867fcf2"
	testParentID   = "8e5d55b1-6356-492b-8296-2b981867fcf2"
	testProductID  = "9e5d55b1-6356-492b-8296-2b981867fcf2"
	testBarcode    = "4006381333931"
)

type fakeRepository struct {
	categories []Category
	category   Category
	products   []Product
	product    Product
	err        error

	categoryInput  CategoryInput
	productInput   ProductInput
	filter         ListFilter
	categoryID     string
	productID      string
	barcode        string
	categoryWrites int
	productWrites  int
}

func (repository *fakeRepository) ListCategories(_ context.Context, filter ListFilter) ([]Category, error) {
	repository.filter = filter
	return repository.categories, repository.err
}

func (repository *fakeRepository) GetCategory(_ context.Context, id string) (Category, error) {
	repository.categoryID = id
	return repository.category, repository.err
}

func (repository *fakeRepository) CreateCategory(_ context.Context, input CategoryInput) (Category, error) {
	repository.categoryWrites++
	repository.categoryInput = input
	return repository.category, repository.err
}

func (repository *fakeRepository) UpdateCategory(_ context.Context, id string, input CategoryInput) (Category, error) {
	repository.categoryWrites++
	repository.categoryID = id
	repository.categoryInput = input
	return repository.category, repository.err
}

func (repository *fakeRepository) DeactivateCategory(_ context.Context, id string) error {
	repository.categoryWrites++
	repository.categoryID = id
	return repository.err
}

func (repository *fakeRepository) ListProducts(_ context.Context, filter ListFilter) ([]Product, error) {
	repository.filter = filter
	return repository.products, repository.err
}

func (repository *fakeRepository) GetProduct(_ context.Context, id string) (Product, error) {
	repository.productID = id
	return repository.product, repository.err
}

func (repository *fakeRepository) GetProductByBarcode(_ context.Context, barcode string) (Product, error) {
	repository.barcode = barcode
	return repository.product, repository.err
}

func (repository *fakeRepository) CreateProduct(_ context.Context, input ProductInput) (Product, error) {
	repository.productWrites++
	repository.productInput = input
	return repository.product, repository.err
}

func (repository *fakeRepository) UpdateProduct(_ context.Context, id string, input ProductInput) (Product, error) {
	repository.productWrites++
	repository.productID = id
	repository.productInput = input
	return repository.product, repository.err
}

func (repository *fakeRepository) DeactivateProduct(_ context.Context, id string) error {
	repository.productWrites++
	repository.productID = id
	return repository.err
}

func TestCategoryMutationRoles(t *testing.T) {
	for _, role := range []string{auth.RoleAdmin, auth.RoleWarehouseManager} {
		t.Run(role+" allowed", func(t *testing.T) {
			repository := &fakeRepository{category: Category{ID: testCategoryID}}
			service := NewService(repository)
			if _, err := service.CreateCategory(context.Background(), Actor{Role: role}, CategoryInput{Name: "Soda"}); err != nil {
				t.Fatalf("CreateCategory() error = %v", err)
			}
			if repository.categoryWrites != 1 {
				t.Fatalf("repository writes = %d, want 1", repository.categoryWrites)
			}
		})
	}

	for _, role := range []string{auth.RolePicker, auth.RoleViewer} {
		t.Run(role+" forbidden", func(t *testing.T) {
			repository := &fakeRepository{}
			service := NewService(repository)
			if _, err := service.CreateCategory(context.Background(), Actor{Role: role}, CategoryInput{Name: "Soda"}); !errors.Is(err, ErrForbidden) {
				t.Fatalf("CreateCategory() error = %v, want ErrForbidden", err)
			}
			if repository.categoryWrites != 0 {
				t.Fatalf("repository writes = %d, want 0", repository.categoryWrites)
			}
		})
	}
}

func TestCreateCategoryNormalizesDefaultsAndParent(t *testing.T) {
	parent := "  " + testParentID + "  "
	repository := &fakeRepository{category: Category{ID: testCategoryID}}
	service := NewService(repository)

	_, err := service.CreateCategory(context.Background(), Actor{Role: auth.RoleAdmin}, CategoryInput{
		Name:     "  Soft Drinks  ",
		ParentID: OptionalString{Set: true, Value: &parent},
	})
	if err != nil {
		t.Fatalf("CreateCategory() error = %v", err)
	}
	if repository.categoryInput.Name != "Soft Drinks" {
		t.Fatalf("Name = %q, want trimmed name", repository.categoryInput.Name)
	}
	if !repository.categoryInput.ParentID.Set || repository.categoryInput.ParentID.Value == nil || *repository.categoryInput.ParentID.Value != testParentID {
		t.Fatalf("ParentID = %#v, want normalized UUID", repository.categoryInput.ParentID)
	}
	if repository.categoryInput.IsActive == nil || !*repository.categoryInput.IsActive {
		t.Fatalf("IsActive = %#v, want true default", repository.categoryInput.IsActive)
	}
}

func TestCreateCategoryWithoutParentWritesExplicitNull(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	if _, err := service.CreateCategory(context.Background(), Actor{Role: auth.RoleAdmin}, CategoryInput{Name: "Water"}); err != nil {
		t.Fatalf("CreateCategory() error = %v", err)
	}
	if !repository.categoryInput.ParentID.Set || repository.categoryInput.ParentID.Value != nil {
		t.Fatalf("ParentID = %#v, want explicit null", repository.categoryInput.ParentID)
	}
}

func TestCategoryValidationStopsRepositoryWrites(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		input  CategoryInput
		update bool
	}{
		{name: "blank name", input: CategoryInput{Name: "   "}},
		{name: "invalid parent", input: CategoryInput{Name: "Water", ParentID: OptionalString{Set: true, Value: stringPointer("bad-id")}}},
		{name: "invalid resource id", id: "bad-id", input: CategoryInput{Name: "Water"}, update: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeRepository{}
			service := NewService(repository)
			var err error
			if test.update {
				_, err = service.UpdateCategory(context.Background(), Actor{Role: auth.RoleAdmin}, test.id, test.input)
			} else {
				_, err = service.CreateCategory(context.Background(), Actor{Role: auth.RoleAdmin}, test.input)
			}
			if !errors.Is(err, ErrValidation) && !errors.Is(err, ErrInvalidID) {
				t.Fatalf("error = %v, want validation or invalid ID", err)
			}
			if repository.categoryWrites != 0 {
				t.Fatalf("repository writes = %d, want 0", repository.categoryWrites)
			}
		})
	}
}

func TestUpdateCategoryPreservesOmittedParent(t *testing.T) {
	repository := &fakeRepository{category: Category{ID: testCategoryID}}
	service := NewService(repository)
	if _, err := service.UpdateCategory(context.Background(), Actor{Role: auth.RoleWarehouseManager}, testCategoryID, CategoryInput{Name: "Juice"}); err != nil {
		t.Fatalf("UpdateCategory() error = %v", err)
	}
	if repository.categoryInput.ParentID.Set {
		t.Fatalf("ParentID.Set = true, want omitted parent preserved")
	}
	if repository.categoryInput.IsActive != nil {
		t.Fatalf("IsActive = %#v, want omitted active preserved", repository.categoryInput.IsActive)
	}
}

func TestListCategoriesNormalizesPagination(t *testing.T) {
	createdAt := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	repository := &fakeRepository{categories: []Category{
		{ID: testCategoryID, CreatedAt: createdAt},
		{ID: testParentID, CreatedAt: createdAt.Add(-time.Minute)},
	}}
	service := NewService(repository)

	page, err := service.ListCategories(context.Background(), Actor{Role: auth.RoleViewer}, ListFilter{Limit: 1, Search: "  soft  "})
	if err != nil {
		t.Fatalf("ListCategories() error = %v", err)
	}
	if repository.filter.Limit != 2 || repository.filter.Search != "soft" {
		t.Fatalf("repository filter = %#v, want limit 2 and trimmed search", repository.filter)
	}
	if len(page.Items) != 1 || !page.Page.HasMore || page.Page.NextCursor == nil {
		t.Fatalf("page = %#v, want one item and next cursor", page)
	}
}

func TestListCategoriesReturnsEmptyArrayAndPreservesErrors(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	page, err := service.ListCategories(context.Background(), Actor{Role: auth.RolePicker}, ListFilter{})
	if err != nil {
		t.Fatalf("ListCategories() error = %v", err)
	}
	if page.Items == nil || len(page.Items) != 0 {
		t.Fatalf("items = %#v, want non-nil empty slice", page.Items)
	}

	repository.err = ErrCategoryNameConflict
	_, err = service.CreateCategory(context.Background(), Actor{Role: auth.RoleAdmin}, CategoryInput{Name: "Water"})
	if !errors.Is(err, ErrCategoryNameConflict) {
		t.Fatalf("CreateCategory() error = %v, want ErrCategoryNameConflict", err)
	}
}

func TestGetAndDeactivateCategoryValidateIDAndForward(t *testing.T) {
	repository := &fakeRepository{category: Category{ID: testCategoryID}}
	service := NewService(repository)
	if _, err := service.GetCategory(context.Background(), Actor{Role: auth.RoleViewer}, testCategoryID); err != nil {
		t.Fatalf("GetCategory() error = %v", err)
	}
	if repository.categoryID != testCategoryID {
		t.Fatalf("category ID = %q, want %q", repository.categoryID, testCategoryID)
	}
	if err := service.DeactivateCategory(context.Background(), Actor{Role: auth.RoleWarehouseManager}, testCategoryID); err != nil {
		t.Fatalf("DeactivateCategory() error = %v", err)
	}
	if repository.categoryWrites != 1 {
		t.Fatalf("category writes = %d, want 1", repository.categoryWrites)
	}
	if _, err := service.GetCategory(context.Background(), Actor{Role: auth.RoleViewer}, "bad-id"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("GetCategory() error = %v, want ErrInvalidID", err)
	}
}

func stringPointer(value string) *string { return &value }

func validProductInput() ProductInput {
	return ProductInput{SKU: "COKE-330", Name: "Coca-Cola 330 ml", Unit: "case"}
}

func TestProductMutationRoles(t *testing.T) {
	for _, role := range []string{auth.RoleAdmin, auth.RoleWarehouseManager} {
		t.Run(role+" allowed", func(t *testing.T) {
			repository := &fakeRepository{product: Product{ID: testProductID}}
			service := NewService(repository)
			if _, err := service.CreateProduct(context.Background(), Actor{Role: role}, validProductInput()); err != nil {
				t.Fatalf("CreateProduct() error = %v", err)
			}
			if repository.productWrites != 1 {
				t.Fatalf("repository writes = %d, want 1", repository.productWrites)
			}
		})
	}

	for _, role := range []string{auth.RolePicker, auth.RoleViewer} {
		t.Run(role+" forbidden", func(t *testing.T) {
			repository := &fakeRepository{}
			service := NewService(repository)
			if _, err := service.CreateProduct(context.Background(), Actor{Role: role}, validProductInput()); !errors.Is(err, ErrForbidden) {
				t.Fatalf("CreateProduct() error = %v, want ErrForbidden", err)
			}
			if repository.productWrites != 0 {
				t.Fatalf("repository writes = %d, want 0", repository.productWrites)
			}
		})
	}
}

func TestCreateProductNormalizesDefaultsAndOptionalValues(t *testing.T) {
	categoryID := "  " + testCategoryID + "  "
	barcode := "  " + testBarcode + "  "
	repository := &fakeRepository{product: Product{ID: testProductID}}
	service := NewService(repository)

	_, err := service.CreateProduct(context.Background(), Actor{Role: auth.RoleAdmin}, ProductInput{
		CategoryID: OptionalString{Set: true, Value: &categoryID},
		SKU:        "  coke-330  ",
		Barcode:    OptionalString{Set: true, Value: &barcode},
		Name:       "  Coca-Cola 330 ml  ",
		Unit:       "  CASE  ",
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}
	input := repository.productInput
	if input.SKU != "COKE-330" || input.Name != "Coca-Cola 330 ml" || input.Unit != "case" {
		t.Fatalf("normalized input = %#v", input)
	}
	if input.CategoryID.Value == nil || *input.CategoryID.Value != testCategoryID {
		t.Fatalf("CategoryID = %#v, want normalized UUID", input.CategoryID)
	}
	if input.Barcode.Value == nil || *input.Barcode.Value != testBarcode {
		t.Fatalf("Barcode = %#v, want normalized barcode", input.Barcode)
	}
	if input.IsLotTracked == nil || !*input.IsLotTracked || input.IsActive == nil || !*input.IsActive {
		t.Fatalf("booleans = %#v %#v, want true defaults", input.IsLotTracked, input.IsActive)
	}
}

func TestCreateProductWithoutOptionalValuesWritesExplicitNull(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	if _, err := service.CreateProduct(context.Background(), Actor{Role: auth.RoleAdmin}, validProductInput()); err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}
	if !repository.productInput.CategoryID.Set || repository.productInput.CategoryID.Value != nil {
		t.Fatalf("CategoryID = %#v, want explicit null", repository.productInput.CategoryID)
	}
	if !repository.productInput.Barcode.Set || repository.productInput.Barcode.Value != nil {
		t.Fatalf("Barcode = %#v, want explicit null", repository.productInput.Barcode)
	}
}

func TestProductValidationStopsRepositoryWrites(t *testing.T) {
	tests := []struct {
		name  string
		input ProductInput
		want  error
	}{
		{name: "blank sku", input: ProductInput{Name: "Water", Unit: "case"}, want: ErrValidation},
		{name: "blank name", input: ProductInput{SKU: "WATER", Unit: "case"}, want: ErrValidation},
		{name: "blank unit", input: ProductInput{SKU: "WATER", Name: "Water"}, want: ErrValidation},
		{name: "invalid category", input: ProductInput{SKU: "WATER", Name: "Water", Unit: "case", CategoryID: OptionalString{Set: true, Value: stringPointer("bad-id")}}, want: ErrInvalidID},
		{name: "invalid barcode", input: ProductInput{SKU: "WATER", Name: "Water", Unit: "case", Barcode: OptionalString{Set: true, Value: stringPointer("123")}}, want: ErrInvalidBarcode},
		{name: "empty barcode", input: ProductInput{SKU: "WATER", Name: "Water", Unit: "case", Barcode: OptionalString{Set: true, Value: stringPointer("  ")}}, want: ErrInvalidBarcode},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeRepository{}
			service := NewService(repository)
			_, err := service.CreateProduct(context.Background(), Actor{Role: auth.RoleAdmin}, test.input)
			if !errors.Is(err, test.want) {
				t.Fatalf("CreateProduct() error = %v, want %v", err, test.want)
			}
			if repository.productWrites != 0 {
				t.Fatalf("repository writes = %d, want 0", repository.productWrites)
			}
		})
	}
}

func TestUpdateProductPreservesOmittedOptionalValues(t *testing.T) {
	repository := &fakeRepository{product: Product{ID: testProductID}}
	service := NewService(repository)
	if _, err := service.UpdateProduct(context.Background(), Actor{Role: auth.RoleWarehouseManager}, testProductID, validProductInput()); err != nil {
		t.Fatalf("UpdateProduct() error = %v", err)
	}
	input := repository.productInput
	if input.CategoryID.Set || input.Barcode.Set || input.IsLotTracked != nil || input.IsActive != nil {
		t.Fatalf("optional fields = %#v, want omitted values preserved", input)
	}
}

func TestUpdateProductPreservesExplicitNullClears(t *testing.T) {
	repository := &fakeRepository{product: Product{ID: testProductID}}
	service := NewService(repository)
	input := validProductInput()
	input.CategoryID = OptionalString{Set: true}
	input.Barcode = OptionalString{Set: true}
	if _, err := service.UpdateProduct(context.Background(), Actor{Role: auth.RoleAdmin}, testProductID, input); err != nil {
		t.Fatalf("UpdateProduct() error = %v", err)
	}
	if !repository.productInput.CategoryID.Set || repository.productInput.CategoryID.Value != nil || !repository.productInput.Barcode.Set || repository.productInput.Barcode.Value != nil {
		t.Fatalf("optional fields = %#v, want explicit null clears", repository.productInput)
	}
}

func TestListProductsNormalizesPaginationAndCategory(t *testing.T) {
	createdAt := time.Date(2026, 8, 11, 9, 0, 0, 0, time.UTC)
	categoryID := "  " + testCategoryID + "  "
	repository := &fakeRepository{products: []Product{
		{ID: testProductID, CreatedAt: createdAt},
		{ID: testParentID, CreatedAt: createdAt.Add(-time.Minute)},
	}}
	service := NewService(repository)

	page, err := service.ListProducts(context.Background(), Actor{Role: auth.RoleViewer}, ListFilter{Limit: 1, Search: "  coke  ", CategoryID: &categoryID})
	if err != nil {
		t.Fatalf("ListProducts() error = %v", err)
	}
	if repository.filter.Limit != 2 || repository.filter.Search != "coke" || repository.filter.CategoryID == nil || *repository.filter.CategoryID != testCategoryID {
		t.Fatalf("repository filter = %#v", repository.filter)
	}
	if len(page.Items) != 1 || !page.Page.HasMore || page.Page.NextCursor == nil {
		t.Fatalf("page = %#v, want one item and next cursor", page)
	}
}

func TestListProductsRejectsInvalidCategoryFilter(t *testing.T) {
	categoryID := "bad-id"
	repository := &fakeRepository{}
	service := NewService(repository)
	if _, err := service.ListProducts(context.Background(), Actor{Role: auth.RolePicker}, ListFilter{CategoryID: &categoryID}); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("ListProducts() error = %v, want ErrInvalidID", err)
	}
}

func TestBarcodeLookupIsReadOnlyAndValidatesChecksum(t *testing.T) {
	repository := &fakeRepository{product: Product{ID: testProductID, Barcode: stringPointer(testBarcode)}}
	service := NewService(repository)
	for _, role := range []string{auth.RoleAdmin, auth.RoleWarehouseManager, auth.RolePicker, auth.RoleViewer} {
		product, err := service.GetProductByBarcode(context.Background(), Actor{Role: role}, "  "+testBarcode+"  ")
		if err != nil || product.ID != testProductID {
			t.Fatalf("role %s lookup = %#v, %v", role, product, err)
		}
	}
	if repository.barcode != testBarcode || repository.productWrites != 0 {
		t.Fatalf("barcode=%q writes=%d, want exact read-only lookup", repository.barcode, repository.productWrites)
	}
	if _, err := service.GetProductByBarcode(context.Background(), Actor{Role: auth.RoleViewer}, "123"); !errors.Is(err, ErrInvalidBarcode) {
		t.Fatalf("invalid lookup error = %v, want ErrInvalidBarcode", err)
	}
}

func TestGetUpdateDeactivateProductValidateIDAndPreserveErrors(t *testing.T) {
	repository := &fakeRepository{product: Product{ID: testProductID}}
	service := NewService(repository)
	if _, err := service.GetProduct(context.Background(), Actor{Role: auth.RoleViewer}, testProductID); err != nil {
		t.Fatalf("GetProduct() error = %v", err)
	}
	if err := service.DeactivateProduct(context.Background(), Actor{Role: auth.RoleAdmin}, testProductID); err != nil {
		t.Fatalf("DeactivateProduct() error = %v", err)
	}
	if _, err := service.UpdateProduct(context.Background(), Actor{Role: auth.RoleAdmin}, "bad-id", validProductInput()); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("UpdateProduct() error = %v, want ErrInvalidID", err)
	}
	repository.err = ErrProductSKUConflict
	if _, err := service.CreateProduct(context.Background(), Actor{Role: auth.RoleAdmin}, validProductInput()); !errors.Is(err, ErrProductSKUConflict) {
		t.Fatalf("CreateProduct() error = %v, want ErrProductSKUConflict", err)
	}
}
