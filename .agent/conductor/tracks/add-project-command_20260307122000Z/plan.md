# Implementation Plan: Implement 'crelay add' Command for Project Registration

**Track ID:** add-project-command_20260307122000Z

## Phase 1: Project Registry

Build the data layer for tracking registered projects.

### Task 1.1: Define project registry types [x]
- Create `internal/project/registry.go`
- Define `Project` struct: `Slug`, `RepoName`, `ProjectDir`, `OriginRemote`, `RegisteredAt`, `Active`
- Define `Registry` struct: `Version`, `Projects map[string]Project`
- Methods: `Load(dataDir) (*Registry, error)`, `Save(dataDir) error`
- `Get(slug) (Project, bool)`, `Add(project Project) error` (errors if slug exists)
- `FindByDir(dir string) (Project, bool)` — look up by project directory
- `List() []Project` — return all projects
- Tests: load/save round-trip, add duplicate errors, find by dir

### Task 1.2: Create project data directory helper [x]
- Add `EnsureProjectDir(dataDir, slug string) error` to create `~/.crelay/projects/<slug>/logs/`
- Tests: verify directory structure created

### Verification 1
- [x] Registry loads, saves, and queries correctly
- [x] Project data directories are created
- [x] Tests pass

## Phase 2: Add Command

Implement the core `crelay add` CLI command.

### Task 2.1: Implement origin remote detection [x]
- Create helper function `detectOriginRemote(ctx context.Context, repoPath string) (string, error)`
- Runs `git -C <path> remote get-url origin`
- Returns empty string (not error) if no origin remote exists
- Tests: mock git command behavior

### Task 2.2: Implement `crelay add` command [x]
- Create `internal/cli/add.go`
- Command: `crelay add [repo-path]` — defaults to cwd if no arg
- Flags: `--name` (override slug), `--origin` (override origin remote URL)
- Flow:
  1. Resolve repo path (arg or cwd)
  2. Verify it's a git repo (check `.git` exists)
  3. Load global config, verify Gitea is running via `CheckVersion()`
  4. Derive slug from basename or `--name`
  5. Load registry, check for duplicate
  6. Detect origin remote (or use `--origin`)
  7. Create Gitea client from config (token or basic auth)
  8. Create repo via `client.CreateRepo(slug)`
  9. Add `gitea` remote to project
  10. Push main branch to Gitea
  11. Create webhook via `client.CreateWebhook(slug, relayPort)`
  12. Create project data directory
  13. Register in projects.json
  14. Print success message with Gitea URL, origin remote, and slug

### Task 2.3: Handle idempotency and error cases [x]
- Already registered: print message and exit cleanly (not an error)
- Gitea not running: error with "run 'crelay init' first"
- Not a git repo: error with clear message
- No origin remote: warn but continue
- Gitea repo already exists (409 conflict): continue gracefully (repo may exist from prior run)
- Git remote `gitea` already exists: remove and re-add

### Task 2.4: Register add command in root.go [x]
- Add `rootCmd.AddCommand(addCmd)` in `internal/cli/root.go`
- Update the `init` command's success message to reference `crelay add <path>` instead of "(coming soon)"

### Verification 2
- [x] `crelay add <path>` creates repo, remote, webhook, and registers project
- [x] `crelay add` with no args uses cwd
- [x] `--name` and `--origin` flags work
- [x] Idempotent: re-running is a no-op
- [x] Errors clearly for: no Gitea, not a git repo
- [x] Origin remote captured and stored

## Phase 3: Projects List Command

### Task 3.1: Implement `crelay projects` command [x]
- Create `internal/cli/projects.go`
- Lists all registered projects from projects.json
- Table output: `SLUG  PATH  ORIGIN  REGISTERED  STATUS`
- If no projects: print helpful message directing to `crelay add`
- Register in root.go

### Task 3.2: Add relay port to global config [x]
- The webhook needs a relay port, but `RelayPort` was removed from global config in the init-docker-compose track
- Add `RelayPort` field back to `Config` struct (default 3001)
- The `add` command reads `cfg.RelayPort` for webhook registration
- If RelayPort is 0 in saved config (legacy), default to 3001

### Verification 3
- [x] `crelay projects` displays registered projects
- [x] Empty registry shows helpful message
- [x] RelayPort available for webhook creation

## Phase 4: Documentation and Cleanup

### Task 4.1: Update README.md [x]
- Add `crelay add` to Commands section with flags and example output
- Add `crelay projects` to Commands section
- Update Quick Start to show full flow: `crelay init` → `crelay add .`
- Add section explaining the origin bridging concept (work locally, push back later)
- Update Architecture diagram to show multi-project model

### Task 4.2: Update docs/ files [x]
- Update `docs/getting-started.md` with add flow
- Update `docs/commands.md` with new command signatures
- Update init success message placeholder

### Task 4.3: Final verification [x]
- `go build ./...` succeeds
- `go test ./...` passes
- Full cycle verified

### Verification 4
- [x] README documents add and projects commands
- [x] Docs are consistent
- [x] Build, tests, and lint pass
