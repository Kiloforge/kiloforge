# Implementation Plan: Restructure Packages into Clean Architecture Layout

**Track ID:** refactor-clean-arch_20260307140000Z

## Phase 1: Create Core Layer Skeleton

### Task 1.1: Create core directories with doc.go files
- [x] ALREADY DONE — `internal/core/{domain,port,service}` created by refactor-domain-ports track

### Verification 1
- [x] `internal/core/{domain,port,service}` directories exist
- [x] Build succeeds

## Phase 2: Move Adapter Packages

### Task 2.1: Move config package
- [x] `git mv internal/config internal/adapter/config`
- [x] Update all imports

### Task 2.2: Move compose package
- [x] `git mv internal/compose internal/adapter/compose`
- [x] Update all imports

### Task 2.3: Move gitea package
- [x] `git mv internal/gitea internal/adapter/gitea`
- [x] Update all imports

### Task 2.4: Move auth package
- [x] `git mv internal/auth internal/adapter/auth`
- [x] Update all imports

### Task 2.5: Move agent package
- [x] `git mv internal/agent internal/adapter/agent`
- [x] Update all imports

### Task 2.6: Move pool package
- [x] `git mv internal/pool internal/adapter/pool`
- [x] Update all imports

### Task 2.7: Move cli package
- [x] `git mv internal/cli internal/adapter/cli`
- [x] Update all imports (including `cmd/crelay/main.go`)

### Task 2.8: Move relay package to adapter/rest
- [x] `git mv internal/relay internal/adapter/rest`
- [x] Update all imports and rename package to `rest`

### Additional moves (not in original plan)
- [x] `git mv internal/lock internal/adapter/lock`
- [x] `git mv internal/dashboard internal/adapter/dashboard`

### Verification 2
- [x] All adapter packages under `internal/adapter/`
- [x] No packages left directly under `internal/` (only `adapter/` and `core/`)
- [x] All imports updated (52 references across 24 files)
- [x] All tests pass with `-race`
- [x] Build succeeds

## Phase 3: Update Project Structure Documentation

### Task 3.1: Update style guide project structure section
- [x] Updated package layout in `.agent/conductor/code_styleguides/go.md`
- [x] Added agent, auth, pool, config, lock, dashboard to adapter listing

### Verification 3
- [x] Style guide matches actual directory structure
- [x] All tests pass
- [x] `go vet ./...` clean

---

**Total: 10 tasks across 3 phases — ALL COMPLETE**
