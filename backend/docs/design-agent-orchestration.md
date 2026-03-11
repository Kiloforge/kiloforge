# Design: Agent Orchestration Lifecycle

**Date:** 2026-03-07
**Status:** Draft

---

## Overview

This document describes the full lifecycle of how kiloforge orchestrates Claude Code agents for track implementation. The system coordinates developer agents through implementation and merge, mediated through the local Gitea instance and webhook orchestrator.

> **Note:** The reviewer agent role described in earlier versions of this document has been removed. The kf-reviewer skill was deleted upstream. Developer agents now implement and merge directly without a separate review cycle.

---

## 1. Trigger: Track Approval

### How a track becomes ready for implementation

A track moves through these states:
```
draft → pending → approved → in-progress → review → complete
```

**Approval mechanism (initial):**
- Human approves via `kf implement <track-id>` CLI command
- This is the trigger that starts the developer agent

**Future automation:**
- Tracks could be auto-approved based on dependency resolution
- A queue system could process approved tracks in priority order

---

## 2. Worktree Pool

### Branch and worktree management

Developer agents need isolated working directories. Rather than creating/destroying worktrees per task, kiloforge maintains a **pool of reusable worktrees**.

### Pool structure
```
/path/to/project/
  .git/                    # bare repo (or main worktree)
  worker-1/                # worktree: developer slot 1
  worker-2/                # worktree: developer slot 2
  worker-3/                # worktree: developer slot 3
  track-generator-1/       # worktree: track generator
```

### Pool lifecycle
```
1. ACQUIRE: Find an idle worktree from the pool
   - Check pool state: which worktrees are idle vs in-use
   - If none idle and pool < max size: create a new worktree
   - If pool at max: queue the request

2. PREPARE: Reset worktree to main and create implementation branch
   - git reset --hard main
   - git checkout -b <track-id>

3. USE: Developer agent works in this worktree
   - Agent has exclusive access during implementation

4. RETURN: After merge or failure, return worktree to pool
   - git checkout main
   - git reset --hard main
   - git branch -D <track-id>  (implementation branch)
   - Mark worktree as idle in pool state
```

### Pool state tracking
```json
{
  "worktrees": {
    "worker-1": {
      "path": "/path/to/project/worker-1",
      "branch": "worker-1",
      "status": "in-use",
      "track_id": "auth-feature_20260307...",
      "agent_id": "uuid-developer",
      "acquired_at": "2026-03-07T12:00:00Z"
    },
    "worker-2": {
      "path": "/path/to/project/worker-2",
      "branch": "worker-2",
      "status": "idle",
      "track_id": null,
      "agent_id": null,
      "acquired_at": null
    }
  },
  "max_size": 3,
  "default_size": 2
}
```

---

## 3. Developer Agent Lifecycle

### Phase 1: Initialization

```
1. User runs: kf implement <track-id>
2. kf validates track is approved/pending
3. kf acquires a worktree from the pool
4. kf creates implementation branch: git checkout -b <track-id>
5. kf spawns Claude Code developer agent:
   - Working directory: acquired worktree
   - Prompt: /kf-developer <track-id>
   - Session ID: generated UUID
   - Flags: --dangerously-skip-permissions (sandboxed)
   - Output: stream-json to log file
6. Agent state recorded:
   - agent_id, session_id, track_id, worktree, role=developer, status=running
```

### Phase 2: Implementation

The developer agent (Claude Code) autonomously:
1. Reads the track spec and plan
2. Implements the code following TDD workflow
3. Runs tests to verify
4. Commits changes to the implementation branch

### Phase 3: PR Creation

When the developer is ready to submit:
1. Developer pushes implementation branch to Gitea: `git push gitea <track-id>`
2. Developer creates a PR via Gitea API:
   - Title: track title
   - Body: summary of changes, link to track spec
   - Base: main
   - Head: <track-id>
3. Developer adds metadata to PR:
   - Label or comment with developer session ID
   - Label with track ID

### Phase 4: Developer Paused

After PR creation:
1. Developer agent signals it is done (exits or is halted)
2. Agent state updated: `status: waiting-review`
3. Session ID preserved for later resume
4. Worktree remains allocated (developer may need to revise)

---

## 4. Review Cycle

### Trigger: PR Opened Webhook

```
Gitea → POST /webhook → orchestrator
  X-Gitea-Event: pull_request
  action: opened
```

The orchestrator:
1. Resolves project from `repository.name`
2. Extracts PR number, branch name (track ID)
3. Records PR metadata: developer session ID from PR body/labels
4. Developer proceeds to merge (reviewer role has been removed)

---

## 5. Revision Cycle

### Developer Resume for Revisions

```
1. Orchestrator receives pull_request_review webhook (changes_requested)
2. Orchestrator looks up developer agent by track_id / PR number
3. Orchestrator resumes developer with saved session ID:
   - claude --resume <developer-session-id>
   - Prompt context: "PR #N has review comments. Address feedback and push updates."
4. Developer state updated: status=running, review_cycle=N+1
```

### Developer Addresses Feedback

1. Developer reads review comments from Gitea API
2. Makes code changes in the worktree
3. Commits and pushes to the same branch
4. Gitea fires `pull_request.synchronize` webhook

### Re-Review

```
1. Orchestrator receives pull_request.synchronize webhook
2. Developer agent signals it has pushed revisions (exits or is halted)
3. Developer state: status=waiting-review
4. Developer proceeds to merge
```

### Cycle Limit

```
review_cycle_count tracking per PR:

  Cycle 1: Developer implements → PR → Reviewer reviews
  Cycle 2: Developer revises → Push → Reviewer re-reviews
  Cycle 3: Developer revises → Push → Reviewer re-reviews
  Cycle 4: ESCALATE — mark PR for human intervention

Escalation:
  - PR labeled: "needs-human-review"
  - Comment on PR: "Review cycle limit reached (3). Human review required."
  - CLI notification: kf status shows escalated PRs
  - All agents for this PR are stopped
```

---

## 6. Merge and Cleanup

### Merge (on approval)

```
1. Orchestrator receives pull_request_review webhook (approved)
2. Orchestrator resumes developer agent
3. Developer merges PR via Gitea API:
   - POST /api/v1/repos/{owner}/{repo}/pulls/{number}/merge
   - Merge method: merge commit (or rebase, configurable)
4. Gitea fires pull_request.closed (merged) webhook
```

### Worktree Cleanup

```
1. Developer switches worktree back to pool branch:
   - git checkout <pool-branch-name>  (e.g., worker-1)
   - git reset --hard main
2. Developer deletes implementation branch:
   - git branch -D <track-id>
   - git push gitea --delete <track-id>  (remote branch cleanup)
3. Pool state updated: worktree status=idle, track_id=null
```

### Agent Cleanup

```
1. Developer posts final comment on merged PR:
   "Merge successful. Implementation branch cleaned up. Track complete."
2. Developer agent exits
3. Developer state updated: status=completed
4. Track state updated: status=complete
5. Reviewer agent already completed (exited after approval)
6. Claude Code processes for both agents are terminated
```

---

## 7. State Machine Summary

### Developer Agent States
```
                    ┌─────────────────────────────────┐
                    │                                 │
                    ▼                                 │
spawned → running → waiting-review → running (revise) ┘
                         │
                         ▼ (approved)
                    merging → cleanup → completed
```

### PR States (kiloforge tracking)
```
created → in-review → changes-requested → in-review → ... → approved → merged
                                                               │
                                                    (cycle > max)
                                                               │
                                                          escalated
```

---

## 8. Data Model

### Agent Record (extended)

```go
type AgentInfo struct {
    ID            string    `json:"id"`
    Role          string    `json:"role"`          // "developer"
    TrackID       string    `json:"track_id"`
    ProjectSlug   string    `json:"project_slug"`
    PRNumber      int       `json:"pr_number,omitempty"`
    SessionID     string    `json:"session_id"`
    PID           int       `json:"pid"`
    WorktreeDir   string    `json:"worktree_dir"`
    LogFile       string    `json:"log_file"`
    Status        string    `json:"status"`        // running, waiting-review, merging, completed, failed
    ReviewCycle   int       `json:"review_cycle"`  // 0-based, incremented on each revision round
    StartedAt     time.Time `json:"started_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}
```

### PR Tracking Record

```go
type PRTracking struct {
    PRNumber          int    `json:"pr_number"`
    TrackID           string `json:"track_id"`
    ProjectSlug       string `json:"project_slug"`
    DeveloperAgentID  string `json:"developer_agent_id"`
    DeveloperSession  string `json:"developer_session_id"`
    ReviewerAgentID   string `json:"reviewer_agent_id,omitempty"`
    ReviewerSession   string `json:"reviewer_session_id,omitempty"`
    ReviewCycleCount  int    `json:"review_cycle_count"`
    MaxReviewCycles   int    `json:"max_review_cycles"`
    Status            string `json:"status"`  // open, in-review, changes-requested, approved, merged, escalated
}
```

---

## 9. CLI Commands

### New commands for orchestration

```
kf implement <track-id>     # Approve track and start developer agent
kf implement --list         # Show tracks available for implementation

kf agents                   # List all agents
kf agents --project <slug>  # Filter by project

kf logs <agent-id>          # View agent log
kf attach <agent-id>        # Halt agent and get resume command

kf escalated                # Show PRs that hit review cycle limit
```

---

## 10. Sequence Diagram

```
Human          kf CLI           Relay          Gitea          Developer CC     Reviewer CC
  │                │              │              │                │                │
  │ implement T1   │              │              │                │                │
  │───────────────>│              │              │                │                │
  │                │ acquire wt   │              │                │                │
  │                │ spawn dev    │              │                │                │
  │                │─────────────────────────────────────────────>│                │
  │                │              │              │                │                │
  │                │              │              │    push branch │                │
  │                │              │              │<───────────────│                │
  │                │              │              │                │                │
  │                │              │              │    create PR   │                │
  │                │              │              │<───────────────│                │
  │                │              │              │                │                │
  │                │              │  PR opened   │                │ (paused)       │
  │                │              │<─────────────│                │                │
  │                │              │              │                │                │
  │                │              │ spawn reviewer                │                │
  │                │              │────────────────────────────────────────────────>│
  │                │              │              │                │                │
  │                │              │              │  post review   │                │
  │                │              │              │<───────────────────────────────│
  │                │              │              │                │                │
  │                │              │  review event│                │  (completed)   │
  │                │              │<─────────────│                │                │
  │                │              │              │                │                │
  │           [if changes requested]             │                │                │
  │                │              │ resume dev   │                │                │
  │                │              │─────────────────────────────>│                │
  │                │              │              │                │                │
  │                │              │              │  push updates  │                │
  │                │              │              │<───────────────│                │
  │                │              │              │                │ (paused)       │
  │                │              │  PR updated  │                │                │
  │                │              │<─────────────│                │                │
  │                │              │ spawn reviewer                │                │
  │                │              │───────────────────────────────────────────────>│
  │                │              │              │                │                │
  │           [if approved]      │              │                │                │
  │                │              │ resume dev   │                │                │
  │                │              │─────────────────────────────>│                │
  │                │              │              │                │                │
  │                │              │              │   merge PR     │                │
  │                │              │              │<───────────────│                │
  │                │              │              │                │                │
  │                │              │              │   cleanup wt   │                │
  │                │              │              │   final comment│                │
  │                │              │              │<───────────────│                │
  │                │              │              │                │ (completed)    │
  │                │              │  PR merged   │                │                │
  │                │              │<─────────────│                │                │
  │                │              │              │                │                │
  │                │ track complete│             │                │                │
  │<───────────────│              │              │                │                │
```

---

## 11. Open Questions

1. **Concurrent tracks** — How many tracks can be implemented simultaneously? Limited by worktree pool size and system resources.

4. **Session resume mechanics** — Claude Code `--resume` resumes a session. How do we inject new context ("you have review comments")? Via prompt flag? Via a file the agent reads?

5. **Agent crash recovery** — If a developer agent crashes mid-implementation, how do we recover? The worktree has partial work. Options: resume session, or start fresh.

6. **PR merge conflicts** — If main advances while a track is in review, the PR may have conflicts. Who resolves them — the developer agent on resume?

7. **Track dependencies** — If track B depends on track A, track B should not start until A is merged. The orchestrator needs dependency awareness.
