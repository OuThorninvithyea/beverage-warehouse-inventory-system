package main

import (
	"fmt"
	"strings"
)

// The curated fixtures in demo_data.go carry the scenarios worth reading. The
// generators here add the bulk: a lot and stock for every remaining product,
// spread across locations and warehouses so picks, transfers and filters all
// have something to work with, and enough rows to push every list past its
// 20-row default page.
//
// Everything is derived from the fixture order rather than from randomness, so
// two seed runs on the same fixtures produce the same ledger.

// Products that belong in a chilled location.
func needsChilled(sku string) bool {
	for _, prefix := range []string{"DRY-", "JUI-", "COF-"} {
		if strings.HasPrefix(sku, prefix) {
			return true
		}
	}
	return false
}

var (
	ambientLocationsPP = []string{
		"PP-CENTRAL/A-01-01", "PP-CENTRAL/A-01-02",
		"PP-CENTRAL/A-02-01", "PP-CENTRAL/B-01-01",
	}
	chilledLocationsPP = []string{"PP-CENTRAL/COLD-01", "PP-CENTRAL/COLD-02"}
	ambientLocationsSR = []string{"SR-DEPOT/A-01-01", "SR-DEPOT/A-01-02"}
	chilledLocationsSR = []string{"SR-DEPOT/COLD-01"}
	ambientLocationsBB = []string{"BB-HUB/A-01-01"}
)

// Floor staff per warehouse. Picks rotate through them so movement history
// has more than one name against it and the actor filter is worth using.
var pickersByWarehouse = map[string][]string{
	warehousePhnomPenh:  {userPickerPP, userPickerPP2, "sokha.picker@bwims.local", "dara.picker@bwims.local", "veasna.picker@bwims.local"},
	warehouseSiemReap:   {userPickerSR, "bopha.picker@bwims.local", "rithy.picker@bwims.local"},
	warehouseBattambang: {userPickerBB, "chanda.picker@bwims.local"},
}

var managersByWarehouse = map[string][]string{
	warehousePhnomPenh:  {userManagerPP, "samnang.mgr@bwims.local"},
	warehouseSiemReap:   {userManagerSR, "kanya.mgr@bwims.local"},
	warehouseBattambang: {userManagerBB},
}

func warehouseManager(warehouse string) string {
	return actorAt(managersByWarehouse, warehouse, 0)
}

// actorAt spreads work deterministically across a warehouse's staff.
func actorAt(staff map[string][]string, warehouse string, index int) string {
	people, ok := staff[warehouse]
	if !ok || len(people) == 0 {
		return userManagerPP
	}
	if index < 0 {
		index = -index
	}
	return people[index%len(people)]
}

// homeLocations picks where a product is stocked. Every product gets a
// Phnom Penh location; some also get Siem Reap or Battambang so warehouse
// filters and cross-warehouse transfers have data.
func homeLocations(index int, sku string) []string {
	chilled := needsChilled(sku)

	primary := ambientLocationsPP[index%len(ambientLocationsPP)]
	if chilled {
		primary = chilledLocationsPP[index%len(chilledLocationsPP)]
	}
	locations := []string{primary}

	// Every third product is stocked in a second Phnom Penh location, which is
	// what makes intra-warehouse transfers and multi-location picks testable.
	if index%3 == 0 {
		secondary := ambientLocationsPP[(index+2)%len(ambientLocationsPP)]
		if chilled {
			secondary = chilledLocationsPP[(index+1)%len(chilledLocationsPP)]
		}
		if secondary != primary {
			locations = append(locations, secondary)
		}
	}
	// Every fourth product also lives in Siem Reap, every seventh in Battambang.
	if index%4 == 0 {
		if chilled {
			locations = append(locations, chilledLocationsSR[0])
		} else {
			locations = append(locations, ambientLocationsSR[index%len(ambientLocationsSR)])
		}
	}
	if index%7 == 0 && !chilled {
		locations = append(locations, ambientLocationsBB[0])
	}
	return locations
}

func curatedLotsBySKU() map[string][]demoLot {
	byProduct := map[string][]demoLot{}
	for _, lot := range curatedLots {
		byProduct[lot.sku] = append(byProduct[lot.sku], lot)
	}
	return byProduct
}

// demoLots returns the curated lots plus a generated lot for every other
// tracked product, with expiry dates spread across the year.
func demoLots() []demoLot {
	existing := curatedLotsBySKU()
	lots := append([]demoLot{}, curatedLots...)

	for index, product := range demoProducts {
		if product.untracked || len(existing[product.sku]) > 0 {
			continue
		}
		lots = append(lots, demoLot{
			sku:             product.sku,
			number:          generatedLotNumber(product.sku),
			expiresInDays:   60 + (index*37)%420,
			receivedDaysAgo: 18 + (index*13)%110,
		})
	}
	return lots
}

func generatedLotNumber(sku string) string {
	parts := strings.Split(sku, "-")
	return "L-" + strings.Join(parts[:min(2, len(parts))], "-") + "-B1"
}

// demoMovements returns the curated movements plus generated receives, picks,
// transfers and adjustments for the rest of the catalog.
func demoMovements() []demoMovement {
	movements := append([]demoMovement{}, curatedMovements...)
	existing := curatedLotsBySKU()
	sequence := 0

	nextReference := func(prefix string) string {
		sequence++
		return fmt.Sprintf("SEED-%s-%04d", prefix, 5000+sequence)
	}

	for index, product := range demoProducts {
		// Inactive products keep no stock: a deactivated SKU with a live
		// balance would be a bug, not a fixture.
		if product.inactive {
			continue
		}
		// Curated products already have their own movements.
		if len(existing[product.sku]) > 0 {
			continue
		}

		lot := ""
		if !product.untracked {
			lot = generatedLotNumber(product.sku)
		}
		unitCost := fmt.Sprintf("%.2f", 4.5+float64((index*7)%38))

		for locationIndex, location := range homeLocations(index, product.sku) {
			warehouse, err := warehouseOf(location)
			if err != nil {
				continue
			}
			manager := warehouseManager(warehouse)

			receivedDaysAgo := 130 - (index*3+locationIndex*11)%110
			quantity := 120 + (index*24+locationIndex*36)%300

			movements = append(movements, demoMovement{
				kind: "receive", sku: product.sku, lot: lot, to: location,
				quantity: fmt.Sprintf("%d", quantity), unitCost: unitCost,
				daysAgo: receivedDaysAgo, reference: nextReference("RCV"), actor: manager,
			})

			// Roughly half of the stocked locations see a pick, always well
			// after the receive and always smaller than it.
			if (index+locationIndex)%2 == 0 {
				picked := quantity / 4
				pickDaysAgo := receivedDaysAgo / 3
				movements = append(movements, demoMovement{
					kind: "pick", sku: product.sku, lot: lot, from: location,
					quantity: fmt.Sprintf("%d", picked),
					daysAgo:  pickDaysAgo, reference: nextReference("PCK"),
					actor: actorAt(pickersByWarehouse, warehouse, index+locationIndex),
				})
			}

			// Recent outbound activity, spread over the last three weeks, so the
			// dashboard charts and the velocity report cover the default 30-day
			// window rather than tailing off weeks ago. Kept small so it cannot
			// outrun what was received.
			for wave := 0; wave < 3; wave++ {
				recentDaysAgo := 1 + (index*5+locationIndex*3+wave*7)%21
				if recentDaysAgo >= receivedDaysAgo {
					continue
				}
				portion := quantity / 12
				if portion < 1 {
					continue
				}
				movements = append(movements, demoMovement{
					kind: "pick", sku: product.sku, lot: lot, from: location,
					quantity: fmt.Sprintf("%d", portion),
					daysAgo:  recentDaysAgo, reference: nextReference("PCK"),
					actor: actorAt(pickersByWarehouse, warehouse, index+wave),
				})
			}

			// A putaway transfer from the receiving dock for a few products,
			// so intra-warehouse transfers appear in the history.
			pickFace := warehouse + "/A-01-01"
			if locationIndex == 0 && index%5 == 0 && location != pickFace {
				moved := quantity / 6
				movements = append(movements, demoMovement{
					kind: "transfer", sku: product.sku, lot: lot,
					from: location, to: pickFace,
					quantity: fmt.Sprintf("%d", moved),
					daysAgo:  receivedDaysAgo / 2, reference: nextReference("TRF"),
					notes: "Replenishment to the pick face", actor: manager,
				})
			}

			// Occasional shrinkage, so adjustments are not all curated.
			if index%9 == 0 && locationIndex == 0 {
				movements = append(movements, demoMovement{
					kind: "adjust", sku: product.sku, lot: lot, from: location,
					quantity: "3", daysAgo: max(1, receivedDaysAgo/4),
					reference: nextReference("ADJ"),
					notes:     "Cycle count variance", actor: manager,
				})
			}
		}
	}

	return movements
}
