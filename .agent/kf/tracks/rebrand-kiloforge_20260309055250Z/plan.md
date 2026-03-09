# Implementation Plan: Rebrand kiloforge to kiloforge (CLI: kf)

**Track ID:** rebrand-kiloforge_20260309055250Z

## Phase 1: Go Module and Import Rename (Core — must be atomic)

- [x] Task 1.1: Update `backend/go.mod` module line from `kiloforge` to `kiloforge`
- [x] Task 1.2: Bulk rename all Go import paths from `"kiloforge/` to `"kiloforge/` across all .go files
- [x] Task 1.3: Rename directory `backend/cmd/kiloforge/` to `backend/cmd/kf/`
- [x] Task 1.4: Update Makefile — binary name to `bin/kf`, build path to `./cmd/kf`
- [x] Task 1.5: Verify: `make build` succeeds

## Phase 2: CLI Identity

- [x] Task 2.1: Update cobra root command `Use` from `"kiloforge"` to `"kf"` in `root.go`
- [x] Task 2.2: Update cobra root command `Long` description to reference "kiloforge"
- [x] Task 2.3: Update any subcommand help text that references "kiloforge" (e.g., init.go, add.go help strings)
- [x] Task 2.4: Verify: `./bin/kf --help` shows correct branding

## Phase 3: Environment Variables and Config Paths

- [x] Task 3.1: Rename env var prefix from `KF_` to `KF_` in `env_adapter.go`
- [x] Task 3.2: Update default data directory from `~/.kiloforge` to `~/.kiloforge` in `defaults.go`
- [x] Task 3.3: Update env var references in `env_adapter_test.go`
- [x] Task 3.4: Update env var references in `resolve_test.go`
- [x] Task 3.5: Search for any other `KF_` references in Go code and update
- [x] Task 3.6: Verify: `make test` passes for config package

## Phase 4: API Specifications

- [x] Task 4.1: Update OpenAPI spec title from "Kiloforge API" to "Kiloforge API" in `backend/api/openapi.yaml`
- [x] Task 4.2: Update AsyncAPI spec title/description in `backend/api/asyncapi.yaml`
- [x] Task 4.3: Regenerate API code: `make gen-api`
- [x] Task 4.4: Verify: `make verify-codegen` passes

## Phase 5: Frontend

- [x] Task 5.1: Update page title in `frontend/index.html` to "kiloforge"
- [x] Task 5.2: Update heading in `frontend/src/App.tsx` from "kiloforge" to "kiloforge"
- [x] Task 5.3: Update help text in `frontend/src/pages/OverviewPage.tsx` — change `kf add` to `kf add`
- [x] Task 5.4: Search for any other "kiloforge" references in frontend/ and update
- [x] Task 5.5: Verify: frontend builds successfully

## Phase 6: Documentation

- [x] Task 6.1: Update `README.md` — all command examples, paths, and project description
- [x] Task 6.2: Update `backend/docs/architecture.md`
- [x] Task 6.3: Update `backend/docs/commands.md`
- [x] Task 6.4: Update `backend/docs/getting-started.md`
- [x] Task 6.5: Update `backend/docs/design-agent-orchestration.md` (if references exist)
- [x] Task 6.6: Update `backend/docs/research-docker-compose.md` (if references exist)
- [x] Task 6.7: Update `backend/docs/research-global-gitea-multiproject.md` (if references exist)

## Phase 7: Conductor Metadata

- [x] Task 7.1: Update `.agent/conductor/product.md` — project name and description
- [x] Task 7.2: Update `.agent/conductor/index.md` — navigation hub title
- [x] Task 7.3: Update `.agent/conductor/setup_state.json` — project_name field
- [x] Task 7.4: Update `.agent/conductor/product-guidelines.md` if it references "kiloforge"
- [x] Task 7.5: Update skill files that reference `KF_RELAY_URL` to `KF_RELAY_URL`

## Phase 8: Final Verification

- [x] Task 8.1: Run `make build` — binary compiles as `bin/kf`
- [x] Task 8.2: Run `make test` — all tests pass
- [x] Task 8.3: Run `make lint` — no lint errors
- [x] Task 8.4: Grep for any remaining "kiloforge" references (case-insensitive) — verify only historical track specs remain
- [x] Task 8.5: Run `./bin/kf --help` — verify CLI branding
