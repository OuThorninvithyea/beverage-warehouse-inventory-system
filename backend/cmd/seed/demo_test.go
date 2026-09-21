package main

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/catalog"
)

// seedReferenceTime keeps the ledger assertions independent of the wall clock.
var seedReferenceTime = time.Date(2026, time.September, 22, 12, 0, 0, 0, time.UTC)

func TestDemoBarcodesPassTheAPIValidator(t *testing.T) {
	seen := map[string]string{}

	for _, product := range demoProducts {
		if product.barcode == "" {
			continue
		}
		if !catalog.ValidateBarcode(product.barcode) {
			t.Errorf("product %s barcode %q is not a valid EAN-13 or UPC-A value",
				product.sku, product.barcode)
		}
		if owner, duplicate := seen[product.barcode]; duplicate {
			t.Errorf("product %s reuses barcode %q already used by %s", product.sku, product.barcode, owner)
		}
		seen[product.barcode] = product.sku
	}

	for _, location := range demoLocations {
		if location.barcode == "" {
			continue
		}
		if !catalog.ValidateBarcode(location.barcode) {
			t.Errorf("location %s/%s barcode %q is not a valid EAN-13 or UPC-A value",
				location.warehouse, location.code, location.barcode)
		}
		if owner, duplicate := seen[location.barcode]; duplicate {
			t.Errorf("location %s/%s reuses barcode %q already used by %s",
				location.warehouse, location.code, location.barcode, owner)
		}
		seen[location.barcode] = location.warehouse + "/" + location.code
	}
}

// Lists default to 20 rows per page, so the dataset has to be bigger than that
// for anyone to exercise pagination by hand.
func TestDemoDataExceedsOneDefaultPage(t *testing.T) {
	const defaultPageSize = 20

	if len(demoProducts) <= defaultPageSize {
		t.Errorf("len(demoProducts) = %d, want more than one %d-row page", len(demoProducts), defaultPageSize)
	}
	if len(demoUsers) <= defaultPageSize/2 {
		t.Errorf("len(demoUsers) = %d, want enough users to fill the user list", len(demoUsers))
	}
	if movements := demoMovements(); len(movements) <= defaultPageSize*2 {
		t.Errorf("len(demoMovements()) = %d, want at least two full pages of history", len(movements))
	}
}

// Every screen has an awkward case behind it; this locks in that the fixtures
// still cover them.
func TestDemoDataCoversTheEdgeCases(t *testing.T) {
	var inactiveProducts, unbarcoded, uncategorised, untracked int
	for _, product := range demoProducts {
		if product.inactive {
			inactiveProducts++
		}
		if product.barcode == "" {
			unbarcoded++
		}
		if product.category == "" {
			uncategorised++
		}
		if product.untracked {
			untracked++
		}
	}
	if inactiveProducts == 0 || unbarcoded == 0 || uncategorised == 0 || untracked == 0 {
		t.Errorf("products: inactive=%d unbarcoded=%d uncategorised=%d untracked=%d, want at least one of each",
			inactiveProducts, unbarcoded, uncategorised, untracked)
	}

	var inactiveUsers, warehouseless int
	roles := map[string]int{}
	for _, user := range demoUsers {
		roles[user.role]++
		if user.inactive {
			inactiveUsers++
		}
		if user.warehouse == "" {
			warehouseless++
		}
	}
	for _, role := range []string{"warehouse_manager", "picker", "viewer"} {
		if roles[role] == 0 {
			t.Errorf("no seeded user has the %s role", role)
		}
	}
	if inactiveUsers == 0 || warehouseless == 0 {
		t.Errorf("users: inactive=%d unscoped=%d, want at least one of each", inactiveUsers, warehouseless)
	}

	var inactiveWarehouses int
	for _, warehouse := range demoWarehouses {
		if warehouse.inactive {
			inactiveWarehouses++
		}
	}
	if inactiveWarehouses == 0 {
		t.Error("no inactive warehouse: deactivation cannot be reviewed in the UI")
	}

	var inactiveLocations, notPickable int
	for _, location := range demoLocations {
		if location.inactive {
			inactiveLocations++
		}
		if location.notPickable {
			notPickable++
		}
	}
	if inactiveLocations == 0 || notPickable == 0 {
		t.Errorf("locations: inactive=%d notPickable=%d, want at least one of each", inactiveLocations, notPickable)
	}

	var inactiveCategories, nested int
	for _, category := range demoCategories {
		if category.inactive {
			inactiveCategories++
		}
		if category.parent != "" {
			nested++
		}
	}
	if inactiveCategories == 0 || nested == 0 {
		t.Errorf("categories: inactive=%d nested=%d, want at least one of each", inactiveCategories, nested)
	}

	var noExpiry, expired int
	for _, lot := range demoLots() {
		if lot.noExpiry {
			noExpiry++
		}
		if !lot.noExpiry && lot.expiresInDays < 0 {
			expired++
		}
	}
	if noExpiry == 0 || expired == 0 {
		t.Errorf("lots: noExpiry=%d expired=%d, want at least one of each", noExpiry, expired)
	}
}

func TestDemoFixturesReferenceSeededRows(t *testing.T) {
	locations := map[string]bool{}
	for _, location := range demoLocations {
		locations[location.warehouse+"/"+location.code] = true
	}
	products := map[string]demoProduct{}
	for _, product := range demoProducts {
		products[product.sku] = product
	}
	lots := map[string]bool{}
	for _, lot := range demoLots() {
		if _, found := products[lot.sku]; !found {
			t.Errorf("lot %s references unknown product %s", lot.number, lot.sku)
		}
		lots[lot.sku+"/"+lot.number] = true
	}
	users := map[string]bool{}
	for _, user := range demoUsers {
		if !user.inactive {
			users[user.email] = true
		}
	}

	for _, movement := range demoMovements() {
		product, found := products[movement.sku]
		if !found {
			t.Errorf("movement %s references unknown product %s", movement.reference, movement.sku)
			continue
		}
		if product.inactive {
			t.Errorf("movement %s moves stock of the inactive product %s", movement.reference, movement.sku)
		}
		if !product.untracked && movement.lot == "" {
			t.Errorf("movement %s must carry a lot for lot-tracked product %s", movement.reference, movement.sku)
		}
		if product.untracked && movement.lot != "" {
			t.Errorf("movement %s sets a lot on the untracked product %s", movement.reference, movement.sku)
		}
		if movement.lot != "" && !lots[movement.sku+"/"+movement.lot] {
			t.Errorf("movement %s references unknown lot %s", movement.reference, movement.lot)
		}
		// An inactive user cannot perform work, so it must not appear as an actor.
		if !users[movement.actor] {
			t.Errorf("movement %s references unknown or inactive actor %s", movement.reference, movement.actor)
		}
		for _, location := range []string{movement.from, movement.to} {
			if location == "" {
				continue
			}
			if !locations[location] {
				t.Errorf("movement %s references unknown location %s", movement.reference, location)
			}
		}
	}
}

// Stock must never sit in a location the warehouse has retired, or in one
// that cannot be picked from, or the UI shows unreachable inventory.
func TestDemoStockSitsInUsableLocations(t *testing.T) {
	usable := map[string]demoLocation{}
	for _, location := range demoLocations {
		usable[location.warehouse+"/"+location.code] = location
	}

	plan, err := buildLedger(demoMovements(), seedReferenceTime)
	if err != nil {
		t.Fatalf("buildLedger error = %v", err)
	}

	for key, quantity := range plan.balances {
		if !quantity.IsPositive() {
			continue
		}
		location := usable[key.location]
		if location.inactive {
			t.Errorf("%s holds %s of %s but the location is inactive", key.location, quantity, key.sku)
		}
		if location.notPickable && key.sku != "PKG-CRATE-24" {
			t.Errorf("%s holds %s of %s but the location is not pickable", key.location, quantity, key.sku)
		}
	}
}

func TestDemoLedgerIsConsistent(t *testing.T) {
	plan, err := buildLedger(demoMovements(), seedReferenceTime)
	if err != nil {
		t.Fatalf("buildLedger error = %v", err)
	}

	for key, quantity := range plan.balances {
		if quantity.IsNegative() {
			t.Errorf("balance %s/%s/%s = %s, want a non-negative quantity",
				key.location, key.sku, key.lot, quantity)
		}
	}

	for key, reservedText := range demoReservations {
		reserved, err := decimal.NewFromString(reservedText)
		if err != nil {
			t.Fatalf("parse reservation for %s/%s: %v", key.location, key.sku, err)
		}
		quantity, found := plan.balances[key]
		if !found {
			t.Errorf("reservation %s/%s/%s has no seeded balance", key.location, key.sku, key.lot)
			continue
		}
		if reserved.GreaterThan(quantity) {
			t.Errorf("reservation %s/%s/%s = %s exceeds the balance %s",
				key.location, key.sku, key.lot, reserved, quantity)
		}
	}

	for _, layer := range plan.layers {
		if layer.remaining.IsNegative() || layer.remaining.GreaterThan(layer.original) {
			t.Errorf("cost layer for %s has remaining %s outside 0..%s",
				layer.sku, layer.remaining, layer.original)
		}
	}
}

func TestDemoLedgerCoversTheKeyScenarios(t *testing.T) {
	movements := demoMovements()

	kinds := map[string]int{}
	warehouses := map[string]int{}
	var expiredLotPicks, crossWarehouseTransfers, intraWarehouseTransfers int

	expiry := map[string]demoLot{}
	for _, lot := range demoLots() {
		expiry[lot.sku+"/"+lot.number] = lot
	}

	for _, movement := range movements {
		kinds[movement.kind]++
		for _, location := range []string{movement.from, movement.to} {
			if location == "" {
				continue
			}
			warehouse, err := warehouseOf(location)
			if err != nil {
				t.Fatalf("warehouseOf(%q) error = %v", location, err)
			}
			warehouses[warehouse]++
		}

		if movement.kind == "pick" {
			if lot, tracked := expiry[movement.sku+"/"+movement.lot]; tracked &&
				!lot.noExpiry && lot.expiresInDays < -movement.daysAgo {
				expiredLotPicks++
			}
		}
		if movement.kind == "transfer" {
			from, _ := warehouseOf(movement.from)
			to, _ := warehouseOf(movement.to)
			if from == to {
				intraWarehouseTransfers++
			} else {
				crossWarehouseTransfers++
			}
		}
	}

	for _, kind := range []string{"receive", "pick", "transfer", "adjust"} {
		if kinds[kind] == 0 {
			t.Errorf("the demo ledger has no %s movement", kind)
		}
	}
	if expiredLotPicks == 0 {
		t.Error("the demo ledger has no expired-lot pick to exercise the FR-19 audit flag")
	}
	if crossWarehouseTransfers == 0 {
		t.Error("the demo ledger has no cross-warehouse transfer to move cost layers")
	}
	if intraWarehouseTransfers == 0 {
		t.Error("the demo ledger has no intra-warehouse transfer (putaway) movement")
	}
	for _, warehouse := range []string{warehousePhnomPenh, warehouseSiemReap, warehouseBattambang} {
		if warehouses[warehouse] == 0 {
			t.Errorf("warehouse %s has no movements, so its filters show nothing", warehouse)
		}
	}

	plan, err := buildLedger(movements, seedReferenceTime)
	if err != nil {
		t.Fatalf("buildLedger error = %v", err)
	}

	// Expiry alerts (FR-12) need stock in each bucket.
	var expiredStock, nearExpiryStock int
	locationsPerProduct := map[string]map[string]bool{}
	for key, quantity := range plan.balances {
		if !quantity.IsPositive() {
			continue
		}
		if locationsPerProduct[key.sku] == nil {
			locationsPerProduct[key.sku] = map[string]bool{}
		}
		locationsPerProduct[key.sku][key.location] = true

		lot, tracked := expiry[key.sku+"/"+key.lot]
		if !tracked || lot.noExpiry {
			continue
		}
		switch {
		case lot.expiresInDays < 0:
			expiredStock++
		case lot.expiresInDays <= 30:
			nearExpiryStock++
		}
	}
	if expiredStock == 0 {
		t.Error("no expired stock remains on hand, so the expiry alert list is empty")
	}
	if nearExpiryStock == 0 {
		t.Error("no balance expires within 30 days, so expiry alerts have nothing to warn about")
	}

	// Picking is location-scoped, so at least some products must be stocked in
	// more than one place for transfers and multi-location picks to be testable.
	multiLocation := 0
	for _, locations := range locationsPerProduct {
		if len(locations) > 1 {
			multiLocation++
		}
	}
	if multiLocation < 5 {
		t.Errorf("only %d products are stocked in more than one location, want at least 5", multiLocation)
	}
}

func TestBuildLedgerRejectsAnOverdraw(t *testing.T) {
	movements := []demoMovement{
		{kind: "receive", sku: "BEV-COLA-330", lot: "L-COLA-2512", to: "PP-CENTRAL/A-01-01",
			quantity: "10", unitCost: "9.20", daysAgo: 2, reference: "TEST-RCV", actor: userManagerPP},
		{kind: "pick", sku: "BEV-COLA-330", lot: "L-COLA-2512", from: "PP-CENTRAL/A-01-01",
			quantity: "11", daysAgo: 1, reference: "TEST-PCK", actor: userPickerPP},
	}

	if _, err := buildLedger(movements, seedReferenceTime); err == nil {
		t.Fatal("buildLedger error = nil, want an overdraw failure")
	}
}

func TestBuildLedgerMovesCostAcrossWarehouses(t *testing.T) {
	movements := []demoMovement{
		{kind: "receive", sku: "BEV-COLA-330", lot: "L-COLA-2512", to: "PP-CENTRAL/A-01-01",
			quantity: "100", unitCost: "9.00", daysAgo: 3, reference: "TEST-RCV", actor: userManagerPP},
		{kind: "transfer", sku: "BEV-COLA-330", lot: "L-COLA-2512", from: "PP-CENTRAL/A-01-01",
			to: "SR-DEPOT/A-01-01", quantity: "40", daysAgo: 1, reference: "TEST-TRF", actor: userManagerPP},
	}

	plan, err := buildLedger(movements, seedReferenceTime)
	if err != nil {
		t.Fatalf("buildLedger error = %v", err)
	}

	source := plan.balances[ledgerKey{location: "PP-CENTRAL/A-01-01", sku: "BEV-COLA-330", lot: "L-COLA-2512"}]
	destination := plan.balances[ledgerKey{location: "SR-DEPOT/A-01-01", sku: "BEV-COLA-330", lot: "L-COLA-2512"}]
	if !source.Equal(decimal.NewFromInt(60)) {
		t.Errorf("source balance = %s, want 60", source)
	}
	if !destination.Equal(decimal.NewFromInt(40)) {
		t.Errorf("destination balance = %s, want 40", destination)
	}

	if len(plan.layers) != 2 {
		t.Fatalf("len(layers) = %d, want 2", len(plan.layers))
	}
	if !plan.layers[0].remaining.Equal(decimal.NewFromInt(60)) {
		t.Errorf("source layer remaining = %s, want 60", plan.layers[0].remaining)
	}
	if plan.layers[1].warehouse != "SR-DEPOT" {
		t.Errorf("destination layer warehouse = %s, want SR-DEPOT", plan.layers[1].warehouse)
	}
	if !plan.layers[1].unitCost.Equal(decimal.RequireFromString("9")) {
		t.Errorf("destination layer unit cost = %s, want 9", plan.layers[1].unitCost)
	}
}
