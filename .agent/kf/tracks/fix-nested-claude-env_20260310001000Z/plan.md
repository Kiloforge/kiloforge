# Implementation Plan: Fix Nested Claude Session Detection

**Track ID:** fix-nested-claude-env_20260310001000Z

## Phase 1: Create Helper and Apply

- [x] Task 1.1: Create `CleanClaudeEnv()` helper in `spawner.go` — filters `CLAUDECODE` from `os.Environ()`
- [x] Task 1.2: Set `cmd.Env = CleanClaudeEnv()` in all spawner.go spawn methods (SpawnDeveloper, SpawnReviewer, SpawnInteractive)
- [x] Task 1.3: Set `cmd.Env = agent.CleanClaudeEnv()` in `SpawnReviewer` and `ResumeDeveloper` (`server.go`)
- [x] Task 1.4: Add local `cleanClaudeEnv()` to `prereq/auth.go` for `CheckClaudeAuth`

## Phase 2: Verification

- [x] Task 2.1: `make test` passes
- [x] Task 2.2: `make build` succeeds
- [x] Task 2.3: All exec.CommandContext("claude",...) calls have cleaned env
