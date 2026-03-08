# Implementation Plan: Rebrand crelay to kiloforge (CLI: kf)

**Track ID:** rebrand-kiloforge_20260309055250Z

## Phase 1: Go Module and Import Rename (Core — must be atomic)

- [ ] Task 1.1: Update `backend/go.mod` module line from `crelay` to `kiloforge`
- [ ] Task 1.2: Bulk rename all Go import paths from `"crelay/` to `"kiloforge/` across all .go files
- [ ] Task 1.3: Rename directory `backend/cmd/crelay/` to `backend/cmd/kf/`
- [ ] Task 1.4: Update Makefile — binary name to `bin/kf`, build path to `./cmd/kf`
- [ ] Task 1.5: Verify: `make build` succeeds

## Phase 2: CLI Identity

- [ ] Task 2.1: Update cobra root command `Use` from `"crelay"` to `"kf"` in `root.go`
- [ ] Task 2.2: Update cobra root command `Long` description to reference "kiloforge"
- [ ] Task 2.3: Update any subcommand help text that references "crelay" (e.g., init.go, add.go help strings)
- [ ] Task 2.4: Verify: `./bin/kf --help` shows correct branding

## Phase 3: Environment Variables and Config Paths

- [ ] Task 3.1: Rename env var prefix from `CRELAY_` to `KF_` in `env_adapter.go`
- [ ] Task 3.2: Update default data directory from `~/.crelay` to `~/.kiloforge` in `defaults.go`
- [ ] Task 3.3: Update env var references in `env_adapter_test.go`
- [ ] Task 3.4: Update env var references in `resolve_test.go`
- [ ] Task 3.5: Search for any other `CRELAY_` references in Go code and update
- [ ] Task 3.6: Verify: `make test` passes for config package

## Phase 4: API Specifications

- [ ] Task 4.1: Update OpenAPI spec title from "Crelay API" to "Kiloforge API" in `backend/api/openapi.yaml`
- [ ] Task 4.2: Update AsyncAPI spec title/description in `backend/api/asyncapi.yaml`
- [ ] Task 4.3: Regenerate API code: `make gen-api`
- [ ] Task 4.4: Verify: `make verify-codegen` passes

## Phase 5: Frontend

- [ ] Task 5.1: Update page title in `frontend/index.html` to "kiloforge"
- [ ] Task 5.2: Update heading in `frontend/src/App.tsx` from "crelay" to "kiloforge"
- [ ] Task 5.3: Update help text in `frontend/src/pages/OverviewPage.tsx` — change `crelay add` to `kf add`
- [ ] Task 5.4: Search for any other "crelay" references in frontend/ and update
- [ ] Task 5.5: Verify: frontend builds successfully

## Phase 6: Documentation

- [ ] Task 6.1: Update `README.md` — all command examples, paths, and project description
- [ ] Task 6.2: Update `backend/docs/architecture.md`
- [ ] Task 6.3: Update `backend/docs/commands.md`
- [ ] Task 6.4: Update `backend/docs/getting-started.md`
- [ ] Task 6.5: Update `backend/docs/design-agent-orchestration.md` (if references exist)
- [ ] Task 6.6: Update `backend/docs/research-docker-compose.md` (if references exist)
- [ ] Task 6.7: Update `backend/docs/research-global-gitea-multiproject.md` (if references exist)

## Phase 7: Conductor Metadata

- [ ] Task 7.1: Update `.agent/conductor/product.md` — project name and description
- [ ] Task 7.2: Update `.agent/conductor/index.md` — navigation hub title
- [ ] Task 7.3: Update `.agent/conductor/setup_state.json` — project_name field
- [ ] Task 7.4: Update `.agent/conductor/product-guidelines.md` if it references "crelay"
- [ ] Task 7.5: Update skill files that reference `CRELAY_RELAY_URL` to `KF_RELAY_URL`

## Phase 8: Final Verification

- [ ] Task 8.1: Run `make build` — binary compiles as `bin/kf`
- [ ] Task 8.2: Run `make test` — all tests pass
- [ ] Task 8.3: Run `make lint` — no lint errors
- [ ] Task 8.4: Grep for any remaining "crelay" references (case-insensitive) — verify only historical track specs remain
- [ ] Task 8.5: Run `./bin/kf --help` — verify CLI branding
