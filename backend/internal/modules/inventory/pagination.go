package inventory

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidCursor = errors.New("inventory cursor is invalid")

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

func balancePage(items []Balance, limit int) Page[Balance] {
	if items == nil {
		items = []Balance{}
	}
	page := Page[Balance]{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		cursor := EncodeCursor(last.CreatedAt, last.ID)
		page.Page = PageInfo{NextCursor: &cursor, HasMore: true}
	}
	return page
}

func movementPage(items []Movement, limit int) Page[Movement] {
	if items == nil {
		items = []Movement{}
	}
	page := Page[Movement]{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[limit-1]
		cursor := EncodeCursor(last.CreatedAt, last.ID)
		page.Page = PageInfo{NextCursor: &cursor, HasMore: true}
	}
	return page
}

func cursorArguments(cursor *Cursor) (any, any) {
	if cursor == nil {
		return nil, nil
	}
	return cursor.CreatedAt, cursor.ID
}
