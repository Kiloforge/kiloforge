# Implementation Plan: Fix Agent Spawner — Add --verbose Flag

**Track ID:** fix-spawner-verbose-flag_20260310003000Z

## Phase 1: Apply Fix

- [x] Task 1.1: Add `"--verbose"` to `args` in `SpawnReviewer` (`spawner.go`)
- [x] Task 1.2: Add `"--verbose"` to `args` in `SpawnDeveloper` (`spawner.go`)
- [x] Task 1.3: Add `"--verbose"` to `args` in `SpawnInteractive` (`spawner.go`)
- [x] Task 1.4: Add `"--verbose"` to `args` in `defaultSpawner.SpawnReviewer` (`server.go`) and `recovery.go`
- [x] Task 1.5: Update `CleanClaudeEnv()` and `cleanClaudeEnv()` to also strip `CLAUDE_CODE_ENTRYPOINT`

## Phase 2: Verification

- [x] Task 2.1: `make test` and `make build` pass
