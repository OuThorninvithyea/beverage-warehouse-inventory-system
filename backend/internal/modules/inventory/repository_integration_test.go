package inventory

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

type testFixture struct {
	pool           *pgxpool.Pool
	suffix         string
	warehouseID    string
	locationAID    string
	locationBID    string
	trackedProduct string
	plainProduct   string
	actorID        string
}

func setupFixture(t *testing.T) testFixture {
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
		t.Fatalf("database ping error = %v", err)
	}

	suffix := strings.ToUpper(fmt.Sprintf("%x", time.Now().UnixNano()))
	fixture := testFixture{pool: pool, suffix: suffix}

	if err := pool.QueryRow(ctx, `
		INSERT INTO warehouses (code, name, is_active) VALUES ($1, $2, TRUE) RETURNING id::text`,
		"WH-"+suffix, "Inventory Test Warehouse "+suffix,
	).Scan(&fixture.warehouseID); err != nil {
		t.Fatalf("seed warehouse error = %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO locations (warehouse_id, code, is_pickable, is_active)
		VALUES ($1, $2, TRUE, TRUE) RETURNING id::text`,
		fixture.warehouseID, "LOC-A-"+suffix,
	).Scan(&fixture.locationAID); err != nil {
		t.Fatalf("seed location A error = %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO locations (warehouse_id, code, is_pickable, is_active)
		VALUES ($1, $2, TRUE, TRUE) RETURNING id::text`,
		fixture.warehouseID, "LOC-B-"+suffix,
	).Scan(&fixture.locationBID); err != nil {
		t.Fatalf("seed location B error = %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO products (sku, name, unit, is_lot_tracked, is_active)
		VALUES ($1, $2, 'case', TRUE, TRUE) RETURNING id::text`,
		"SKU-TRACKED-"+suffix, "Tracked Product "+suffix,
	).Scan(&fixture.trackedProduct); err != nil {
		t.Fatalf("seed tracked product error = %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO products (sku, name, unit, is_lot_tracked, is_active)
		VALUES ($1, $2, 'case', FALSE, TRUE) RETURNING id::text`,
		"SKU-PLAIN-"+suffix, "Plain Product "+suffix,
	).Scan(&fixture.plainProduct); err != nil {
		t.Fatalf("seed plain product error = %v", err)
	}

	// Deliberately not 'admin': stock_movements.performed_by will end up
	// referencing this user, and stock_movements is append-only (see the
	// stock_movements_immutable trigger), so this row can never actually be
	// deleted once the test runs a movement. If it held the admin role it
	// would permanently inflate the global active-admin count and silently
	// break the users module's "last admin" invariant tests, which count
	// admins across the whole table, not scoped to any one test's fixture
	// (this exact failure mode was hit and diagnosed during development).
	var roleID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM roles WHERE code = 'picker'`).Scan(&roleID); err != nil {
		t.Fatalf("look up picker role error = %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (role_id, email, password_hash, full_name, is_active)
		VALUES ($1, $2, 'x', 'Inventory Test Actor', TRUE) RETURNING id::text`,
		roleID, "inventory-actor-"+strings.ToLower(suffix)+"@bwims.test",
	).Scan(&fixture.actorID); err != nil {
		t.Fatalf("seed actor user error = %v", err)
	}

	// Only cost_layers and inventory_balances are actually deletable here.
	// stock_movements (and therefore any product/lot/location/user it
	// references) is permanently retained by the append-only trigger — that
	// is correct audit-trail behavior, not a cleanup bug, so this test
	// database is expected to accumulate uniquely-suffixed rows across runs
	// rather than being fully reset.
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM cost_layers WHERE product_id IN ($1, $2)", fixture.trackedProduct, fixture.plainProduct)
		_, _ = pool.Exec(cleanupCtx, "DELETE FROM inventory_balances WHERE product_id IN ($1, $2)", fixture.trackedProduct, fixture.plainProduct)
	})

	return fixture
}

func TestPostgresInventoryLifecycle(t *testing.T) {
	fixture := setupFixture(t)
	ctx := context.Background()
	repository := NewPostgresRepository(fixture.pool)

	empty, err := repository.ListBalances(ctx, BalanceListFilter{Limit: 10, ProductID: &fixture.trackedProduct})
	if err != nil {
		t.Fatalf("ListBalances(empty) error = %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("ListBalances(empty) = %#v, want no rows before any receive", empty)
	}

	emptyLots, err := repository.ListLots(ctx, fixture.trackedProduct, nil)
	if err != nil {
		t.Fatalf("ListLots(empty) error = %v", err)
	}
	if len(emptyLots) != 0 {
		t.Fatalf("ListLots(empty) = %#v, want no lots before any receive", emptyLots)
	}

	// --- Receive ---

	movement, balance, err := repository.Receive(ctx, fixture.actorID, nil, ReceiveInput{
		LocationID: fixture.locationAID, ProductID: fixture.trackedProduct,
		Quantity: "50.000", UnitCost: "1.2500",
		LotNumber: OptionalString{Set: true, Value: strPointer("LOT-A")},
		ExpirationDate: OptionalString{Set: true, Value: strPointer("2030-01-01")},
	})
	if err != nil {
		t.Fatalf("Receive() error = %v", err)
	}
	if movement.MovementType != "receive" || movement.ToLocationID == nil || *movement.ToLocationID != fixture.locationAID {
		t.Fatalf("movement = %#v, want receive into locationA", movement)
	}
	if balance.Quantity != "50.000" || balance.LotID == nil {
		t.Fatalf("balance = %#v, want quantity 50.000 with a lot", balance)
	}

	var costLayerCount int
	if err := fixture.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cost_layers WHERE source_movement_id = $1`, movement.ID,
	).Scan(&costLayerCount); err != nil {
		t.Fatalf("count cost layers error = %v", err)
	}
	if costLayerCount != 1 {
		t.Fatalf("costLayerCount = %d, want 1", costLayerCount)
	}

	missingLocation := "00000000-0000-0000-0000-000000000000"
	_, _, err = repository.Receive(ctx, fixture.actorID, nil, ReceiveInput{
		LocationID: missingLocation, ProductID: fixture.trackedProduct,
		Quantity: "5.000", UnitCost: "1.0000",
		LotNumber: OptionalString{Set: true, Value: strPointer("LOT-B")},
	})
	if !errors.Is(err, ErrLocationNotFound) {
		t.Fatalf("Receive(missing location) error = %v, want ErrLocationNotFound", err)
	}

	otherWarehouse := "22222222-2222-2222-2222-222222222222"
	_, _, err = repository.Receive(ctx, fixture.actorID, &otherWarehouse, ReceiveInput{
		LocationID: fixture.locationAID, ProductID: fixture.trackedProduct,
		Quantity: "5.000", UnitCost: "1.0000",
		LotNumber: OptionalString{Set: true, Value: strPointer("LOT-C")},
	})
	if !errors.Is(err, ErrWarehouseMismatch) {
		t.Fatalf("Receive(wrong warehouse actor) error = %v, want ErrWarehouseMismatch", err)
	}

	// --- Pick (FEFO physical, FIFO cost) ---

	// Second lot at the same location, expiring sooner than LOT-A, received later.
	_, _, err = repository.Receive(ctx, fixture.actorID, nil, ReceiveInput{
		LocationID: fixture.locationAID, ProductID: fixture.trackedProduct,
		Quantity: "30.000", UnitCost: "2.0000",
		LotNumber:      OptionalString{Set: true, Value: strPointer("LOT-EXPIRES-SOON")},
		ExpirationDate: OptionalString{Set: true, Value: strPointer("2026-09-01")},
	})
	if err != nil {
		t.Fatalf("Receive(second lot) error = %v", err)
	}

	// Pick 60 units with no lot override: FEFO must exhaust the sooner-expiring
	// lot (30 units) before drawing 30 more from LOT-A.
	movements, err := repository.Pick(ctx, fixture.actorID, nil, PickInput{
		LocationID: fixture.locationAID, ProductID: fixture.trackedProduct, Quantity: "60.000",
	})
	if err != nil {
		t.Fatalf("Pick() error = %v", err)
	}
	if len(movements) != 2 {
		t.Fatalf("len(movements) = %d, want 2 (split across both lots)", len(movements))
	}
	if movements[0].Quantity != "30.000" {
		t.Fatalf("movements[0].Quantity = %s, want 30.000 from the sooner-expiring lot first", movements[0].Quantity)
	}
	if movements[1].Quantity != "30.000" {
		t.Fatalf("movements[1].Quantity = %s, want 30.000 remaining from LOT-A", movements[1].Quantity)
	}

	var remainingOnEarliestReceivedLayer, remainingOnSecondLayer string
	if err := fixture.pool.QueryRow(ctx, `
		SELECT remaining_quantity::text FROM cost_layers
		WHERE product_id = $1 ORDER BY received_at ASC, id ASC LIMIT 1`, fixture.trackedProduct,
	).Scan(&remainingOnEarliestReceivedLayer); err != nil {
		t.Fatalf("read first cost layer error = %v", err)
	}
	if remainingOnEarliestReceivedLayer != "0.000" {
		t.Fatalf("first-received cost layer remaining = %s, want 0.000 (FIFO drains it first regardless of FEFO lot order)", remainingOnEarliestReceivedLayer)
	}
	if err := fixture.pool.QueryRow(ctx, `
		SELECT remaining_quantity::text FROM cost_layers
		WHERE product_id = $1 ORDER BY received_at ASC, id ASC OFFSET 1 LIMIT 1`, fixture.trackedProduct,
	).Scan(&remainingOnSecondLayer); err != nil {
		t.Fatalf("read second cost layer error = %v", err)
	}
	if remainingOnSecondLayer != "20.000" {
		t.Fatalf("second cost layer remaining = %s, want 20.000 (50 - 30 already picked FIFO)", remainingOnSecondLayer)
	}

	_, err = repository.Pick(ctx, fixture.actorID, nil, PickInput{
		LocationID: fixture.locationAID, ProductID: fixture.trackedProduct, Quantity: "1000.000",
	})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("Pick(too much) error = %v, want ErrInsufficientStock", err)
	}

	// --- Transfer ---

	_, _, _, err = repository.Transfer(ctx, fixture.actorID, nil, TransferInput{
		ProductID: fixture.plainProduct, Quantity: "5.000",
		FromLocationID: fixture.locationAID, ToLocationID: fixture.locationBID,
	})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("Transfer(no stock yet) error = %v, want ErrInsufficientStock", err)
	}

	_, _, err = repository.Receive(ctx, fixture.actorID, nil, ReceiveInput{
		LocationID: fixture.locationAID, ProductID: fixture.plainProduct,
		Quantity: "20.000", UnitCost: "0.5000",
	})
	if err != nil {
		t.Fatalf("Receive(plain product) error = %v", err)
	}

	transferMovement, sourceBalance, destinationBalance, err := repository.Transfer(ctx, fixture.actorID, nil, TransferInput{
		ProductID: fixture.plainProduct, Quantity: "5.000",
		FromLocationID: fixture.locationAID, ToLocationID: fixture.locationBID,
	})
	if err != nil {
		t.Fatalf("Transfer() error = %v", err)
	}
	if transferMovement.MovementType != "transfer" {
		t.Fatalf("transferMovement.MovementType = %s, want transfer", transferMovement.MovementType)
	}
	if sourceBalance.Quantity != "15.000" {
		t.Fatalf("sourceBalance.Quantity = %s, want 15.000", sourceBalance.Quantity)
	}
	if destinationBalance.Quantity != "5.000" {
		t.Fatalf("destinationBalance.Quantity = %s, want 5.000", destinationBalance.Quantity)
	}

	// --- Adjust ---

	increaseMovement, increasedBalance, err := repository.Adjust(ctx, fixture.actorID, nil, AdjustInput{
		LocationID: fixture.locationBID, ProductID: fixture.plainProduct,
		Direction: "increase", Quantity: "3.000",
	})
	if err != nil {
		t.Fatalf("Adjust(increase) error = %v", err)
	}
	if increaseMovement.ToLocationID == nil || *increaseMovement.ToLocationID != fixture.locationBID {
		t.Fatalf("increaseMovement = %#v, want to_location_id = locationB", increaseMovement)
	}
	if increasedBalance.Quantity != "8.000" {
		t.Fatalf("increasedBalance.Quantity = %s, want 8.000 (5 transferred in + 3 adjusted in)", increasedBalance.Quantity)
	}

	decreaseMovement, decreasedBalance, err := repository.Adjust(ctx, fixture.actorID, nil, AdjustInput{
		LocationID: fixture.locationBID, ProductID: fixture.plainProduct,
		Direction: "decrease", Quantity: "2.000",
	})
	if err != nil {
		t.Fatalf("Adjust(decrease) error = %v", err)
	}
	if decreaseMovement.FromLocationID == nil || *decreaseMovement.FromLocationID != fixture.locationBID {
		t.Fatalf("decreaseMovement = %#v, want from_location_id = locationB", decreaseMovement)
	}
	if decreasedBalance.Quantity != "6.000" {
		t.Fatalf("decreasedBalance.Quantity = %s, want 6.000", decreasedBalance.Quantity)
	}

	_, _, err = repository.Adjust(ctx, fixture.actorID, nil, AdjustInput{
		LocationID: fixture.locationBID, ProductID: fixture.plainProduct,
		Direction: "decrease", Quantity: "1000.000",
	})
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("Adjust(decrease too much) error = %v, want ErrInsufficientStock", err)
	}

	// --- ListMovements / GetMovement ---

	allMovements, err := repository.ListMovements(ctx, MovementListFilter{Limit: 20, ProductID: &fixture.plainProduct})
	if err != nil {
		t.Fatalf("ListMovements() error = %v", err)
	}
	if len(allMovements) < 3 {
		t.Fatalf("len(allMovements) = %d, want at least 3 (receive, transfer, 2x adjust)", len(allMovements))
	}

	fetched, err := repository.GetMovement(ctx, transferMovement.ID)
	if err != nil || fetched.ID != transferMovement.ID {
		t.Fatalf("GetMovement() = %#v, %v, want transferMovement", fetched, err)
	}

	missingMovement := "00000000-0000-0000-0000-000000000000"
	if _, err := repository.GetMovement(ctx, missingMovement); !errors.Is(err, ErrMovementNotFound) {
		t.Fatalf("GetMovement(missing) error = %v, want ErrMovementNotFound", err)
	}

	// --- FR-19: picking an already-expired lot is allowed, but flagged ---

	_, _, err = repository.Receive(ctx, fixture.actorID, nil, ReceiveInput{
		LocationID: fixture.locationBID, ProductID: fixture.trackedProduct,
		Quantity: "10.000", UnitCost: "1.0000",
		LotNumber:      OptionalString{Set: true, Value: strPointer("LOT-ALREADY-EXPIRED")},
		ExpirationDate: OptionalString{Set: true, Value: strPointer("2020-01-01")},
	})
	if err != nil {
		t.Fatalf("Receive(expired lot) error = %v", err)
	}

	expiredPickMovements, err := repository.Pick(ctx, fixture.actorID, nil, PickInput{
		LocationID: fixture.locationBID, ProductID: fixture.trackedProduct, Quantity: "4.000",
	})
	if err != nil {
		t.Fatalf("Pick(expired lot) error = %v, want success (FR-19 allows it with an audit flag)", err)
	}
	if len(expiredPickMovements) != 1 {
		t.Fatalf("len(expiredPickMovements) = %d, want 1", len(expiredPickMovements))
	}

	var auditCount int
	var auditAction, auditEntityType string
	if err := fixture.pool.QueryRow(ctx, `
		SELECT COUNT(*), MAX(action), MAX(entity_type) FROM audit_records
		WHERE entity_id = $1`, expiredPickMovements[0].ID,
	).Scan(&auditCount, &auditAction, &auditEntityType); err != nil {
		t.Fatalf("query audit_records error = %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("auditCount = %d, want 1 audit_records row for the expired-lot pick", auditCount)
	}
	if auditAction != "EXPIRED_LOT_PICK" || auditEntityType != "stock_movements" {
		t.Fatalf("audit action/entity_type = %s/%s, want EXPIRED_LOT_PICK/stock_movements", auditAction, auditEntityType)
	}

	// A non-expired pick must not create an audit_records row.
	nonExpiredPickMovements, err := repository.Pick(ctx, fixture.actorID, nil, PickInput{
		LocationID: fixture.locationAID, ProductID: fixture.plainProduct, Quantity: "1.000",
	})
	if err != nil {
		t.Fatalf("Pick(non-expired) error = %v", err)
	}
	var nonExpiredAuditCount int
	if err := fixture.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_records WHERE entity_id = $1`, nonExpiredPickMovements[0].ID,
	).Scan(&nonExpiredAuditCount); err != nil {
		t.Fatalf("query audit_records error = %v", err)
	}
	if nonExpiredAuditCount != 0 {
		t.Fatalf("nonExpiredAuditCount = %d, want 0 (no false-positive audit flag)", nonExpiredAuditCount)
	}
}
