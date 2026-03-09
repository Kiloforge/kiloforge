# Kiloforge — Agent Instructions

## Worktree Build Fix (MANDATORY)

This repo uses git worktrees. Go's VCS stamping fails in worktrees because `.git` is a file, not a directory. **Never use `-buildvcs=false`.**

Before running `go build`, `go test`, or any `go` command that touches VCS metadata, export these env vars:

```bash
export GIT_DIR=$(git rev-parse --git-common-dir)
export GIT_WORK_TREE=$(git rev-parse --show-toplevel)
```

Or prefix inline:

```bash
GIT_DIR=$(git rev-parse --git-common-dir) GIT_WORK_TREE=$(git rev-parse --show-toplevel) go build ./...
GIT_DIR=$(git rev-parse --git-common-dir) GIT_WORK_TREE=$(git rev-parse --show-toplevel) go test ./...
```

`make build` and `make test` already do this — prefer Makefile targets when available.

## Project Structure

- `backend/` — Go backend (Cobra CLI, REST server, Gitea adapter)
- `frontend/` — React dashboard (Vite, TanStack Query)
- `.agent/conductor/` — Track-based project management artifacts

## Guidelines

- See `.agent/conductor/code_styleguides/` for Go and build conventions
- See `.agent/conductor/product-guidelines.md` for design principles
