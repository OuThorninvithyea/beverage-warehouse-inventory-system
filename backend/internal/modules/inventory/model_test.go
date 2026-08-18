package inventory

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBalanceJSONIncludesAvailableQuantity(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	balance := Balance{
		ID: "b1", LocationID: "loc1", ProductID: "prod1", LotID: nil,
		Quantity: "120.000", ReservedQuantity: "20.000", AvailableQuantity: "100.000",
		CreatedAt: now, UpdatedAt: now,
	}
	raw, err := json.Marshal(balance)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["available_quantity"] != "100.000" {
		t.Fatalf("available_quantity = %v, want 100.000", decoded["available_quantity"])
	}
	if decoded["lot_id"] != nil {
		t.Fatalf("lot_id = %v, want null", decoded["lot_id"])
	}
}

func TestMovementJSONShape(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	movement := Movement{
		ID: "m1", MovementType: "receive", ProductID: "prod1", LotID: nil,
		FromLocationID: nil, ToLocationID: strPointer("loc1"), Quantity: "50.000",
		UnitCost: strPointer("1.2500"), Reference: strPointer("PO-1"), Notes: nil,
		PerformedBy: strPointer("user1"), CreatedAt: now,
	}
	raw, err := json.Marshal(movement)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["movement_type"] != "receive" {
		t.Fatalf("movement_type = %v, want receive", decoded["movement_type"])
	}
	if decoded["from_location_id"] != nil {
		t.Fatalf("from_location_id = %v, want null", decoded["from_location_id"])
	}
}

func strPointer(value string) *string { return &value }
