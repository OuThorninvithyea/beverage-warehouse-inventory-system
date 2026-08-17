package catalog

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestCatalogCursorRoundTripUsesRFC3339NanoPipeAndUUID(t *testing.T) {
	createdAt := time.Date(2026, 8, 11, 7, 59, 10, 827123456, time.UTC)
	id := "7e5d55b1-6356-492b-8296-2b981867fcf2"

	encoded := EncodeCursor(createdAt, id)
	wantRaw := createdAt.Format(time.RFC3339Nano) + "|" + id
	if gotRawBytes, err := base64.RawURLEncoding.DecodeString(encoded); err != nil || string(gotRawBytes) != wantRaw {
		t.Fatalf("encoded raw = %q, %v, want %q", string(gotRawBytes), err, wantRaw)
	}

	got, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeCursor() error = %v", err)
	}
	if got.CreatedAt != createdAt || got.ID != id {
		t.Fatalf("cursor = %#v, want CreatedAt %v ID %q", got, createdAt, id)
	}
}

func TestDecodeCursorRejectsInvalidValues(t *testing.T) {
	validTime := time.Date(2026, 8, 11, 7, 59, 10, 0, time.UTC).Format(time.RFC3339Nano)
	tests := []struct{ name, value string }{
		{"malformed base64", "not+a+cursor"},
		{"malformed payload", base64.RawURLEncoding.EncodeToString([]byte(validTime))},
		{"invalid timestamp", base64.RawURLEncoding.EncodeToString([]byte("not-a-time|7e5d55b1-6356-492b-8296-2b981867fcf2"))},
		{"empty id", base64.RawURLEncoding.EncodeToString([]byte(validTime + "|"))},
		{"non uuid id", base64.RawURLEncoding.EncodeToString([]byte(validTime + "|not-a-uuid"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := DecodeCursor(tt.value); !errors.Is(err, ErrInvalidCursor) {
				t.Fatalf("DecodeCursor() error = %v, want ErrInvalidCursor", err)
			}
		})
	}
}

func TestNormalizeLimitDefaultsAndClamps(t *testing.T) {
	for _, input := range []int{0, -1} {
		if got := normalizeLimit(input); got != 20 {
			t.Fatalf("normalizeLimit(%d) = %d, want 20", input, got)
		}
	}
	if got := normalizeLimit(101); got != 100 {
		t.Fatalf("normalizeLimit(101) = %d, want 100", got)
	}
	if got := normalizeLimit(50); got != 50 {
		t.Fatalf("normalizeLimit(50) = %d, want 50", got)
	}
}

func TestCategoryPageSerializesNilItemsAsEmptyArray(t *testing.T) {
	page := categoryPage(nil, 20)
	body, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(body) != `{"items":[],"page":{"next_cursor":null,"has_more":false}}` {
		t.Fatalf("json = %s, want empty items array", body)
	}
}

func TestCategoryPageExposesNextCursorOnlyWhenLimitPlusOneExists(t *testing.T) {
	items := []Category{
		{ID: "7e5d55b1-6356-492b-8296-2b981867fcf2", CreatedAt: time.Date(2026, 8, 11, 7, 0, 0, 0, time.UTC)},
		{ID: "8e5d55b1-6356-492b-8296-2b981867fcf2", CreatedAt: time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)},
	}
	page := categoryPage(items, 1)
	if len(page.Items) != 1 || page.Items[0].ID != items[0].ID {
		t.Fatalf("items = %#v, want only requested limit", page.Items)
	}
	if !page.Page.HasMore || page.Page.NextCursor == nil {
		t.Fatalf("page info = %#v, want has more with next cursor", page.Page)
	}
	cursor, err := DecodeCursor(*page.Page.NextCursor)
	if err != nil || cursor.ID != items[0].ID || cursor.CreatedAt != items[0].CreatedAt {
		t.Fatalf("next cursor = %#v, %v, want last returned item", cursor, err)
	}

	page = categoryPage(items[:1], 1)
	if page.Page.HasMore || page.Page.NextCursor != nil {
		t.Fatalf("page info = %#v, want no next cursor", page.Page)
	}
}

func TestProductPageExposesNextCursorOnlyWhenLimitPlusOneExists(t *testing.T) {
	items := []Product{
		{ID: "7e5d55b1-6356-492b-8296-2b981867fcf2", CreatedAt: time.Date(2026, 8, 11, 7, 0, 0, 0, time.UTC)},
		{ID: "8e5d55b1-6356-492b-8296-2b981867fcf2", CreatedAt: time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)},
	}
	page := productPage(items, 1)
	if len(page.Items) != 1 || page.Items[0].ID != items[0].ID {
		t.Fatalf("items = %#v, want only requested limit", page.Items)
	}
	if !page.Page.HasMore || page.Page.NextCursor == nil {
		t.Fatalf("page info = %#v, want has more with next cursor", page.Page)
	}
}
