package catalog

import (
	"encoding/json"
	"testing"
)

func TestOptionalStringUnmarshalOmittedLeavesUnset(t *testing.T) {
	var input CategoryInput
	if err := json.Unmarshal([]byte(`{"name":"Soda"}`), &input); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if input.ParentID.Set {
		t.Fatalf("ParentID.Set = true, want false")
	}
	if input.ParentID.Value != nil {
		t.Fatalf("ParentID.Value = %v, want nil", *input.ParentID.Value)
	}
}

func TestOptionalStringUnmarshalNullSetsNilValue(t *testing.T) {
	var input CategoryInput
	if err := json.Unmarshal([]byte(`{"name":"Soda","parent_id":null}`), &input); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !input.ParentID.Set {
		t.Fatalf("ParentID.Set = false, want true")
	}
	if input.ParentID.Value != nil {
		t.Fatalf("ParentID.Value = %v, want nil", *input.ParentID.Value)
	}
}

func TestOptionalStringUnmarshalStringSetsPointerValue(t *testing.T) {
	var input ProductInput
	if err := json.Unmarshal([]byte(`{"category_id":"7e5d55b1-6356-492b-8296-2b981867fcf2","barcode":"4006381333931"}`), &input); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !input.CategoryID.Set || input.CategoryID.Value == nil || *input.CategoryID.Value != "7e5d55b1-6356-492b-8296-2b981867fcf2" {
		t.Fatalf("CategoryID = %#v, want set pointer to decoded string", input.CategoryID)
	}
	if !input.Barcode.Set || input.Barcode.Value == nil || *input.Barcode.Value != "4006381333931" {
		t.Fatalf("Barcode = %#v, want set pointer to decoded string", input.Barcode)
	}
}

func TestOptionalStringUnmarshalRejectsNonString(t *testing.T) {
	var input CategoryInput
	if err := json.Unmarshal([]byte(`{"name":"Soda","parent_id":123}`), &input); err == nil {
		t.Fatalf("Unmarshal() error = nil, want error")
	}
}
