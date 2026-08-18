package inventory

import (
	"testing"
	"time"
)

func TestEncodeDecodeCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	id := "33333333-3333-3333-3333-333333333333"
	encoded := EncodeCursor(createdAt, id)
	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeCursor() error = %v", err)
	}
	if !decoded.CreatedAt.Equal(createdAt) || decoded.ID != id {
		t.Fatalf("decoded = %+v, want CreatedAt=%v ID=%v", decoded, createdAt, id)
	}
}

func TestDecodeCursorRejectsGarbage(t *testing.T) {
	if _, err := DecodeCursor("not-base64!!"); err == nil {
		t.Fatalf("DecodeCursor() error = nil, want ErrInvalidCursor")
	}
}

func TestNormalizeLimitBounds(t *testing.T) {
	cases := map[int]int{0: 20, -5: 20, 1: 1, 100: 100, 250: 100}
	for input, want := range cases {
		if got := normalizeLimit(input); got != want {
			t.Fatalf("normalizeLimit(%d) = %d, want %d", input, got, want)
		}
	}
}

func TestBalancePageTruncatesAndSetsCursor(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	items := make([]Balance, 3)
	for i := range items {
		items[i] = Balance{ID: string(rune('a' + i)), CreatedAt: now.Add(-time.Duration(i) * time.Minute)}
	}
	page := balancePage(items, 2)
	if len(page.Items) != 2 || !page.Page.HasMore || page.Page.NextCursor == nil {
		t.Fatalf("page = %+v, want 2 items with HasMore and a cursor", page)
	}
}

func TestBalancePageHandlesNilItems(t *testing.T) {
	page := balancePage(nil, 20)
	if page.Items == nil {
		t.Fatalf("Items = nil, want empty slice (JSON must encode [] not null)")
	}
}

func TestMovementPageTruncatesAndSetsCursor(t *testing.T) {
	now := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	items := make([]Movement, 2)
	for i := range items {
		items[i] = Movement{ID: string(rune('a' + i)), CreatedAt: now.Add(-time.Duration(i) * time.Minute)}
	}
	page := movementPage(items, 1)
	if len(page.Items) != 1 || !page.Page.HasMore {
		t.Fatalf("page = %+v, want 1 item with HasMore", page)
	}
}
