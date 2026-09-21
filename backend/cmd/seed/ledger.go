package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// ledgerKey identifies one inventory_balances row. lot is empty for products
// that are not lot tracked, which matches the NULL lot_id in the table.
type ledgerKey struct {
	location string
	sku      string
	lot      string
}

// costLayerPlan is a FIFO layer that will become one cost_layers row. It points
// back at the movement that opened it, because cost_layers.source_movement_id
// is NOT NULL and unique.
type costLayerPlan struct {
	movementIndex int
	warehouse     string
	sku           string
	lot           string
	original      decimal.Decimal
	remaining     decimal.Decimal
	unitCost      decimal.Decimal
	receivedAt    time.Time
}

// ledgerPlan is the derived end state of the demo movements.
type ledgerPlan struct {
	order    []int // movement indexes, oldest first
	balances map[ledgerKey]decimal.Decimal
	layers   []*costLayerPlan
}

func warehouseOf(location string) (string, error) {
	warehouse, _, found := strings.Cut(location, "/")
	if !found || warehouse == "" {
		return "", fmt.Errorf("location %q is not in warehouse/code form", location)
	}
	return warehouse, nil
}

// buildLedger replays the demo movements in chronological order and derives the
// balances and FIFO cost layers they imply. Cost layers are consumed per
// warehouse and product, exactly like the inventory repository does at runtime,
// so the seeded valuation matches what the API would have produced.
func buildLedger(movements []demoMovement, now time.Time) (*ledgerPlan, error) {
	order := make([]int, len(movements))
	for index := range movements {
		order[index] = index
	}
	sort.SliceStable(order, func(left, right int) bool {
		return movements[order[left]].daysAgo > movements[order[right]].daysAgo
	})

	plan := &ledgerPlan{order: order, balances: map[ledgerKey]decimal.Decimal{}}

	for _, index := range order {
		movement := movements[index]
		quantity, err := decimal.NewFromString(movement.quantity)
		if err != nil {
			return nil, fmt.Errorf("movement %s: parse quantity: %w", movement.reference, err)
		}
		if !quantity.IsPositive() {
			return nil, fmt.Errorf("movement %s: quantity must be positive", movement.reference)
		}
		occurredAt := now.AddDate(0, 0, -movement.daysAgo)

		switch movement.kind {
		case "receive":
			if err := plan.credit(movement, quantity); err != nil {
				return nil, err
			}
			if err := plan.openLayer(index, movement, movement.to, quantity, movement.unitCost, occurredAt); err != nil {
				return nil, err
			}
		case "pick":
			if err := plan.debit(movement, quantity); err != nil {
				return nil, err
			}
			if _, err := plan.consume(movement, movement.from, quantity); err != nil {
				return nil, err
			}
		case "transfer":
			if err := plan.debit(movement, quantity); err != nil {
				return nil, err
			}
			if err := plan.credit(movement, quantity); err != nil {
				return nil, err
			}
			source, err := warehouseOf(movement.from)
			if err != nil {
				return nil, err
			}
			destination, err := warehouseOf(movement.to)
			if err != nil {
				return nil, err
			}
			if source == destination {
				break
			}
			// A cross-warehouse transfer moves value as well as stock: drain the
			// source layers and reopen a single layer at the weighted average of
			// what was consumed.
			consumedCost, err := plan.consume(movement, movement.from, quantity)
			if err != nil {
				return nil, err
			}
			unitCost := consumedCost.Div(quantity).Round(4)
			if err := plan.openLayer(index, movement, movement.to, quantity, unitCost.String(), occurredAt); err != nil {
				return nil, err
			}
		case "adjust":
			switch {
			case movement.to != "" && movement.from == "":
				if err := plan.credit(movement, quantity); err != nil {
					return nil, err
				}
				if err := plan.openLayer(index, movement, movement.to, quantity, movement.unitCost, occurredAt); err != nil {
					return nil, err
				}
			case movement.from != "" && movement.to == "":
				if err := plan.debit(movement, quantity); err != nil {
					return nil, err
				}
				if _, err := plan.consume(movement, movement.from, quantity); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("movement %s: an adjustment needs exactly one location", movement.reference)
			}
		default:
			return nil, fmt.Errorf("movement %s: unknown movement type %q", movement.reference, movement.kind)
		}
	}

	return plan, nil
}

func (p *ledgerPlan) credit(movement demoMovement, quantity decimal.Decimal) error {
	if movement.to == "" {
		return fmt.Errorf("movement %s: a destination location is required", movement.reference)
	}
	key := ledgerKey{location: movement.to, sku: movement.sku, lot: movement.lot}
	p.balances[key] = p.balances[key].Add(quantity)
	return nil
}

func (p *ledgerPlan) debit(movement demoMovement, quantity decimal.Decimal) error {
	if movement.from == "" {
		return fmt.Errorf("movement %s: a source location is required", movement.reference)
	}
	key := ledgerKey{location: movement.from, sku: movement.sku, lot: movement.lot}
	remaining := p.balances[key].Sub(quantity)
	if remaining.IsNegative() {
		return fmt.Errorf("movement %s: %s would leave %s negative at %s",
			movement.reference, movement.sku, movement.lot, movement.from)
	}
	p.balances[key] = remaining
	return nil
}

func (p *ledgerPlan) openLayer(
	movementIndex int,
	movement demoMovement,
	location string,
	quantity decimal.Decimal,
	unitCost string,
	receivedAt time.Time,
) error {
	if unitCost == "" {
		return fmt.Errorf("movement %s: a unit cost is required to open a cost layer", movement.reference)
	}
	cost, err := decimal.NewFromString(unitCost)
	if err != nil {
		return fmt.Errorf("movement %s: parse unit cost: %w", movement.reference, err)
	}
	warehouse, err := warehouseOf(location)
	if err != nil {
		return err
	}
	p.layers = append(p.layers, &costLayerPlan{
		movementIndex: movementIndex,
		warehouse:     warehouse,
		sku:           movement.sku,
		lot:           movement.lot,
		original:      quantity,
		remaining:     quantity,
		unitCost:      cost,
		receivedAt:    receivedAt,
	})
	return nil
}

// consume drains FIFO layers for the warehouse and product and returns the
// total value taken out.
func (p *ledgerPlan) consume(
	movement demoMovement,
	location string,
	quantity decimal.Decimal,
) (decimal.Decimal, error) {
	warehouse, err := warehouseOf(location)
	if err != nil {
		return decimal.Zero, err
	}

	outstanding := quantity
	consumedCost := decimal.Zero
	for _, layer := range p.layers {
		if outstanding.IsZero() {
			break
		}
		if layer.warehouse != warehouse || layer.sku != movement.sku || !layer.remaining.IsPositive() {
			continue
		}
		taken := decimal.Min(layer.remaining, outstanding)
		layer.remaining = layer.remaining.Sub(taken)
		outstanding = outstanding.Sub(taken)
		consumedCost = consumedCost.Add(taken.Mul(layer.unitCost))
	}
	if outstanding.IsPositive() {
		return decimal.Zero, fmt.Errorf("movement %s: %s has no cost layer left in %s",
			movement.reference, movement.sku, warehouse)
	}
	return consumedCost, nil
}

// sortedBalances returns the balances in a stable order so repeated seed runs
// write the same rows in the same sequence.
func (p *ledgerPlan) sortedBalances() []ledgerKey {
	keys := make([]ledgerKey, 0, len(p.balances))
	for key := range p.balances {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		if keys[left].location != keys[right].location {
			return keys[left].location < keys[right].location
		}
		if keys[left].sku != keys[right].sku {
			return keys[left].sku < keys[right].sku
		}
		return keys[left].lot < keys[right].lot
	})
	return keys
}
