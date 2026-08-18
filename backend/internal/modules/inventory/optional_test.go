package inventory

import (
	"encoding/json"
	"testing"
)

func TestOptionalStringUnmarshalJSON(t *testing.T) {
	type wrapper struct {
		Field OptionalString `json:"field"`
	}

	t.Run("omitted field leaves Set false", func(t *testing.T) {
		var w wrapper
		if err := json.Unmarshal([]byte(`{}`), &w); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if w.Field.Set {
			t.Fatalf("Set = true, want false for omitted field")
		}
	})

	t.Run("null clears the field", func(t *testing.T) {
		var w wrapper
		if err := json.Unmarshal([]byte(`{"field":null}`), &w); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if !w.Field.Set || w.Field.Value != nil {
			t.Fatalf("got Set=%v Value=%v, want Set=true Value=nil", w.Field.Set, w.Field.Value)
		}
	})

	t.Run("string value is captured", func(t *testing.T) {
		var w wrapper
		if err := json.Unmarshal([]byte(`{"field":"LOT-1"}`), &w); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if !w.Field.Set || w.Field.Value == nil || *w.Field.Value != "LOT-1" {
			t.Fatalf("got Set=%v Value=%v, want Set=true Value=LOT-1", w.Field.Set, w.Field.Value)
		}
	})

	t.Run("non-string value is rejected", func(t *testing.T) {
		var w wrapper
		if err := json.Unmarshal([]byte(`{"field":42}`), &w); err == nil {
			t.Fatalf("Unmarshal() error = nil, want error for non-string field")
		}
	})
}
