package catalog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCatalogLifecycle(t *testing.T) {
	databaseURL := os.Getenv("BWIMS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("BWIMS_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("database ping error = %v", err)
	}

	repository := NewPostgresRepository(pool)
	suffix := strings.ToUpper(fmt.Sprintf("%x", time.Now().UnixNano()))
	prefix := "CATALOG TEST " + suffix
	active := true

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM products WHERE sku LIKE $1", "TST-"+suffix+"%")
		_, _ = pool.Exec(context.Background(), "UPDATE categories SET parent_id = NULL WHERE name LIKE $1", prefix+"%")
		_, _ = pool.Exec(context.Background(), "DELETE FROM categories WHERE name LIKE $1", prefix+"%")
	})

	root, err := repository.CreateCategory(ctx, CategoryInput{
		Name: prefix + " ROOT", ParentID: OptionalString{Set: true}, IsActive: &active,
	})
	if err != nil {
		t.Fatalf("CreateCategory(root) error = %v", err)
	}
	childParent := root.ID
	child, err := repository.CreateCategory(ctx, CategoryInput{
		Name:     prefix + " CHILD",
		ParentID: OptionalString{Set: true, Value: &childParent},
		IsActive: &active,
	})
	if err != nil {
		t.Fatalf("CreateCategory(child) error = %v", err)
	}

	_, err = repository.CreateCategory(ctx, CategoryInput{
		Name: strings.ToLower(prefix + " ROOT"), ParentID: OptionalString{Set: true}, IsActive: &active,
	})
	if !errors.Is(err, ErrCategoryNameConflict) {
		t.Fatalf("duplicate category error = %v, want ErrCategoryNameConflict", err)
	}

	cycleParent := child.ID
	_, err = repository.UpdateCategory(ctx, root.ID, CategoryInput{
		Name: root.Name, ParentID: OptionalString{Set: true, Value: &cycleParent},
	})
	if !errors.Is(err, ErrCategoryCycle) {
		t.Fatalf("cycle update error = %v, want ErrCategoryCycle", err)
	}
	if err := repository.DeactivateCategory(ctx, root.ID); !errors.Is(err, ErrCategoryInUse) {
		t.Fatalf("category with active child error = %v, want ErrCategoryInUse", err)
	}

	barcode := "4006381333931"
	categoryID := root.ID
	product, err := repository.CreateProduct(ctx, ProductInput{
		CategoryID:   OptionalString{Set: true, Value: &categoryID},
		SKU:          "TST-" + suffix + "-COLA",
		Barcode:      OptionalString{Set: true, Value: &barcode},
		Name:         "Catalog Test Cola",
		Unit:         "case",
		IsLotTracked: &active,
		IsActive:     &active,
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}

	_, err = repository.CreateProduct(ctx, ProductInput{
		SKU: strings.ToLower(product.SKU), Barcode: OptionalString{Set: true},
		Name: "Duplicate SKU", Unit: "case", IsLotTracked: &active, IsActive: &active,
	})
	if !errors.Is(err, ErrProductSKUConflict) {
		t.Fatalf("duplicate SKU error = %v, want ErrProductSKUConflict", err)
	}
	_, err = repository.CreateProduct(ctx, ProductInput{
		SKU: "TST-" + suffix + "-OTHER", Barcode: OptionalString{Set: true, Value: &barcode},
		Name: "Duplicate Barcode", Unit: "case", IsLotTracked: &active, IsActive: &active,
	})
	if !errors.Is(err, ErrProductBarcodeConflict) {
		t.Fatalf("duplicate barcode error = %v, want ErrProductBarcodeConflict", err)
	}

	lookedUp, err := repository.GetProductByBarcode(ctx, barcode)
	if err != nil || lookedUp.ID != product.ID {
		t.Fatalf("GetProductByBarcode() = %#v, %v, want product %s", lookedUp, err, product.ID)
	}

	categories, err := repository.ListCategories(ctx, ListFilter{Limit: 10, Search: suffix, IsActive: &active})
	if err != nil || len(categories) != 2 {
		t.Fatalf("ListCategories() = %#v, %v, want two categories", categories, err)
	}
	products, err := repository.ListProducts(ctx, ListFilter{Limit: 10, Search: suffix, CategoryID: &root.ID, IsActive: &active})
	if err != nil || len(products) != 1 || products[0].ID != product.ID {
		t.Fatalf("ListProducts() = %#v, %v, want created product", products, err)
	}

	if err := repository.DeactivateCategory(ctx, root.ID); !errors.Is(err, ErrCategoryInUse) {
		t.Fatalf("category with active product error = %v, want ErrCategoryInUse", err)
	}

	updated, err := repository.UpdateProduct(ctx, product.ID, ProductInput{
		CategoryID: OptionalString{Set: true},
		SKU:        product.SKU,
		Barcode:    OptionalString{Set: true},
		Name:       "Catalog Test Cola Updated",
		Unit:       "case",
	})
	if err != nil {
		t.Fatalf("UpdateProduct() error = %v", err)
	}
	if updated.CategoryID != nil || updated.Barcode != nil {
		t.Fatalf("updated optional values = %#v %#v, want nil", updated.CategoryID, updated.Barcode)
	}

	if err := repository.DeactivateProduct(ctx, product.ID); err != nil {
		t.Fatalf("DeactivateProduct() error = %v", err)
	}
	if err := repository.DeactivateProduct(ctx, product.ID); err != nil {
		t.Fatalf("second DeactivateProduct() error = %v", err)
	}
	if _, err := repository.GetProductByBarcode(ctx, barcode); !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("inactive barcode lookup error = %v, want ErrProductNotFound", err)
	}

	if err := repository.DeactivateCategory(ctx, child.ID); err != nil {
		t.Fatalf("DeactivateCategory(child) error = %v", err)
	}
	if err := repository.DeactivateCategory(ctx, root.ID); err != nil {
		t.Fatalf("DeactivateCategory(root) error = %v", err)
	}
	if err := repository.DeactivateCategory(ctx, root.ID); err != nil {
		t.Fatalf("second DeactivateCategory(root) error = %v", err)
	}

	inactiveCategory := root.ID
	_, err = repository.CreateProduct(ctx, ProductInput{
		CategoryID: OptionalString{Set: true, Value: &inactiveCategory},
		SKU:        "TST-" + suffix + "-INACTIVE", Barcode: OptionalString{Set: true},
		Name: "Inactive Category Product", Unit: "case", IsLotTracked: &active, IsActive: &active,
	})
	if !errors.Is(err, ErrCategoryNotFound) {
		t.Fatalf("inactive category error = %v, want ErrCategoryNotFound", err)
	}
}
