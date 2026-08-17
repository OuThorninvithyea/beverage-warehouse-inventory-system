package users

import "time"

type Actor struct {
	ID   string
	Role string
}

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	FullName    string    `json:"full_name"`
	Role        string    `json:"role"`
	WarehouseID *string   `json:"warehouse_id"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserCreateInput struct {
	Email       string
	FullName    string
	Role        string
	WarehouseID OptionalString
	Password    string
}

type UserUpdateInput struct {
	FullName    string
	Role        string
	WarehouseID OptionalString
	IsActive    *bool
}

type PasswordResetInput struct {
	Password string
}

type Cursor struct {
	CreatedAt time.Time
	ID        string
}

type ListFilter struct {
	Limit       int
	After       *Cursor
	Search      string
	Role        string
	WarehouseID *string
	IsActive    *bool
}

type PageInfo struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

type Page[T any] struct {
	Items []T      `json:"items"`
	Page  PageInfo `json:"page"`
}
