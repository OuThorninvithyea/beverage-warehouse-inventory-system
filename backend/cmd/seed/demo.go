package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OuThorninvithyea/beverage-warehouse-inventory-system/backend/internal/modules/auth"
)

// seedDemoData loads the development dataset described in demo_data.go:
// warehouses, locations, role users, catalog, lots, movements, FIFO cost
// layers and inventory balances. It runs in one transaction and is safe to
// repeat: reference data is upserted by its natural key, and the append-only
// movement ledger is written only when it is not already there.
func seedDemoData(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, password string) error {
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash demo password: %w", err)
	}

	now := time.Now().UTC()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin demo seed: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := insertWarehouses(ctx, tx); err != nil {
		return err
	}
	if err := insertLocations(ctx, tx); err != nil {
		return err
	}
	if err := insertUsers(ctx, tx, passwordHash); err != nil {
		return err
	}
	if err := insertCategories(ctx, tx); err != nil {
		return err
	}
	if err := insertProducts(ctx, tx); err != nil {
		return err
	}
	if err := insertLots(ctx, tx, now); err != nil {
		return err
	}

	seeded, err := movementsAlreadySeeded(ctx, tx)
	if err != nil {
		return err
	}
	if seeded {
		logger.Info("demo movements already present, keeping the existing ledger")
	} else if err := insertLedger(ctx, tx, now); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit demo seed: %w", err)
	}

	logger.Info("demo data ready",
		"warehouses", len(demoWarehouses),
		"locations", len(demoLocations),
		"users", len(demoUsers),
		"products", len(demoProducts),
		"lots", len(demoLots()),
		"movements", len(demoMovements()),
		"ledger_written", !seeded,
	)
	return nil
}

func insertWarehouses(ctx context.Context, tx pgx.Tx) error {
	for _, warehouse := range demoWarehouses {
		if _, err := tx.Exec(ctx, `
			INSERT INTO warehouses (code, name, address, is_active)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT ((LOWER(code))) DO UPDATE
			SET name = EXCLUDED.name,
			    address = EXCLUDED.address,
			    is_active = EXCLUDED.is_active,
			    updated_at = NOW()`,
			warehouse.code, warehouse.name, warehouse.address, !warehouse.inactive); err != nil {
			return fmt.Errorf("seed warehouse %s: %w", warehouse.code, err)
		}
	}
	return nil
}

func insertLocations(ctx context.Context, tx pgx.Tx) error {
	for _, location := range demoLocations {
		if _, err := tx.Exec(ctx, `
			INSERT INTO locations (warehouse_id, code, zone, aisle, rack, shelf, barcode, is_pickable, is_active)
			SELECT w.id, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), $8, $9
			FROM warehouses w
			WHERE LOWER(w.code) = LOWER($1)
			ON CONFLICT (warehouse_id, (LOWER(code))) DO UPDATE
			SET zone = EXCLUDED.zone,
			    aisle = EXCLUDED.aisle,
			    rack = EXCLUDED.rack,
			    shelf = EXCLUDED.shelf,
			    barcode = EXCLUDED.barcode,
			    is_pickable = EXCLUDED.is_pickable,
			    is_active = EXCLUDED.is_active,
			    updated_at = NOW()`,
			location.warehouse, location.code, location.zone, location.aisle,
			location.rack, location.shelf, location.barcode,
			!location.notPickable, !location.inactive); err != nil {
			return fmt.Errorf("seed location %s/%s: %w", location.warehouse, location.code, err)
		}
	}
	return nil
}

func insertUsers(ctx context.Context, tx pgx.Tx, passwordHash string) error {
	for _, user := range demoUsers {
		tag, err := tx.Exec(ctx, `
			INSERT INTO users (role_id, warehouse_id, email, password_hash, full_name, is_active)
			SELECT r.id, w.id, $1, $2, $3, $6
			FROM roles r
			LEFT JOIN warehouses w ON LOWER(w.code) = LOWER(NULLIF($4, ''))
			WHERE r.code = $5
			ON CONFLICT ((LOWER(email))) DO UPDATE
			SET password_hash = EXCLUDED.password_hash,
			    full_name = EXCLUDED.full_name,
			    role_id = EXCLUDED.role_id,
			    warehouse_id = EXCLUDED.warehouse_id,
			    is_active = EXCLUDED.is_active,
			    updated_at = NOW()`,
			user.email, passwordHash, user.fullName, user.warehouse, user.role, !user.inactive)
		if err != nil {
			return fmt.Errorf("seed user %s: %w", user.email, err)
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("seed user %s: role %q is missing", user.email, user.role)
		}
	}
	return nil
}

func insertCategories(ctx context.Context, tx pgx.Tx) error {
	// Roots first so a child can resolve its parent by name.
	for _, category := range demoCategories {
		if _, err := tx.Exec(ctx, `
			INSERT INTO categories (parent_id, name, is_active)
			SELECT p.id, $1, $3
			FROM (SELECT 1) AS anchor
			LEFT JOIN categories p ON LOWER(p.name) = LOWER(NULLIF($2, ''))
			ON CONFLICT ((LOWER(name))) DO UPDATE
			SET parent_id = EXCLUDED.parent_id,
			    is_active = EXCLUDED.is_active,
			    updated_at = NOW()`,
			category.name, category.parent, !category.inactive); err != nil {
			return fmt.Errorf("seed category %s: %w", category.name, err)
		}
	}
	return nil
}

func insertProducts(ctx context.Context, tx pgx.Tx) error {
	for _, product := range demoProducts {
		if _, err := tx.Exec(ctx, `
			INSERT INTO products (category_id, sku, barcode, name, unit, is_lot_tracked, is_active)
			SELECT c.id, $1, NULLIF($2, ''), $3, $4, $5, $7
			FROM (SELECT 1) AS anchor
			LEFT JOIN categories c ON LOWER(c.name) = LOWER(NULLIF($6, ''))
			ON CONFLICT ((LOWER(sku))) DO UPDATE
			SET category_id = EXCLUDED.category_id,
			    barcode = EXCLUDED.barcode,
			    name = EXCLUDED.name,
			    unit = EXCLUDED.unit,
			    is_lot_tracked = EXCLUDED.is_lot_tracked,
			    is_active = EXCLUDED.is_active,
			    updated_at = NOW()`,
			product.sku, product.barcode, product.name, product.unit,
			!product.untracked, product.category, !product.inactive); err != nil {
			return fmt.Errorf("seed product %s: %w", product.sku, err)
		}
	}
	return nil
}

func insertLots(ctx context.Context, tx pgx.Tx, now time.Time) error {
	for _, lot := range demoLots() {
		var expiration *time.Time
		if !lot.noExpiry {
			expiresOn := now.AddDate(0, 0, lot.expiresInDays)
			expiration = &expiresOn
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO lots (product_id, lot_number, expiration_date, received_at)
			SELECT p.id, $2, $3::date, $4
			FROM products p
			WHERE LOWER(p.sku) = LOWER($1)
			ON CONFLICT (product_id, lot_number) DO UPDATE
			SET expiration_date = EXCLUDED.expiration_date,
			    received_at = EXCLUDED.received_at`,
			lot.sku, lot.number, expiration, now.AddDate(0, 0, -lot.receivedDaysAgo)); err != nil {
			return fmt.Errorf("seed lot %s/%s: %w", lot.sku, lot.number, err)
		}
	}
	return nil
}

func movementsAlreadySeeded(ctx context.Context, tx pgx.Tx) (bool, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM stock_movements WHERE reference LIKE $1)`,
		demoReference+"%").Scan(&exists); err != nil {
		return false, fmt.Errorf("check seeded movements: %w", err)
	}
	return exists, nil
}

// identifiers holds the database ids of the reference rows the ledger needs,
// keyed by the lower-cased natural keys used in the fixtures.
type identifiers struct {
	warehouses map[string]string
	locations  map[string]string // "warehouse/code"
	products   map[string]string // sku
	lots       map[string]string // "sku/lot number"
	users      map[string]string // email
}

func loadIdentifiers(ctx context.Context, tx pgx.Tx) (identifiers, error) {
	queries := map[string]string{
		"warehouses": `SELECT LOWER(code), id::text FROM warehouses`,
		"locations": `SELECT LOWER(w.code) || '/' || LOWER(l.code), l.id::text
		              FROM locations l JOIN warehouses w ON w.id = l.warehouse_id`,
		"products": `SELECT LOWER(sku), id::text FROM products`,
		"lots": `SELECT LOWER(p.sku) || '/' || LOWER(l.lot_number), l.id::text
		         FROM lots l JOIN products p ON p.id = l.product_id`,
		"users": `SELECT LOWER(email), id::text FROM users`,
	}

	loaded := identifiers{}
	for name, query := range queries {
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return identifiers{}, fmt.Errorf("load %s identifiers: %w", name, err)
		}
		values := map[string]string{}
		for rows.Next() {
			var key, id string
			if err := rows.Scan(&key, &id); err != nil {
				rows.Close()
				return identifiers{}, fmt.Errorf("scan %s identifiers: %w", name, err)
			}
			values[key] = id
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return identifiers{}, fmt.Errorf("read %s identifiers: %w", name, err)
		}

		switch name {
		case "warehouses":
			loaded.warehouses = values
		case "locations":
			loaded.locations = values
		case "products":
			loaded.products = values
		case "lots":
			loaded.lots = values
		case "users":
			loaded.users = values
		}
	}
	return loaded, nil
}

func (i identifiers) lookup(set map[string]string, kind, key string) (string, error) {
	id, found := set[strings.ToLower(key)]
	if !found {
		return "", fmt.Errorf("%s %q was not seeded", kind, key)
	}
	return id, nil
}

// optionalLot returns the lot id for a fixture reference, or nil for products
// that are not lot tracked.
func (i identifiers) optionalLot(sku, lot string) (*string, error) {
	if lot == "" {
		return nil, nil
	}
	id, err := i.lookup(i.lots, "lot", sku+"/"+lot)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (i identifiers) optionalLocation(location string) (*string, error) {
	if location == "" {
		return nil, nil
	}
	id, err := i.lookup(i.locations, "location", location)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func insertLedger(ctx context.Context, tx pgx.Tx, now time.Time) error {
	movements := demoMovements()
	plan, err := buildLedger(movements, now)
	if err != nil {
		return err
	}
	ids, err := loadIdentifiers(ctx, tx)
	if err != nil {
		return err
	}

	expiryByLot := map[string]time.Time{}
	for _, lot := range demoLots() {
		if !lot.noExpiry {
			expiryByLot[strings.ToLower(lot.sku+"/"+lot.number)] = now.AddDate(0, 0, lot.expiresInDays)
		}
	}

	movementIDs := make(map[int]string, len(movements))
	for _, index := range plan.order {
		movement := movements[index]
		occurredAt := now.AddDate(0, 0, -movement.daysAgo)

		productID, err := ids.lookup(ids.products, "product", movement.sku)
		if err != nil {
			return err
		}
		actorID, err := ids.lookup(ids.users, "user", movement.actor)
		if err != nil {
			return err
		}
		lotID, err := ids.optionalLot(movement.sku, movement.lot)
		if err != nil {
			return err
		}
		fromID, err := ids.optionalLocation(movement.from)
		if err != nil {
			return err
		}
		toID, err := ids.optionalLocation(movement.to)
		if err != nil {
			return err
		}

		var unitCost *string
		if movement.unitCost != "" {
			cost := movement.unitCost
			unitCost = &cost
		}
		var notes *string
		if movement.notes != "" {
			text := movement.notes
			notes = &text
		}

		var movementID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO stock_movements
			    (movement_type, product_id, lot_id, from_location_id, to_location_id,
			     quantity, unit_cost, reference, notes, performed_by, created_at)
			VALUES ($1, $2, $3, $4, $5, $6::numeric, $7::numeric, $8, $9, $10, $11)
			RETURNING id::text`,
			movement.kind, productID, lotID, fromID, toID, movement.quantity,
			unitCost, movement.reference, notes, actorID, occurredAt,
		).Scan(&movementID); err != nil {
			return fmt.Errorf("seed movement %s: %w", movement.reference, err)
		}
		movementIDs[index] = movementID

		expiry, tracked := expiryByLot[strings.ToLower(movement.sku+"/"+movement.lot)]
		if movement.kind == "pick" && tracked && expiry.Before(occurredAt) {
			if err := recordExpiredLotPick(ctx, tx, actorID, movementID, productID,
				lotID, fromID, movement.quantity, occurredAt); err != nil {
				return err
			}
		}
	}

	for _, layer := range plan.layers {
		movement := movements[layer.movementIndex]
		warehouseID, err := ids.lookup(ids.warehouses, "warehouse", layer.warehouse)
		if err != nil {
			return err
		}
		productID, err := ids.lookup(ids.products, "product", layer.sku)
		if err != nil {
			return err
		}
		lotID, err := ids.optionalLot(layer.sku, layer.lot)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO cost_layers
			    (warehouse_id, product_id, lot_id, source_movement_id,
			     original_quantity, remaining_quantity, unit_cost, received_at)
			VALUES ($1, $2, $3, $4, $5::numeric, $6::numeric, $7::numeric, $8)`,
			warehouseID, productID, lotID, movementIDs[layer.movementIndex],
			layer.original.String(), layer.remaining.String(), layer.unitCost.String(),
			layer.receivedAt); err != nil {
			return fmt.Errorf("seed cost layer for %s: %w", movement.reference, err)
		}
	}

	for _, key := range plan.sortedBalances() {
		locationID, err := ids.lookup(ids.locations, "location", key.location)
		if err != nil {
			return err
		}
		productID, err := ids.lookup(ids.products, "product", key.sku)
		if err != nil {
			return err
		}
		lotID, err := ids.optionalLot(key.sku, key.lot)
		if err != nil {
			return err
		}
		reserved, found := demoReservations[key]
		if !found {
			reserved = "0"
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO inventory_balances (location_id, product_id, lot_id, quantity, reserved_quantity)
			VALUES ($1, $2, $3, $4::numeric, $5::numeric)
			ON CONFLICT (location_id, product_id, lot_id) DO UPDATE
			SET quantity = EXCLUDED.quantity,
			    reserved_quantity = EXCLUDED.reserved_quantity,
			    updated_at = NOW()`,
			locationID, productID, lotID, plan.balances[key].String(), reserved); err != nil {
			return fmt.Errorf("seed balance %s/%s: %w", key.location, key.sku, err)
		}
	}

	return nil
}

// recordExpiredLotPick mirrors the audit row the inventory repository writes
// when a pick draws from an expired lot (FR-19), so the seeded history shows
// the flag instead of hiding it.
func recordExpiredLotPick(
	ctx context.Context,
	tx pgx.Tx,
	actorID, movementID, productID string,
	lotID, locationID *string,
	quantity string,
	occurredAt time.Time,
) error {
	afterState, err := json.Marshal(map[string]any{
		"movement_id": movementID,
		"product_id":  productID,
		"lot_id":      lotID,
		"location_id": locationID,
		"quantity":    quantity,
	})
	if err != nil {
		return fmt.Errorf("marshal expired lot pick audit payload: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_records (actor_user_id, action, entity_type, entity_id, after_state, occurred_at)
		VALUES ($1, 'EXPIRED_LOT_PICK', 'stock_movements', $2, $3::jsonb, $4)`,
		actorID, movementID, afterState, occurredAt); err != nil {
		return fmt.Errorf("record expired lot pick audit: %w", err)
	}
	return nil
}
