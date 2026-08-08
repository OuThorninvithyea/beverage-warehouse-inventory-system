package warehouse

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

const (
	testWarehouseID      = "7e5d55b1-6356-492b-8296-2b981867fcf2"
	testOtherWarehouseID = "a0f31ce4-8c52-4441-9904-8088c47757fd"
	testLocationID       = "70ac37cc-afb9-4c9c-8b9b-a96430956341"
)

type fakeRepository struct {
	warehouses                     []Warehouse
	warehouse                      Warehouse
	warehouseErr                   error
	createdWarehouseInput          WarehouseInput
	updatedWarehouseID             string
	updatedWarehouseInput          WarehouseInput
	deactivatedWarehouse           string
	warehouseScope                 *string
	warehouseFilter                ListFilter
	locations                      []Location
	location                       Location
	locationErr                    error
	createdLocationWarehouseID     string
	createdLocationInput           LocationInput
	updatedLocationWarehouseID     string
	updatedLocationID              string
	updatedLocationInput           LocationInput
	deactivatedLocationWarehouseID string
	deactivatedLocationID          string
	locationListWarehouseID        string
	locationFilter                 ListFilter
}

func (r *fakeRepository) ListWarehouses(_ context.Context, scope *string, filter ListFilter) ([]Warehouse, error) {
	r.warehouseScope = scope
	r.warehouseFilter = filter
	return r.warehouses, r.warehouseErr
}

func (r *fakeRepository) GetWarehouse(_ context.Context, _ string) (Warehouse, error) {
	return r.warehouse, r.warehouseErr
}

func (r *fakeRepository) CreateWarehouse(_ context.Context, input WarehouseInput) (Warehouse, error) {
	r.createdWarehouseInput = input
	if r.warehouseErr != nil {
		return Warehouse{}, r.warehouseErr
	}
	return Warehouse{Code: input.Code, Name: input.Name, Address: input.Address, IsActive: *input.IsActive}, nil
}

func (r *fakeRepository) UpdateWarehouse(_ context.Context, id string, input WarehouseInput) (Warehouse, error) {
	r.updatedWarehouseID = id
	r.updatedWarehouseInput = input
	if r.warehouseErr != nil {
		return Warehouse{}, r.warehouseErr
	}
	return Warehouse{ID: id, Code: input.Code, Name: input.Name, Address: input.Address}, nil
}

func (r *fakeRepository) DeactivateWarehouse(_ context.Context, id string) error {
	r.deactivatedWarehouse = id
	return r.warehouseErr
}

func (r *fakeRepository) ListLocations(_ context.Context, warehouseID string, filter ListFilter) ([]Location, error) {
	r.locationListWarehouseID = warehouseID
	r.locationFilter = filter
	return r.locations, r.locationErr
}

func (r *fakeRepository) GetLocation(_ context.Context, _, _ string) (Location, error) {
	return r.location, r.locationErr
}

func (r *fakeRepository) CreateLocation(_ context.Context, warehouseID string, input LocationInput) (Location, error) {
	r.createdLocationWarehouseID = warehouseID
	r.createdLocationInput = input
	if r.locationErr != nil {
		return Location{}, r.locationErr
	}
	return Location{
		WarehouseID: warehouseID,
		Code:        input.Code,
		Zone:        input.Zone,
		Aisle:       input.Aisle,
		Rack:        input.Rack,
		Shelf:       input.Shelf,
		Barcode:     input.Barcode,
		IsPickable:  *input.IsPickable,
		IsActive:    *input.IsActive,
	}, nil
}

func (r *fakeRepository) UpdateLocation(_ context.Context, warehouseID, locationID string, input LocationInput) (Location, error) {
	r.updatedLocationWarehouseID = warehouseID
	r.updatedLocationID = locationID
	r.updatedLocationInput = input
	if r.locationErr != nil {
		return Location{}, r.locationErr
	}
	return Location{ID: locationID, WarehouseID: warehouseID, Code: input.Code}, nil
}

func (r *fakeRepository) DeactivateLocation(_ context.Context, warehouseID, locationID string) error {
	r.deactivatedLocationWarehouseID = warehouseID
	r.deactivatedLocationID = locationID
	return r.locationErr
}

func TestCreateWarehouseNormalizesBusinessFields(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	address := "  Sen Sok  "

	got, err := service.CreateWarehouse(context.Background(), Actor{Role: auth.RoleAdmin}, WarehouseInput{
		Code: " pp-01 ", Name: " Phnom Penh Main ", Address: &address,
	})
	if err != nil {
		t.Fatalf("CreateWarehouse() error = %v", err)
	}
	if repository.createdWarehouseInput.Code != "PP-01" {
		t.Fatalf("repository code = %q, want PP-01", repository.createdWarehouseInput.Code)
	}
	if got.Name != "Phnom Penh Main" {
		t.Fatalf("name = %q, want Phnom Penh Main", got.Name)
	}
	if got.Address == nil || *got.Address != "Sen Sok" {
		t.Fatalf("address = %#v, want Sen Sok", got.Address)
	}
	if !got.IsActive {
		t.Fatal("is_active = false, want create default true")
	}
}

func TestCreateWarehouseTurnsBlankAddressIntoNil(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	address := "   "

	_, err := service.CreateWarehouse(context.Background(), Actor{Role: auth.RoleAdmin}, WarehouseInput{
		Code: "PP-01", Name: "Main", Address: &address,
	})
	if err != nil {
		t.Fatalf("CreateWarehouse() error = %v", err)
	}
	if repository.createdWarehouseInput.Address != nil {
		t.Fatalf("repository address = %#v, want nil", repository.createdWarehouseInput.Address)
	}
}

func TestCreateWarehouseRejectsInvalidOrUnauthorizedInput(t *testing.T) {
	tests := []struct {
		name  string
		actor Actor
		input WarehouseInput
		err   error
	}{
		{name: "manager cannot create", actor: Actor{Role: auth.RoleWarehouseManager}, input: WarehouseInput{Code: "PP-01", Name: "Main"}, err: ErrForbidden},
		{name: "blank code", actor: Actor{Role: auth.RoleAdmin}, input: WarehouseInput{Name: "Main"}, err: ErrValidation},
		{name: "blank name", actor: Actor{Role: auth.RoleAdmin}, input: WarehouseInput{Code: "PP-01"}, err: ErrValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewService(&fakeRepository{}).CreateWarehouse(context.Background(), tt.actor, tt.input)
			if !errors.Is(err, tt.err) {
				t.Fatalf("CreateWarehouse() error = %v, want %v", err, tt.err)
			}
		})
	}
}

func TestCreateWarehousePreservesCodeConflict(t *testing.T) {
	service := NewService(&fakeRepository{warehouseErr: ErrWarehouseCodeConflict})
	_, err := service.CreateWarehouse(context.Background(), Actor{Role: auth.RoleAdmin}, WarehouseInput{Code: "PP-01", Name: "Main"})
	if !errors.Is(err, ErrWarehouseCodeConflict) {
		t.Fatalf("CreateWarehouse() error = %v, want ErrWarehouseCodeConflict", err)
	}
}

func TestListWarehousesScopesManagerAndBuildsCursorPage(t *testing.T) {
	assigned := testWarehouseID
	now := time.Date(2026, 8, 8, 8, 0, 0, 0, time.UTC)
	repository := &fakeRepository{warehouses: []Warehouse{
		{ID: testWarehouseID, CreatedAt: now},
		{ID: testOtherWarehouseID, CreatedAt: now.Add(-time.Minute)},
		{ID: testLocationID, CreatedAt: now.Add(-2 * time.Minute)},
	}}

	page, err := NewService(repository).ListWarehouses(context.Background(), Actor{
		Role: auth.RoleWarehouseManager, WarehouseID: &assigned,
	}, ListFilter{Limit: 2, Search: " main "})
	if err != nil {
		t.Fatalf("ListWarehouses() error = %v", err)
	}
	if repository.warehouseScope == nil || *repository.warehouseScope != assigned {
		t.Fatalf("scope = %#v, want %s", repository.warehouseScope, assigned)
	}
	if repository.warehouseFilter.Limit != 3 {
		t.Fatalf("repository limit = %d, want 3", repository.warehouseFilter.Limit)
	}
	if repository.warehouseFilter.Search != "main" {
		t.Fatalf("search = %q, want main", repository.warehouseFilter.Search)
	}
	if len(page.Items) != 2 || !page.Page.HasMore || page.Page.NextCursor == nil {
		t.Fatalf("page = %#v, want two items and next cursor", page)
	}
	decoded, err := DecodeCursor(*page.Page.NextCursor)
	if err != nil {
		t.Fatalf("DecodeCursor() error = %v", err)
	}
	if decoded.ID != testOtherWarehouseID {
		t.Fatalf("cursor id = %s, want %s", decoded.ID, testOtherWarehouseID)
	}
}

func TestListWarehousesDefaultsAndCapsLimit(t *testing.T) {
	tests := []struct {
		name      string
		requested int
		wantRepo  int
	}{
		{name: "default", requested: 0, wantRepo: 21},
		{name: "cap", requested: 500, wantRepo: 101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeRepository{}
			_, err := NewService(repository).ListWarehouses(context.Background(), Actor{Role: auth.RoleAdmin}, ListFilter{Limit: tt.requested})
			if err != nil {
				t.Fatalf("ListWarehouses() error = %v", err)
			}
			if repository.warehouseFilter.Limit != tt.wantRepo {
				t.Fatalf("repository limit = %d, want %d", repository.warehouseFilter.Limit, tt.wantRepo)
			}
		})
	}
}

func TestWarehouseReadScopeRejectsMissingOrDifferentAssignment(t *testing.T) {
	tests := []struct {
		name       string
		assignment *string
	}{
		{name: "missing assignment"},
		{name: "different warehouse", assignment: stringPointer(testOtherWarehouseID)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewService(&fakeRepository{}).GetWarehouse(context.Background(), Actor{
				Role: auth.RoleViewer, WarehouseID: tt.assignment,
			}, testWarehouseID)
			if !errors.Is(err, ErrForbidden) {
				t.Fatalf("GetWarehouse() error = %v, want ErrForbidden", err)
			}
		})
	}
}

func TestWarehouseMutationsRequireAdmin(t *testing.T) {
	managerWarehouse := testWarehouseID
	service := NewService(&fakeRepository{})
	manager := Actor{Role: auth.RoleWarehouseManager, WarehouseID: &managerWarehouse}

	if _, err := service.UpdateWarehouse(context.Background(), manager, testWarehouseID, WarehouseInput{Code: "PP-01", Name: "Main"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("UpdateWarehouse() error = %v, want ErrForbidden", err)
	}
	if err := service.DeactivateWarehouse(context.Background(), manager, testWarehouseID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("DeactivateWarehouse() error = %v, want ErrForbidden", err)
	}
}

func TestUpdateAndDeactivateWarehouseUseValidatedID(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	admin := Actor{Role: auth.RoleAdmin}

	_, err := service.UpdateWarehouse(context.Background(), admin, testWarehouseID, WarehouseInput{Code: " pp-01 ", Name: " Main "})
	if err != nil {
		t.Fatalf("UpdateWarehouse() error = %v", err)
	}
	if repository.updatedWarehouseID != testWarehouseID || repository.updatedWarehouseInput.Code != "PP-01" {
		t.Fatalf("update = id %q input %#v", repository.updatedWarehouseID, repository.updatedWarehouseInput)
	}
	if err := service.DeactivateWarehouse(context.Background(), admin, testWarehouseID); err != nil {
		t.Fatalf("DeactivateWarehouse() error = %v", err)
	}
	if repository.deactivatedWarehouse != testWarehouseID {
		t.Fatalf("deactivated id = %q, want %q", repository.deactivatedWarehouse, testWarehouseID)
	}
}

func TestGetWarehouseRejectsInvalidUUID(t *testing.T) {
	_, err := NewService(&fakeRepository{}).GetWarehouse(context.Background(), Actor{Role: auth.RoleAdmin}, "not-a-uuid")
	if !errors.Is(err, ErrInvalidID) {
		t.Fatalf("GetWarehouse() error = %v, want ErrInvalidID", err)
	}
}

func TestManagerCreatesLocationOnlyInAssignedWarehouse(t *testing.T) {
	assigned := testWarehouseID
	manager := Actor{Role: auth.RoleWarehouseManager, WarehouseID: &assigned}
	service := NewService(&fakeRepository{})

	_, err := service.CreateLocation(context.Background(), manager, testOtherWarehouseID, LocationInput{Code: "A-01"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("CreateLocation() error = %v, want ErrForbidden", err)
	}
}

func TestCreateLocationNormalizesFieldsAndDefaultsBooleans(t *testing.T) {
	assigned := testWarehouseID
	zone := "  Ambient  "
	empty := "   "
	barcode := "  LOC000001  "
	repository := &fakeRepository{}

	got, err := NewService(repository).CreateLocation(context.Background(), Actor{
		Role: auth.RoleWarehouseManager, WarehouseID: &assigned,
	}, assigned, LocationInput{
		Code: " a-01-r02-s03 ", Zone: &zone, Aisle: &empty, Barcode: &barcode,
	})
	if err != nil {
		t.Fatalf("CreateLocation() error = %v", err)
	}
	if repository.createdLocationWarehouseID != assigned {
		t.Fatalf("warehouse id = %q, want %q", repository.createdLocationWarehouseID, assigned)
	}
	if repository.createdLocationInput.Code != "A-01-R02-S03" {
		t.Fatalf("code = %q, want A-01-R02-S03", repository.createdLocationInput.Code)
	}
	if got.Zone == nil || *got.Zone != "Ambient" {
		t.Fatalf("zone = %#v, want Ambient", got.Zone)
	}
	if got.Aisle != nil {
		t.Fatalf("aisle = %#v, want nil", got.Aisle)
	}
	if got.Barcode == nil || *got.Barcode != "LOC000001" {
		t.Fatalf("barcode = %#v, want LOC000001", got.Barcode)
	}
	if !got.IsPickable || !got.IsActive {
		t.Fatalf("is_pickable=%v is_active=%v, want true true", got.IsPickable, got.IsActive)
	}
}

func TestLocationMutationsEnforceRoles(t *testing.T) {
	assigned := testWarehouseID
	picker := Actor{Role: auth.RolePicker, WarehouseID: &assigned}
	service := NewService(&fakeRepository{})

	if _, err := service.CreateLocation(context.Background(), picker, assigned, LocationInput{Code: "A-01"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("CreateLocation() error = %v, want ErrForbidden", err)
	}
	if _, err := service.UpdateLocation(context.Background(), picker, assigned, testLocationID, LocationInput{Code: "A-01"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("UpdateLocation() error = %v, want ErrForbidden", err)
	}
	if err := service.DeactivateLocation(context.Background(), picker, assigned, testLocationID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("DeactivateLocation() error = %v, want ErrForbidden", err)
	}
}

func TestLocationReadsAllowAssignedViewer(t *testing.T) {
	assigned := testWarehouseID
	repository := &fakeRepository{location: Location{ID: testLocationID, WarehouseID: assigned, Code: "A-01"}}
	service := NewService(repository)
	viewer := Actor{Role: auth.RoleViewer, WarehouseID: &assigned}

	got, err := service.GetLocation(context.Background(), viewer, assigned, testLocationID)
	if err != nil {
		t.Fatalf("GetLocation() error = %v", err)
	}
	if got.ID != testLocationID {
		t.Fatalf("location id = %q, want %q", got.ID, testLocationID)
	}
}

func TestListLocationsBuildsCursorPage(t *testing.T) {
	assigned := testWarehouseID
	now := time.Date(2026, 8, 8, 8, 0, 0, 0, time.UTC)
	repository := &fakeRepository{locations: []Location{
		{ID: testLocationID, CreatedAt: now},
		{ID: testOtherWarehouseID, CreatedAt: now.Add(-time.Minute)},
		{ID: testWarehouseID, CreatedAt: now.Add(-2 * time.Minute)},
	}}
	pickable := true

	page, err := NewService(repository).ListLocations(context.Background(), Actor{
		Role: auth.RoleWarehouseManager, WarehouseID: &assigned,
	}, assigned, ListFilter{Limit: 2, Search: " rack ", IsPickable: &pickable})
	if err != nil {
		t.Fatalf("ListLocations() error = %v", err)
	}
	if repository.locationListWarehouseID != assigned {
		t.Fatalf("warehouse id = %q, want %q", repository.locationListWarehouseID, assigned)
	}
	if repository.locationFilter.Limit != 3 || repository.locationFilter.Search != "rack" {
		t.Fatalf("filter = %#v, want limit 3 search rack", repository.locationFilter)
	}
	if len(page.Items) != 2 || !page.Page.HasMore || page.Page.NextCursor == nil {
		t.Fatalf("page = %#v, want two items and next cursor", page)
	}
}

func TestUpdateLocationPreservesOmittedBooleans(t *testing.T) {
	assigned := testWarehouseID
	repository := &fakeRepository{}
	service := NewService(repository)
	manager := Actor{Role: auth.RoleWarehouseManager, WarehouseID: &assigned}

	_, err := service.UpdateLocation(context.Background(), manager, assigned, testLocationID, LocationInput{Code: " a-01 "})
	if err != nil {
		t.Fatalf("UpdateLocation() error = %v", err)
	}
	if repository.updatedLocationInput.IsPickable != nil || repository.updatedLocationInput.IsActive != nil {
		t.Fatalf("update booleans = %#v, %#v; want nil, nil", repository.updatedLocationInput.IsPickable, repository.updatedLocationInput.IsActive)
	}
	if repository.updatedLocationWarehouseID != assigned || repository.updatedLocationID != testLocationID {
		t.Fatalf("update ids = %q %q", repository.updatedLocationWarehouseID, repository.updatedLocationID)
	}
}

func TestLocationConflictsAndNotFoundArePreserved(t *testing.T) {
	assigned := testWarehouseID
	manager := Actor{Role: auth.RoleWarehouseManager, WarehouseID: &assigned}

	for _, wantErr := range []error{ErrLocationCodeConflict, ErrLocationBarcodeConflict, ErrLocationNotFound} {
		service := NewService(&fakeRepository{locationErr: wantErr})
		_, err := service.CreateLocation(context.Background(), manager, assigned, LocationInput{Code: "A-01"})
		if !errors.Is(err, wantErr) {
			t.Fatalf("CreateLocation() error = %v, want %v", err, wantErr)
		}
	}
}

func TestLocationOperationsRejectInvalidIDs(t *testing.T) {
	service := NewService(&fakeRepository{})
	admin := Actor{Role: auth.RoleAdmin}

	if _, err := service.GetLocation(context.Background(), admin, "bad", testLocationID); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("GetLocation() warehouse error = %v, want ErrInvalidID", err)
	}
	if _, err := service.GetLocation(context.Background(), admin, testWarehouseID, "bad"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("GetLocation() location error = %v, want ErrInvalidID", err)
	}
}

func stringPointer(value string) *string {
	return &value
}
