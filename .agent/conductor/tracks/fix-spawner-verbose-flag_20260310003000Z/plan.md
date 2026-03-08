# Implementation Plan: Fix Agent Spawner — Add --verbose Flag

**Track ID:** fix-spawner-verbose-flag_20260310003000Z

## Phase 1: Apply Fix

- [ ] Task 1.1: Add `"--verbose"` to `args` in `SpawnReviewer` (`spawner.go:172`)
- [ ] Task 1.2: Add `"--verbose"` to `args` in `SpawnDeveloper` (`spawner.go:275`)
- [ ] Task 1.3: Add `"--verbose"` to `args` in `SpawnInteractive` (`spawner.go:385`)
- [ ] Task 1.4: Add `"--verbose"` to `args` in `defaultSpawner.SpawnReviewer` (`server.go:172`)
- [ ] Task 1.5: Update `CleanClaudeEnv()` to also strip `CLAUDE_CODE_ENTRYPOINT`

## Phase 2: Verification

- [ ] Task 2.1: `make test` passes
