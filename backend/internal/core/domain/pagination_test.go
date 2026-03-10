package domain

import "testing"

func TestPageOpts_Normalize(t *testing.T) {
	tests := []struct {
		name  string
		input PageOpts
		want  int
	}{
		{"zero defaults", PageOpts{Limit: 0}, DefaultPageLimit},
		{"negative defaults", PageOpts{Limit: -1}, DefaultPageLimit},
		{"within range", PageOpts{Limit: 25}, 25},
		{"at max", PageOpts{Limit: MaxPageLimit}, MaxPageLimit},
		{"over max clamped", PageOpts{Limit: 500}, MaxPageLimit},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.Normalize()
			if tt.input.Limit != tt.want {
				t.Errorf("Normalize() limit = %d, want %d", tt.input.Limit, tt.want)
			}
		})
	}
}

func TestCursorRoundTrip(t *testing.T) {
	encoded := EncodeCursor("2026-03-10T12:00:00Z", "abc-123")
	if encoded == "" {
		t.Fatal("EncodeCursor returned empty string")
	}

	decoded := DecodeCursor(encoded)
	if decoded.SortVal != "2026-03-10T12:00:00Z" {
		t.Errorf("SortVal = %q, want %q", decoded.SortVal, "2026-03-10T12:00:00Z")
	}
	if decoded.ID != "abc-123" {
		t.Errorf("ID = %q, want %q", decoded.ID, "abc-123")
	}
}

func TestDecodeCursor_Empty(t *testing.T) {
	d := DecodeCursor("")
	if d.SortVal != "" || d.ID != "" {
		t.Errorf("expected zero value, got %+v", d)
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	d := DecodeCursor("not-valid-base64!!!")
	if d.SortVal != "" || d.ID != "" {
		t.Errorf("expected zero value for invalid cursor, got %+v", d)
	}
}
