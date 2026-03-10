package domain

import (
	"encoding/base64"
	"encoding/json"
)

// DefaultPageLimit is the default number of items per page.
const DefaultPageLimit = 50

// MaxPageLimit is the maximum allowed page size.
const MaxPageLimit = 200

// PageOpts holds pagination parameters for list queries.
type PageOpts struct {
	Limit  int
	Cursor string
}

// Normalize ensures Limit is within [1, MaxPageLimit] and defaults to DefaultPageLimit.
func (p *PageOpts) Normalize() {
	if p.Limit <= 0 {
		p.Limit = DefaultPageLimit
	}
	if p.Limit > MaxPageLimit {
		p.Limit = MaxPageLimit
	}
}

// Page holds a page of results with cursor-based pagination metadata.
type Page[T any] struct {
	Items      []T
	NextCursor string
	TotalCount int
}

// CursorData holds the encoded cursor state for keyset pagination.
type CursorData struct {
	// SortVal is the value of the sort column at the cursor position.
	SortVal string `json:"s"`
	// ID is the unique identifier at the cursor position (tiebreaker).
	ID string `json:"id"`
}

// EncodeCursor encodes cursor data to an opaque base64 string.
func EncodeCursor(sortVal, id string) string {
	d := CursorData{SortVal: sortVal, ID: id}
	b, err := json.Marshal(d)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor decodes an opaque cursor string. Returns zero value if invalid.
func DecodeCursor(cursor string) CursorData {
	if cursor == "" {
		return CursorData{}
	}
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return CursorData{}
	}
	var d CursorData
	if err := json.Unmarshal(b, &d); err != nil {
		return CursorData{}
	}
	return d
}
