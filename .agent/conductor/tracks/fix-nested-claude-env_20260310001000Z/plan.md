# Implementation Plan: Fix Nested Claude Session Detection

**Track ID:** fix-nested-claude-env_20260310001000Z

## Phase 1: Create Helper and Apply

- [ ] Task 1.1: Create `cleanClaudeEnv()` helper in `spawner.go` — filters `CLAUDECODE` from `os.Environ()`
- [ ] Task 1.2: Set `cmd.Env = cleanClaudeEnv()` in `SpawnInteractive` (`spawner.go`)
- [ ] Task 1.3: Set `cmd.Env = cleanClaudeEnv()` in `SpawnReviewer` (`server.go`)
- [ ] Task 1.4: Set `cmd.Env = cleanClaudeEnv()` in `ResumeDeveloper` (`server.go`)

## Phase 2: Verification

- [ ] Task 2.1: `make test` passes
- [ ] Task 2.2: Interactive agent spawn works from dashboard when running inside Claude Code session
- [ ] Task 2.3: Track generation from dashboard produces actual output (not "exit 0")
