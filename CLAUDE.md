# Kiloforge — Agent Instructions

## Worktree Build Fix (MANDATORY)

This repo uses git worktrees. Go's VCS stamping fails in worktrees because `.git` is a file, not a directory. **Never use `-buildvcs=false`.**

Before running `go build`, `go test`, or any `go` command that touches VCS metadata, source the worktree env helper:

```bash
. .agent/kf/bin/kf-worktree-env
```

This exports `GIT_DIR` and `GIT_WORK_TREE` automatically. Use `KF_QUIET=1` to suppress output. Run `kf-worktree-env --help` for details.

`make build` and `make test` already handle this — prefer Makefile targets when available.

## Project Structure

- `backend/` — Go backend (Cobra CLI, REST server, Gitea adapter)
- `frontend/` — React dashboard (Vite, TanStack Query)
- `.agent/conductor/` — Track-based project management artifacts

## Guidelines

- See `.agent/conductor/code_styleguides/` for Go and build conventions
- See `.agent/conductor/product-guidelines.md` for design principles
