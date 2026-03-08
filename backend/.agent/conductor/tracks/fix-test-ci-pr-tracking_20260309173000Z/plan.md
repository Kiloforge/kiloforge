# Implementation Plan: Fix CI Test Failure — PR Tracking Store References

**Track ID:** fix-test-ci-pr-tracking_20260309173000Z

## Phase 1: Fix Test Store References

- [x] Task 1.1: Update `backend/internal/adapter/rest/server_test.go` — replace jsonfile PR tracking store with in-memory SQLite
- [x] Task 1.2: Update `backend/internal/adapter/rest/routes_test.go` — replace all jsonfile PR tracking and agent store references with SQLite
- [x] Task 1.3: Fix `backend/internal/adapter/rest/integration_test.go` — update `NewServer` call to include all required parameters using SQLite stores
- [x] Task 1.4: Check for any other test files using jsonfile stores where production uses SQLite — update if found

## Phase 2: Verification

- [x] Task 2.1: Run `go test -count=1 ./...` (no cache) — all tests pass
- [x] Task 2.2: Run `go test -count=1 ./internal/adapter/rest/...` specifically — verify REST tests pass
- [x] Task 2.3: Verify `go vet ./...` and `go build ./...` pass
