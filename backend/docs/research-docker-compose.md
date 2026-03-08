# Research: Docker Compose for Init with v1/v2 CLI Compatibility

**Track:** research-docker-compose-init_20260307120000Z
**Date:** 2026-03-07
**Status:** Complete

## Executive Summary

Replace the raw `docker run` approach in `internal/gitea/manager.go` with docker-compose for the Gitea lifecycle. This gives us a declarative config, easy multi-service extension, and standard lifecycle commands. Both `docker compose` (v2 plugin) and `docker-compose` (v1 standalone) must be supported for Colima compatibility.

**Recommendation:** Adopt docker-compose with an embedded Go template, v2-first detection with v1 fallback.

---

## 1. Docker Compose CLI Variants

### v2 — `docker compose` (Docker CLI Plugin)

- Ships with Docker Desktop (macOS, Windows, Linux)
- Installed as a Go binary plugin at `~/.docker/cli-plugins/docker-compose`
- Invoked as a subcommand: `docker compose up -d`
- Version check: `docker compose version`
- Compose file spec: supports all modern features (profiles, `depends_on.condition`, etc.)

### v1 — `docker-compose` (Standalone Binary)

- Originally Python, later rewritten in Go (`compose-switch` or direct Go port)
- Installed separately via package managers: `brew install docker-compose`, `apt install docker-compose`
- Invoked as a standalone binary: `docker-compose up -d`
- Version check: `docker-compose version`
- Colima on macOS often only provides v1 (via `brew install docker-compose`)

### Syntax Differences

For our use case (single-service, basic compose file), there are **no meaningful syntax differences** between v1 and v2 in terms of CLI flags or compose file format. Both support:

- `up -d`, `down`, `ps`, `logs`, `exec`
- Compose file version 3.x (and both read `docker-compose.yml` by default)
- `-f` flag to specify a custom compose file path
- `--project-name` / `-p` for project naming

The only difference is invocation: `docker compose` vs `docker-compose`.

### Minimum Compose File Version

Use **version 3.8** (or omit version entirely — modern compose ignores it). Our compose file uses only basic features (services, ports, volumes, environment, healthcheck) that work across all versions.

---

## 2. CLI Detection Strategy

### Proposed Logic

Try v2 first (Docker CLI plugin), fall back to v1 (standalone binary). Fail with a clear error if neither is available.

### Pseudocode

```go
// ComposeRunner abstracts docker compose CLI invocation.
type ComposeRunner struct {
    args []string // e.g., ["docker", "compose"] or ["docker-compose"]
}

// Detect returns a ComposeRunner configured for the available CLI variant.
func DetectCompose() (*ComposeRunner, error) {
    // Try v2: docker compose version
    if err := exec.Command("docker", "compose", "version").Run(); err == nil {
        return &ComposeRunner{args: []string{"docker", "compose"}}, nil
    }

    // Try v1: docker-compose version
    if err := exec.Command("docker-compose", "version").Run(); err == nil {
        return &ComposeRunner{args: []string{"docker-compose"}}, nil
    }

    return nil, fmt.Errorf(
        "docker compose not found: install Docker Desktop (includes compose v2) " +
        "or install docker-compose standalone")
}

// Run executes a compose command (e.g., "up", "-d").
func (c *ComposeRunner) Run(ctx context.Context, projectDir string, composeFile string, args ...string) error {
    cmdArgs := append(c.args, "-f", composeFile, "-p", "kiloforge")
    cmdArgs = append(cmdArgs, args...)
    cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
    cmd.Dir = projectDir
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

### Edge Cases

| Case | Handling |
|------|----------|
| Neither installed | Clear error message with install instructions |
| v2 installed but Docker daemon not running | `docker compose version` succeeds (it's just the plugin), but `up` will fail — handle at `up` time |
| Both installed | v2 takes priority (more actively maintained) |
| Broken install (binary exists but crashes) | `version` command will fail, fall through to next variant |

---

## 3. Colima-Specific Considerations

### `host.docker.internal`

- **Docker Desktop:** Automatically resolves `host.docker.internal` to the host machine.
- **Colima:** Does NOT provide `host.docker.internal` by default.
- **Workaround:** Use `--network host` or add `extra_hosts: ["host.docker.internal:host-gateway"]` in the compose file. The `host-gateway` magic string is supported by Docker Engine 20.10+ and resolves to the host IP.
- **Current code impact:** `init.go:106` registers the webhook with `host.docker.internal`. This already works with Docker Desktop but will need the `extra_hosts` entry for Colima.

### Volume Mounts

- Colima uses a Linux VM (via Lima) — bind mounts go through the VM's filesystem.
- Performance is generally fine for small data dirs like Gitea's `/data`.
- Named volumes (recommended for compose) perform the same as with Docker Desktop.
- **Recommendation:** Use a named volume in the compose file instead of the current bind mount (`-v ${DataDir}/gitea-data:/data`). This simplifies path handling and avoids cross-platform path issues.

### Network Mode

- Default bridge networking works identically on Colima and Docker Desktop.
- No special configuration needed for our single-service setup.

### Known Issues

- Colima's Docker socket is at `~/.colima/default/docker.sock` (not `/var/run/docker.sock`). This doesn't affect compose usage since the `docker` CLI handles socket discovery.
- File ownership in volumes can differ (Colima's VM user vs Docker Desktop's mapping). Gitea handles this internally, so no impact expected.

---

## 4. Proposed docker-compose.yml

```yaml
services:
  gitea:
    image: gitea/gitea:latest
    container_name: conductor-gitea
    restart: unless-stopped
    ports:
      - "${KF_GITEA_PORT:-3000}:3000"
    volumes:
      - gitea-data:/data
    environment:
      - GITEA__security__INSTALL_LOCK=true
      - GITEA__server__ROOT_URL=http://localhost:${KF_GITEA_PORT:-3000}/
      - GITEA__server__HTTP_PORT=3000
      - GITEA__database__DB_TYPE=sqlite3
      - GITEA__service__DISABLE_REGISTRATION=true
      - GITEA__webhook__ALLOWED_HOST_LIST=*
    extra_hosts:
      - "host.docker.internal:host-gateway"
    healthcheck:
      test: ["CMD", "curl", "-sf", "http://localhost:3000/api/v1/version"]
      interval: 5s
      timeout: 3s
      retries: 12
      start_period: 10s

volumes:
  gitea-data:
```

### Design Decisions

| Decision | Rationale |
|----------|-----------|
| Named volume `gitea-data` | Avoids host path issues across platforms; compose manages volume lifecycle |
| `extra_hosts` with `host-gateway` | Ensures `host.docker.internal` works on Colima |
| Environment variable interpolation for port | Allows runtime configuration without file modification |
| Built-in healthcheck | Replaces the manual `waitReady()` polling loop — compose tracks readiness natively |
| `restart: unless-stopped` | Container survives reboots; stops only with explicit `down` |
| `container_name` preserved | Backward compatibility; other commands reference it |

### Migration from Named Volume

The current code uses a bind mount at `${DataDir}/gitea-data:/data`. Switching to a named volume means existing data won't automatically transfer. Options:

1. **Document migration:** Users run `docker cp conductor-gitea:/data ./backup` before switching
2. **First-run detection:** If `${DataDir}/gitea-data` exists, use bind mount; otherwise use named volume
3. **Always use bind mount in compose:** Keep `${DataDir}/gitea-data:/data` syntax in compose

**Recommendation:** Option 3 for backward compatibility. Use bind mount in compose to preserve existing data directories. Named volumes can be a future optimization.

Updated compose file with bind mount:

```yaml
services:
  gitea:
    image: gitea/gitea:latest
    container_name: conductor-gitea
    restart: unless-stopped
    ports:
      - "${KF_GITEA_PORT:-3000}:3000"
    volumes:
      - "${KF_DATA_DIR}/gitea-data:/data"
    environment:
      - GITEA__security__INSTALL_LOCK=true
      - GITEA__server__ROOT_URL=http://localhost:${KF_GITEA_PORT:-3000}/
      - GITEA__server__HTTP_PORT=3000
      - GITEA__database__DB_TYPE=sqlite3
      - GITEA__service__DISABLE_REGISTRATION=true
      - GITEA__webhook__ALLOWED_HOST_LIST=*
    extra_hosts:
      - "host.docker.internal:host-gateway"
    healthcheck:
      test: ["CMD", "curl", "-sf", "http://localhost:3000/api/v1/version"]
      interval: 5s
      timeout: 3s
      retries: 12
      start_period: 10s
```

---

## 5. Compose File Management Strategy

### Options Evaluated

| Option | Approach | Pros | Cons |
|--------|----------|------|------|
| A. Template file | Ship `docker-compose.yml.tmpl`, copy to data dir | User can edit, visible | Extra file to manage, template drift |
| B. Go generation | Build YAML programmatically | Type-safe, no template syntax | Verbose, hard to read, YAML formatting issues |
| C. Embed + template | `//go:embed` with `text/template` | Single binary, no external files, templated | User can't easily customize |

### Recommendation: Option C — Embed with Go template

```go
import "embed"

//go:embed docker-compose.yml.tmpl
var composeTemplate string
```

At `init` time:
1. Render the template with config values (port, data dir)
2. Write the rendered `docker-compose.yml` to `${DataDir}/docker-compose.yml`
3. Use `-f ${DataDir}/docker-compose.yml` for all compose commands

This keeps the template in the binary (no external files to lose), renders with actual config values (no env var interpolation needed at runtime), and writes a concrete file the user can inspect or override.

---

## 6. Impact Analysis — Code Change Map

### `internal/gitea/manager.go`

| Current | Proposed | Lines |
|---------|----------|-------|
| `Start()` — `docker run` with 10+ flags | `compose.Run("up", "-d")` | ~20 lines simplified to ~5 |
| `Start()` — `docker inspect` for status check | `compose.Run("ps", "--format", "json")` or keep `docker inspect` | ~8 lines |
| `Start()` — `docker start` for restart | Removed — `up -d` handles both create and start | ~3 lines removed |
| `waitReady()` — manual curl polling loop | Can leverage compose healthcheck + `docker compose up --wait` (v2 only) or keep polling | ~15 lines potentially simplified |

**New code needed:**
- `ComposeRunner` struct and `DetectCompose()` function (~30 lines)
- Compose template rendering and file writing (~20 lines)
- Manager gains `composeFile string` field and `ComposeRunner` reference

**Net effect:** Slight increase in total lines, but much better maintainability and extensibility.

### `internal/cli/init.go`

| Current | Proposed |
|---------|----------|
| `giteaManager.Start(ctx)` | Same call — Manager interface unchanged |
| No compose file handling | Add compose file generation before `Start()` |

Minimal changes — the Manager abstracts the lifecycle.

### `internal/cli/destroy.go`

| Current | Proposed |
|---------|----------|
| `docker stop` + `docker rm` (2 exec calls) | `compose.Run("down")` (1 call) |
| Manual container name reference | Compose handles by project name |
| `--data` flag removes data dir | Add `compose.Run("down", "-v")` to also remove volumes |

### `internal/cli/status.go`

| Current | Proposed |
|---------|----------|
| `docker inspect -f {{.State.Status}}` | `compose.Run("ps", "--format", "json")` or keep `docker inspect` |

Either approach works. Keeping `docker inspect` is simpler and avoids parsing compose JSON output.

### `internal/config/config.go`

| Current | Proposed |
|---------|----------|
| `ContainerName` constant used everywhere | Still used — compose file sets `container_name` |
| `GiteaImage` constant | Move to compose template (single source of truth) |

### Summary of Affected Functions

```
internal/gitea/manager.go
  ├── NewManager()      — Add ComposeRunner field
  ├── Start()           — Replace docker run with compose up
  ├── waitReady()       — Optionally simplify with compose healthcheck
  ├── NEW: generateComposeFile() — Render and write compose template
  └── NEW: DetectCompose()       — CLI variant detection

internal/cli/init.go
  └── runInit()         — Add compose file generation step

internal/cli/destroy.go
  └── runDestroy()      — Replace docker stop/rm with compose down

internal/cli/status.go
  └── runStatus()       — Minor: could use compose ps (optional)

internal/config/config.go
  └── Constants         — GiteaImage moves to compose template
```

---

## 7. Tradeoffs Summary

| Factor | docker run (current) | docker-compose (proposed) |
|--------|---------------------|---------------------------|
| Simplicity | Single command, no files | Requires compose file + detection |
| Extensibility | Hard (more flags, more code) | Easy (add services to YAML) |
| Readability | Flags buried in Go code | Declarative YAML |
| Multi-service | Manual orchestration | Native support |
| Health checks | Manual polling | Built-in compose healthcheck |
| Cleanup | Manual stop + rm | Single `down` command |
| Dependency | Docker only | Docker + compose |
| Colima compat | Works | Works with detection logic |

---

## 8. Recommended Implementation Approach

1. **Add `ComposeRunner`** to `internal/gitea/` — handles CLI detection and command execution
2. **Embed compose template** in `internal/gitea/compose.yml.tmpl`
3. **Render at init time** to `${DataDir}/docker-compose.yml`
4. **Update `Manager.Start()`** to use `compose up -d` instead of `docker run`
5. **Update `destroy.go`** to use `compose down`
6. **Keep `docker inspect`** in `status.go` for now (simpler, works with both approaches)
7. **Add `extra_hosts`** for Colima `host.docker.internal` support
8. **Keep `waitReady()` polling** as-is — the compose healthcheck is useful for `docker compose up --wait` (v2 only), but our polling approach works with both v1 and v2

This can be implemented in a single track with ~3 phases: compose runner + template, manager refactor, CLI updates.
