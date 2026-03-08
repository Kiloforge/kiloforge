# Implementation Plan: Migrate All Skill References from conductor-* to kf-*

**Track ID:** skill-ref-migration_20260309173002Z

## Phase 1: Backend Source Code

- [x] Task 1.1: Update `backend/internal/adapter/agent/spawner.go` — change `/conductor-developer` to `/kf-developer`, `/conductor-reviewer` to `/kf-reviewer`, update comment
- [x] Task 1.2: Update `backend/internal/adapter/rest/server.go` — change `/conductor-reviewer` to `/kf-reviewer`
- [x] Task 1.3: Update `backend/internal/adapter/skills/installer_test.go` — change `conductor-developer/` and `conductor-reviewer/` to `kf-developer/` and `kf-reviewer/`
- [x] Task 1.4: Search for any other conductor-* references in Go source — update if found

## Phase 2: Documentation

- [x] Task 2.1: Update `backend/docs/architecture.md` — fix spawner command examples
- [x] Task 2.2: Update `backend/docs/design-agent-orchestration.md` — fix agent prompt references
- [x] Task 2.3: Update `backend/docs/getting-started.md` — fix CLI examples
- [x] Task 2.4: Update `README.md` — fix implements command description

## Phase 3: Conductor Artifacts

- [x] Task 3.1: Update `.agent/conductor/index.md` — change Getting Started references to `/kf-track-generator` and `/kf-new-track`
- [x] Task 3.2: Update `.agent/conductor/tracks.md` — change comment to `<!-- Tracks registered by /kf-new-track -->`

## Phase 4: Verification

- [x] Task 4.1: Run `go test ./...` — all tests pass
- [x] Task 4.2: Run `go build ./...` — builds succeed
- [x] Task 4.3: Grep for remaining `conductor-` references in Go source (excluding _archive, track specs) — confirm none remain
