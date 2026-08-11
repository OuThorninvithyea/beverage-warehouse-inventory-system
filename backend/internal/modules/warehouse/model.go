package warehouse

import "time"

type Actor struct {
	Role        string
	WarehouseID *string
}

type Warehouse struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Address   *string   `json:"address,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Location struct {
	ID          string    `json:"id"`
	WarehouseID string    `json:"warehouse_id"`
	Code        string    `json:"code"`
	Zone        *string   `json:"zone,omitempty"`
	Aisle       *string   `json:"aisle,omitempty"`
	Rack        *string   `json:"rack,omitempty"`
	Shelf       *string   `json:"shelf,omitempty"`
	Barcode     *string   `json:"barcode,omitempty"`
	IsPickable  bool      `json:"is_pickable"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type WarehouseInput struct {
	Code     string
	Name     string
	Address  *string
	IsActive *bool
}

type LocationInput struct {
	Code       string
	Zone       *string
	Aisle      *string
	Rack       *string
	Shelf      *string
	Barcode    *string
	IsPickable *bool
	IsActive   *bool
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
	IsPickable *bool
}

type PageInfo struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

type Page[T any] struct {
	Items []T      `json:"items"`
	Page  PageInfo `json:"page"`
}
