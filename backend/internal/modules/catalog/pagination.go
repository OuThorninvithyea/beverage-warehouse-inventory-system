package catalog

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCursor = errors.New("catalog cursor is invalid")

func EncodeCursor(createdAt time.Time, id string) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(value string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Cursor{}, ErrInvalidCursor
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	return Cursor{CreatedAt: createdAt.UTC(), ID: parts[1]}, nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func categoryPage(items []Category, limit int) Page[Category] {
	return pageFromItems(items, limit)
}

func productPage(items []Product, limit int) Page[Product] {
	return pageFromItems(items, limit)
}

func pageFromItems[T interface{ Category | Product }](items []T, limit int) Page[T] {
	if items == nil {
		items = []T{}
	}
	page := Page[T]{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		cursor := cursorFor(page.Items[limit-1])
		page.Page = PageInfo{NextCursor: &cursor, HasMore: true}
	}
	return page
}

func cursorFor[T interface{ Category | Product }](item T) string {
	switch v := any(item).(type) {
	case Category:
		return EncodeCursor(v.CreatedAt, v.ID)
	case Product:
		return EncodeCursor(v.CreatedAt, v.ID)
	default:
		return ""
	}
}
