package inventory

import (
	"context"
	"errors"
	"testing"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

type fakeRepository struct {
	listBalancesFn  func(BalanceListFilter) ([]Balance, error)
	listLotsFn      func(string, *string) ([]Lot, error)
	receiveFn       func(string, *string, ReceiveInput) (Movement, Balance, error)
	pickFn          func(string, *string, PickInput) ([]Movement, error)
	transferFn      func(string, *string, TransferInput) (Movement, Balance, Balance, error)
	adjustFn        func(string, *string, AdjustInput) (Movement, Balance, error)
	listMovementsFn func(MovementListFilter) ([]Movement, error)
	getMovementFn   func(string) (Movement, error)
}

func (f *fakeRepository) ListBalances(_ context.Context, filter BalanceListFilter) ([]Balance, error) {
	return f.listBalancesFn(filter)
}
func (f *fakeRepository) ListLots(_ context.Context, productID string, warehouseID *string) ([]Lot, error) {
	return f.listLotsFn(productID, warehouseID)
}
func (f *fakeRepository) Receive(_ context.Context, actorID string, actorWarehouseID *string, input ReceiveInput) (Movement, Balance, error) {
	return f.receiveFn(actorID, actorWarehouseID, input)
}
func (f *fakeRepository) Pick(_ context.Context, actorID string, actorWarehouseID *string, input PickInput) ([]Movement, error) {
	return f.pickFn(actorID, actorWarehouseID, input)
}
func (f *fakeRepository) Transfer(_ context.Context, actorID string, actorWarehouseID *string, input TransferInput) (Movement, Balance, Balance, error) {
	return f.transferFn(actorID, actorWarehouseID, input)
}
func (f *fakeRepository) Adjust(_ context.Context, actorID string, actorWarehouseID *string, input AdjustInput) (Movement, Balance, error) {
	return f.adjustFn(actorID, actorWarehouseID, input)
}
func (f *fakeRepository) ListMovements(_ context.Context, filter MovementListFilter) ([]Movement, error) {
	return f.listMovementsFn(filter)
}
func (f *fakeRepository) GetMovement(_ context.Context, id string) (Movement, error) {
	return f.getMovementFn(id)
}

func adminActor() Actor { return Actor{ID: "admin-1", Role: auth.RoleAdmin} }
func pickerActor(warehouseID string) Actor {
	return Actor{ID: "picker-1", Role: auth.RolePicker, WarehouseID: &warehouseID}
}

func TestListBalancesRejectsNonAdminWithoutWarehouse(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.ListBalances(context.Background(), Actor{ID: "x", Role: auth.RoleViewer}, BalanceListFilter{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestListBalancesScopesNonAdminToAssignedWarehouse(t *testing.T) {
	var capturedFilter BalanceListFilter
	repo := &fakeRepository{listBalancesFn: func(filter BalanceListFilter) ([]Balance, error) {
		capturedFilter = filter
		return nil, nil
	}}
	service := NewService(repo)
	warehouseID := "wh-1"
	if _, err := service.ListBalances(context.Background(), pickerActor(warehouseID), BalanceListFilter{}); err != nil {
		t.Fatalf("ListBalances() error = %v", err)
	}
	if capturedFilter.WarehouseID == nil || *capturedFilter.WarehouseID != warehouseID {
		t.Fatalf("capturedFilter.WarehouseID = %v, want %s", capturedFilter.WarehouseID, warehouseID)
	}
}

func TestListBalancesRejectsCrossWarehouseFilterForNonAdmin(t *testing.T) {
	service := NewService(&fakeRepository{})
	warehouseID := "wh-1"
	other := "wh-2"
	_, err := service.ListBalances(context.Background(), pickerActor(warehouseID), BalanceListFilter{WarehouseID: &other})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestListLotsRejectsInvalidProductID(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.ListLots(context.Background(), adminActor(), "not-a-uuid")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestReceiveRejectsViewerAndPicker(t *testing.T) {
	service := NewService(&fakeRepository{})
	for _, role := range []string{auth.RoleViewer} {
		_, _, err := service.Receive(context.Background(), Actor{ID: "x", Role: role}, ReceiveInput{
			LocationID: "11111111-1111-1111-1111-111111111111",
			ProductID:  "22222222-2222-2222-2222-222222222222",
			Quantity:   "1.000", UnitCost: "1.0000",
		})
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("role %s: err = %v, want ErrForbidden", role, err)
		}
	}
}

func TestReceiveRejectsNonPositiveQuantity(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, _, err := service.Receive(context.Background(), adminActor(), ReceiveInput{
		LocationID: "11111111-1111-1111-1111-111111111111",
		ProductID:  "22222222-2222-2222-2222-222222222222",
		Quantity:   "0.000", UnitCost: "1.0000",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestReceivePassesThroughToRepository(t *testing.T) {
	var capturedWarehouseID *string
	repo := &fakeRepository{receiveFn: func(actorID string, actorWarehouseID *string, input ReceiveInput) (Movement, Balance, error) {
		capturedWarehouseID = actorWarehouseID
		return Movement{ID: "m1"}, Balance{ID: "b1"}, nil
	}}
	service := NewService(repo)
	warehouseID := "wh-1"
	movement, balance, err := service.Receive(context.Background(), pickerActor(warehouseID), ReceiveInput{
		LocationID: "11111111-1111-1111-1111-111111111111",
		ProductID:  "22222222-2222-2222-2222-222222222222",
		Quantity:   "5.000", UnitCost: "1.0000",
	})
	if err != nil {
		t.Fatalf("Receive() error = %v", err)
	}
	if movement.ID != "m1" || balance.ID != "b1" {
		t.Fatalf("got movement=%+v balance=%+v, want passthrough from repository", movement, balance)
	}
	if capturedWarehouseID == nil || *capturedWarehouseID != warehouseID {
		t.Fatalf("capturedWarehouseID = %v, want %s", capturedWarehouseID, warehouseID)
	}
}

func TestPickRejectsTransferOnlyRoles(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.Pick(context.Background(), Actor{ID: "x", Role: auth.RoleViewer}, PickInput{
		LocationID: "11111111-1111-1111-1111-111111111111",
		ProductID:  "22222222-2222-2222-2222-222222222222",
		Quantity:   "1.000",
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestPickPassesThroughToRepository(t *testing.T) {
	repo := &fakeRepository{pickFn: func(actorID string, actorWarehouseID *string, input PickInput) ([]Movement, error) {
		return []Movement{{ID: "m1"}}, nil
	}}
	service := NewService(repo)
	movements, err := service.Pick(context.Background(), adminActor(), PickInput{
		LocationID: "11111111-1111-1111-1111-111111111111",
		ProductID:  "22222222-2222-2222-2222-222222222222",
		Quantity:   "5.000",
	})
	if err != nil || len(movements) != 1 {
		t.Fatalf("Pick() = %#v, %v, want one movement", movements, err)
	}
}

func TestTransferRejectsPickerAndViewer(t *testing.T) {
	service := NewService(&fakeRepository{})
	for _, role := range []string{auth.RolePicker, auth.RoleViewer} {
		_, _, _, err := service.Transfer(context.Background(), Actor{ID: "x", Role: role}, TransferInput{
			ProductID: "22222222-2222-2222-2222-222222222222", Quantity: "1.000",
			FromLocationID: "11111111-1111-1111-1111-111111111111",
			ToLocationID:   "33333333-3333-3333-3333-333333333333",
		})
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("role %s: err = %v, want ErrForbidden", role, err)
		}
	}
}

func TestTransferRejectsSameLocation(t *testing.T) {
	service := NewService(&fakeRepository{})
	sameLocation := "11111111-1111-1111-1111-111111111111"
	_, _, _, err := service.Transfer(context.Background(), adminActor(), TransferInput{
		ProductID: "22222222-2222-2222-2222-222222222222", Quantity: "1.000",
		FromLocationID: sameLocation, ToLocationID: sameLocation,
	})
	if !errors.Is(err, ErrSameLocationTransfer) {
		t.Fatalf("err = %v, want ErrSameLocationTransfer", err)
	}
}

func TestTransferPassesThroughToRepository(t *testing.T) {
	repo := &fakeRepository{transferFn: func(actorID string, actorWarehouseID *string, input TransferInput) (Movement, Balance, Balance, error) {
		return Movement{ID: "m1"}, Balance{ID: "src"}, Balance{ID: "dst"}, nil
	}}
	service := NewService(repo)
	movement, source, destination, err := service.Transfer(context.Background(), adminActor(), TransferInput{
		ProductID: "22222222-2222-2222-2222-222222222222", Quantity: "1.000",
		FromLocationID: "11111111-1111-1111-1111-111111111111",
		ToLocationID:   "33333333-3333-3333-3333-333333333333",
	})
	if err != nil || movement.ID != "m1" || source.ID != "src" || destination.ID != "dst" {
		t.Fatalf("Transfer() = %+v %+v %+v %v, want passthrough", movement, source, destination, err)
	}
}

func TestAdjustRejectsInvalidDirection(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, _, err := service.Adjust(context.Background(), adminActor(), AdjustInput{
		LocationID: "11111111-1111-1111-1111-111111111111",
		ProductID:  "22222222-2222-2222-2222-222222222222",
		Direction:  "sideways", Quantity: "1.000",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestAdjustRejectsPickerAndViewer(t *testing.T) {
	service := NewService(&fakeRepository{})
	for _, role := range []string{auth.RolePicker, auth.RoleViewer} {
		_, _, err := service.Adjust(context.Background(), Actor{ID: "x", Role: role}, AdjustInput{
			LocationID: "11111111-1111-1111-1111-111111111111",
			ProductID:  "22222222-2222-2222-2222-222222222222",
			Direction:  "increase", Quantity: "1.000",
		})
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("role %s: err = %v, want ErrForbidden", role, err)
		}
	}
}

func TestAdjustPassesThroughToRepository(t *testing.T) {
	repo := &fakeRepository{adjustFn: func(actorID string, actorWarehouseID *string, input AdjustInput) (Movement, Balance, error) {
		return Movement{ID: "m1"}, Balance{ID: "b1"}, nil
	}}
	service := NewService(repo)
	movement, balance, err := service.Adjust(context.Background(), adminActor(), AdjustInput{
		LocationID: "11111111-1111-1111-1111-111111111111",
		ProductID:  "22222222-2222-2222-2222-222222222222",
		Direction:  "decrease", Quantity: "1.000",
	})
	if err != nil || movement.ID != "m1" || balance.ID != "b1" {
		t.Fatalf("Adjust() = %+v %+v %v, want passthrough", movement, balance, err)
	}
}

func TestListMovementsRejectsNonAdminWithoutWarehouse(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.ListMovements(context.Background(), Actor{ID: "x", Role: auth.RoleViewer}, MovementListFilter{})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestGetMovementRejectsInvalidID(t *testing.T) {
	service := NewService(&fakeRepository{})
	_, err := service.GetMovement(context.Background(), adminActor(), "not-a-uuid")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestGetMovementPassesThroughToRepository(t *testing.T) {
	repo := &fakeRepository{getMovementFn: func(id string) (Movement, error) {
		return Movement{ID: id}, nil
	}}
	service := NewService(repo)
	movement, err := service.GetMovement(context.Background(), adminActor(), "11111111-1111-1111-1111-111111111111")
	if err != nil || movement.ID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("GetMovement() = %+v, %v, want passthrough", movement, err)
	}
}
