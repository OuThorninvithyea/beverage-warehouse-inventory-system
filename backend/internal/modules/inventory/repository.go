package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	ErrProductNotFound          = errors.New("product was not found or is not active")
	ErrLocationNotFound         = errors.New("location was not found")
	ErrLotNotFound              = errors.New("lot was not found for this product")
	ErrInsufficientStock        = errors.New("insufficient available stock for this operation")
	ErrWarehouseMismatch        = errors.New("location does not belong to the actor's assigned warehouse")
	ErrInventoryOperationFailed = errors.New("inventory operation could not be completed")
	ErrMovementNotFound         = errors.New("movement was not found")
	ErrLotExpiryRequired        = errors.New("lot number and expiration date are required for lot-tracked products")
)

type Repository interface {
	ListBalances(ctx context.Context, filter BalanceListFilter) ([]Balance, error)
	ListLots(ctx context.Context, productID string, warehouseID *string) ([]Lot, error)
	ListExpiryAlerts(ctx context.Context, filter ExpiryAlertFilter) ([]ExpiryAlert, error)
	Receive(ctx context.Context, actorID string, actorWarehouseID *string, input ReceiveInput) (Movement, Balance, error)
	Pick(ctx context.Context, actorID string, actorWarehouseID *string, input PickInput) ([]Movement, error)
	Transfer(ctx context.Context, actorID string, actorWarehouseID *string, input TransferInput) (Movement, Balance, Balance, error)
	Adjust(ctx context.Context, actorID string, actorWarehouseID *string, input AdjustInput) (Movement, Balance, error)
	ListMovements(ctx context.Context, filter MovementListFilter) ([]Movement, error)
	GetMovement(ctx context.Context, id string) (Movement, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) Repository {
	return &PostgresRepository{pool: pool}
}

type rowScanner interface {
	Scan(...any) error
}

func scanBalance(row rowScanner) (Balance, error) {
	var balance Balance
	var quantityText, reservedText string
	err := row.Scan(&balance.ID, &balance.LocationID, &balance.ProductID, &balance.LotID,
		&quantityText, &reservedText, &balance.CreatedAt, &balance.UpdatedAt)
	if err != nil {
		return Balance{}, err
	}
	balance.Quantity = quantityText
	balance.ReservedQuantity = reservedText
	quantity, err := decimal.NewFromString(quantityText)
	if err != nil {
		return Balance{}, fmt.Errorf("parse balance quantity: %w", err)
	}
	reserved, err := decimal.NewFromString(reservedText)
	if err != nil {
		return Balance{}, fmt.Errorf("parse balance reserved quantity: %w", err)
	}
	balance.AvailableQuantity = quantity.Sub(reserved).String()
	return balance, nil
}

func scanMovement(row rowScanner) (Movement, error) {
	var movement Movement
	err := row.Scan(&movement.ID, &movement.MovementType, &movement.ProductID, &movement.LotID,
		&movement.FromLocationID, &movement.ToLocationID, &movement.Quantity,
		&movement.UnitCost, &movement.Reference, &movement.Notes, &movement.PerformedBy, &movement.CreatedAt)
	return movement, err
}

func resolveLocationWarehouse(ctx context.Context, tx pgx.Tx, locationID string) (string, error) {
	var warehouseID string
	err := tx.QueryRow(ctx, `SELECT warehouse_id::text FROM locations WHERE id = $1`, locationID).Scan(&warehouseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrLocationNotFound
	}
	if err != nil {
		return "", fmt.Errorf("resolve location warehouse: %w", err)
	}
	return warehouseID, nil
}

func checkWarehouseScope(actorWarehouseID *string, resolvedWarehouseID string) error {
	if actorWarehouseID != nil && *actorWarehouseID != resolvedWarehouseID {
		return ErrWarehouseMismatch
	}
	return nil
}

type productRef struct {
	IsLotTracked bool
}

func lockActiveProduct(ctx context.Context, tx pgx.Tx, productID string) (productRef, error) {
	var product productRef
	err := tx.QueryRow(ctx, `
		SELECT is_lot_tracked FROM products WHERE id = $1 AND is_active = TRUE FOR UPDATE`, productID,
	).Scan(&product.IsLotTracked)
	if errors.Is(err, pgx.ErrNoRows) {
		return productRef{}, ErrProductNotFound
	}
	if err != nil {
		return productRef{}, fmt.Errorf("lock active product: %w", err)
	}
	return product, nil
}

func lockLotForProduct(ctx context.Context, tx pgx.Tx, lotID, productID string) error {
	var exists string
	err := tx.QueryRow(ctx, `SELECT id::text FROM lots WHERE id = $1 AND product_id = $2`, lotID, productID).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrLotNotFound
	}
	if err != nil {
		return fmt.Errorf("validate lot: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListBalances(ctx context.Context, filter BalanceListFilter) ([]Balance, error) {
	afterTime, afterID := cursorArguments(filter.After)
	rows, err := r.pool.Query(ctx, `
		SELECT b.id::text, b.location_id::text, b.product_id::text, b.lot_id::text,
		       b.quantity::text, b.reserved_quantity::text, b.created_at, b.updated_at
		FROM inventory_balances b
		JOIN locations loc ON loc.id = b.location_id
		WHERE ($1::uuid IS NULL OR b.location_id = $1)
		  AND ($2::uuid IS NULL OR b.product_id = $2)
		  AND ($3::uuid IS NULL OR loc.warehouse_id = $3)
		  AND ($4::uuid IS NULL OR b.lot_id = $4)
		  AND ($5::timestamptz IS NULL OR (b.created_at, b.id) < ($5::timestamptz, $6::uuid))
		ORDER BY b.created_at DESC, b.id DESC
		LIMIT $7`,
		filter.LocationID, filter.ProductID, filter.WarehouseID, filter.LotID,
		afterTime, afterID, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list balances: %w", err)
	}
	defer rows.Close()

	balances := make([]Balance, 0)
	for rows.Next() {
		balance, err := scanBalance(rows)
		if err != nil {
			return nil, fmt.Errorf("scan balance: %w", err)
		}
		balances = append(balances, balance)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list balances: %w", err)
	}
	return balances, nil
}

func (r *PostgresRepository) ListLots(ctx context.Context, productID string, warehouseID *string) ([]Lot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT l.id::text, l.product_id::text, l.lot_number, l.expiration_date, l.received_at,
		       COALESCE(SUM(b.quantity - b.reserved_quantity)
		           FILTER (WHERE $2::uuid IS NULL OR loc.warehouse_id = $2), 0)::text
		FROM lots l
		LEFT JOIN inventory_balances b ON b.lot_id = l.id
		LEFT JOIN locations loc ON loc.id = b.location_id
		WHERE l.product_id = $1
		GROUP BY l.id
		ORDER BY l.expiration_date ASC NULLS LAST, l.received_at ASC`,
		productID, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("list lots: %w", err)
	}
	defer rows.Close()

	lots := make([]Lot, 0)
	for rows.Next() {
		var lot Lot
		var expirationDate *time.Time
		if err := rows.Scan(&lot.ID, &lot.ProductID, &lot.LotNumber, &expirationDate, &lot.ReceivedAt, &lot.AvailableQuantity); err != nil {
			return nil, fmt.Errorf("scan lot: %w", err)
		}
		if expirationDate != nil {
			formatted := expirationDate.Format("2006-01-02")
			lot.ExpirationDate = &formatted
		}
		lots = append(lots, lot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list lots: %w", err)
	}
	return lots, nil
}

// ListExpiryAlerts reports lots that have expired or will expire within the
// requested window and still have stock on hand (FR-12). Already-expired lots
// are always included: they are the most urgent case, and FR-19 keeps them
// pickable, so hiding them would defeat the alert.
func (r *PostgresRepository) ListExpiryAlerts(ctx context.Context, filter ExpiryAlertFilter) ([]ExpiryAlert, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id::text, p.sku, p.name,
		       l.id::text, l.lot_number, l.expiration_date,
		       (l.expiration_date - CURRENT_DATE)::int,
		       w.id::text, w.code, loc.id::text, loc.code,
		       b.quantity::text, b.reserved_quantity::text,
		       (b.quantity - b.reserved_quantity)::text
		FROM inventory_balances b
		JOIN lots l ON l.id = b.lot_id
		JOIN products p ON p.id = b.product_id
		JOIN locations loc ON loc.id = b.location_id
		JOIN warehouses w ON w.id = loc.warehouse_id
		WHERE l.expiration_date IS NOT NULL
		  AND b.quantity > 0
		  AND l.expiration_date <= CURRENT_DATE + $1::int
		  AND ($2::uuid IS NULL OR loc.warehouse_id = $2)
		ORDER BY l.expiration_date ASC, w.code ASC, loc.code ASC, p.sku ASC
		LIMIT $3`,
		*filter.WithinDays, filter.WarehouseID, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list expiry alerts: %w", err)
	}
	defer rows.Close()

	alerts := make([]ExpiryAlert, 0)
	for rows.Next() {
		var alert ExpiryAlert
		var expirationDate time.Time
		if err := rows.Scan(&alert.ProductID, &alert.SKU, &alert.ProductName,
			&alert.LotID, &alert.LotNumber, &expirationDate, &alert.DaysRemaining,
			&alert.WarehouseID, &alert.WarehouseCode, &alert.LocationID, &alert.LocationCode,
			&alert.Quantity, &alert.ReservedQuantity, &alert.AvailableQuantity); err != nil {
			return nil, fmt.Errorf("scan expiry alert: %w", err)
		}
		alert.ExpirationDate = expirationDate.Format("2006-01-02")
		alert.Status = ExpiryStatusExpiring
		if alert.DaysRemaining < 0 {
			alert.Status = ExpiryStatusExpired
		}
		alerts = append(alerts, alert)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list expiry alerts: %w", err)
	}
	return alerts, nil
}

func resolveOrCreateLot(ctx context.Context, tx pgx.Tx, productID, lotNumber string, expirationDate *string) (string, error) {
	var lotID string
	err := tx.QueryRow(ctx, `
		INSERT INTO lots (product_id, lot_number, expiration_date)
		VALUES ($1, $2, $3::date)
		ON CONFLICT (product_id, lot_number) DO UPDATE SET lot_number = EXCLUDED.lot_number
		RETURNING id::text`,
		productID, lotNumber, expirationDate,
	).Scan(&lotID)
	if err != nil {
		return "", fmt.Errorf("resolve or create lot: %w", err)
	}
	return lotID, nil
}

func upsertBalanceIncrement(ctx context.Context, tx pgx.Tx, locationID, productID string, lotID *string, delta string) (Balance, error) {
	return scanBalance(tx.QueryRow(ctx, `
		INSERT INTO inventory_balances (location_id, product_id, lot_id, quantity, created_at)
		VALUES ($1, $2, $3, $4::numeric, NOW())
		ON CONFLICT (location_id, product_id, lot_id) DO UPDATE
		    SET quantity = inventory_balances.quantity + EXCLUDED.quantity, updated_at = NOW()
		RETURNING id::text, location_id::text, product_id::text, lot_id::text,
		          quantity::text, reserved_quantity::text, created_at, updated_at`,
		locationID, productID, lotID, delta))
}

func insertMovement(
	ctx context.Context, tx pgx.Tx,
	movementType, productID string, lotID, fromLocationID, toLocationID *string,
	quantity string, unitCost, reference, notes *string, performedBy string,
) (Movement, error) {
	movement, err := scanMovement(tx.QueryRow(ctx, `
		INSERT INTO stock_movements
		    (movement_type, product_id, lot_id, from_location_id, to_location_id,
		     quantity, unit_cost, reference, notes, performed_by)
		VALUES ($1, $2, $3, $4, $5, $6::numeric, $7::numeric, $8, $9, $10)
		RETURNING id::text, movement_type, product_id::text, lot_id::text,
		          from_location_id::text, to_location_id::text, quantity::text,
		          unit_cost::text, reference, notes, performed_by::text, created_at`,
		movementType, productID, lotID, fromLocationID, toLocationID,
		quantity, unitCost, reference, notes, performedBy))
	if err != nil {
		return Movement{}, fmt.Errorf("insert movement: %w", err)
	}
	return movement, nil
}

func optionalStringPtr(value OptionalString) *string {
	if !value.Set {
		return nil
	}
	return value.Value
}

func optionalDateValue(value OptionalString) *string {
	if !value.Set {
		return nil
	}
	return value.Value
}

func (r *PostgresRepository) Receive(ctx context.Context, actorID string, actorWarehouseID *string, input ReceiveInput) (Movement, Balance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Movement{}, Balance{}, fmt.Errorf("begin receive: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	warehouseID, err := resolveLocationWarehouse(ctx, tx, input.LocationID)
	if err != nil {
		return Movement{}, Balance{}, err
	}
	if err := checkWarehouseScope(actorWarehouseID, warehouseID); err != nil {
		return Movement{}, Balance{}, err
	}

	product, err := lockActiveProduct(ctx, tx, input.ProductID)
	if err != nil {
		return Movement{}, Balance{}, err
	}
	if product.IsLotTracked && (!input.LotNumber.Set || input.LotNumber.Value == nil || strings.TrimSpace(*input.LotNumber.Value) == "" || !input.ExpirationDate.Set || input.ExpirationDate.Value == nil || strings.TrimSpace(*input.ExpirationDate.Value) == "") {
		return Movement{}, Balance{}, ErrLotExpiryRequired
	}

	var lotID *string
	if product.IsLotTracked && input.LotNumber.Set && input.LotNumber.Value != nil {
		id, err := resolveOrCreateLot(ctx, tx, input.ProductID, *input.LotNumber.Value, optionalDateValue(input.ExpirationDate))
		if err != nil {
			return Movement{}, Balance{}, err
		}
		lotID = &id
	}

	movement, err := insertMovement(ctx, tx, "receive", input.ProductID, lotID, nil, &input.LocationID,
		input.Quantity, &input.UnitCost, optionalStringPtr(input.Reference), optionalStringPtr(input.Notes), actorID)
	if err != nil {
		return Movement{}, Balance{}, err
	}

	balance, err := upsertBalanceIncrement(ctx, tx, input.LocationID, input.ProductID, lotID, input.Quantity)
	if err != nil {
		return Movement{}, Balance{}, fmt.Errorf("increment balance on receive: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO cost_layers
		    (warehouse_id, product_id, lot_id, source_movement_id, original_quantity, remaining_quantity, unit_cost, received_at)
		VALUES ($1, $2, $3, $4, $5::numeric, $5::numeric, $6::numeric, NOW())`,
		warehouseID, input.ProductID, lotID, movement.ID, input.Quantity, input.UnitCost); err != nil {
		return Movement{}, Balance{}, fmt.Errorf("open cost layer: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Movement{}, Balance{}, fmt.Errorf("commit receive: %w", err)
	}
	return movement, balance, nil
}

func (r *PostgresRepository) Pick(ctx context.Context, actorID string, actorWarehouseID *string, input PickInput) ([]Movement, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin pick: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	warehouseID, err := resolveLocationWarehouse(ctx, tx, input.LocationID)
	if err != nil {
		return nil, err
	}
	if err := checkWarehouseScope(actorWarehouseID, warehouseID); err != nil {
		return nil, err
	}
	if _, err := lockActiveProduct(ctx, tx, input.ProductID); err != nil {
		return nil, err
	}

	requested, err := decimal.NewFromString(input.Quantity)
	if err != nil {
		return nil, fmt.Errorf("parse pick quantity: %w", err)
	}

	type candidate struct {
		BalanceID  string
		LotID      *string
		Available  decimal.Decimal
		ExpiredLot bool
	}
	var candidates []candidate

	if input.LotID.Set && input.LotID.Value != nil {
		if err := lockLotForProduct(ctx, tx, *input.LotID.Value, input.ProductID); err != nil {
			return nil, err
		}
		var id, availableText string
		var expiredLot bool
		err := tx.QueryRow(ctx, `
			SELECT b.id::text, (b.quantity - b.reserved_quantity)::text,
			       (l.expiration_date IS NOT NULL AND l.expiration_date < CURRENT_DATE)
			FROM inventory_balances b
			LEFT JOIN lots l ON l.id = b.lot_id
			WHERE b.location_id = $1 AND b.product_id = $2 AND b.lot_id = $3
			FOR UPDATE OF b`, input.LocationID, input.ProductID, *input.LotID.Value,
		).Scan(&id, &availableText, &expiredLot)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInsufficientStock
		}
		if err != nil {
			return nil, fmt.Errorf("lock pick balance: %w", err)
		}
		available, err := decimal.NewFromString(availableText)
		if err != nil {
			return nil, fmt.Errorf("parse pick balance available quantity: %w", err)
		}
		candidates = append(candidates, candidate{BalanceID: id, LotID: input.LotID.Value, Available: available, ExpiredLot: expiredLot})
	} else {
		rows, err := tx.Query(ctx, `
			SELECT b.id::text, b.lot_id::text, (b.quantity - b.reserved_quantity)::text,
			       (l.expiration_date IS NOT NULL AND l.expiration_date < CURRENT_DATE)
			FROM inventory_balances b
			LEFT JOIN lots l ON l.id = b.lot_id
			WHERE b.location_id = $1 AND b.product_id = $2
			  AND (b.quantity - b.reserved_quantity) > 0
			ORDER BY l.expiration_date ASC NULLS LAST, l.received_at ASC NULLS LAST, b.id ASC
			FOR UPDATE OF b`, input.LocationID, input.ProductID)
		if err != nil {
			return nil, fmt.Errorf("query pick candidates: %w", err)
		}
		for rows.Next() {
			var c candidate
			var availableText string
			if err := rows.Scan(&c.BalanceID, &c.LotID, &availableText, &c.ExpiredLot); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan pick candidate: %w", err)
			}
			c.Available, err = decimal.NewFromString(availableText)
			if err != nil {
				rows.Close()
				return nil, fmt.Errorf("parse pick candidate available quantity: %w", err)
			}
			candidates = append(candidates, c)
		}
		rowsErr := rows.Err()
		rows.Close()
		if rowsErr != nil {
			return nil, fmt.Errorf("query pick candidates: %w", rowsErr)
		}
	}

	remaining := requested
	var movements []Movement
	for _, c := range candidates {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
		take := c.Available
		if take.GreaterThan(remaining) {
			take = remaining
		}
		if take.LessThanOrEqual(decimal.Zero) {
			continue
		}
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_balances SET quantity = quantity - $2::numeric, updated_at = NOW() WHERE id = $1`,
			c.BalanceID, take.String()); err != nil {
			return nil, fmt.Errorf("decrement pick balance: %w", err)
		}
		movement, err := insertMovement(ctx, tx, "pick", input.ProductID, c.LotID, &input.LocationID, nil,
			take.String(), nil, optionalStringPtr(input.Reference), optionalStringPtr(input.Notes), actorID)
		if err != nil {
			return nil, err
		}
		if c.ExpiredLot {
			if err := recordExpiredLotPick(ctx, tx, actorID, movement); err != nil {
				return nil, err
			}
		}
		movements = append(movements, movement)
		remaining = remaining.Sub(take)
	}

	if remaining.GreaterThan(decimal.Zero) {
		return nil, ErrInsufficientStock
	}

	if err := consumeCostLayers(ctx, tx, warehouseID, input.ProductID, requested); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit pick: %w", err)
	}
	return movements, nil
}

// recordExpiredLotPick writes an append-only audit_records entry when a pick
// draws from a lot whose expiration_date has already passed. Picking expired
// stock is allowed (FR-19: allow-with-audit-flag, a deliberate product
// decision — not blocked and not silent), so this never fails the pick
// itself; it only fails the whole transaction if the audit write itself
// errors, which would indicate a real problem worth surfacing.
func recordExpiredLotPick(ctx context.Context, tx pgx.Tx, actorID string, movement Movement) error {
	afterState, err := json.Marshal(map[string]any{
		"movement_id": movement.ID,
		"product_id":  movement.ProductID,
		"lot_id":      movement.LotID,
		"location_id": movement.FromLocationID,
		"quantity":    movement.Quantity,
	})
	if err != nil {
		return fmt.Errorf("marshal expired lot pick audit payload: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_records (actor_user_id, action, entity_type, entity_id, after_state)
		VALUES ($1, 'EXPIRED_LOT_PICK', 'stock_movements', $2, $3::jsonb)`,
		actorID, movement.ID, afterState); err != nil {
		return fmt.Errorf("record expired lot pick audit: %w", err)
	}
	return nil
}

func consumeCostLayers(ctx context.Context, tx pgx.Tx, warehouseID, productID string, quantity decimal.Decimal) error {
	rows, err := tx.Query(ctx, `
		SELECT id::text, remaining_quantity::text
		FROM cost_layers
		WHERE warehouse_id = $1 AND product_id = $2 AND remaining_quantity > 0
		ORDER BY received_at ASC, id ASC
		FOR UPDATE`, warehouseID, productID)
	if err != nil {
		return fmt.Errorf("query cost layers: %w", err)
	}

	type layer struct {
		ID        string
		Remaining decimal.Decimal
	}
	var layers []layer
	for rows.Next() {
		var l layer
		var remainingText string
		if err := rows.Scan(&l.ID, &remainingText); err != nil {
			rows.Close()
			return fmt.Errorf("scan cost layer: %w", err)
		}
		l.Remaining, err = decimal.NewFromString(remainingText)
		if err != nil {
			rows.Close()
			return fmt.Errorf("parse cost layer remaining quantity: %w", err)
		}
		layers = append(layers, l)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return fmt.Errorf("query cost layers: %w", rowsErr)
	}

	remaining := quantity
	for _, l := range layers {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
		take := l.Remaining
		if take.GreaterThan(remaining) {
			take = remaining
		}
		if _, err := tx.Exec(ctx, `
			UPDATE cost_layers SET remaining_quantity = remaining_quantity - $2::numeric WHERE id = $1`,
			l.ID, take.String()); err != nil {
			return fmt.Errorf("decrement cost layer: %w", err)
		}
		remaining = remaining.Sub(take)
	}

	if remaining.GreaterThan(decimal.Zero) {
		return ErrInventoryOperationFailed
	}
	return nil
}

func decrementBalance(ctx context.Context, tx pgx.Tx, locationID, productID string, lotID *string, amount decimal.Decimal) (Balance, error) {
	var id, quantityText, reservedText string
	err := tx.QueryRow(ctx, `
		SELECT id::text, quantity::text, reserved_quantity::text
		FROM inventory_balances
		WHERE location_id = $1 AND product_id = $2 AND lot_id IS NOT DISTINCT FROM $3
		FOR UPDATE`, locationID, productID, lotID,
	).Scan(&id, &quantityText, &reservedText)
	if errors.Is(err, pgx.ErrNoRows) {
		return Balance{}, ErrInsufficientStock
	}
	if err != nil {
		return Balance{}, fmt.Errorf("lock balance for decrement: %w", err)
	}
	quantity, err := decimal.NewFromString(quantityText)
	if err != nil {
		return Balance{}, fmt.Errorf("parse balance quantity: %w", err)
	}
	reserved, err := decimal.NewFromString(reservedText)
	if err != nil {
		return Balance{}, fmt.Errorf("parse balance reserved quantity: %w", err)
	}
	if quantity.Sub(reserved).LessThan(amount) {
		return Balance{}, ErrInsufficientStock
	}
	return scanBalance(tx.QueryRow(ctx, `
		UPDATE inventory_balances
		SET quantity = quantity - $2::numeric, updated_at = NOW()
		WHERE id = $1
		RETURNING id::text, location_id::text, product_id::text, lot_id::text,
		          quantity::text, reserved_quantity::text, created_at, updated_at`,
		id, amount.String()))
}

func (r *PostgresRepository) Transfer(ctx context.Context, actorID string, actorWarehouseID *string, input TransferInput) (Movement, Balance, Balance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, fmt.Errorf("begin transfer: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	fromWarehouseID, err := resolveLocationWarehouse(ctx, tx, input.FromLocationID)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}
	if err := checkWarehouseScope(actorWarehouseID, fromWarehouseID); err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}
	toWarehouseID, err := resolveLocationWarehouse(ctx, tx, input.ToLocationID)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}
	if err := checkWarehouseScope(actorWarehouseID, toWarehouseID); err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}

	if _, err := lockActiveProduct(ctx, tx, input.ProductID); err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}

	var lotID *string
	if input.LotID.Set && input.LotID.Value != nil {
		if err := lockLotForProduct(ctx, tx, *input.LotID.Value, input.ProductID); err != nil {
			return Movement{}, Balance{}, Balance{}, err
		}
		lotID = input.LotID.Value
	}

	quantity, err := decimal.NewFromString(input.Quantity)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, fmt.Errorf("parse transfer quantity: %w", err)
	}

	sourceBalance, err := decrementBalance(ctx, tx, input.FromLocationID, input.ProductID, lotID, quantity)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}

	destinationBalance, err := upsertBalanceIncrement(ctx, tx, input.ToLocationID, input.ProductID, lotID, input.Quantity)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, fmt.Errorf("increment balance on transfer: %w", err)
	}

	movement, err := insertMovement(ctx, tx, "transfer", input.ProductID, lotID, &input.FromLocationID, &input.ToLocationID,
		input.Quantity, nil, optionalStringPtr(input.Reference), optionalStringPtr(input.Notes), actorID)
	if err != nil {
		return Movement{}, Balance{}, Balance{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Movement{}, Balance{}, Balance{}, fmt.Errorf("commit transfer: %w", err)
	}
	return movement, sourceBalance, destinationBalance, nil
}

func (r *PostgresRepository) Adjust(ctx context.Context, actorID string, actorWarehouseID *string, input AdjustInput) (Movement, Balance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Movement{}, Balance{}, fmt.Errorf("begin adjust: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	warehouseID, err := resolveLocationWarehouse(ctx, tx, input.LocationID)
	if err != nil {
		return Movement{}, Balance{}, err
	}
	if err := checkWarehouseScope(actorWarehouseID, warehouseID); err != nil {
		return Movement{}, Balance{}, err
	}
	if _, err := lockActiveProduct(ctx, tx, input.ProductID); err != nil {
		return Movement{}, Balance{}, err
	}

	var lotID *string
	if input.LotID.Set && input.LotID.Value != nil {
		if err := lockLotForProduct(ctx, tx, *input.LotID.Value, input.ProductID); err != nil {
			return Movement{}, Balance{}, err
		}
		lotID = input.LotID.Value
	}

	quantity, err := decimal.NewFromString(input.Quantity)
	if err != nil {
		return Movement{}, Balance{}, fmt.Errorf("parse adjust quantity: %w", err)
	}

	var balance Balance
	var fromLocationID, toLocationID *string
	if input.Direction == "increase" {
		balance, err = upsertBalanceIncrement(ctx, tx, input.LocationID, input.ProductID, lotID, input.Quantity)
		toLocationID = &input.LocationID
	} else {
		balance, err = decrementBalance(ctx, tx, input.LocationID, input.ProductID, lotID, quantity)
		fromLocationID = &input.LocationID
	}
	if err != nil {
		return Movement{}, Balance{}, err
	}

	movement, err := insertMovement(ctx, tx, "adjust", input.ProductID, lotID, fromLocationID, toLocationID,
		input.Quantity, nil, nil, optionalStringPtr(input.Notes), actorID)
	if err != nil {
		return Movement{}, Balance{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Movement{}, Balance{}, fmt.Errorf("commit adjust: %w", err)
	}
	return movement, balance, nil
}

func (r *PostgresRepository) ListMovements(ctx context.Context, filter MovementListFilter) ([]Movement, error) {
	afterTime, afterID := cursorArguments(filter.After)
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, movement_type, product_id::text, lot_id::text,
		       from_location_id::text, to_location_id::text, quantity::text,
		       unit_cost::text, reference, notes, performed_by::text, created_at
		FROM stock_movements
		WHERE ($1::uuid IS NULL OR product_id = $1)
		  AND ($2::uuid IS NULL OR from_location_id = $2 OR to_location_id = $2)
		  AND ($3::text IS NULL OR movement_type = $3)
		  AND ($4::timestamptz IS NULL OR created_at >= $4)
		  AND ($5::timestamptz IS NULL OR created_at <= $5)
		  AND ($6::timestamptz IS NULL OR (created_at, id) < ($6::timestamptz, $7::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $8`,
		filter.ProductID, filter.LocationID, filter.MovementType, filter.From, filter.To,
		afterTime, afterID, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list movements: %w", err)
	}
	defer rows.Close()

	movements := make([]Movement, 0)
	for rows.Next() {
		movement, err := scanMovement(rows)
		if err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}
		movements = append(movements, movement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list movements: %w", err)
	}
	return movements, nil
}

func (r *PostgresRepository) GetMovement(ctx context.Context, id string) (Movement, error) {
	movement, err := scanMovement(r.pool.QueryRow(ctx, `
		SELECT id::text, movement_type, product_id::text, lot_id::text,
		       from_location_id::text, to_location_id::text, quantity::text,
		       unit_cost::text, reference, notes, performed_by::text, created_at
		FROM stock_movements WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Movement{}, ErrMovementNotFound
	}
	if err != nil {
		return Movement{}, fmt.Errorf("get movement: %w", err)
	}
	return movement, nil
}
