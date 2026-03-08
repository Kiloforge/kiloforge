package prereq

import (
	"context"
	"errors"
	"testing"
)

func TestContainsAuthError(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want bool
	}{
		{"not logged in", "Error: You are not logged in. Please run claude to authenticate.", true},
		{"authentication required", "Authentication required to continue", true},
		{"login required", "Please login first", true},
		{"unauthorized", "Unauthorized: invalid credentials", true},
		{"api key", "Missing API key", true},
		{"sign in", "Please sign in to continue", true},
		{"unrelated error", "connection timeout", false},
		{"empty", "", false},
		{"case insensitive", "NOT LOGGED IN", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsAuthError(tt.msg)
			if got != tt.want {
				t.Errorf("containsAuthError(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

func TestAuthError_Error(t *testing.T) {
	err := &AuthError{
		Message: "Claude CLI is not logged in",
		Hint:    "Run 'claude' in a terminal to authenticate.",
	}
	got := err.Error()
	if got != "Claude CLI is not logged in — Run 'claude' in a terminal to authenticate." {
		t.Errorf("unexpected error string: %s", got)
	}
}

func TestAuthError_Is(t *testing.T) {
	err := &AuthError{Message: "test", Hint: "hint"}
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Error("expected errors.As to match *AuthError")
	}
}

func TestCheckClaudeAuthCached_ReturnsSameResult(t *testing.T) {
	ResetAuthCache()
	defer ResetAuthCache()

	// First call with a fresh cache.
	ctx := context.Background()
	err1 := CheckClaudeAuthCached(ctx)
	err2 := CheckClaudeAuthCached(ctx)

	// Both should return the same result (whatever it is on this machine).
	if err1 != err2 {
		t.Errorf("cached results differ: %v vs %v", err1, err2)
	}
}

func TestResetAuthCache(t *testing.T) {
	ResetAuthCache()
	// After reset, the once should be fresh — no panic.
	ResetAuthCache()
}
