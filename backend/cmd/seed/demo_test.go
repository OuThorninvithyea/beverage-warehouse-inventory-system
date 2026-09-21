package main

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/catalog"
)

func TestDemoProductBarcodesPassTheAPIValidator(t *testing.T) {
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
			t.Errorf("product %s reuses barcode %q already used by %s",
				product.sku, product.barcode, owner)
		}
		seen[product.barcode] = product.sku
	}
}

func TestDemoLocationBarcodesPassTheAPIValidator(t *testing.T) {
	seen := map[string]string{}
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
	for _, lot := range demoLots {
		if _, found := products[lot.sku]; !found {
			t.Errorf("lot %s references unknown product %s", lot.number, lot.sku)
		}
		lots[lot.sku+"/"+lot.number] = true
	}
	users := map[string]bool{}
	for _, user := range demoUsers {
		users[user.email] = true
	}

	for _, movement := range demoMovements {
		product, found := products[movement.sku]
		if !found {
			t.Errorf("movement %s references unknown product %s", movement.reference, movement.sku)
			continue
		}
		if product.lotTracked && movement.lot == "" {
			t.Errorf("movement %s must carry a lot for lot-tracked product %s",
				movement.reference, movement.sku)
		}
		if !product.lotTracked && movement.lot != "" {
			t.Errorf("movement %s sets a lot on the untracked product %s",
				movement.reference, movement.sku)
		}
		if movement.lot != "" && !lots[movement.sku+"/"+movement.lot] {
			t.Errorf("movement %s references unknown lot %s", movement.reference, movement.lot)
		}
		if !users[movement.actor] {
			t.Errorf("movement %s references unknown actor %s", movement.reference, movement.actor)
		}
		for _, location := range []string{movement.from, movement.to} {
			if location != "" && !locations[location] {
				t.Errorf("movement %s references unknown location %s", movement.reference, location)
			}
		}
	}
}

func TestDemoLedgerIsConsistent(t *testing.T) {
	now := time.Date(2026, time.September, 21, 12, 0, 0, 0, time.UTC)

	plan, err := buildLedger(demoMovements, now)
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
			t.Errorf("reservation %s/%s/%s has no seeded balance",
				key.location, key.sku, key.lot)
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

// The seeded ledger must contain the scenarios the demo is meant to show.
func TestDemoLedgerCoversTheKeyScenarios(t *testing.T) {
	now := time.Date(2026, time.September, 21, 12, 0, 0, 0, time.UTC)

	kinds := map[string]int{}
	var expiredLotPicks, crossWarehouseTransfers int
	expiry := map[string]int{}
	for _, lot := range demoLots {
		if lot.hasExpiry {
			expiry[lot.sku+"/"+lot.number] = lot.expiresInDays
		}
	}

	for _, movement := range demoMovements {
		kinds[movement.kind]++
		if movement.kind == "pick" {
			if days, tracked := expiry[movement.sku+"/"+movement.lot]; tracked && days < -movement.daysAgo {
				expiredLotPicks++
			}
		}
		if movement.kind == "transfer" {
			from, err := warehouseOf(movement.from)
			if err != nil {
				t.Fatalf("warehouseOf(%q) error = %v", movement.from, err)
			}
			to, err := warehouseOf(movement.to)
			if err != nil {
				t.Fatalf("warehouseOf(%q) error = %v", movement.to, err)
			}
			if from != to {
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

	plan, err := buildLedger(demoMovements, now)
	if err != nil {
		t.Fatalf("buildLedger error = %v", err)
	}
	var nearExpiryBalances int
	for key, quantity := range plan.balances {
		if days, tracked := expiry[key.sku+"/"+key.lot]; tracked && quantity.IsPositive() && days >= 0 && days <= 30 {
			nearExpiryBalances++
		}
	}
	if nearExpiryBalances == 0 {
		t.Error("no seeded balance expires within 30 days, so expiry alerts have nothing to show")
	}
}

func TestBuildLedgerRejectsAnOverdraw(t *testing.T) {
	now := time.Date(2026, time.September, 21, 12, 0, 0, 0, time.UTC)
	movements := []demoMovement{
		{kind: "receive", sku: "BEV-COLA-330", lot: "L-COLA-2512", to: "PP-CENTRAL/A-01-01",
			quantity: "10", unitCost: "9.20", daysAgo: 2, reference: "TEST-RCV", actor: userManager},
		{kind: "pick", sku: "BEV-COLA-330", lot: "L-COLA-2512", from: "PP-CENTRAL/A-01-01",
			quantity: "11", daysAgo: 1, reference: "TEST-PCK", actor: userPicker},
	}

	if _, err := buildLedger(movements, now); err == nil {
		t.Fatal("buildLedger error = nil, want an overdraw failure")
	}
}

func TestBuildLedgerMovesCostAcrossWarehouses(t *testing.T) {
	now := time.Date(2026, time.September, 21, 12, 0, 0, 0, time.UTC)
	movements := []demoMovement{
		{kind: "receive", sku: "BEV-COLA-330", lot: "L-COLA-2512", to: "PP-CENTRAL/A-01-01",
			quantity: "100", unitCost: "9.00", daysAgo: 3, reference: "TEST-RCV", actor: userManager},
		{kind: "transfer", sku: "BEV-COLA-330", lot: "L-COLA-2512", from: "PP-CENTRAL/A-01-01",
			to: "SR-DEPOT/A-01-01", quantity: "40", daysAgo: 1, reference: "TEST-TRF", actor: userManager},
	}

	plan, err := buildLedger(movements, now)
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
