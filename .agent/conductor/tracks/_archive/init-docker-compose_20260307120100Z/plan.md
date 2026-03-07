# Implementation Plan: Implement Init Command with Docker Compose and Global Gitea

**Track ID:** init-docker-compose_20260307120100Z

## Phase 1: Docker Compose Runner

Build the compose CLI abstraction layer.

### Task 1.1: Create compose CLI detection and runner
- Create `internal/compose/runner.go`
- Implement `Detect() (Runner, error)` that tries `docker compose version` then `docker-compose version`
- `Runner` struct stores the resolved command (e.g., `["docker", "compose"]` or `["docker-compose"]`)
- Methods: `Up(ctx, dir)`, `Down(ctx, dir)`, `Ps(ctx, dir)`, `Exec(ctx, dir, service, cmd...)`, `Version() string`
- All methods accept a working directory where the `docker-compose.yml` lives
- Tests: mock exec commands, verify detection logic and fallback behavior

### Task 1.2: Create compose file generator
- Create `internal/compose/template.go`
- Function `GenerateComposeFile(cfg ComposeConfig) ([]byte, error)` that produces a `docker-compose.yml`
- `ComposeConfig` struct: `GiteaPort`, `DataDir`, `GiteaImage`
- Compose file defines: gitea service, port mapping, named volume, environment vars, health check, restart policy
- Tests: verify generated YAML is valid and contains expected fields

### Verification 1
- [x] `compose.Detect()` finds the correct CLI variant
- [x] `compose.GenerateComposeFile()` produces valid YAML
- [x] Unit tests pass

## Phase 2: Config Evolution

Split config into global-only schema.

### Task 2.1: Refactor config to global-only
- Remove `RepoName` and `ProjectDir` from `Config` struct
- Add `ComposeFile` field (path to generated docker-compose.yml)
- Update `Save()` / `Load()` / `LoadFrom()` accordingly
- Update all call sites that reference removed fields (compile errors will guide this)
- Temporarily stub out or comment project-specific CLI commands that depend on removed fields (`agents`, `logs`, `attach`, `stop`) — these will be restored in a future track when project context is added back
- Tests: verify config serialization/deserialization with new schema

### Verification 2
- [x] Config struct contains only global fields
- [x] Config saves and loads correctly
- [x] Project compiles with updated call sites

## Phase 3: Rewrite Init Command

Replace the init flow with compose-based Gitea startup.

### Task 3.1: Rewrite `runInit` in `internal/cli/init.go`
- Remove all project-specific steps (repo creation, git remote, webhook, relay start)
- New flow: detect compose CLI → create data dir → generate compose file → `compose up -d` → wait ready → configure admin → save config → print success
- Use `compose.Runner` for all Docker operations
- Keep `--gitea-port` and `--data-dir` flags; remove `--repo` and `--relay-port` flags for now
- Add idempotency: if Gitea is already running, print status and exit cleanly

### Task 3.2: Rewrite `internal/gitea/manager.go`
- Remove `Start()` method (replaced by compose runner)
- Keep `waitReady()` (still needed to poll Gitea API after compose up)
- Keep `Configure()` but update `docker exec` to use compose runner's `Exec()` method
- Remove `SetupGitRemote()` (project-specific, moves to future `crelay add`)
- Update `NewManager` to accept compose runner

### Task 3.3: Integration test for init flow
- Test the full init sequence end-to-end (may require Docker available)
- Verify: compose file created, Gitea accessible, admin user exists, config saved
- If Docker not available in CI, mark as integration test with build tag

### Verification 3
- [x] `crelay init` starts Gitea via docker-compose
- [x] Gitea web UI accessible at configured port
- [x] Running `crelay init` again is idempotent
- [x] No project-specific operations in init

## Phase 4: Update Destroy and Status

Make destroy and status compose-aware.

### Task 4.1: Rewrite `internal/cli/destroy.go`
- Replace `docker stop` + `docker rm` with `compose.Runner.Down()`
- `--data` flag triggers `docker compose down --volumes` plus data dir removal
- Remove git remote removal (project-specific)
- Load config to find compose file location

### Task 4.2: Rewrite `internal/cli/status.go`
- Replace `docker inspect` with compose-aware check (try compose ps, fall back to API health check)
- Remove project-specific output (repo name, project dir)
- Show: Gitea status + URL, data directory, compose file location
- Remove relay and agent status for now (will return with project context)

### Verification 4
- [x] `crelay destroy` tears down via compose
- [x] `crelay destroy --data` removes volumes and data
- [x] `crelay status` correctly reports Gitea state

## Phase 5: README and Documentation

### Task 5.1: Update README.md
- Update Prerequisites section (add docker-compose requirement note for Colima users)
- Rewrite Quick Start to show the new `crelay init` flow
- Update Commands section: `init`, `status`, `destroy` with new behavior
- Remove references to project-specific init behavior (repo, remote, webhook)
- Add note that project registration (`crelay add`) is coming
- Update Architecture diagram to show global Gitea model
- Update Data Directory section with compose file

### Task 5.2: Update docs/ files
- Update `docs/getting-started.md` with new init flow
- Update `docs/commands.md` with new command signatures
- Update `docs/architecture.md` with global Gitea architecture

### Task 5.3: Temporarily disable project-specific CLI commands
- Comment out or gate `agents`, `logs`, `attach`, `stop` command registration in `root.go`
- These commands depend on project context that no longer exists in global config
- Add TODO comments noting they'll be restored with `crelay add` track
- Keep the source files — just don't register the commands

### Verification 5
- [x] README accurately describes current behavior
- [x] `crelay --help` shows only working commands (init, status, destroy)
- [x] Docs are consistent with implementation
- [x] Project builds and all tests pass
