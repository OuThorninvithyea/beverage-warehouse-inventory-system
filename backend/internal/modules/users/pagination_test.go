package users

import (
	"testing"
	"time"
)

func TestEncodeDecodeCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	id := "22222222-2222-2222-2222-222222222222"

	encoded := EncodeCursor(createdAt, id)
	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeCursor() error = %v", err)
	}
	if !decoded.CreatedAt.Equal(createdAt) {
		t.Fatalf("CreatedAt = %v, want %v", decoded.CreatedAt, createdAt)
	}
	if decoded.ID != id {
		t.Fatalf("ID = %v, want %v", decoded.ID, id)
	}
}

func TestDecodeCursorRejectsGarbage(t *testing.T) {
	if _, err := DecodeCursor("not-base64!!"); err == nil {
		t.Fatalf("DecodeCursor() error = nil, want ErrInvalidCursor")
	}
	if _, err := DecodeCursor(""); err == nil {
		t.Fatalf("DecodeCursor(\"\") error = nil, want ErrInvalidCursor")
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

func TestUserPageTruncatesAndSetsCursor(t *testing.T) {
	now := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	items := make([]User, 3)
	for i := range items {
		items[i] = User{ID: string(rune('a' + i)), CreatedAt: now.Add(-time.Duration(i) * time.Minute)}
	}

	page := userPage(items, 2)

	if len(page.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(page.Items))
	}
	if !page.Page.HasMore {
		t.Fatalf("HasMore = false, want true")
	}
	if page.Page.NextCursor == nil {
		t.Fatalf("NextCursor = nil, want a cursor")
	}
}

func TestUserPageWithoutOverflowHasNoCursor(t *testing.T) {
	items := []User{{ID: "a"}}
	page := userPage(items, 2)

	if page.Page.HasMore {
		t.Fatalf("HasMore = true, want false")
	}
	if page.Page.NextCursor != nil {
		t.Fatalf("NextCursor = %v, want nil", page.Page.NextCursor)
	}
}

func TestUserPageHandlesNilItems(t *testing.T) {
	page := userPage(nil, 20)
	if page.Items == nil {
		t.Fatalf("Items = nil, want empty slice (JSON must encode [] not null)")
	}
}
