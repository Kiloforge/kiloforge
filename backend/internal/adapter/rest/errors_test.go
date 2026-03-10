package rest

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"kiloforge/internal/core/domain"
)

func TestMapServiceError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantCode   int
		wantSubstr string
	}{
		{
			name:       "nil error",
			err:        nil,
			wantCode:   http.StatusInternalServerError,
			wantSubstr: "unknown error",
		},
		{
			name:       "board not found",
			err:        fmt.Errorf("get board: %w", domain.ErrBoardNotFound),
			wantCode:   http.StatusNotFound,
			wantSubstr: "board not found",
		},
		{
			name:       "card not found",
			err:        fmt.Errorf("move: %w", domain.ErrCardNotFound),
			wantCode:   http.StatusNotFound,
			wantSubstr: "card not found",
		},
		{
			name:       "project not found",
			err:        domain.ErrProjectNotFound,
			wantCode:   http.StatusNotFound,
			wantSubstr: "project not found",
		},
		{
			name:       "agent not found",
			err:        domain.ErrAgentNotFound,
			wantCode:   http.StatusNotFound,
			wantSubstr: "agent not found",
		},
		{
			name:       "invalid column",
			err:        fmt.Errorf("%w: done", domain.ErrInvalidColumn),
			wantCode:   http.StatusUnprocessableEntity,
			wantSubstr: "invalid column",
		},
		{
			name:       "project exists conflict",
			err:        domain.ErrProjectExists,
			wantCode:   http.StatusConflict,
			wantSubstr: "project already registered",
		},
		{
			name:       "pool exhausted conflict",
			err:        domain.ErrPoolExhausted,
			wantCode:   http.StatusConflict,
			wantSubstr: "pool exhausted",
		},
		{
			name:       "forbidden",
			err:        domain.ErrForbidden,
			wantCode:   http.StatusForbidden,
			wantSubstr: "forbidden",
		},
		{
			name:       "generic error maps to 500",
			err:        errors.New("something broke"),
			wantCode:   http.StatusInternalServerError,
			wantSubstr: "internal error",
		},
		{
			name:       "wrapped generic error maps to 500",
			err:        fmt.Errorf("save: %w", errors.New("disk full")),
			wantCode:   http.StatusInternalServerError,
			wantSubstr: "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, msg := mapServiceError(tt.err)
			if code != tt.wantCode {
				t.Errorf("mapServiceError(%v) code = %d, want %d", tt.err, code, tt.wantCode)
			}
			if msg == "" {
				t.Error("mapServiceError returned empty message")
			}
			if tt.wantSubstr != "" && !containsCI(msg, tt.wantSubstr) {
				t.Errorf("mapServiceError(%v) msg = %q, want substring %q", tt.err, msg, tt.wantSubstr)
			}
		})
	}
}

func TestMapServiceError_DoesNotLeakInternals(t *testing.T) {
	t.Parallel()

	internalErr := fmt.Errorf("save board: sqlite: disk I/O error at /var/data/kf.db")
	_, msg := mapServiceError(internalErr)
	for _, forbidden := range []string{"sqlite", "/var/data", "disk I/O"} {
		if containsCI(msg, forbidden) {
			t.Errorf("mapServiceError leaked internal detail %q in message: %q", forbidden, msg)
		}
	}
}

func containsCI(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstr(s, substr))
}

func findSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if eqFold(s[i:i+len(sub)], sub) {
			return true
		}
	}
	return false
}

func eqFold(a, b string) bool {
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
