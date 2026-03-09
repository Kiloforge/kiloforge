# Implementation Plan: Research: Graceful Agent Shutdown, Session Persistence, and Recovery

**Track ID:** research-graceful-shutdown-recovery_20260307150001Z

## Phase 1: CC Session Resume Behavior (3 tasks)

### Task 1.1: Test CC session resume basics
- [x] Spawn agent, capture session ID, SIGINT, then `claude --resume <id>`
- [x] Document: Does it pick up where it left off? Re-read context?
- [x] Document: Output format on successful resume
- [x] Document: Exit codes and error messages on failed resume

### Task 1.2: Test resume edge cases
- [x] Resume after modified working directory (changed files)
- [x] Resume after git branch change
- [x] Resume after long delay (if possible to test)
- [x] Resume with deleted worktree
- [x] Document each failure mode and error output

### Task 1.3: Test signal handling
- [x] Send SIGINT to running agent — does it save state gracefully?
- [x] Send SIGTERM — different behavior?
- [x] Test shutdown of multiple agents simultaneously
- [x] Document timing: how long does graceful shutdown take?

## Phase 2: Architecture Design (3 tasks)

### Task 2.1: Design graceful shutdown sequence
- [x] Define signal cascade: relay receives SIGINT → agents receive SIGINT → wait → force kill
- [x] Define timeout per agent and total shutdown timeout
- [x] Design state snapshot: persist "was-running" flag distinct from "running"
- [x] Consider: `kf down` vs ctrl-c vs process kill scenarios

### Task 2.2: Design auto-resume on startup
- [x] Define startup sequence: relay starts → reads state → identifies resumable agents → resumes
- [x] Design resume prioritization (developers before reviewers? by track dependency?)
- [x] Define error handling: what to do when resume fails per agent
- [x] Design user-facing output: "Restoring agents... 3/4 restored, 1 failed: session expired"

### Task 2.3: Design state persistence format
- [x] Extend state.json or create separate shutdown-state.json?
- [x] What additional fields needed? (shutdown_time, was_healthy, last_known_phase)
- [x] Worktree state preservation requirements
- [x] PR tracking state across restarts

## Phase 3: Documentation and Proposal (2 tasks)

### Task 3.1: Write research findings document
- [x] Compile all findings into `research.md` in track directory
- [x] Include test results, error catalogs, architecture proposals
- [x] Document limitations and risks

### Task 3.2: Propose implementation track(s)
- [x] Based on findings, draft scope for implementation tracks
- [x] Split if needed: shutdown track + recovery track
- [x] Identify dependencies on CC features or limitations

---

**Total: 8 tasks across 3 phases — ALL COMPLETE**
