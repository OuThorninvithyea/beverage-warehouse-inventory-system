package reports

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Dashboard(ctx context.Context, filter Filter) (Dashboard, error)
	Valuation(ctx context.Context, filter Filter) (Valuation, error)
	MovementSummary(ctx context.Context, filter Filter) (MovementSummary, error)
	Velocity(ctx context.Context, filter Filter) (Velocity, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) Repository {
	return &PostgresRepository{pool: pool}
}

// movementTypes is the fixed set from the stock_movements check constraint.
// Reports always return all four keys, including zeros, so a chart does not
// change shape just because nothing was adjusted this week.
var movementTypes = []string{"receive", "pick", "transfer", "adjust"}

// Dashboard aggregates the whole warehouse in one round trip. Every branch is
// warehouse-scoped through the same $1 parameter: NULL means "all warehouses",
// which only an admin can ask for.
func (r *PostgresRepository) Dashboard(ctx context.Context, filter Filter) (Dashboard, error) {
	dashboard := Dashboard{
		GeneratedAt:     time.Now().UTC(),
		MovementWindow:  filter.Days,
		MovementsByType: map[string]int{},
	}

	err := r.pool.QueryRow(ctx, `
		WITH scoped_locations AS (
		    SELECT l.id
		    FROM locations l
		    WHERE ($1::uuid IS NULL OR l.warehouse_id = $1)
		),
		stock AS (
		    SELECT b.quantity, b.reserved_quantity, b.lot_id
		    FROM inventory_balances b
		    JOIN scoped_locations sl ON sl.id = b.location_id
		    WHERE b.quantity > 0
		)
		SELECT
		    (SELECT COUNT(*) FROM products WHERE is_active),
		    (SELECT COUNT(*) FROM warehouses WHERE is_active
		         AND ($1::uuid IS NULL OR id = $1)),
		    (SELECT COUNT(*) FROM locations WHERE is_active
		         AND ($1::uuid IS NULL OR warehouse_id = $1)),
		    COALESCE((SELECT SUM(quantity) FROM stock), 0)::text,
		    COALESCE((SELECT SUM(reserved_quantity) FROM stock), 0)::text,
		    COALESCE((SELECT SUM(quantity - reserved_quantity) FROM stock), 0)::text,
		    COALESCE((
		        SELECT SUM(cl.remaining_quantity * cl.unit_cost)
		        FROM cost_layers cl
		        WHERE cl.remaining_quantity > 0
		          AND ($1::uuid IS NULL OR cl.warehouse_id = $1)
		    ), 0)::text,
		    (SELECT COUNT(DISTINCT lot_id) FROM stock WHERE lot_id IS NOT NULL),
		    (SELECT COUNT(DISTINCT b.lot_id)
		     FROM inventory_balances b
		     JOIN scoped_locations sl ON sl.id = b.location_id
		     JOIN lots l ON l.id = b.lot_id
		     WHERE b.quantity > 0 AND l.expiration_date < CURRENT_DATE),
		    (SELECT COUNT(DISTINCT b.lot_id)
		     FROM inventory_balances b
		     JOIN scoped_locations sl ON sl.id = b.location_id
		     JOIN lots l ON l.id = b.lot_id
		     WHERE b.quantity > 0
		       AND l.expiration_date >= CURRENT_DATE
		       AND l.expiration_date <= CURRENT_DATE + 30)`,
		filter.WarehouseID,
	).Scan(&dashboard.ActiveProducts, &dashboard.ActiveWarehouses, &dashboard.ActiveLocations,
		&dashboard.TotalQuantity, &dashboard.ReservedQuantity, &dashboard.AvailableQuantity,
		&dashboard.StockValue, &dashboard.LotsOnHand, &dashboard.ExpiredLots,
		&dashboard.ExpiringSoonLots)
	if err != nil {
		return Dashboard{}, fmt.Errorf("dashboard totals: %w", err)
	}

	for _, movementType := range movementTypes {
		dashboard.MovementsByType[movementType] = 0
	}
	rows, err := r.pool.Query(ctx, `
		SELECT m.movement_type, COUNT(*)
		FROM stock_movements m
		WHERE m.created_at >= NOW() - ($2::int * INTERVAL '1 day')
		  AND ($1::uuid IS NULL OR EXISTS (
		      SELECT 1 FROM locations l
		      WHERE l.warehouse_id = $1
		        AND l.id IN (m.from_location_id, m.to_location_id)
		  ))
		GROUP BY m.movement_type`,
		filter.WarehouseID, filter.Days)
	if err != nil {
		return Dashboard{}, fmt.Errorf("dashboard movement counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var movementType string
		var count int
		if err := rows.Scan(&movementType, &count); err != nil {
			return Dashboard{}, fmt.Errorf("scan dashboard movement count: %w", err)
		}
		dashboard.MovementsByType[movementType] = count
	}
	if err := rows.Err(); err != nil {
		return Dashboard{}, fmt.Errorf("dashboard movement counts: %w", err)
	}

	return dashboard, nil
}

// Valuation reads the remaining FIFO cost layers, which is the only honest
// source for stock value: it reflects what was actually paid for the units
// still on hand, not a list price.
func (r *PostgresRepository) Valuation(ctx context.Context, filter Filter) (Valuation, error) {
	valuation := Valuation{GeneratedAt: time.Now().UTC(), Rows: []ValuationRow{}}

	rows, err := r.pool.Query(ctx, `
		SELECT w.id::text, w.code, p.id::text, p.sku, p.name,
		       SUM(cl.remaining_quantity)::text,
		       (SUM(cl.remaining_quantity * cl.unit_cost)
		            / NULLIF(SUM(cl.remaining_quantity), 0))::numeric(18,4)::text,
		       SUM(cl.remaining_quantity * cl.unit_cost)::numeric(18,4)::text
		FROM cost_layers cl
		JOIN warehouses w ON w.id = cl.warehouse_id
		JOIN products p ON p.id = cl.product_id
		WHERE cl.remaining_quantity > 0
		  AND ($1::uuid IS NULL OR cl.warehouse_id = $1)
		GROUP BY w.id, w.code, p.id, p.sku, p.name
		ORDER BY SUM(cl.remaining_quantity * cl.unit_cost) DESC, p.sku ASC
		LIMIT $2`,
		filter.WarehouseID, filter.Limit)
	if err != nil {
		return Valuation{}, fmt.Errorf("valuation rows: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var row ValuationRow
		if err := rows.Scan(&row.WarehouseID, &row.WarehouseCode, &row.ProductID,
			&row.SKU, &row.ProductName, &row.RemainingQuantity,
			&row.AverageUnitCost, &row.TotalValue); err != nil {
			return Valuation{}, fmt.Errorf("scan valuation row: %w", err)
		}
		valuation.Rows = append(valuation.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return Valuation{}, fmt.Errorf("valuation rows: %w", err)
	}

	// The grand total covers the whole scope, not just the rows returned under
	// the limit, so a truncated table cannot understate the total.
	if err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(remaining_quantity * unit_cost), 0)::numeric(18,4)::text
		FROM cost_layers
		WHERE remaining_quantity > 0 AND ($1::uuid IS NULL OR warehouse_id = $1)`,
		filter.WarehouseID).Scan(&valuation.TotalValue); err != nil {
		return Valuation{}, fmt.Errorf("valuation total: %w", err)
	}

	return valuation, nil
}

func (r *PostgresRepository) MovementSummary(ctx context.Context, filter Filter) (MovementSummary, error) {
	summary := MovementSummary{
		GeneratedAt: time.Now().UTC(),
		WindowDays:  filter.Days,
		Totals:      map[string]int{},
		Rows:        []MovementSummaryRow{},
	}
	for _, movementType := range movementTypes {
		summary.Totals[movementType] = 0
	}

	rows, err := r.pool.Query(ctx, `
		SELECT TO_CHAR(DATE_TRUNC('day', m.created_at), 'YYYY-MM-DD'),
		       m.movement_type, COUNT(*), SUM(m.quantity)::text
		FROM stock_movements m
		WHERE m.created_at >= NOW() - ($2::int * INTERVAL '1 day')
		  AND ($1::uuid IS NULL OR EXISTS (
		      SELECT 1 FROM locations l
		      WHERE l.warehouse_id = $1
		        AND l.id IN (m.from_location_id, m.to_location_id)
		  ))
		GROUP BY 1, 2
		ORDER BY 1 DESC, 2 ASC`,
		filter.WarehouseID, filter.Days)
	if err != nil {
		return MovementSummary{}, fmt.Errorf("movement summary: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var row MovementSummaryRow
		if err := rows.Scan(&row.Day, &row.MovementType, &row.Movements, &row.Quantity); err != nil {
			return MovementSummary{}, fmt.Errorf("scan movement summary row: %w", err)
		}
		summary.Totals[row.MovementType] += row.Movements
		summary.Rows = append(summary.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return MovementSummary{}, fmt.Errorf("movement summary: %w", err)
	}

	return summary, nil
}

// Velocity ranks products by how much left the building, which is what drives
// replenishment decisions. Only picks count: a transfer moves stock without
// consuming it.
func (r *PostgresRepository) Velocity(ctx context.Context, filter Filter) (Velocity, error) {
	velocity := Velocity{
		GeneratedAt: time.Now().UTC(),
		WindowDays:  filter.Days,
		Rows:        []VelocityRow{},
	}

	rows, err := r.pool.Query(ctx, `
		SELECT p.id::text, p.sku, p.name,
		       SUM(m.quantity)::text, COUNT(*),
		       (SUM(m.quantity) / GREATEST($2::int, 1))::numeric(18,3)::text
		FROM stock_movements m
		JOIN products p ON p.id = m.product_id
		WHERE m.movement_type = 'pick'
		  AND m.created_at >= NOW() - ($2::int * INTERVAL '1 day')
		  AND ($1::uuid IS NULL OR EXISTS (
		      SELECT 1 FROM locations l
		      WHERE l.warehouse_id = $1 AND l.id = m.from_location_id
		  ))
		GROUP BY p.id, p.sku, p.name
		ORDER BY SUM(m.quantity) DESC, p.sku ASC
		LIMIT $3`,
		filter.WarehouseID, filter.Days, filter.Limit)
	if err != nil {
		return Velocity{}, fmt.Errorf("velocity rows: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var row VelocityRow
		if err := rows.Scan(&row.ProductID, &row.SKU, &row.ProductName,
			&row.PickedQuantity, &row.PickCount, &row.DailyAverage); err != nil {
			return Velocity{}, fmt.Errorf("scan velocity row: %w", err)
		}
		velocity.Rows = append(velocity.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return Velocity{}, fmt.Errorf("velocity rows: %w", err)
	}

	return velocity, nil
}
