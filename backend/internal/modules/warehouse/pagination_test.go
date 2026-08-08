package warehouse

import (
	"errors"
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	want := Cursor{
		CreatedAt: time.Date(2026, 8, 8, 8, 0, 0, 123, time.UTC),
		ID:        "7e5d55b1-6356-492b-8296-2b981867fcf2",
	}

	got, err := DecodeCursor(EncodeCursor(want.CreatedAt, want.ID))
	if err != nil {
		t.Fatalf("DecodeCursor() error = %v", err)
	}
	if got != want {
		t.Fatalf("cursor = %#v, want %#v", got, want)
	}
}

func TestDecodeCursorRejectsMalformedValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "not base64", value: "not-a-cursor"},
		{name: "missing id", value: "MjAyNi0wOC0wOFQwODowMDowMFoK"},
		{name: "invalid timestamp", value: "bm90LWEtdGltZQppZA"},
		{name: "extra field", value: "MjAyNi0wOC0wOFQwODowMDowMFoKaWQKZXh0cmE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := DecodeCursor(tt.value); !errors.Is(err, ErrInvalidCursor) {
				t.Fatalf("DecodeCursor() error = %v, want ErrInvalidCursor", err)
			}
		})
	}
}

func TestEncodeCursorRejectsEmptyIDWhenDecoded(t *testing.T) {
	encoded := EncodeCursor(time.Date(2026, 8, 8, 8, 0, 0, 0, time.UTC), "")
	if _, err := DecodeCursor(encoded); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("DecodeCursor() error = %v, want ErrInvalidCursor", err)
	}
}
