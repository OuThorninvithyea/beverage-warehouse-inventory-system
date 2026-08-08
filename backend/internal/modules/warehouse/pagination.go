package warehouse

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

var ErrInvalidCursor = errors.New("invalid pagination cursor")

func EncodeCursor(createdAt time.Time, id string) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "\n" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(value string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}

	parts := strings.Split(string(raw), "\n")
	if len(parts) != 2 || parts[0] == "" || strings.TrimSpace(parts[1]) == "" {
		return Cursor{}, ErrInvalidCursor
	}

	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}

	return Cursor{CreatedAt: createdAt.UTC(), ID: parts[1]}, nil
}
