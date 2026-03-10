# Build Conventions

## Frontend Embed Pattern

The React dashboard is embedded into the Go binary using `//go:embed dist/*` in `backend/internal/adapter/dashboard/embed.go`. The pattern works as follows:

- **`dist/` is gitignored** — `backend/internal/adapter/dashboard/dist/` is listed in `.gitignore` and must never be committed. Real assets are produced by `make build-frontend` (npm ci + npm run build).
- **`ensure-dist` Makefile target** — Because `//go:embed dist/*` fails if the directory is empty or missing, the Makefile provides an `ensure-dist` target that creates a placeholder `index.html` when no real assets exist. This runs automatically as a dependency of `build-backend`, `test`, `dev`, and all test targets.
- **No build tags, no `.gitkeep`** — The embed directive is unconditional. There is no `//go:build embed_frontend` guard and no `.gitkeep` file. The `ensure-dist` placeholder approach is simpler and eliminates conditional compilation.

## Build Entry Point

All builds go through the **Makefile**. Never run `go build` or `go test` directly from the repo root.

| Command | Purpose |
|---------|---------|
| `make build` | Full build (frontend + backend) |
| `make build-backend` | Backend only (auto-runs `ensure-dist`) |
| `make build-frontend` | Frontend only (npm ci + npm run build) |
| `make dev` | Run backend + frontend dev servers concurrently |
| `make test` | Unit tests with race detector |
| `make test-smoke` | Smoke tests (binary builds, routes, CLI commands) |
| `make test-integration` | Integration tests (`-tags=integration`) |
| `make test-all` | All tests including integration |
| `make gen-api` | Regenerate OpenAPI server/client code |
| `make verify-codegen` | Verify generated code is up to date |

### Output directory

Build output goes to **`.build/`** (gitignored). The legacy `bin/` directory is not used.

## VCS Stamping

Prefer VCS stamping in local builds. Go's built-in VCS stamping embeds commit hash and dirty state into the binary, which is valuable for debugging and version reporting.

- **Local builds (normal repo / worktree):** Full VCS metadata is active by default.
- **Bare repo root:** VCS detection is unavailable (`git status` returns exit 128). The Makefile automatically detects this and passes `-buildvcs=false`. Version info is still stamped via ldflags — only the `vcs.modified` dirty flag is lost, which is an acceptable tradeoff.
- **CI environments:** If git metadata is unavailable (e.g., shallow clones), set `GOFLAGS=-buildvcs=false` as an environment variable rather than hardcoding the flag.
