# Implementation Plan: Update README Prerequisites to Docker + Claude Code

## Phase 1: Update README (3 tasks)

### Task 1.1: Restructure Prerequisites section
- **File:** `README.md`
- Split current Prerequisites into "Prerequisites" (runtime: Docker, Claude Code) and "Building from Source" (Go, Node.js)
- Keep runtime deps first and prominent
- Move Colima note under the Docker prerequisite

### Task 1.2: Verify no broken links or formatting
- Review the full README for consistency after the edit
- Ensure markdown renders correctly

### Task 1.3: Commit
- `git add README.md`
- Commit with descriptive message
