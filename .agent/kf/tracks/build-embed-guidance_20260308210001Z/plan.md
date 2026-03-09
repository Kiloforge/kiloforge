# Implementation Plan: Build and Embed Pattern Guidance

## Phase 1: Create Guidance Document (3 tasks)

### Task 1.1: Create build guidance document
- [x] **File:** `.agent/conductor/code_styleguides/build.md`
- Document the three key patterns:
  1. Frontend embed pattern (ensure-dist, //go:embed dist/*, no build tags, no .gitkeep)
  2. Build entry point (Makefile-only, .build/ output, ensure-dist dependency chain)
  3. VCS support (no -buildvcs=false, env var for CI only)
- Include rationale for each decision (link to lmf-wt as reference pattern)

### Task 1.2: Update product guidelines
- [x] **File:** `.agent/conductor/product-guidelines.md`
- Add a "Build Conventions" section or link to the new guidance document
- Keep it brief — reference the full document for details

### Task 1.3: Verify guidance accuracy against current Makefile and embed.go
- [x] Cross-check all statements in the guidance against actual code
- Ensure no stale references

## Phase 2: Commit and Verify (2 tasks)

### Task 2.1: Commit guidance artifacts
- [x] `git add` the new/modified files
- Commit with descriptive message

### Task 2.2: Verify no code changes introduced
- [x] `git diff --stat` should show only `.md` file changes
- `make test-smoke` passes (no code impact)
