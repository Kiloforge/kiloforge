# Implementation Plan: Restructure Packages into Clean Architecture Layout

**Track ID:** refactor-clean-arch_20260307140000Z

## Phase 1: Create Core Layer Skeleton

### Task 1.1: Create core directories with doc.go files
- Create `internal/core/domain/doc.go`, `internal/core/port/doc.go`, `internal/core/service/doc.go`
- Each doc.go declares the package and documents its role
- Tests: `go build ./...` succeeds

### Verification 1
- [ ] `internal/core/{domain,port,service}` directories exist
- [ ] Build succeeds

## Phase 2: Move Adapter Packages

### Task 2.1: Move config package
- `git mv internal/config internal/adapter/config`
- Update all imports: `crelay/internal/config` → `crelay/internal/adapter/config`
- Tests: `go build ./...` and `go test ./...`

### Task 2.2: Move compose package
- `git mv internal/compose internal/adapter/compose`
- Update all imports
- Tests: build and test

### Task 2.3: Move gitea package
- `git mv internal/gitea internal/adapter/gitea`
- Update all imports
- Tests: build and test

### Task 2.4: Move auth package
- `git mv internal/auth internal/adapter/auth`
- Update all imports
- Tests: build and test

### Task 2.5: Move agent package
- `git mv internal/agent internal/adapter/agent`
- Update all imports
- Tests: build and test

### Task 2.6: Move pool package
- `git mv internal/pool internal/adapter/pool`
- Update all imports
- Tests: build and test

### Task 2.7: Move cli package
- `git mv internal/cli internal/adapter/cli`
- Update all imports (including `cmd/crelay/main.go`)
- Tests: build and test

### Task 2.8: Move relay package to adapter/rest
- `git mv internal/relay internal/adapter/rest`
- Update all imports: `crelay/internal/relay` → `crelay/internal/adapter/rest`
- Tests: build and test

### Verification 2
- [ ] All adapter packages under `internal/adapter/`
- [ ] No packages left directly under `internal/` except `project/`, `state/`, `orchestration/`
- [ ] All imports updated
- [ ] All tests pass
- [ ] Build succeeds

## Phase 3: Update Project Structure Documentation

### Task 3.1: Update style guide project structure section
- Update the package layout in `.agent/conductor/code_styleguides/go.md` to reflect actual paths
- Update README if it references package paths

### Verification 3
- [ ] Style guide matches actual directory structure
- [ ] All tests pass
- [ ] `go vet ./...` clean
