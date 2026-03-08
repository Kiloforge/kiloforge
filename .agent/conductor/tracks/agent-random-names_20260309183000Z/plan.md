# Implementation Plan: Random Human-Friendly Agent Names

**Track ID:** agent-random-names_20260309183000Z

## Phase 1: Name Generator and Domain

- [ ] Task 1.1: Create `backend/internal/adapter/agent/namegen.go` — word pools (30+ adverbs, 50+ adjectives, 50+ names) and `GenerateName()` function
- [ ] Task 1.2: Add `Name string` field to `AgentInfo` in `backend/internal/core/domain/agent.go`
- [ ] Task 1.3: Unit test for `GenerateName()` — verify format, non-empty, randomness over multiple calls

## Phase 2: Backend Integration

- [ ] Task 2.1: Wire `GenerateName()` into all three spawn methods in `spawner.go` — set `info.Name` at creation time
- [ ] Task 2.2: Add `name TEXT NOT NULL DEFAULT ''` to SQLite agents table migration
- [ ] Task 2.3: Update OpenAPI spec — add `name` field to Agent schema

## Phase 3: Frontend Display

- [ ] Task 3.1: Update `AgentCard` component — show name as primary heading, ID as secondary subtitle
- [ ] Task 3.2: Update agent detail page (if present) — show name as page heading

## Phase 4: Verification

- [ ] Task 4.1: Verify `go test ./...` passes
- [ ] Task 4.2: Verify frontend builds without errors
- [ ] Task 4.3: Verify spawned agents display names in dashboard
