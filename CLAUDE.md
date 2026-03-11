# Kiloforge — Agent Instructions

## Worktree Build Fix (MANDATORY)

This repo uses git worktrees. Go's VCS stamping fails in worktrees because `.git` is a file, not a directory. **Do not manually pass `-buildvcs=false`** — the Makefile auto-detects VCS availability and falls back gracefully when needed (e.g., bare repo root).

Before running `go build`, `go test`, or any `go` command that touches VCS metadata, source the worktree env helper:

```bash
eval "$(.agent/kf/bin/kf-worktree-env.py)"
```

This exports `GIT_DIR` and `GIT_WORK_TREE` automatically. Use `KF_QUIET=1` to suppress output. Run `kf-worktree-env.py --help` for details.

`make build` and `make test` already handle this — prefer Makefile targets when available.

## Never Commit Build Artifacts

**`backend/internal/adapter/dashboard/dist/`** is in `.gitignore`. Never `git add` it. The frontend is built via `make build-frontend` and embedded into the Go binary at compile time. If `dist/` is missing, run the build — do not commit artifacts.

## Project Structure

- `backend/` — Go backend (Cobra CLI, REST server, Gitea adapter)
- `frontend/` — React dashboard (Vite, TanStack Query)
- `.agent/kf/` — Kiloforge project management artifacts

## Guidelines

- See `.agent/kf/code_styleguides/` for Go and build conventions
- See `.agent/kf/product-guidelines.md` for design principles
