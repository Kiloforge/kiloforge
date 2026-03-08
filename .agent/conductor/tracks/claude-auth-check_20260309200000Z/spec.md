# Specification: Claude CLI Authentication Check Before Agent Spawning

**Track ID:** claude-auth-check_20260309200000Z
**Type:** Feature
**Created:** 2026-03-09T20:00:00Z
**Status:** Draft

## Summary

Add a Claude CLI login verification as the first pre-flight check before any agent spawning. If the user isn't logged in, block spawning with a clear message directing them to run `claude` interactively to authenticate.

## Context

The spawn pre-flight chain currently validates: skills installed → permissions consent → spawn. But if the user hasn't logged into the Claude CLI at all, the agent process fails at runtime with cryptic auth errors. This check should be the very first gate — before skill checks and consent — since there's no point validating skills or asking for permission consent if Claude itself can't authenticate.

## Codebase Analysis

### Current pre-flight checks

- **`backend/internal/adapter/prereq/check.go`** — `Check()` validates `git`, `docker`, `docker compose`, `claude` binary existence via `exec.LookPath`. Called during `kf init` only.
- **`backend/internal/adapter/skills/checker.go`** — `CheckRequired()` validates skill installation. Called from `spawner.ValidateSkills()`.
- **`backend/internal/adapter/agent/spawner.go`** — `ValidateSkills()` runs before spawn. No auth check exists.
- **`backend/internal/adapter/cli/implement.go`** — CLI entry point calls `ValidateSkills()` then `SpawnDeveloper()`. No auth gate.

### Claude CLI auth detection

The Claude CLI does not expose a `claude auth status` command (GitHub issue #1886). However, we can detect auth state by:

1. **Probe command**: `claude -p "." --max-turns 0` — exits quickly with 0 if authenticated, non-zero with auth error message on stderr if not
2. **Timeout**: 10s timeout on the probe to avoid hanging
3. **Error parsing**: Check stderr for keywords like "not logged in", "authentication", "login", "unauthorized"

### Spawn flow after this track

```
1. Claude auth check      ← NEW (this track)
2. Skill pre-flight check  (existing)
3. Permissions consent     (existing)
4. Spawn agent            (existing)
```

## Acceptance Criteria

- [ ] New `CheckClaudeAuth()` function in `prereq` package that probes Claude CLI authentication
- [ ] Probe uses `claude -p "." --max-turns 0` with 10s timeout
- [ ] Returns structured error with clear message: "Claude CLI is not logged in. Run `claude` in a terminal to authenticate."
- [ ] Spawner calls `CheckClaudeAuth()` before `ValidateSkills()` in all three spawn paths (developer, reviewer, interactive)
- [ ] CLI `kf implement` shows auth error with login instructions before reaching skill/consent checks
- [ ] REST API spawn endpoints return 401 with auth error message if Claude is not authenticated
- [ ] Auth check result is cached for the lifetime of the process (avoid re-probing on every spawn)
- [ ] `kf init` prereq check also includes auth verification (warn, don't block — user may log in later)
- [ ] `go test ./...` passes
- [ ] Probe failure due to timeout treated as "unknown" — warn but don't block (network issues shouldn't prevent local work)

## Dependencies

None.

## Blockers

None.

## Conflict Risk

- LOW against `agent-list-monitoring-ui` — FE only, no spawn flow changes
- LOW against `tanstack-query-migration` — FE only
- LOW against `rebrand-historical-records` — metadata files only

## Out of Scope

- Implementing a full OAuth/credential refresh flow
- Modifying Claude CLI itself to add `auth status`
- Storing Claude credentials in Kiloforge's database
- Checking for specific Claude subscription tier or quota

## Technical Notes

### Auth probe implementation

```go
// backend/internal/adapter/prereq/auth.go

// CheckClaudeAuth verifies the Claude CLI is authenticated by running
// a lightweight probe command. Returns nil if authenticated.
func CheckClaudeAuth(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, "claude", "-p", ".", "--max-turns", "0")
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
        // Non-auth error (timeout, crash) — warn but don't block
        return nil
    }
    return nil
}
```

### Process-level cache

```go
var (
    authChecked   bool
    authCheckErr  error
    authCheckOnce sync.Once
)

func CheckClaudeAuthCached(ctx context.Context) error {
    authCheckOnce.Do(func() {
        authCheckErr = CheckClaudeAuth(ctx)
        authChecked = true
    })
    return authCheckErr
}
```

### Spawner integration

In `spawner.go`, add as the first validation before `ValidateSkills()`:

```go
func (s *Spawner) preflightChecks(ctx context.Context) error {
    // 1. Auth check (cached)
    if err := prereq.CheckClaudeAuthCached(ctx); err != nil {
        return fmt.Errorf("claude auth: %w", err)
    }
    // 2. Skill check
    if err := s.ValidateSkills(); err != nil {
        return err
    }
    // 3. Consent check (handled by caller for CLI, by middleware for REST)
    return nil
}
```

---

_Generated by conductor-track-generator from prompt: "check if the user has already logged in to claude before spawn checks"_
