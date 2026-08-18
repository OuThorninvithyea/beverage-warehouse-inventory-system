package inventory

import "time"

type Actor struct {
	ID          string
	Role        string
	WarehouseID *string
}

type Balance struct {
	ID                string    `json:"id"`
	LocationID        string    `json:"location_id"`
	ProductID         string    `json:"product_id"`
	LotID             *string   `json:"lot_id"`
	Quantity          string    `json:"quantity"`
	ReservedQuantity  string    `json:"reserved_quantity"`
	AvailableQuantity string    `json:"available_quantity"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Lot struct {
	ID                string    `json:"id"`
	ProductID         string    `json:"product_id"`
	LotNumber         string    `json:"lot_number"`
	ExpirationDate    *string   `json:"expiration_date"`
	ReceivedAt        time.Time `json:"received_at"`
	AvailableQuantity string    `json:"available_quantity"`
}

type Movement struct {
	ID             string    `json:"id"`
	MovementType   string    `json:"movement_type"`
	ProductID      string    `json:"product_id"`
	LotID          *string   `json:"lot_id"`
	FromLocationID *string   `json:"from_location_id"`
	ToLocationID   *string   `json:"to_location_id"`
	Quantity       string    `json:"quantity"`
	UnitCost       *string   `json:"unit_cost"`
	Reference      *string   `json:"reference"`
	Notes          *string   `json:"notes"`
	PerformedBy    *string   `json:"performed_by"`
	CreatedAt      time.Time `json:"created_at"`
}

type ReceiveInput struct {
	LocationID     string
	ProductID      string
	Quantity       string
	UnitCost       string
	LotNumber      OptionalString
	ExpirationDate OptionalString
	Reference      OptionalString
	Notes          OptionalString
}

type PickInput struct {
	LocationID string
	ProductID  string
	Quantity   string
	LotID      OptionalString
	Reference  OptionalString
	Notes      OptionalString
}

type TransferInput struct {
	ProductID      string
	LotID          OptionalString
	Quantity       string
	FromLocationID string
	ToLocationID   string
	Reference      OptionalString
	Notes          OptionalString
}

type AdjustInput struct {
	LocationID string
	ProductID  string
	LotID      OptionalString
	Direction  string
	Quantity   string
	Notes      OptionalString
}

type Cursor struct {
	CreatedAt time.Time
	ID        string
}

type BalanceListFilter struct {
	Limit       int
	After       *Cursor
	LocationID  *string
	ProductID   *string
	WarehouseID *string
	LotID       *string
}

type MovementListFilter struct {
	Limit        int
	After        *Cursor
	ProductID    *string
	LocationID   *string
	MovementType *string
	From         *time.Time
	To           *time.Time
}

type PageInfo struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

type Page[T any] struct {
	Items []T      `json:"items"`
	Page  PageInfo `json:"page"`
}
