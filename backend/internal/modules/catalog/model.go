package catalog

import "time"

type Actor struct {
	Role string
}

type Category struct {
	ID        string    `json:"id"`
	ParentID  *string   `json:"parent_id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Product struct {
	ID           string    `json:"id"`
	CategoryID   *string   `json:"category_id"`
	SKU          string    `json:"sku"`
	Barcode      *string   `json:"barcode"`
	Name         string    `json:"name"`
	Unit         string    `json:"unit"`
	IsLotTracked bool      `json:"is_lot_tracked"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CategoryInput struct {
	Name     string         `json:"name"`
	ParentID OptionalString `json:"parent_id"`
	IsActive *bool          `json:"is_active"`
}

type ProductInput struct {
	CategoryID   OptionalString `json:"category_id"`
	SKU          string         `json:"sku"`
	Barcode      OptionalString `json:"barcode"`
	Name         string         `json:"name"`
	Unit         string         `json:"unit"`
	IsLotTracked *bool          `json:"is_lot_tracked"`
	IsActive     *bool          `json:"is_active"`
}

type Cursor struct {
	CreatedAt time.Time
	ID        string
}

type ListFilter struct {
	Limit      int
	After      *Cursor
	Search     string
	IsActive   *bool
	ParentID   *string
	CategoryID *string
}

type PageInfo struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

type Page[T any] struct {
	Items []T      `json:"items"`
	Page  PageInfo `json:"page"`
}
