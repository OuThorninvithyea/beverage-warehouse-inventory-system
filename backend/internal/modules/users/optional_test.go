package users

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
		if w.Field.Value != nil {
			t.Fatalf("Value = %v, want nil for omitted field", w.Field.Value)
		}
	})

	t.Run("null clears the field", func(t *testing.T) {
		var w wrapper
		if err := json.Unmarshal([]byte(`{"field":null}`), &w); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if !w.Field.Set {
			t.Fatalf("Set = false, want true for null field")
		}
		if w.Field.Value != nil {
			t.Fatalf("Value = %v, want nil for null field", w.Field.Value)
		}
	})

	t.Run("string value is captured", func(t *testing.T) {
		var w wrapper
		if err := json.Unmarshal([]byte(`{"field":"abc-123"}`), &w); err != nil {
			t.Fatalf("Unmarshal() error = %v", err)
		}
		if !w.Field.Set {
			t.Fatalf("Set = false, want true for string field")
		}
		if w.Field.Value == nil || *w.Field.Value != "abc-123" {
			t.Fatalf("Value = %v, want \"abc-123\"", w.Field.Value)
		}
	})

	t.Run("non-string value is rejected", func(t *testing.T) {
		var w wrapper
		if err := json.Unmarshal([]byte(`{"field":42}`), &w); err == nil {
			t.Fatalf("Unmarshal() error = nil, want error for non-string field")
		}
	})
}
