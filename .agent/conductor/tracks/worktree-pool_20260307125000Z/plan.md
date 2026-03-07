# Implementation Plan: Worktree Pool Management

**Track ID:** worktree-pool_20260307125000Z

## Phase 1: Core Pool Types and State

### Task 1.1: Define pool types and state persistence [x]
- Create `internal/pool/pool.go`
- Define `Worktree` struct: Name, Path, Status (idle/in-use), TrackID, AgentID, AcquiredAt
- Define `Pool` struct: Worktrees map, MaxSize, project root path
- `Load(dataDir) (*Pool, error)` and `Save(dataDir) error` for JSON persistence
- Tests: load/save round-trip

### Task 1.2: Implement worktree creation [x]
- `createWorktree(name string) error` — runs `git worktree add <path> -b <branch>`
- Worktree path: `{projectRoot}/worker-N/`
- Branch name: `worker-N` (stable pool branch)
- Tests: verify git commands (may need integration test)

### Task 1.3: Implement Acquire method [x]
- `Acquire() (*Worktree, error)` — find first idle worktree
- If none idle and len < MaxSize: create new worktree, add to pool
- If all in-use and at max: return error "pool exhausted"
- Mark acquired worktree as in-use
- Save pool state
- Tests: acquire idle, acquire with creation, acquire exhausted

### Verification 1
- [x] Pool state persists correctly
- [x] Worktrees created via git
- [x] Acquire logic handles all cases
- [x] Tests pass

## Phase 2: Prepare and Return

### Task 2.1: Implement Prepare method [x]
- `Prepare(worktree *Worktree, trackID string) error`
- In the worktree: `git checkout <pool-branch> && git reset --hard main && git checkout -b <trackID>`
- Update worktree state: TrackID set, status remains in-use
- Save pool state
- Tests: verify branch creation

### Task 2.2: Implement Return method [x]
- `Return(worktree *Worktree) error`
- In the worktree: `git checkout <pool-branch> && git reset --hard main`
- Delete implementation branch: `git branch -D <trackID>`
- Update state: status=idle, TrackID=nil, AgentID=nil
- Save pool state
- Tests: verify cleanup

### Task 2.3: Implement Status method [x]
- `Status() []WorktreeStatus` — returns summary of all worktrees
- Used by CLI and health checks

### Verification 2
- [x] Prepare creates implementation branch
- [x] Return cleans up and resets
- [x] Status reports correct state
- [x] Tests pass

## Phase 3: CLI and Integration

### Task 3.1: Implement `crelay pool` command [x]
- Create `internal/cli/pool.go`
- Table output: `NAME  STATUS  TRACK  AGENT  ACQUIRED`
- Register in root.go
- If pool not initialized, show helpful message

### Task 3.2: Pool initialization [x]
- Pool created on first `Acquire()` if pool.json doesn't exist
- Or explicitly via `crelay pool --init` (optional)
- Default max size from config or flag

### Task 3.3: Update docs [x]
- Document pool concept in README
- Add `crelay pool` to commands docs

### Verification 3
- [x] `crelay pool` displays worktree status
- [x] Pool auto-initializes on first use
- [x] Build and tests pass
