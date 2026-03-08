# Research: Global Gitea Server for Multi-Project Coordination

**Track ID:** research-global-gitea-multiproject_20260307120001Z
**Date:** 2026-03-07
**Status:** Draft

---

## 1. Architecture: Global vs Per-Project Separation

### Current Model (Single-Project)

```
~/.kiloforge/
  config.json      # One project: ports, repo name, project dir, token
  state.json       # Flat agent list, no project association
  logs/            # All agent logs
  gitea-data/      # Docker volume (single Gitea instance)
```

Everything is singular: one config, one state file, one repo, one relay, one Gitea container named `conductor-gitea`.

### Proposed Model (Multi-Project)

**Principle:** Gitea and relay are **global** (workstation-level). Projects are **registered** with the global instance.

#### What is Global

| Component | Rationale |
|-----------|-----------|
| Gitea container | Single instance serves all projects — avoids port conflicts and resource waste |
| Relay server | Single webhook receiver — Gitea can only target one webhook URL per repo anyway |
| Ports (Gitea, relay) | Fixed per workstation — no per-project port management |
| API token | One admin user, one token — all repos under same Gitea user |
| Docker volume | Gitea data is inherently global (all repos live in one Gitea DB) |

#### What is Per-Project

| Component | Rationale |
|-----------|-----------|
| Gitea repo | Each project gets its own repo in the Gitea instance |
| Webhooks | Each repo has its own webhook pointing to the relay |
| Git remote | Each project dir gets a `gitea` remote added |
| Agents | Agents are spawned in a project context (worktree, track) |
| Logs | Agent logs belong to a project context |

### Proposed Directory Structure

```
~/.kiloforge/
  config.json              # Global config (ports, token, Gitea settings)
  projects.json            # Registry of all known projects
  gitea-data/              # Docker volume (shared)
  projects/
    <project-slug>/
      state.json           # Agent state for this project
      logs/
        <agent-id>.log
    <another-project>/
      state.json
      logs/
        ...
```

**Project slug** is derived from the repo name (e.g., `kiloforge`, `my-app`). It must be unique across all registered projects.

### Should the Relay Be Global?

**Recommendation: Yes, single global relay.**

Reasons:
1. Gitea webhooks fire to a URL — having one relay on one port is simplest
2. The relay already receives `repository.full_name` in every webhook payload — it can route by repo name
3. Running multiple relays on different ports adds complexity with no benefit
4. One relay can serve as the central orchestration point for all projects

The relay maintains an in-memory map of `repo_name -> project_config` loaded at startup from `projects.json`.

---

## 2. Config Schema Evolution

### Current Schema (`config.json`)

```json
{
  "gitea_port": 3000,
  "relay_port": 3001,
  "repo_name": "kiloforge",
  "project_dir": "/Users/dev/kiloforge",
  "data_dir": "/Users/dev/.kiloforge",
  "api_token": "abc123..."
}
```

Single-project fields mixed with global settings.

### Proposed Schema: Global Config (`config.json`)

```json
{
  "version": 2,
  "gitea_port": 3000,
  "relay_port": 3001,
  "data_dir": "/Users/dev/.kiloforge",
  "api_token": "abc123...",
  "container_name": "conductor-gitea"
}
```

Removed: `repo_name`, `project_dir` — these move to project registration.

### Proposed Schema: Project Registry (`projects.json`)

```json
{
  "version": 1,
  "projects": {
    "kiloforge": {
      "slug": "kiloforge",
      "repo_name": "kiloforge",
      "project_dir": "/Users/dev/kiloforge",
      "registered_at": "2026-03-07T12:00:00Z",
      "active": true
    },
    "my-app": {
      "slug": "my-app",
      "repo_name": "my-app",
      "project_dir": "/Users/dev/my-app",
      "registered_at": "2026-03-07T13:00:00Z",
      "active": true
    }
  }
}
```

### Go Structs

```go
// Global config — workstation-level settings
type GlobalConfig struct {
    Version       int    `json:"version"`
    GiteaPort     int    `json:"gitea_port"`
    RelayPort     int    `json:"relay_port"`
    DataDir       string `json:"data_dir"`
    APIToken      string `json:"api_token"`
    ContainerName string `json:"container_name"`
}

// Project entry in the registry
type Project struct {
    Slug         string    `json:"slug"`
    RepoName     string    `json:"repo_name"`
    ProjectDir   string    `json:"project_dir"`
    RegisteredAt time.Time `json:"registered_at"`
    Active       bool      `json:"active"`
}

// Project registry
type ProjectRegistry struct {
    Version  int                `json:"version"`
    Projects map[string]Project `json:"projects"`
}
```

### Config Loading Strategy

The current `Config` struct is used everywhere. Rather than changing all call sites at once, a **facade** can wrap both:

```go
// ProjectConfig provides a per-project view compatible with existing code
type ProjectConfig struct {
    Global  *GlobalConfig
    Project *Project
}

// Backward-compatible accessors
func (pc *ProjectConfig) GiteaPort() int    { return pc.Global.GiteaPort }
func (pc *ProjectConfig) RelayPort() int    { return pc.Global.RelayPort }
func (pc *ProjectConfig) RepoName() string  { return pc.Project.RepoName }
func (pc *ProjectConfig) ProjectDir() string { return pc.Project.ProjectDir }
func (pc *ProjectConfig) GiteaURL() string  { return fmt.Sprintf("http://localhost:%d", pc.Global.GiteaPort) }
```

This allows incremental migration — existing code that takes `*config.Config` can be updated to take an interface or the new struct gradually.

---

## 3. State Model Evolution

### Current State Schema (`state.json`)

```json
{
  "agents": [
    {
      "id": "uuid-1",
      "role": "developer",
      "ref": "auth_track",
      "status": "running",
      "session_id": "uuid-session",
      "pid": 12345,
      "worktree_dir": "/Users/dev/kiloforge",
      "log_file": "/Users/dev/.kiloforge/logs/uuid-1.log",
      "started_at": "...",
      "updated_at": "..."
    }
  ]
}
```

No project association. All agents in one flat list.

### Proposed State Schema

**Option A: Separate state files per project** (recommended)

```
~/.kiloforge/projects/kiloforge/state.json
~/.kiloforge/projects/my-app/state.json
```

Each state file has the same structure as today:
```json
{
  "agents": [...]
}
```

**Option B: Single state file with project field**

```json
{
  "agents": [
    {
      "id": "uuid-1",
      "project": "kiloforge",
      "role": "developer",
      ...
    }
  ]
}
```

### Recommendation: Option A (Separate State Files)

Reasons:
1. **Isolation** — one project's state can't corrupt another's
2. **Simplicity** — no need for project-aware queries in the Store struct
3. **Cleanup** — removing a project means deleting its directory
4. **Compatibility** — state file format is unchanged, only its location moves
5. **Cross-project queries** — `kf agents` (no project flag) can iterate all project state files and merge results

### Store Evolution

```go
// Load state for a specific project
func LoadProject(dataDir, projectSlug string) (*Store, error) {
    path := filepath.Join(dataDir, "projects", projectSlug, "state.json")
    return loadFrom(path)
}

// Load all agents across all projects (for `kf agents` without --project)
func LoadAll(dataDir string) (map[string]*Store, error) {
    // Iterate ~/.kiloforge/projects/*/state.json
    // Return map[projectSlug]*Store
}
```

### Log Files

Logs move to per-project directories:
```
~/.kiloforge/projects/kiloforge/logs/uuid-1.log
~/.kiloforge/projects/my-app/logs/uuid-2.log
```

---

## 4. Project Onboarding Flow

### Command: `kf add`

Run from within a project directory (or with `--project-dir` flag).

```
$ cd ~/my-app
$ kf add
```

#### Flow

```
1. Resolve project directory (cwd or --project-dir)
2. Derive slug from directory name (or --name flag)
3. Check: is global Gitea running?
   - No → start it (same as current init, minus repo creation)
   - Yes → continue
4. Check: is project already registered?
   - Yes → error "project already registered, use kf remove first"
   - No → continue
5. Create repo in Gitea: POST /api/v1/user/repos {name: slug}
6. Add git remote: git remote add gitea http://.../<slug>.git
7. Push main branch: git push gitea main
8. Create webhook: POST /api/v1/repos/<owner>/<slug>/hooks
9. Create project directory: mkdir -p ~/.kiloforge/projects/<slug>/logs/
10. Register in projects.json
11. Notify relay to reload project registry (if relay is running)
```

#### Edge Cases

| Scenario | Handling |
|----------|----------|
| Project already registered | Error with suggestion to `kf remove` first |
| Project dir moved | `kf add` from new location; old registration is stale |
| Repo name conflict | Error — user must choose different `--name` |
| Gitea not running | Auto-start Gitea (global init) |
| No git repo in project dir | Error — must be a git repository |

### Command: `kf remove`

```
$ cd ~/my-app
$ kf remove
```

#### Flow

```
1. Resolve project from cwd (or --name flag)
2. Stop all agents for this project
3. Remove webhook from Gitea
4. Remove git remote
5. Optionally delete Gitea repo (--delete-repo flag)
6. Remove project from projects.json
7. Remove ~/.kiloforge/projects/<slug>/ directory
```

---

## 5. Webhook Routing Strategy

### Current Behavior

The relay receives all webhooks at `POST /webhook`. It doesn't distinguish which project triggered the event because there's only one project.

### Proposed: Single Relay with Repo-Name Routing

Gitea webhook payloads include `repository.full_name` (e.g., `conductor/kiloforge`). The relay uses this to look up the project.

#### Implementation

```go
func (s *Server) handleWebhook(c *fiber.Ctx) error {
    var payload map[string]any
    json.Unmarshal(c.Body(), &payload)

    // Extract repo name from payload
    repo := payload["repository"].(map[string]any)
    repoName := repo["name"].(string) // e.g., "kiloforge"

    // Look up project
    project, ok := s.projects[repoName]
    if !ok {
        s.logger.Printf("webhook from unknown repo: %s", repoName)
        return c.JSON(fiber.Map{"status": "ignored"})
    }

    // Route with project context
    eventType := c.Get("X-Gitea-Event")
    switch eventType {
    case "pull_request":
        return s.handlePullRequest(c, payload, project)
    // ...
    }
}
```

#### Why Not Per-Project Webhook URLs?

Option B (`/webhook/{project}`) was considered but rejected:
- Adds URL complexity for no gain — the payload already identifies the repo
- Requires custom webhook URLs per project during setup
- The current `/webhook` endpoint is simpler and works with any repo

#### Relay Project Registry Reload

When a project is added/removed, the relay needs to update its routing table. Options:
1. **File watch** — relay watches `projects.json` for changes (inotify/fsnotify)
2. **Signal** — `kf add` sends SIGHUP to relay process, relay reloads
3. **API endpoint** — `POST /api/reload` triggers reload

**Recommendation:** Option 2 (SIGHUP) — simple, standard Unix pattern, no extra dependencies.

---

## 6. CLI Command Restructuring

### Current Commands

```
kf init       # Set up everything (Gitea + repo + relay) — single project
kf status     # Show system status
kf agents     # List agents
kf logs       # View agent logs
kf attach     # Halt agent and get resume command
kf stop       # Halt agent
kf destroy    # Tear down everything
```

### Proposed Commands

```
kf init                   # Global: start Gitea + relay (no project)
kf add [--name slug]      # Register current project with Gitea
kf remove [--name slug]   # Unregister project
kf projects               # List all registered projects

kf status                 # Global status + per-project summary
kf agents [--project X]   # List agents (all or filtered)
kf logs <id>              # View agent logs (auto-resolves project)
kf attach <id>            # Halt agent and resume
kf stop <id>              # Halt agent

kf destroy                # Tear down Gitea container + all data
kf destroy --project X    # Remove just one project (alias for remove)
```

### Project Context Resolution

Many commands need to know which project they're operating on. Resolution order:

1. **Explicit flag:** `--project kiloforge`
2. **Current directory:** detect git root, look up in `projects.json` by `project_dir`
3. **All projects:** if neither, operate on all (e.g., `kf agents` shows all agents)

```go
func resolveProject(flagValue string) (*Project, error) {
    if flagValue != "" {
        return registry.Get(flagValue)
    }
    cwd, _ := os.Getwd()
    gitRoot := findGitRoot(cwd)
    for _, p := range registry.Projects {
        if p.ProjectDir == gitRoot {
            return &p, nil
        }
    }
    return nil, nil // no project context — caller decides behavior
}
```

### `init` Changes

Current `init` does everything. Proposed split:

| Current `init` step | New location |
|---------------------|-------------|
| Create data dir | `init` |
| Start Gitea container | `init` |
| Create admin user + token | `init` |
| Start relay server | `init` |
| Create repo | `add` |
| Add git remote | `add` |
| Push to Gitea | `add` |
| Create webhook | `add` |

`init` becomes idempotent — safe to run multiple times. If Gitea is already running, it's a no-op. If the relay is already running, it reconnects or reports.

---

## 7. Migration Path

### Detection

On any CLI command, check `config.json` for a `version` field:
- **Missing or `version: 1`** → old single-project format
- **`version: 2`** → new multi-project format

### Migration Steps

```
1. Read old config.json
2. Create new config.json (version: 2) with global fields only
3. Create projects.json with single project entry from old config
4. Create ~/.kiloforge/projects/<slug>/ directory
5. Move state.json → ~/.kiloforge/projects/<slug>/state.json
6. Move logs/* → ~/.kiloforge/projects/<slug>/logs/
7. Back up old config.json as config.json.v1.bak
```

### Code

```go
func MigrateIfNeeded(dataDir string) error {
    cfg, err := loadRaw(dataDir) // raw JSON map
    if err != nil { return err }

    version, _ := cfg["version"].(float64)
    if version >= 2 { return nil } // already migrated

    // Extract fields
    repoName := cfg["repo_name"].(string)
    projectDir := cfg["project_dir"].(string)

    // Create new global config
    global := GlobalConfig{
        Version:       2,
        GiteaPort:     int(cfg["gitea_port"].(float64)),
        RelayPort:     int(cfg["relay_port"].(float64)),
        DataDir:       dataDir,
        APIToken:      cfg["api_token"].(string),
        ContainerName: "conductor-gitea",
    }

    // Create project entry
    project := Project{
        Slug:         repoName,
        RepoName:     repoName,
        ProjectDir:   projectDir,
        RegisteredAt: time.Now(),
        Active:       true,
    }

    // Write new files
    saveGlobalConfig(dataDir, global)
    saveProjectRegistry(dataDir, map[string]Project{repoName: project})

    // Move state and logs
    projectDir := filepath.Join(dataDir, "projects", repoName)
    os.MkdirAll(filepath.Join(projectDir, "logs"), 0755)
    os.Rename(filepath.Join(dataDir, "state.json"), filepath.Join(projectDir, "state.json"))
    moveLogFiles(filepath.Join(dataDir, "logs"), filepath.Join(projectDir, "logs"))

    // Backup old config
    os.Rename(filepath.Join(dataDir, "config.json"), filepath.Join(dataDir, "config.json.v1.bak"))

    return nil
}
```

### Safety

- Migration is **non-destructive** — old config is backed up
- Migration runs **once** — version field prevents re-running
- If migration fails midway, the old config still exists as backup
- Gitea data volume is untouched — repos, webhooks, and DB are preserved

---

## 8. Open Questions (Require User Input)

1. **Global Gitea lifecycle** — Should `kf init` start Gitea as a long-running background daemon? Or should the user explicitly start/stop it? Current behavior blocks the terminal.

2. **Project naming** — Should the slug always match the directory name, or allow arbitrary names? Allowing arbitrary names adds complexity but avoids conflicts when two projects share a directory name.

3. **Relay as daemon** — Currently the relay runs in the foreground (blocks terminal in `init`). For multi-project, the relay should be a background process. Options: daemonize with PID file, or use `kf start`/`kf stop` commands.

4. **Gitea organizations** — Should each project get its own Gitea org for namespace isolation? Or keep all repos under the single `conductor` user? Orgs add isolation but complexity.

5. **SQLite migration** — The tech stack mentions SQLite for future state persistence. Should the multi-project migration target SQLite directly instead of evolving JSON files? This would simplify querying but adds a migration step.

---

## 9. Summary of Decisions

| Decision | Recommendation | Confidence |
|----------|---------------|------------|
| Gitea: global or per-project? | Global (one instance) | High |
| Relay: global or per-project? | Global (one instance, routes by repo) | High |
| Config: split or unified? | Split: global config + project registry | High |
| State: shared or per-project? | Per-project state files | High |
| Webhook routing | Repo-name from payload (no URL changes) | High |
| CLI restructuring | `init` (global) + `add/remove` (project) | High |
| Migration | Auto-detect v1, migrate to v2, backup old | High |
| Relay reload | SIGHUP signal | Medium |
| Project naming | Directory name by default, `--name` override | Medium |
| Relay lifecycle | Background daemon with PID file | Needs discussion |
| Gitea organizations | Single user (flat namespace) | Medium |
