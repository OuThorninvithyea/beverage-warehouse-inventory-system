package reports

import (
	"context"
	"errors"
	"testing"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

type fakeRepository struct {
	captured Filter
}

func (f *fakeRepository) Dashboard(_ context.Context, filter Filter) (Dashboard, error) {
	f.captured = filter
	return Dashboard{}, nil
}
func (f *fakeRepository) Valuation(_ context.Context, filter Filter) (Valuation, error) {
	f.captured = filter
	return Valuation{}, nil
}
func (f *fakeRepository) MovementSummary(_ context.Context, filter Filter) (MovementSummary, error) {
	f.captured = filter
	return MovementSummary{}, nil
}
func (f *fakeRepository) Velocity(_ context.Context, filter Filter) (Velocity, error) {
	f.captured = filter
	return Velocity{}, nil
}

const (
	warehouseA = "11111111-1111-1111-1111-111111111111"
	warehouseB = "22222222-2222-2222-2222-222222222222"
)

func managerActor(warehouseID string) Actor {
	return Actor{ID: "manager-1", Role: auth.RoleWarehouseManager, WarehouseID: &warehouseID}
}

// Every report shares resolveFilter, so the boundary is checked once per
// report rather than assumed.
func eachReport(t *testing.T, run func(t *testing.T, call func(Service, Actor, Filter) error)) {
	t.Helper()
	calls := map[string]func(Service, Actor, Filter) error{
		"dashboard": func(s Service, a Actor, f Filter) error {
			_, err := s.Dashboard(context.Background(), a, f)
			return err
		},
		"valuation": func(s Service, a Actor, f Filter) error {
			_, err := s.Valuation(context.Background(), a, f)
			return err
		},
		"movement-summary": func(s Service, a Actor, f Filter) error {
			_, err := s.MovementSummary(context.Background(), a, f)
			return err
		},
		"velocity": func(s Service, a Actor, f Filter) error {
			_, err := s.Velocity(context.Background(), a, f)
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) { run(t, call) })
	}
}

func TestReportsRejectPickersAndViewers(t *testing.T) {
	eachReport(t, func(t *testing.T, call func(Service, Actor, Filter) error) {
		for _, role := range []string{auth.RolePicker, auth.RoleViewer} {
			err := call(NewService(&fakeRepository{}), Actor{ID: "x", Role: role}, Filter{})
			if !errors.Is(err, ErrForbidden) {
				t.Fatalf("role %s: err = %v, want ErrForbidden", role, err)
			}
		}
	})
}

func TestReportsRejectManagerWithoutWarehouse(t *testing.T) {
	eachReport(t, func(t *testing.T, call func(Service, Actor, Filter) error) {
		err := call(NewService(&fakeRepository{}),
			Actor{ID: "x", Role: auth.RoleWarehouseManager}, Filter{})
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("err = %v, want ErrForbidden", err)
		}
	})
}

func TestReportsScopeManagerToAssignedWarehouse(t *testing.T) {
	eachReport(t, func(t *testing.T, call func(Service, Actor, Filter) error) {
		repo := &fakeRepository{}
		if err := call(NewService(repo), managerActor(warehouseA), Filter{}); err != nil {
			t.Fatalf("call error = %v", err)
		}
		if repo.captured.WarehouseID == nil || *repo.captured.WarehouseID != warehouseA {
			t.Fatalf("WarehouseID = %v, want %s", repo.captured.WarehouseID, warehouseA)
		}
	})
}

func TestReportsRejectCrossWarehouseRequests(t *testing.T) {
	eachReport(t, func(t *testing.T, call func(Service, Actor, Filter) error) {
		other := warehouseB
		err := call(NewService(&fakeRepository{}), managerActor(warehouseA),
			Filter{WarehouseID: &other})
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("err = %v, want ErrForbidden", err)
		}
	})
}

func TestReportsLetAdminsReadEveryWarehouse(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)
	admin := Actor{ID: "admin-1", Role: auth.RoleAdmin}

	if _, err := service.Dashboard(context.Background(), admin, Filter{}); err != nil {
		t.Fatalf("Dashboard() error = %v", err)
	}
	if repo.captured.WarehouseID != nil {
		t.Fatalf("WarehouseID = %v, want nil so the report spans all warehouses", repo.captured.WarehouseID)
	}

	target := warehouseB
	if _, err := service.Dashboard(context.Background(), admin, Filter{WarehouseID: &target}); err != nil {
		t.Fatalf("Dashboard(scoped) error = %v", err)
	}
	if repo.captured.WarehouseID == nil || *repo.captured.WarehouseID != target {
		t.Fatalf("WarehouseID = %v, want %s", repo.captured.WarehouseID, target)
	}
}

func TestReportsRejectAnInvalidWarehouseID(t *testing.T) {
	invalid := "not-a-uuid"
	_, err := NewService(&fakeRepository{}).Valuation(context.Background(),
		Actor{ID: "admin-1", Role: auth.RoleAdmin}, Filter{WarehouseID: &invalid})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestReportsClampTheWindowAndLimit(t *testing.T) {
	cases := []struct {
		name      string
		filter    Filter
		wantDays  int
		wantLimit int
	}{
		{name: "defaults", filter: Filter{}, wantDays: defaultWindowDays, wantLimit: defaultRowLimit},
		{name: "explicit values pass through", filter: Filter{Days: 7, Limit: 10}, wantDays: 7, wantLimit: 10},
		{name: "oversized values are clamped", filter: Filter{Days: 10_000, Limit: 10_000},
			wantDays: maxWindowDays, wantLimit: maxRowLimit},
		{name: "negatives fall back to the defaults", filter: Filter{Days: -5, Limit: -5},
			wantDays: defaultWindowDays, wantLimit: defaultRowLimit},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			repo := &fakeRepository{}
			if _, err := NewService(repo).Velocity(context.Background(),
				Actor{ID: "admin-1", Role: auth.RoleAdmin}, testCase.filter); err != nil {
				t.Fatalf("Velocity() error = %v", err)
			}
			if repo.captured.Days != testCase.wantDays {
				t.Errorf("Days = %d, want %d", repo.captured.Days, testCase.wantDays)
			}
			if repo.captured.Limit != testCase.wantLimit {
				t.Errorf("Limit = %d, want %d", repo.captured.Limit, testCase.wantLimit)
			}
		})
	}
}
