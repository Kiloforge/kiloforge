# Implementation Plan: Migrate to Goose for Database Schema Migrations

**Track ID:** goose-migrations_20260309184000Z

## Phase 1: Setup Goose Infrastructure

- [x] Task 1.1: Add `github.com/pressly/goose/v3` to `go.mod`
- [x] Task 1.2: Create `backend/internal/adapter/persistence/sqlite/migrations/` directory
- [x] Task 1.3: Create `001_initial_schema.sql` — full V1 schema as goose up/down migration (up = all CREATE TABLE/INDEX statements, down = DROP in reverse order)

## Phase 2: Replace Custom Migrator

- [x] Task 2.1: Rewrite `migrate.go` — replace custom `Migrate()` with goose-based migration using `embed.FS`
- [x] Task 2.2: Add bridge logic — detect existing `schema_version` table, seed `goose_db_version` with V1 entry, drop `schema_version`
- [x] Task 2.3: Update `db.go` `Open()` if needed — no changes needed, `Migrate(db)` API unchanged
- [x] Task 2.4: Remove old `migration` struct, `migrations` slice, and `currentVersion()` function

## Phase 3: Tests

- [x] Task 3.1: Update `db_test.go` — test fresh database migration via goose
- [x] Task 3.2: Test idempotent migration (run `goose.Up` twice — no error)
- [x] Task 3.3: Test bridge — create database with old `schema_version` table, verify goose detects V1 as applied
- [x] Task 3.4: Test down migration — verify `goose.Down` rolls back cleanly

## Phase 4: Verification

- [x] Task 4.1: Verify `go test ./...` passes
- [x] Task 4.2: Verify `go build ./...` succeeds
- [x] Task 4.3: Verify existing `~/.kiloforge/kiloforge.db` upgrades cleanly (manual test)
