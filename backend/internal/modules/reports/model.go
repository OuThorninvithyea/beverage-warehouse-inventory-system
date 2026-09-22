package reports

import "time"

// Actor mirrors the other modules: a non-admin is pinned to their assigned
// warehouse, an admin may look across all of them.
type Actor struct {
	ID          string
	Role        string
	WarehouseID *string
}

// Filter is shared by every report. WarehouseID is resolved from the actor for
// non-admins, so a manager cannot read another site's numbers.
type Filter struct {
	WarehouseID *string
	Days        int
	Limit       int
}

// Dashboard is the single aggregate the dashboard screen needs. Computing it
// in one query avoids the client stitching it together from paginated lists,
// which silently under-reports as soon as the data outgrows one page.
type Dashboard struct {
	GeneratedAt time.Time `json:"generated_at"`

	ActiveProducts   int `json:"active_products"`
	ActiveWarehouses int `json:"active_warehouses"`
	ActiveLocations  int `json:"active_locations"`

	TotalQuantity     string `json:"total_quantity"`
	ReservedQuantity  string `json:"reserved_quantity"`
	AvailableQuantity string `json:"available_quantity"`
	StockValue        string `json:"stock_value"`

	LotsOnHand       int `json:"lots_on_hand"`
	ExpiredLots      int `json:"expired_lots"`
	ExpiringSoonLots int `json:"expiring_soon_lots"`

	MovementsByType map[string]int `json:"movements_by_type"`
	MovementWindow  int            `json:"movement_window_days"`
}

// ValuationRow is one product's stock value in one warehouse, taken from the
// remaining FIFO cost layers rather than from a nominal price.
type ValuationRow struct {
	WarehouseID       string `json:"warehouse_id"`
	WarehouseCode     string `json:"warehouse_code"`
	ProductID         string `json:"product_id"`
	SKU               string `json:"sku"`
	ProductName       string `json:"product_name"`
	RemainingQuantity string `json:"remaining_quantity"`
	AverageUnitCost   string `json:"average_unit_cost"`
	TotalValue        string `json:"total_value"`
}

type Valuation struct {
	GeneratedAt time.Time      `json:"generated_at"`
	TotalValue  string         `json:"total_value"`
	Rows        []ValuationRow `json:"rows"`
}

// MovementSummaryRow counts one movement type on one day.
type MovementSummaryRow struct {
	Day          string `json:"day"`
	MovementType string `json:"movement_type"`
	Movements    int    `json:"movements"`
	Quantity     string `json:"quantity"`
}

type MovementSummary struct {
	GeneratedAt time.Time            `json:"generated_at"`
	WindowDays  int                  `json:"window_days"`
	Totals      map[string]int       `json:"totals"`
	Rows        []MovementSummaryRow `json:"rows"`
}

// VelocityRow is one product's outbound throughput over the window.
type VelocityRow struct {
	ProductID      string `json:"product_id"`
	SKU            string `json:"sku"`
	ProductName    string `json:"product_name"`
	PickedQuantity string `json:"picked_quantity"`
	PickCount      int    `json:"pick_count"`
	DailyAverage   string `json:"daily_average"`
}

type Velocity struct {
	GeneratedAt time.Time     `json:"generated_at"`
	WindowDays  int           `json:"window_days"`
	Rows        []VelocityRow `json:"rows"`
}
