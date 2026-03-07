# Implementation Plan: 'crelay implement' Command — Spawn Developer Agent

**Track ID:** implement-command_20260307125001Z

## Phase 1: Track Resolution and Agent Spawning

### Task 1.1: Implement track discovery
- [x] Read conductor tracks from project's `.agent/conductor/tracks.md` and track directories
- [x] Filter for pending/approved status
- [x] `--list` flag shows available tracks in a table
- [x] Tests: parse tracks.md, filter by status

### Task 1.2: Extend agent spawner for worktree context
- [x] Update `SpawnDeveloper()` in `internal/agent/spawner.go`
- [x] Accept worktree path, track ID, project context via SpawnDeveloperOpts
- [x] Set working directory to worktree
- [x] Generate and use session ID
- [x] Stream output to project-specific log dir

### Task 1.3: Implement PR tracking data model
- [x] Create `internal/orchestration/tracking.go`
- [x] `PRTracking` struct with all required fields
- [x] Load/Save to `~/.crelay/projects/<slug>/pr-tracking.json`
- [x] Tests: load/save round-trip

### Verification 1
- [x] Tracks discoverable and filterable
- [x] Spawner works with worktree context
- [x] PR tracking model persists
- [x] Tests pass

## Phase 2: Implement Command

### Task 2.1: Create `crelay implement` command
- [x] Create `internal/cli/implement.go`
- [x] Accepts `<track-id>` argument
- [x] `--list` flag shows available tracks
- [x] `--project` flag specifies project (or auto-detect from cwd)
- [x] Flow: validate track -> resolve project -> acquire worktree -> prepare -> spawn -> record state
- [x] Print agent ID, session ID, worktree path, log file
- [x] Register in root.go

### Task 2.2: Wire PR webhook to create tracking record
- [x] When relay receives `pull_request.opened` for a branch matching a track ID:
  - [x] Create PRTracking record linking PR to developer agent
  - [x] Extract developer info from state store by matching track ID
  - [x] Update developer agent status: `waiting-review`
- [x] Tests: verify tracking record creation from webhook

### Task 2.3: Add track ID to PR metadata
- [x] Relay infers track ID from branch name (head ref)
- [x] Branch name = track ID convention used by conductor-developer

### Verification 2
- [x] `crelay implement <track-id>` spawns developer in worktree
- [x] `crelay implement --list` shows available tracks
- [x] PR creation triggers tracking record
- [x] Developer status updated to waiting-review
- [x] Tests pass

## Phase 3: Integration and Docs

### Task 3.1: End-to-end integration
- [x] Handle edge cases: track already complete, track in-progress, pool exhausted
- [x] Pool state, agent state, PR tracking state all consistent

### Task 3.2: Re-enable agent CLI commands
- [x] Re-enable `agents`, `logs`, `attach`, `stop` commands in root.go
- [x] Commands load agent state from global DataDir

### Task 3.3: Update README and docs
- [x] Document `crelay implement` command
- [x] Document agents, logs, stop, attach commands
- [x] Updated data directory structure

### Verification 3
- [x] Agent commands restored and working
- [x] Docs updated
- [x] Build and tests pass
