# Implementation Plan: Migrate All Skill References from conductor-* to kf-*

**Track ID:** skill-ref-migration_20260309173002Z

## Phase 1: Backend Source Code

- [ ] Task 1.1: Update `backend/internal/adapter/agent/spawner.go` — change `/conductor-developer` to `/kf-developer`, `/conductor-reviewer` to `/kf-reviewer`, update comment
- [ ] Task 1.2: Update `backend/internal/adapter/rest/server.go` — change `/conductor-reviewer` to `/kf-reviewer`
- [ ] Task 1.3: Update `backend/internal/adapter/skills/installer_test.go` — change `conductor-developer/` and `conductor-reviewer/` to `kf-developer/` and `kf-reviewer/`
- [ ] Task 1.4: Search for any other conductor-* references in Go source — update if found

## Phase 2: Documentation

- [ ] Task 2.1: Update `backend/docs/architecture.md` — fix spawner command examples
- [ ] Task 2.2: Update `backend/docs/design-agent-orchestration.md` — fix agent prompt references
- [ ] Task 2.3: Update `backend/docs/getting-started.md` — fix CLI examples
- [ ] Task 2.4: Update `README.md` — fix implements command description

## Phase 3: Conductor Artifacts

- [ ] Task 3.1: Update `.agent/conductor/index.md` — change Getting Started references to `/kf-track-generator` and `/kf-new-track`
- [ ] Task 3.2: Update `.agent/conductor/tracks.md` — change comment to `<!-- Tracks registered by /kf-new-track -->`

## Phase 4: Verification

- [ ] Task 4.1: Run `go test ./...` — all tests pass
- [ ] Task 4.2: Run `go build ./...` — builds succeed
- [ ] Task 4.3: Grep for remaining `conductor-` references in Go source (excluding _archive, track specs) — confirm none remain
