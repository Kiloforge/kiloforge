# Implementation Plan: 'crelay implement' Command — Spawn Developer Agent

**Track ID:** implement-command_20260307125001Z

## Phase 1: Track Resolution and Agent Spawning

### Task 1.1: Implement track discovery
- Read conductor tracks from project's `.agent/conductor/tracks.md` and track directories
- Filter for pending/approved status
- `--list` flag shows available tracks in a table
- Tests: parse tracks.md, filter by status

### Task 1.2: Extend agent spawner for worktree context
- Update `SpawnDeveloper()` in `internal/agent/spawner.go`
- Accept worktree path, track ID, project context
- Set working directory to worktree
- Generate and use session ID
- Stream output to project-specific log dir
- Tests: verify spawner configuration

### Task 1.3: Implement PR tracking data model
- Create `internal/orchestration/tracking.go`
- `PRTracking` struct: PRNumber, TrackID, ProjectSlug, DeveloperAgentID, DeveloperSession, ReviewerAgentID, ReviewerSession, ReviewCycleCount, MaxReviewCycles, Status
- Load/Save to `~/.crelay/projects/<slug>/pr-tracking.json`
- Tests: load/save round-trip

### Verification 1
- [ ] Tracks discoverable and filterable
- [ ] Spawner works with worktree context
- [ ] PR tracking model persists
- [ ] Tests pass

## Phase 2: Implement Command

### Task 2.1: Create `crelay implement` command
- Create `internal/cli/implement.go`
- Accepts `<track-id>` argument
- `--list` flag shows available tracks
- `--project` flag specifies project (or auto-detect from cwd)
- Flow: validate track → resolve project → acquire worktree → prepare → spawn → record state
- Print agent ID, session ID, worktree path, log file
- Register in root.go

### Task 2.2: Wire PR webhook to create tracking record
- When relay receives `pull_request.opened` for a branch matching a track ID:
  - Create PRTracking record linking PR to developer agent
  - Extract session ID from PR body/labels
  - Update developer agent status: `waiting-review`
- Tests: verify tracking record creation from webhook

### Task 2.3: Add track ID to PR metadata
- The conductor-developer skill creates PRs — ensure the PR body includes:
  - Track ID
  - Developer session ID
  - These are parseable by the relay for tracking
- If conductor-developer doesn't add this, the relay infers from branch name (branch = track ID)

### Verification 2
- [ ] `crelay implement <track-id>` spawns developer in worktree
- [ ] `crelay implement --list` shows available tracks
- [ ] PR creation triggers tracking record
- [ ] Developer paused after PR creation
- [ ] Tests pass

## Phase 3: Integration and Docs

### Task 3.1: End-to-end integration
- Test full flow: implement → developer works → PR created → developer paused
- Verify pool state, agent state, PR tracking state all consistent
- Handle edge cases: track already in-progress, pool exhausted

### Task 3.2: Re-enable agent CLI commands
- Re-enable `agents`, `logs`, `attach`, `stop` commands in root.go
- Update to load agent state from project-specific paths
- `crelay agents` shows developer/reviewer agents with track context

### Task 3.3: Update README and docs
- Document `crelay implement` command
- Document agent lifecycle overview
- Add to getting-started guide

### Verification 3
- [ ] Full flow works end-to-end
- [ ] Agent commands restored and working
- [ ] Docs updated
- [ ] Build and tests pass
