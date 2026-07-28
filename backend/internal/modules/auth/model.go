package auth

import "time"

const (
	RoleAdmin            = "admin"
	RoleWarehouseManager = "warehouse_manager"
	RolePicker           = "picker"
	RoleViewer           = "viewer"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	FullName     string
	Role         string
	WarehouseID  *string
	IsActive     bool
}

type PublicUser struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	FullName    string  `json:"full_name"`
	Role        string  `json:"role"`
	WarehouseID *string `json:"warehouse_id,omitempty"`
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:          u.ID,
		Email:       u.Email,
		FullName:    u.FullName,
		Role:        u.Role,
		WarehouseID: u.WarehouseID,
	}
}

type TokenPair struct {
	AccessToken           string     `json:"access_token"`
	RefreshToken          string     `json:"refresh_token"`
	TokenType             string     `json:"token_type"`
	AccessTokenExpiresAt  time.Time  `json:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time  `json:"refresh_token_expires_at"`
	User                  PublicUser `json:"user"`
}
