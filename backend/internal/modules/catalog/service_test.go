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
