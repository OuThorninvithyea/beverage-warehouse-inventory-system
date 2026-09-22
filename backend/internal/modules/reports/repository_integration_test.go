package reports

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The reports read across every inventory table, so the fixture builds a
// small warehouse by hand: two products, two lots, known quantities and known
// unit costs, so every number the report returns can be checked by hand.
type fixture struct {
	pool        *pgxpool.Pool
	warehouseID string
	otherID     string
	locationID  string
	otherLoc    string
	productA    string
	productB    string
	actorID     string
}

func setupFixture(t *testing.T) fixture {
	t.Helper()
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
		t.Fatalf("ping error = %v", err)
	}

	suffix := strings.ToUpper(fmt.Sprintf("%x", time.Now().UnixNano()))
	f := fixture{pool: pool}

	mustScan := func(query string, args ...any) string {
		var id string
		if err := pool.QueryRow(ctx, query, args...).Scan(&id); err != nil {
			t.Fatalf("fixture query failed: %v\n%s", err, query)
		}
		return id
	}

	f.warehouseID = mustScan(`INSERT INTO warehouses (code, name) VALUES ($1, $2) RETURNING id::text`,
		"RPT-"+suffix, "Reports Warehouse "+suffix)
	f.otherID = mustScan(`INSERT INTO warehouses (code, name) VALUES ($1, $2) RETURNING id::text`,
		"RPT2-"+suffix, "Other Warehouse "+suffix)
	f.locationID = mustScan(`INSERT INTO locations (warehouse_id, code, zone) VALUES ($1, $2, 'AMBIENT') RETURNING id::text`,
		f.warehouseID, "RPT-A-"+suffix)
	f.otherLoc = mustScan(`INSERT INTO locations (warehouse_id, code, zone) VALUES ($1, $2, 'AMBIENT') RETURNING id::text`,
		f.otherID, "RPT-B-"+suffix)
	f.productA = mustScan(`INSERT INTO products (sku, name, unit) VALUES ($1, $2, 'case') RETURNING id::text`,
		"RPT-SKU-A-"+suffix, "Reports Product A "+suffix)
	f.productB = mustScan(`INSERT INTO products (sku, name, unit) VALUES ($1, $2, 'case') RETURNING id::text`,
		"RPT-SKU-B-"+suffix, "Reports Product B "+suffix)
	// The actor only has to exist to satisfy stock_movements.performed_by, and
	// it must NOT be an admin. Movements are append-only, so this user can
	// never be deleted; a leftover admin would add a second active admin to
	// the shared test database and break the users module's "last active
	// admin" invariant from an unrelated package.
	f.actorID = mustScan(`
		INSERT INTO users (role_id, warehouse_id, email, password_hash, full_name)
		SELECT id, $1, $2, 'x', 'Reports Actor' FROM roles WHERE code = 'warehouse_manager'
		RETURNING id::text`, f.warehouseID, "reports-"+strings.ToLower(suffix)+"@bwims.local")

	return f
}

// seedStock writes a lot, a balance, a receive movement and its cost layer:
// the same shape the inventory repository produces, so the reports see
// realistic data.
func (f fixture) seedStock(t *testing.T, productID, lotNumber string, expiresInDays int,
	quantity, reserved, unitCost string, receivedDaysAgo int) string {
	t.Helper()
	ctx := context.Background()

	var lotID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO lots (product_id, lot_number, expiration_date, received_at)
		VALUES ($1, $2, CURRENT_DATE + $3::int, NOW() - ($4::int * INTERVAL '1 day'))
		RETURNING id::text`,
		productID, lotNumber, expiresInDays, receivedDaysAgo).Scan(&lotID); err != nil {
		t.Fatalf("seed lot error = %v", err)
	}

	if _, err := f.pool.Exec(ctx, `
		INSERT INTO inventory_balances (location_id, product_id, lot_id, quantity, reserved_quantity)
		VALUES ($1, $2, $3, $4::numeric, $5::numeric)`,
		f.locationID, productID, lotID, quantity, reserved); err != nil {
		t.Fatalf("seed balance error = %v", err)
	}

	var movementID string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO stock_movements
		    (movement_type, product_id, lot_id, to_location_id, quantity, unit_cost, reference, performed_by, created_at)
		VALUES ('receive', $1, $2, $3, $4::numeric, $5::numeric, 'RPT-TEST', $6,
		        NOW() - ($7::int * INTERVAL '1 day'))
		RETURNING id::text`,
		productID, lotID, f.locationID, quantity, unitCost, f.actorID, receivedDaysAgo).Scan(&movementID); err != nil {
		t.Fatalf("seed movement error = %v", err)
	}

	if _, err := f.pool.Exec(ctx, `
		INSERT INTO cost_layers
		    (warehouse_id, product_id, lot_id, source_movement_id,
		     original_quantity, remaining_quantity, unit_cost, received_at)
		VALUES ($1, $2, $3, $4, $5::numeric, $5::numeric, $6::numeric,
		        NOW() - ($7::int * INTERVAL '1 day'))`,
		f.warehouseID, productID, lotID, movementID, quantity, unitCost, receivedDaysAgo); err != nil {
		t.Fatalf("seed cost layer error = %v", err)
	}

	return lotID
}

func (f fixture) seedPick(t *testing.T, productID, lotID, quantity string, daysAgo int) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), `
		INSERT INTO stock_movements
		    (movement_type, product_id, lot_id, from_location_id, quantity, reference, performed_by, created_at)
		VALUES ('pick', $1, $2, $3, $4::numeric, 'RPT-TEST', $5, NOW() - ($6::int * INTERVAL '1 day'))`,
		productID, lotID, f.locationID, quantity, f.actorID, daysAgo); err != nil {
		t.Fatalf("seed pick error = %v", err)
	}
}

func TestPostgresDashboardReport(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()
	repository := NewPostgresRepository(f.pool)

	// 100 cases at 2.50 = 250.00, with 10 reserved, expiring in 10 days.
	lotA := f.seedStock(t, f.productA, "RPT-LOT-A", 10, "100.000", "10.000", "2.5000", 20)
	// 40 cases at 5.00 = 200.00, already expired.
	f.seedStock(t, f.productB, "RPT-LOT-B", -3, "40.000", "0.000", "5.0000", 40)
	f.seedPick(t, f.productA, lotA, "15.000", 2)

	dashboard, err := repository.Dashboard(ctx, Filter{WarehouseID: &f.warehouseID, Days: 30, Limit: 50})
	if err != nil {
		t.Fatalf("Dashboard() error = %v", err)
	}

	if dashboard.TotalQuantity != "140.000" {
		t.Errorf("TotalQuantity = %s, want 140.000", dashboard.TotalQuantity)
	}
	if dashboard.ReservedQuantity != "10.000" {
		t.Errorf("ReservedQuantity = %s, want 10.000", dashboard.ReservedQuantity)
	}
	if dashboard.AvailableQuantity != "130.000" {
		t.Errorf("AvailableQuantity = %s, want 130.000", dashboard.AvailableQuantity)
	}
	// 100 * 2.50 + 40 * 5.00 = 450.00
	if !strings.HasPrefix(dashboard.StockValue, "450") {
		t.Errorf("StockValue = %s, want 450", dashboard.StockValue)
	}
	if dashboard.LotsOnHand != 2 {
		t.Errorf("LotsOnHand = %d, want 2", dashboard.LotsOnHand)
	}
	if dashboard.ExpiredLots != 1 {
		t.Errorf("ExpiredLots = %d, want 1", dashboard.ExpiredLots)
	}
	if dashboard.ExpiringSoonLots != 1 {
		t.Errorf("ExpiringSoonLots = %d, want 1", dashboard.ExpiringSoonLots)
	}
	// Only product A's receive (20 days ago) and the pick fall inside the
	// 30-day window; product B was received 40 days ago and is excluded.
	if dashboard.MovementsByType["receive"] != 1 || dashboard.MovementsByType["pick"] != 1 {
		t.Errorf("MovementsByType = %v, want 1 receive and 1 pick inside the 30-day window",
			dashboard.MovementsByType)
	}
	// Widening the window past the older receive must pick it up, which proves
	// the window is doing the filtering rather than something else.
	wider, err := repository.Dashboard(ctx, Filter{WarehouseID: &f.warehouseID, Days: 60, Limit: 50})
	if err != nil {
		t.Fatalf("Dashboard(60) error = %v", err)
	}
	if wider.MovementsByType["receive"] != 2 {
		t.Errorf("60-day window receives = %d, want 2", wider.MovementsByType["receive"])
	}
	// Every movement type is present even at zero, so charts keep their shape.
	for _, movementType := range movementTypes {
		if _, ok := dashboard.MovementsByType[movementType]; !ok {
			t.Errorf("MovementsByType is missing the %s key", movementType)
		}
	}

	// The other warehouse shares nothing, so its dashboard must be empty.
	other, err := repository.Dashboard(ctx, Filter{WarehouseID: &f.otherID, Days: 30, Limit: 50})
	if err != nil {
		t.Fatalf("Dashboard(other) error = %v", err)
	}
	if other.TotalQuantity != "0" || other.LotsOnHand != 0 {
		t.Errorf("other warehouse dashboard = %+v, want empty totals", other)
	}
}

func TestPostgresValuationReport(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()
	repository := NewPostgresRepository(f.pool)

	f.seedStock(t, f.productA, "RPT-VAL-A", 60, "100.000", "0.000", "2.5000", 15)
	f.seedStock(t, f.productB, "RPT-VAL-B", 90, "40.000", "0.000", "5.0000", 10)

	valuation, err := repository.Valuation(ctx, Filter{WarehouseID: &f.warehouseID, Days: 30, Limit: 50})
	if err != nil {
		t.Fatalf("Valuation() error = %v", err)
	}
	if len(valuation.Rows) != 2 {
		t.Fatalf("len(Rows) = %d, want 2", len(valuation.Rows))
	}
	// Ordered by value descending: A is worth 250.00, B is worth 200.00.
	if valuation.Rows[0].TotalValue != "250.0000" {
		t.Errorf("Rows[0].TotalValue = %s, want 250.0000", valuation.Rows[0].TotalValue)
	}
	if valuation.Rows[0].AverageUnitCost != "2.5000" {
		t.Errorf("Rows[0].AverageUnitCost = %s, want 2.5000", valuation.Rows[0].AverageUnitCost)
	}
	if valuation.Rows[1].TotalValue != "200.0000" {
		t.Errorf("Rows[1].TotalValue = %s, want 200.0000", valuation.Rows[1].TotalValue)
	}
	if valuation.TotalValue != "450.0000" {
		t.Errorf("TotalValue = %s, want 450.0000", valuation.TotalValue)
	}

	// A limit truncates the rows but must not understate the grand total.
	limited, err := repository.Valuation(ctx, Filter{WarehouseID: &f.warehouseID, Days: 30, Limit: 1})
	if err != nil {
		t.Fatalf("Valuation(limit 1) error = %v", err)
	}
	if len(limited.Rows) != 1 {
		t.Fatalf("len(Rows) = %d, want 1", len(limited.Rows))
	}
	if limited.TotalValue != "450.0000" {
		t.Errorf("TotalValue under a limit = %s, want the full 450.0000", limited.TotalValue)
	}
}

func TestPostgresMovementSummaryAndVelocity(t *testing.T) {
	f := setupFixture(t)
	ctx := context.Background()
	repository := NewPostgresRepository(f.pool)

	lotA := f.seedStock(t, f.productA, "RPT-VEL-A", 60, "200.000", "0.000", "2.0000", 20)
	lotB := f.seedStock(t, f.productB, "RPT-VEL-B", 60, "200.000", "0.000", "3.0000", 20)
	f.seedPick(t, f.productA, lotA, "30.000", 3)
	f.seedPick(t, f.productA, lotA, "20.000", 1)
	f.seedPick(t, f.productB, lotB, "10.000", 1)
	// Outside a 10-day window, so it must be excluded from short reports.
	f.seedPick(t, f.productB, lotB, "90.000", 40)

	summary, err := repository.MovementSummary(ctx, Filter{WarehouseID: &f.warehouseID, Days: 10, Limit: 50})
	if err != nil {
		t.Fatalf("MovementSummary() error = %v", err)
	}
	if summary.Totals["pick"] != 3 {
		t.Errorf("Totals[pick] = %d, want 3 inside the 10-day window", summary.Totals["pick"])
	}
	if summary.Totals["receive"] != 0 {
		t.Errorf("Totals[receive] = %d, want 0: the receives are 20 days old", summary.Totals["receive"])
	}
	if len(summary.Rows) == 0 {
		t.Error("Rows is empty, want one row per day and movement type")
	}

	velocity, err := repository.Velocity(ctx, Filter{WarehouseID: &f.warehouseID, Days: 10, Limit: 50})
	if err != nil {
		t.Fatalf("Velocity() error = %v", err)
	}
	if len(velocity.Rows) != 2 {
		t.Fatalf("len(Rows) = %d, want 2 products picked in the window", len(velocity.Rows))
	}
	// Product A moved 50 in the window, product B only 10.
	if velocity.Rows[0].PickedQuantity != "50.000" || velocity.Rows[0].PickCount != 2 {
		t.Errorf("Rows[0] = %+v, want 50.000 across 2 picks", velocity.Rows[0])
	}
	if velocity.Rows[1].PickedQuantity != "10.000" {
		t.Errorf("Rows[1].PickedQuantity = %s, want 10.000", velocity.Rows[1].PickedQuantity)
	}
	// 50 over a 10-day window averages 5 a day.
	if velocity.Rows[0].DailyAverage != "5.000" {
		t.Errorf("Rows[0].DailyAverage = %s, want 5.000", velocity.Rows[0].DailyAverage)
	}

	// A wider window picks the old movement back up.
	wide, err := repository.Velocity(ctx, Filter{WarehouseID: &f.warehouseID, Days: 90, Limit: 50})
	if err != nil {
		t.Fatalf("Velocity(90) error = %v", err)
	}
	if wide.Rows[0].PickedQuantity != "100.000" {
		t.Errorf("widest window Rows[0].PickedQuantity = %s, want 100.000 for product B", wide.Rows[0].PickedQuantity)
	}
}
