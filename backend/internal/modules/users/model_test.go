package users

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserJSONNeverIncludesPassword(t *testing.T) {
	now := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	user := User{
		ID:          "11111111-1111-1111-1111-111111111111",
		Email:       "manager@bwims.test",
		FullName:    "Sinat Chantha",
		Role:        "warehouse_manager",
		WarehouseID: nil,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	raw, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if _, exists := decoded["password_hash"]; exists {
		t.Fatalf("encoded user must never include password_hash")
	}
	if _, exists := decoded["password"]; exists {
		t.Fatalf("encoded user must never include password")
	}
	if decoded["warehouse_id"] != nil {
		t.Fatalf("warehouse_id = %v, want null", decoded["warehouse_id"])
	}
	if decoded["role"] != "warehouse_manager" {
		t.Fatalf("role = %v, want warehouse_manager", decoded["role"])
	}
}
