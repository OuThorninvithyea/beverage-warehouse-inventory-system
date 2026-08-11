package warehouse

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

func TestPostgresRepositoryWarehouseAndLocationLifecycle(t *testing.T) {
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
	warehouseCode := "TST-" + suffix
	locationCode := "A-" + suffix
	barcode := "LOC-" + suffix
	active := true

	createdWarehouse, err := repository.CreateWarehouse(ctx, WarehouseInput{
		Code: warehouseCode, Name: "Repository Test " + suffix, IsActive: &active,
	})
	if err != nil {
		t.Fatalf("CreateWarehouse() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
			DELETE FROM locations
			WHERE warehouse_id IN (SELECT id FROM warehouses WHERE LOWER(code) = LOWER($1))`, warehouseCode)
		_, _ = pool.Exec(context.Background(), "DELETE FROM warehouses WHERE LOWER(code) = LOWER($1)", warehouseCode)
	})

	_, err = repository.CreateWarehouse(ctx, WarehouseInput{
		Code: strings.ToLower(warehouseCode), Name: "Duplicate", IsActive: &active,
	})
	if !errors.Is(err, ErrWarehouseCodeConflict) {
		t.Fatalf("lower-case duplicate warehouse error = %v, want ErrWarehouseCodeConflict", err)
	}

	createdLocation, err := repository.CreateLocation(ctx, createdWarehouse.ID, LocationInput{
		Code: locationCode, Barcode: &barcode, IsPickable: &active, IsActive: &active,
	})
	if err != nil {
		t.Fatalf("CreateLocation() error = %v", err)
	}

	_, err = repository.CreateLocation(ctx, createdWarehouse.ID, LocationInput{
		Code: strings.ToLower(locationCode), IsPickable: &active, IsActive: &active,
	})
	if !errors.Is(err, ErrLocationCodeConflict) {
		t.Fatalf("lower-case duplicate location error = %v, want ErrLocationCodeConflict", err)
	}

	_, err = repository.CreateLocation(ctx, createdWarehouse.ID, LocationInput{
		Code: "B-" + suffix, Barcode: &barcode, IsPickable: &active, IsActive: &active,
	})
	if !errors.Is(err, ErrLocationBarcodeConflict) {
		t.Fatalf("duplicate barcode error = %v, want ErrLocationBarcodeConflict", err)
	}

	_, err = repository.CreateLocation(ctx, testOtherWarehouseID, LocationInput{
		Code: "MISSING-" + suffix, IsPickable: &active, IsActive: &active,
	})
	if !errors.Is(err, ErrWarehouseNotFound) {
		t.Fatalf("missing warehouse error = %v, want ErrWarehouseNotFound", err)
	}

	if _, err := repository.GetLocation(ctx, testOtherWarehouseID, createdLocation.ID); !errors.Is(err, ErrLocationNotFound) {
		t.Fatalf("cross-warehouse location error = %v, want ErrLocationNotFound", err)
	}

	items, err := repository.ListWarehouses(ctx, nil, ListFilter{Limit: 2, Search: suffix})
	if err != nil {
		t.Fatalf("ListWarehouses() error = %v", err)
	}
	if len(items) != 1 || items[0].ID != createdWarehouse.ID {
		t.Fatalf("ListWarehouses() = %#v, want created warehouse", items)
	}

	if err := repository.DeactivateLocation(ctx, createdWarehouse.ID, createdLocation.ID); err != nil {
		t.Fatalf("DeactivateLocation() error = %v", err)
	}
	if err := repository.DeactivateLocation(ctx, createdWarehouse.ID, createdLocation.ID); err != nil {
		t.Fatalf("second DeactivateLocation() error = %v", err)
	}
	gotLocation, err := repository.GetLocation(ctx, createdWarehouse.ID, createdLocation.ID)
	if err != nil {
		t.Fatalf("GetLocation() error = %v", err)
	}
	if gotLocation.IsActive {
		t.Fatal("location is_active = true after deactivation")
	}

	if err := repository.DeactivateWarehouse(ctx, createdWarehouse.ID); err != nil {
		t.Fatalf("DeactivateWarehouse() error = %v", err)
	}
	if err := repository.DeactivateWarehouse(ctx, createdWarehouse.ID); err != nil {
		t.Fatalf("second DeactivateWarehouse() error = %v", err)
	}
	gotWarehouse, err := repository.GetWarehouse(ctx, createdWarehouse.ID)
	if err != nil {
		t.Fatalf("GetWarehouse() error = %v", err)
	}
	if gotWarehouse.IsActive {
		t.Fatal("warehouse is_active = true after deactivation")
	}
}
