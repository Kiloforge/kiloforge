package prereq

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// AuthError indicates the Claude CLI is not authenticated.
type AuthError struct {
	Message string
	Hint    string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("%s — %s", e.Message, e.Hint)
}

// authKeywords are substrings in Claude CLI stderr that indicate an auth failure.
var authKeywords = []string{
	"not logged in",
	"authentication",
	"login",
	"unauthorized",
	"unauthenticated",
	"sign in",
	"api key",
}

// containsAuthError checks if the message contains auth-related error keywords.
func containsAuthError(msg string) bool {
	lower := strings.ToLower(msg)
	for _, kw := range authKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// CheckClaudeAuth verifies the Claude CLI is authenticated by running
// a lightweight probe command. Returns nil if authenticated or if the
// probe fails for non-auth reasons (timeout, crash).
func CheckClaudeAuth(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "-p", ".", "--max-turns", "0")
	cmd.Env = cleanClaudeEnv()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if containsAuthError(msg) {
			return &AuthError{
				Message: "Claude CLI is not logged in",
				Hint:    "Run 'claude' in a terminal to authenticate, then retry.",
			}
		}
		// Non-auth error (timeout, crash, etc.) — warn but don't block.
		return nil
	}
	return nil
}

var (
	authCheckErr  error
	authCheckOnce sync.Once
)

// CheckClaudeAuthCached calls CheckClaudeAuth once and caches the result
// for the lifetime of the process.
func CheckClaudeAuthCached(ctx context.Context) error {
	authCheckOnce.Do(func() {
		authCheckErr = CheckClaudeAuth(ctx)
	})
	return authCheckErr
}

// ResetAuthCache resets the cached auth check result. Intended for testing.
func ResetAuthCache() {
	authCheckOnce = sync.Once{}
	authCheckErr = nil
}

// cleanClaudeEnv returns os.Environ() with Claude-internal env vars removed
// to prevent "nested session" detection in child claude processes.
func cleanClaudeEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "CLAUDECODE=") ||
			strings.HasPrefix(e, "CLAUDE_CODE_ENTRYPOINT=") {
			continue
		}
		env = append(env, e)
	}
	return env
}
