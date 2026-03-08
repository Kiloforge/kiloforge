# Implementation Plan: Fix Init Ctrl+C, Build Failure Propagation, and VCS Stamping

## Phase 1: Fix Makefile Build Issues (2 tasks)

### Task 1.1: Add -buildvcs=false to backend build
- **File:** `Makefile`
- Add `-buildvcs=false` flag to `go build` command in `build-backend` target
- Prevents VCS stamping failure in worktree/bare-repo setups

### Task 1.2: Make build fail on frontend build failure
- **File:** `Makefile`
- Change `build` target so it stops if `build-frontend` fails
- Use `$(MAKE) build-frontend && $(MAKE) build-backend` or separate recipe lines with error checking
- `ensure-dist` should only be a dependency of `build-backend` when run standalone (not via `build`)
- Verify: `make build` with a broken frontend should fail, not produce a binary with stub HTML

## Phase 2: Fix Ctrl+C During Init (2 tasks)

### Task 2.1: Pass context to offerSkillsInstall and all interactive prompts
- **File:** `backend/internal/adapter/cli/init.go`
- Pass `ctx` to `offerSkillsInstall(ctx, cfg)`
- **File:** `backend/internal/adapter/cli/skills.go`
- Update `offerSkillsInstall` signature to accept `context.Context`
- Replace bare `fmt.Scanln()` with context-aware input: read in goroutine, select on ctx.Done()
- On cancellation: print newline and return cleanly

### Task 2.2: Audit all interactive prompts in init flow for context cancellation
- **Files:** `backend/internal/adapter/cli/init.go`, `skills.go`, any other files with `fmt.Scanln` or `bufio.Scanner`
- Ensure every blocking read respects context cancellation
- Verify Ctrl+C works at every stage of `crelay init`

## Phase 3: Verification (1 task)

### Task 3.1: Manual verification and test suite
- `make build` with working frontend → succeeds, produces binary with real dashboard
- `make build` with broken frontend (e.g., introduce TS error) → fails immediately
- `make build-backend` standalone → succeeds with ensure-dist stub (dev convenience)
- `crelay init` → Ctrl+C at each interactive prompt → clean exit
- Run existing test suite: `make test`
