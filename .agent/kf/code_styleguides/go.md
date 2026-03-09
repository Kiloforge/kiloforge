# Go Style Guide & Architecture Guidelines

## Architecture: Layered Clean Architecture

Kiloforge follows a layered architecture with strict dependency direction: **domain → port → service → adapter → CLI**. Inner layers never import outer layers.

### Package Layout

```
cmd/
  kf/              — Entry point. Wires dependencies and starts the app.

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
    agent/          — Claude agent spawning & lifecycle.
    auth/           — SSH key detection, password generation.
    pool/           — Git worktree pool management.
    config/         — Config resolution (defaults, JSON, env, flags).
    lock/           — File locking for webhook concurrency.
    dashboard/      — Web dashboard with SSE real-time updates.
```

### Dependency Rules

1. **`domain/`** imports nothing from `internal/`. It defines entities, value objects, sentinel errors, and authorization primitives.
2. **`port/`** imports only `domain/`. It defines interfaces that adapters implement.
3. **`service/`** imports `domain/` and `port/`. It contains business logic and orchestrates operations through port interfaces.
4. **Adapters** (`rest/`, `gitea/`, `persistence/`, `compose/`, `cli/`, `agent/`, `auth/`, `pool/`, `config/`, `lock/`, `dashboard/`) import `domain/`, `port/`, and `service/`. They implement port interfaces and wire things together.
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

Every non-trivial package must have tests. Tests are first-class code — they document behavior, catch regressions, and guide design.

### Principles

- **Test every layer independently.** Domain, service, and adapter layers each have their own test suites.
- **Test both happy paths and error paths.** Every function that can fail must have tests for its failure modes.
- **Tests are documentation.** A reader should understand the behavior of a function by reading its tests.
- **No mocking frameworks.** Use interface-based design with in-memory adapter implementations and targeted test doubles.

### Test Organization

```
internal/core/domain/*_test.go                — Pure logic, no I/O. Highest coverage.
internal/core/service/*_test.go               — Business logic with in-memory adapters.
internal/core/testutil/                       — Shared mocks and test helpers.
internal/adapter/persistence/*_test.go        — Adapter integration tests.
internal/adapter/rest/*_test.go               — HTTP handler tests with httptest.
test/standalone/                              — Full-stack in-process integration tests.
test/e2e/                                     — End-to-end tests against running services.
```

### Build Tags for Test Separation

Use build tags to separate test tiers that have different requirements:

```go
//go:build standalone

// Package standalone contains integration tests that validate core workflows
// against an in-process server.
//
// Run with: go test ./test/standalone/ -v -tags=standalone -timeout 2m
package standalone
```

| Tag | Purpose | Dependencies |
|-----|---------|-------------|
| (none) | Unit tests, always run | None |
| `standalone` | In-process integration tests | In-memory stores, embedded services |
| `e2e` | End-to-end against running server | Running service at `CONTROL_PLANE_URL` |
| `dockertest` | Tests requiring Docker daemon | Docker engine |

### Table-Driven Tests

The primary pattern for parametric testing. Use for any function with multiple input/output scenarios.

```go
func TestRepoNameFromURL(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        url     string
        want    string
        wantErr bool
    }{
        {name: "ssh standard", url: "git@github.com:user/repo.git", want: "repo"},
        {name: "https with suffix", url: "https://github.com/user/repo.git", want: "repo"},
        {name: "https without suffix", url: "https://github.com/user/repo", want: "repo"},
        {name: "ssh no suffix", url: "git@github.com:user/my-project", want: "my-project"},
        {name: "empty url", url: "", wantErr: true},
        {name: "just a word", url: "repo", wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            got, err := repoNameFromURL(tt.url)
            if (err != nil) != tt.wantErr {
                t.Fatalf("repoNameFromURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("repoNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
            }
        })
    }
}
```

### Test Naming

```
TestFunctionName                        — Basic behavior
TestFunctionName_Scenario               — Specific scenario
TestFunctionName_Scenario_Expected      — When scenario name alone is ambiguous
```

Examples:
- `TestResolve_FlagsOverrideEnv`
- `TestAddProject_DuplicateSlug_ReturnsError`
- `TestClaimJob_SkipsPausedJobs`
- `TestUpdateJob_StoreError_Returns500`

### Test Doubles Strategy

**Three levels of test doubles, from simplest to most capable:**

#### 1. Stub implementations (for simple cases)

Override specific methods by embedding a base mock:

```go
type errUpdateStore struct {
    mockProjectStore // Embeds base with defaults
}

func (m *errUpdateStore) Add(ctx context.Context, p domain.Project) error {
    return fmt.Errorf("db: connection refused")
}
```

#### 2. In-memory adapter implementations (for stateful tests)

Full implementations backed by maps. Reusable across test suites.

```go
// testutil/mocks.go

type MockProjectStore struct {
    Projects map[string]domain.Project
}

func (m *MockProjectStore) Get(ctx context.Context, slug string) (domain.Project, error) {
    p, ok := m.Projects[slug]
    if !ok {
        return domain.Project{}, domain.ErrProjectNotFound
    }
    return p, nil
}

func (m *MockProjectStore) Add(ctx context.Context, p domain.Project) error {
    if m.Projects == nil {
        m.Projects = make(map[string]domain.Project)
    }
    if _, exists := m.Projects[p.Slug]; exists {
        return domain.ErrProjectExists
    }
    m.Projects[p.Slug] = p
    return nil
}
```

#### 3. Shared testutil package (for cross-package test infrastructure)

```go
// internal/core/testutil/mocks.go

// MockLogger is a silent logger for tests.
type MockLogger struct{}

func (l *MockLogger) Debug(msg string, kv ...interface{}) {}
func (l *MockLogger) Info(msg string, kv ...interface{})  {}
func (l *MockLogger) Warn(msg string, kv ...interface{})  {}
func (l *MockLogger) Error(msg string, kv ...interface{}) {}
func (l *MockLogger) With(kv ...interface{}) port.Logger  { return l }
```

### What Must Be Tested

#### Domain layer (highest coverage)

- All validation logic (every validation rule, both pass and fail)
- Sentinel error conditions
- State transitions and business rules
- Authorization: `HasPermission()` for each role × activity combination
- Value object parsing and formatting

#### Service layer

- Happy path for each operation
- Error propagation from port dependencies
- Business logic orchestration (e.g., "when dependency A fails, B should not run")
- Edge cases: empty inputs, duplicate operations, concurrent access

#### Adapter layer

- Persistence: save/load round-trips, not-found cases, duplicate handling
- HTTP handlers: correct status codes, request validation, error responses
- HTTP handlers: error details must NOT leak to clients (no DB connection strings, stack traces)
- External API clients: mock the HTTP server with `httptest.NewServer()`

#### CLI layer

- Argument and flag parsing
- Error messages for invalid input

### Error Path Testing

Every function that returns an error must have tests for its error conditions. Test that:

1. The correct sentinel error is returned (use `errors.Is()`)
2. Error context is preserved (wrapped errors contain useful messages)
3. Partial state is not left behind on failure
4. HTTP handlers return correct status codes for domain errors

```go
func TestAddProject_DuplicateSlug_ReturnsError(t *testing.T) {
    store := &testutil.MockProjectStore{
        Projects: map[string]domain.Project{
            "existing": {Slug: "existing"},
        },
    }
    svc := service.NewProjectService(store)

    err := svc.Add(ctx, domain.Project{Slug: "existing"})
    if !errors.Is(err, domain.ErrProjectExists) {
        t.Fatalf("expected ErrProjectExists, got: %v", err)
    }
}
```

### HTTP Handler Testing

Use `httptest` for handler tests. Verify status codes, response bodies, and that internal errors are not leaked.

```go
func TestWebhook_InvalidEvent_Returns400(t *testing.T) {
    srv := newTestServer(t)

    req := httptest.NewRequest("POST", "/webhook", strings.NewReader("{}"))
    req.Header.Set("X-Gitea-Event", "unknown_event")
    rec := httptest.NewRecorder()

    srv.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Errorf("status = %d, want 200", rec.Code)
    }
}

func TestWebhook_StoreError_DoesNotLeakDetails(t *testing.T) {
    // Verify that internal error messages (DB paths, connection strings)
    // are never exposed in HTTP responses.
    srv := newTestServer(t, withFailingStore())

    // ... make request ...

    body := rec.Body.String()
    for _, forbidden := range []string{"connection refused", "/var/run", "sqlite"} {
        if strings.Contains(body, forbidden) {
            t.Errorf("response leaked internal detail: %q", forbidden)
        }
    }
}
```

### Async and Concurrent Testing

For event-driven or goroutine-based code, use WaitGroup with timeout:

```go
func waitOrTimeout(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
    t.Helper()
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()
    select {
    case <-done:
    case <-time.After(timeout):
        t.Fatal("timed out waiting for async operation")
    }
}
```

### Test Execution

```bash
# Unit tests (always)
go test ./... -race -count=1

# With coverage
go test ./... -race -coverprofile=coverage.out
go tool cover -func=coverage.out

# Standalone integration
go test ./test/standalone/ -v -tags=standalone -timeout 2m

# E2E
go test ./test/e2e/ -v -tags=e2e -timeout 5m
```

### General Rules

- Use `t.Parallel()` for tests that don't share mutable state. Mark non-parallel tests with a comment explaining why.
- Use `t.Helper()` in all test helper functions.
- Use `t.TempDir()` for filesystem tests — automatic cleanup.
- Use `t.Setenv()` for environment variable tests (incompatible with `t.Parallel()`).
- Use `t.Context()` (Go 1.21+) for test-scoped context.
- Verify interface compliance at compile time: `var _ port.ProjectStore = (*JSONProjectStore)(nil)`
- Do not use `testify` unless it's already in the dependency graph. Standard `testing` is preferred.

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
- Environment variable prefix: `KF_` (e.g., `KF_GITEA_PORT`).

## SQL (SQLite)

- Use parameterized queries. Never interpolate user input into SQL strings.
- Use migrations for schema changes.
- Keep queries in the persistence adapter layer, never in domain or service.

## API Design: Schema-First Workflow

All HTTP endpoints and event-driven interfaces follow a **schema-first** approach. The schema is the source of truth — code is generated from it, never written by hand.

### When to Use OpenAPI vs AsyncAPI

| Protocol | Schema | Use Case |
|----------|--------|----------|
| REST/HTTP request-response | OpenAPI 3.1 | All `/-/api/*` endpoints, `/health` |
| Server-Sent Events (SSE) | AsyncAPI 3.0 | `/-/events` channel |
| Webhook payloads (consumed) | AsyncAPI 3.0 | Gitea webhook events at `/webhook` |
| Future WebSocket | AsyncAPI 3.0 | Bidirectional agent communication |

### Adding a New REST Endpoint

1. **Update the OpenAPI schema** (`backend/api/openapi.yaml`) — add path, request/response schemas
2. **Regenerate code** — `make gen-api` produces `*.gen.go` files
3. **Implement the generated interface** — write the handler method on your server struct
4. **Write tests** — test the handler using `httptest`
5. **Verify** — `make verify-codegen` ensures generated code matches schema

### Adding a New Event/Message Type

1. **Update the AsyncAPI schema** (`backend/api/asyncapi.yaml`) — add channel, message, and payload schema
2. **Implement the handler** — write Go code matching the documented schema
3. **Write tests** — test event serialization and handling

### Non-Standard Responses

Some endpoints return non-JSON content (SVG badges, SSE streams). These are handled alongside generated code:

- **SVG badge endpoints**: Documented in OpenAPI with `content: image/svg+xml` response type. Handler implementation is manual since code generators don't produce SVG renderers.
- **SSE endpoints**: Documented in AsyncAPI as channels. Handler implementation is manual — uses `text/event-stream` content type with chunked transfer encoding.
- **Webhook ingestion**: Documented in AsyncAPI as consumed channels. Payload parsing is manual since Gitea defines the types.

### Code Generation Conventions

- Generated files use `.gen.go` suffix — **never edit these files manually**
- Generation config lives in `backend/api/cfg.yaml`
- Run `make gen-api` after any schema change
- CI runs `make verify-codegen` to ensure generated code is up to date
- The `oapi-codegen` strict server mode generates an interface; adapters implement it
- Keep hand-written handler code in separate files from generated code
- **Always prefer strict typing** — use `oapi-codegen` strict server mode to generate typed interfaces. Avoid `interface{}` or `any` in generated models. Use typed enums, typed IDs, and strongly-typed request/response structs. If the generator produces loose types, tighten the schema constraints rather than accepting `any`
