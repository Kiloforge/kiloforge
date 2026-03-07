# Go Style Guide

## Formatting

- Use `gofmt` / `goimports` for all code. No exceptions.
- Line length: no hard limit, but prefer readability. Break long function signatures across lines.

## Naming

- **Packages**: Short, lowercase, single-word names. No underscores or mixed caps. (`state`, `relay`, `agent`)
- **Exported names**: PascalCase. Descriptive but not verbose. (`SpawnReviewer`, `LoadState`)
- **Unexported names**: camelCase. (`handleWebhook`, `parseEvent`)
- **Interfaces**: Name by behavior, not by the type implementing them. Use `-er` suffix when appropriate. (`Spawner`, `StateStore`)
- **Acronyms**: All caps when exported (`HTTPServer`, `APIURL`), all lowercase when unexported (`httpServer`, `apiURL`)
- **Test files**: `*_test.go` in the same package for unit tests. `*_test.go` in `_test` package for black-box tests.

## Project Structure

```
cmd/           — Main applications (one per binary)
internal/      — Private application code (not importable)
  cli/         — CLI command definitions
  agent/       — Agent lifecycle management
  config/      — Configuration loading/saving
  gitea/       — Gitea API client and Docker management
  relay/       — Webhook relay HTTP server
  state/       — State persistence
```

## Error Handling

- Return errors, don't panic. Reserve `panic` for truly unrecoverable programmer errors.
- Wrap errors with context using `fmt.Errorf("operation: %w", err)`.
- Use sentinel errors (`var ErrNotFound = errors.New(...)`) for errors callers need to check.
- Check errors immediately. Never ignore returned errors without explicit justification.

## Functions

- Keep functions short and focused. A function should do one thing.
- Prefer returning `(result, error)` over using output parameters.
- Use named return values sparingly — only when they improve readability of short functions.
- Constructor functions: `NewXxx(...)` pattern.

## Concurrency

- Document goroutine ownership. Every goroutine should have a clear owner responsible for its lifecycle.
- Use `context.Context` for cancellation and timeouts. Pass it as the first parameter.
- Prefer channels for communication, mutexes for state protection.
- Always handle channel closure and context cancellation.

## Testing

- Table-driven tests for multiple cases.
- Use `testify` assertions if available, otherwise standard `testing` package.
- Test function names: `TestFunctionName_Scenario_ExpectedBehavior`.
- Use `t.Helper()` in test helper functions.
- Use `t.Parallel()` where safe.
- Prefer interfaces for dependency injection over mocking frameworks.

## Dependencies

- Minimize external dependencies. Prefer the standard library.
- Vet new dependencies for maintenance status, license compatibility, and transitive dependency count.

## Comments

- Package comments: one per package, in `doc.go` or the primary file.
- Exported functions: godoc-style comments starting with the function name.
- Don't comment obvious code. Comment _why_, not _what_.

## SQL (SQLite)

- Use parameterized queries. Never interpolate user input into SQL strings.
- Use migrations for schema changes.
- Keep queries in the data layer, not in business logic.

## API Design (OpenAPI / Fiber)

- Define the OpenAPI spec first, then generate server stubs.
- Use strict types mode — all request/response types are generated.
- Handlers should be thin: validate input, call business logic, return response.
- Use proper HTTP status codes and structured error responses.
