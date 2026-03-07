# Go Style Guide & Architecture Guidelines

## Architecture: Layered Clean Architecture

crelay follows a layered architecture with strict dependency direction: **domain → port → service → adapter → CLI**. Inner layers never import outer layers.

### Package Layout

```
cmd/
  crelay/          — Entry point. Wires dependencies and starts the app.

internal/
  core/
    domain/        — Entities, value objects, sentinel errors, authorization.
                     Pure Go. No external dependencies. No I/O.
    port/          — Interfaces (contracts) for persistence and services.
                     Depends only on domain.
    service/       — Business logic orchestration. Depends on domain + port.
                     Never imports adapters directly.

  adapter/
    rest/           — HTTP handlers, middleware, webhook relay.
    gitea/          — Gitea REST API client and Docker management.
    compose/        — Docker Compose CLI abstraction.
    persistence/    — Data access implementations (JSON files, SQLite, etc.).
    cli/            — Cobra command definitions. Thin: parse args, call services.

  agent/           — Claude agent spawning & lifecycle.
  auth/            — SSH key detection, password generation.
  pool/            — Git worktree pool management.
```

### Dependency Rules

1. **`domain/`** imports nothing from `internal/`. It defines entities, value objects, sentinel errors, and authorization primitives.
2. **`port/`** imports only `domain/`. It defines interfaces that adapters implement.
3. **`service/`** imports `domain/` and `port/`. It contains business logic and orchestrates operations through port interfaces.
4. **Adapters** (`rest/`, `gitea/`, `persistence/`, `compose/`, `cli/`) import `domain/`, `port/`, and `service/`. They implement port interfaces and wire things together.
5. **`cmd/`** imports adapters and services to wire the dependency graph.

### Why This Matters

- Domain logic is testable without I/O.
- Swapping persistence (JSON → SQLite) requires no domain changes.
- Services are testable with interface mocks — no real HTTP, filesystem, or Docker needed.

---

## Domain Layer

### Entities and Value Objects

Define domain types as plain Go structs. Use typed constants for enumerations.

```go
// domain/project.go

type ProjectStatus string

const (
    ProjectStatusActive   ProjectStatus = "active"
    ProjectStatusInactive ProjectStatus = "inactive"
)

type Project struct {
    Slug         string
    RepoName     string
    ProjectDir   string
    OriginRemote string
    Status       ProjectStatus
    RegisteredAt time.Time
}
```

### Sentinel Errors

Define sentinel errors in the domain package for errors that callers need to check.

```go
// domain/errors.go

var (
    ErrProjectNotFound = errors.New("project not found")
    ErrProjectExists   = errors.New("project already registered")
    ErrForbidden       = errors.New("forbidden: insufficient permissions")
    ErrGiteaUnreachable = errors.New("gitea server is not reachable")
)
```

Callers check with `errors.Is(err, domain.ErrProjectNotFound)`.

### Authorization: Activity-Based RBAC

Authorization uses an **activity model** — each operation declares what permission it requires, and roles map to sets of activities.

#### Activities

```go
// domain/authz.go

type Activity string

type Authorizable interface {
    RequiredActivity() Activity
}

const (
    ActivityProjectAdd    Activity = "project:add"
    ActivityProjectList   Activity = "project:list"
    ActivityAgentSpawn    Activity = "agent:spawn"
    ActivityAgentStop     Activity = "agent:stop"
    ActivityPRMerge       Activity = "pr:merge"
    ActivityPREscalate    Activity = "pr:escalate"
)
```

#### Role → Activity Mapping

```go
var RoleActivities = map[string][]Activity{
    "admin":     {/* implicitly has all */},
    "developer": {ActivityProjectList, ActivityAgentSpawn, ActivityPRMerge},
    "reviewer":  {ActivityProjectList, ActivityPREscalate},
}

func HasPermission(role string, activity Activity) bool {
    if role == "admin" { return true }
    for _, a := range RoleActivities[role] {
        if a == activity { return true }
    }
    return false
}
```

#### Authorization Registry

The registry maps operation IDs (e.g., HTTP route names, CLI command names) to their required activity. Built at init time.

```go
// domain/authz_registry.go

type AuthzRegistry struct {
    ops map[string]Activity
}

func (r *AuthzRegistry) Register(operationID string, cmd Authorizable)
func (r *AuthzRegistry) ActivityForOperation(operationID string) (Activity, bool)
```

Middleware looks up the operation, gets the required activity, and checks `HasPermission(role, activity)`.

---

## Port Layer

### Interface Design

Define interfaces in `port/` — one file per concern. Interfaces belong to the **consumer**, not the implementor.

```go
// port/project_store.go

type ProjectStore interface {
    Get(ctx context.Context, slug string) (domain.Project, error)
    List(ctx context.Context) ([]domain.Project, error)
    Add(ctx context.Context, p domain.Project) error
    FindByRepoName(ctx context.Context, repoName string) (domain.Project, bool, error)
}
```

#### Conventions

- Accept `context.Context` as the first parameter.
- Return domain types, not adapter-specific types.
- Lookup-by-ID methods return `domain.ErrXxxNotFound` when not found.
- Document the not-found convention in `port/doc.go`.
- Keep interfaces small and focused — prefer multiple small interfaces over one large one.

```go
// port/doc.go

// Package port defines persistence and service interfaces.
//
// Store methods that look up a single entity by ID return a domain sentinel
// error (e.g., domain.ErrProjectNotFound) when the entity is not found.
// Callers check with errors.Is(err, domain.Err*NotFound).
package port
```

### Logger Interface

```go
// port/logger.go

type Logger interface {
    Debug(msg string, keysAndValues ...interface{})
    Info(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
    With(keysAndValues ...interface{}) Logger
}
```

---

## Service Layer

Services contain business logic and depend only on domain types and port interfaces. They receive dependencies via constructor injection.

```go
// service/orchestrator.go

type Orchestrator struct {
    logger   port.Logger
    projects port.ProjectStore
    agents   port.AgentSpawner
    pool     port.WorktreePool
}

func NewOrchestrator(
    logger port.Logger,
    projects port.ProjectStore,
    agents port.AgentSpawner,
    pool port.WorktreePool,
) *Orchestrator {
    return &Orchestrator{
        logger:   logger,
        projects: projects,
        agents:   agents,
        pool:     pool,
    }
}
```

Optional dependencies use setter methods:

```go
func (o *Orchestrator) SetAuditLogger(al port.AuditLogger) {
    o.audit = al
}
```

---

## Adapter Layer

Adapters implement port interfaces. Each adapter lives in its own package under `adapter/`.

### Persistence Adapters

```
adapter/
  persistence/
    jsonfile/      — JSON file-based implementations
    memory/        — In-memory implementations (for tests)
    sqlitedb/      — SQLite implementations (future)
```

Each implementation satisfies one or more port interfaces:

```go
// adapter/persistence/jsonfile/project_store.go

type ProjectStore struct {
    dataDir string
}

func NewProjectStore(dataDir string) *ProjectStore { ... }

func (s *ProjectStore) Get(ctx context.Context, slug string) (domain.Project, error) { ... }
func (s *ProjectStore) Add(ctx context.Context, p domain.Project) error { ... }
```

### REST / HTTP Adapters

Handlers are thin — validate input, call a service, return a response.

```go
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    event := r.Header.Get("X-Gitea-Event")
    payload, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }
    if err := h.orchestrator.ProcessEvent(r.Context(), event, payload); err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusOK)
}
```

### CLI Adapters

CLI commands are thin wrappers. They parse flags, construct services, and call methods.

```go
func runAdd(cmd *cobra.Command, args []string) error {
    cfg, err := config.Resolve(...)
    // ... construct dependencies
    svc := service.NewProjectService(store, giteaClient)
    return svc.AddProject(ctx, args[0])
}
```

---

## Formatting

- Use `gofmt` / `goimports` for all code. No exceptions.
- Line length: no hard limit, but prefer readability. Break long function signatures across lines.

## Naming

- **Packages**: Short, lowercase, single-word. No underscores or mixed caps. (`domain`, `relay`, `agent`)
- **Exported names**: PascalCase. Descriptive but not verbose. (`SpawnReviewer`, `LoadState`)
- **Unexported names**: camelCase. (`handleWebhook`, `parseEvent`)
- **Interfaces**: Name by behavior, not by the type implementing them. Use `-er` suffix when appropriate. (`Spawner`, `ProjectStore`)
- **Acronyms**: All caps when exported (`HTTPServer`, `APIURL`), all lowercase when unexported (`httpServer`, `apiURL`)
- **Test files**: `*_test.go` in the same package for unit tests. `*_test.go` in `_test` package for black-box tests.

## Build

- Build artifacts go in `.build/` (gitignored). Never output binaries to the project root.
- The Makefile sets `GIT_WORK_TREE` for worktree compatibility.

## Error Handling

- Return errors, don't panic. Reserve `panic` for truly unrecoverable programmer errors.
- Wrap errors with context using `fmt.Errorf("operation: %w", err)`.
- Use sentinel errors (`var ErrNotFound = errors.New(...)`) in the domain package for errors callers need to check.
- Check errors immediately. Never ignore returned errors without explicit justification.
- Adapter layers translate external errors into domain sentinel errors when appropriate.

## Functions

- Keep functions short and focused. A function should do one thing.
- Prefer returning `(result, error)` over using output parameters.
- Use named return values sparingly — only when they improve readability of short functions.
- Constructor functions: `NewXxx(...)` pattern with all required dependencies as parameters.

## Concurrency

- Document goroutine ownership. Every goroutine should have a clear owner responsible for its lifecycle.
- Use `context.Context` for cancellation and timeouts. Pass it as the first parameter.
- Prefer channels for communication, mutexes for state protection.
- Always handle channel closure and context cancellation.

## Testing

- Table-driven tests for multiple cases.
- Use standard `testing` package. `testify` assertions allowed if already in the dependency graph.
- Test function names: `TestFunctionName_Scenario_ExpectedBehavior`.
- Use `t.Helper()` in test helper functions.
- Use `t.Parallel()` where safe.
- Prefer in-memory adapter implementations as test doubles over mocking frameworks.
- Interface-based design eliminates the need for mocking libraries.

### Test Organization

```
internal/core/service/orchestrator_test.go   — Unit tests with in-memory adapters
internal/adapter/persistence/jsonfile/*_test.go — Adapter integration tests
test/standalone/                              — Full-stack single-process tests
test/e2e/                                     — Multi-instance integration tests
```

## Dependencies

- Minimize external dependencies. Prefer the standard library.
- Vet new dependencies for maintenance status, license compatibility, and transitive dependency count.

## Comments

- Package comments: one per package, in `doc.go` or the primary file.
- Exported functions: godoc-style comments starting with the function name.
- Don't comment obvious code. Comment _why_, not _what_.

## Configuration

- Use the port/adapter pattern for configuration.
- `port.Config` interface abstracts config access.
- Adapters: defaults, JSON file, environment variables, CLI flags.
- Resolution order (highest priority last): defaults → JSON → env → flags.
- Environment variable prefix: `CRELAY_` (e.g., `CRELAY_GITEA_PORT`).

## SQL (SQLite)

- Use parameterized queries. Never interpolate user input into SQL strings.
- Use migrations for schema changes.
- Keep queries in the persistence adapter layer, never in domain or service.
