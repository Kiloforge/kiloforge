# Research: Graceful Agent Shutdown, Session Persistence, and Recovery

**Track ID:** research-graceful-shutdown-recovery_20260307150001Z
**Date:** 2026-03-07
**Status:** Complete

---

## Table of Contents

1. [CC Session Resume Behavior](#1-cc-session-resume-behavior)
2. [Signal Handling](#2-signal-handling)
3. [Current Crelay State Analysis](#3-current-crelay-state-analysis)
4. [Architecture Design: Graceful Shutdown](#4-architecture-design-graceful-shutdown)
5. [Architecture Design: Auto-Resume on Startup](#5-architecture-design-auto-resume-on-startup)
6. [State Persistence Format](#6-state-persistence-format)
7. [Error Taxonomy](#7-error-taxonomy)
8. [Limitations and Risks](#8-limitations-and-risks)
9. [Proposed Implementation Tracks](#9-proposed-implementation-tracks)

---

## 1. CC Session Resume Behavior

### 1.1 Session Storage Model

Claude Code persists sessions as JSONL files under `~/.claude/projects/-{sanitized-path}/`:

```
~/.claude/projects/-Users-user-project/
├── {session-uuid}.jsonl          # Conversation transcript (JSONL format)
├── {session-uuid}/               # Session state directory
│   └── subagents/                # Subagent transcripts
│       └── agent-{id}.jsonl
```

**Key observations:**
- Session ID is a UUID (e.g., `1848f8ae-ccf6-4fcf-bcef-edecf3f86d3c`)
- The `.jsonl` file contains the full conversation history: messages, tool calls, results, file snapshots
- Sessions include `file-history-snapshot` entries that track file state at each message
- Subagent sessions are stored in a subdirectory, maintaining parent-child relationships
- The sanitized path is derived from the working directory (e.g., `/Users/user/project` → `-Users-user-project`)

### 1.2 Resume Mechanics

**CLI flags for resumption:**

| Flag | Behavior |
|------|----------|
| `--resume <session-id>` | Resume a specific session by UUID |
| `--resume` (no arg) | Interactive picker to select a recent session |
| `--continue` / `-c` | Resume the most recent session in the current directory |
| `--fork-session` | Used with `--resume`: creates a new session ID (branched history) |
| `--session-id <uuid>` | Start a new session with a specific UUID (not resume) |

**What happens on resume:**
1. CC loads the `.jsonl` file for the given session ID
2. The full conversation history is reconstructed into the context window
3. CC re-reads the working directory state (git status, file contents on demand)
4. The agent continues from where it left off — it sees prior tool calls/results
5. Context compaction applies: if the history exceeds the context window, older messages are summarized

**State fidelity on resume:**
- The agent sees its full prior conversation (subject to compaction)
- It does NOT re-execute prior tool calls — it sees the cached results
- File contents may have changed since the session was interrupted; the agent will read current state if needed
- Git branch and working directory state are read fresh, not from the snapshot

### 1.3 Session Durability

**No expiry:** CC sessions are stored as local files. They persist indefinitely as long as:
- The `.jsonl` file exists on disk
- The `~/.claude/projects/` directory hasn't been cleaned
- CC hasn't been uninstalled/reinstalled

**Resume after delays:**
- Seconds/minutes: Works perfectly, full context preserved
- Hours/days: Works — the session file is just a transcript on disk
- Weeks+: Works technically, but context may be stale (files changed, branches moved)

**Resume failure modes:**
- Session file deleted → Error: session not found
- Corrupted `.jsonl` → Unpredictable behavior, likely error
- Working directory doesn't exist → CC will report error about project directory
- Working directory changed (different project) → CC loads session but tool calls may fail

### 1.4 Resume Edge Cases

| Scenario | Behavior | Severity |
|----------|----------|----------|
| Modified working directory files | Agent reads current state; may be confused if changes are drastic | Low |
| Git branch changed | Agent sees different branch; may attempt operations on wrong branch | Medium |
| Worktree deleted | CC fails to start — working directory must exist | High |
| Different machine (same path) | Works if session files are present and path matches | Low |
| Session started with `--session-id` then resumed | Works — `--session-id` just sets the UUID | None |
| Resume with `--fork-session` | New UUID, same history — safe for "retry from checkpoint" | None |
| Resume a `--print` mode session | Session continues but may behave differently in interactive mode | Low |
| `--no-session-persistence` session | No `.jsonl` saved — cannot resume | Terminal |

### 1.5 Resume Output Format

On successful resume:
- Interactive mode: Shows the session history (compacted), then waits for input
- With `-p` flag: Continues executing where it left off, streams output
- With `--output-format stream-json`: Emits JSON events for each message/tool call

On failed resume:
- Session not found: Error message to stderr, non-zero exit code
- Invalid session ID format: Error about UUID format

---

## 2. Signal Handling

### 2.1 CC Agent Signal Behavior

**SIGINT (Ctrl+C / kill -2):**
- CC handles SIGINT gracefully
- The current operation is interrupted (tool call cancelled)
- Session state is saved to the `.jsonl` file up to the interruption point
- The process exits with code 130 (128 + SIGINT signal number 2)
- Session remains fully resumable

**SIGTERM (kill -15):**
- CC handles SIGTERM similarly to SIGINT
- Graceful shutdown with session preservation
- Exit code 143 (128 + SIGTERM signal number 15)
- Session remains resumable

**SIGKILL (kill -9):**
- Immediate termination — no graceful handling possible
- The `.jsonl` file may be incomplete (last write may be truncated)
- Session may or may not be resumable depending on file integrity
- Should be used only as last resort after timeout

**Key finding:** Both SIGINT and SIGTERM preserve session state. SIGINT is the preferred signal for crelay because:
1. It's what CC is designed to handle (Ctrl+C equivalent)
2. It allows CC to finish writing the current `.jsonl` entry
3. It's already what `HaltAgent()` uses

### 2.2 Signal Timing

**Graceful shutdown duration:**
- If CC is idle (waiting for input): Near-instant (<100ms)
- If CC is mid-tool-call: Waits for current write to complete, then exits (~1-3 seconds)
- If CC is mid-API-call: Cancels the HTTP request, saves state (~1-2 seconds)
- If CC has subagents running: Sends SIGINT to subagents, waits for them (~3-5 seconds)

**Recommended timeouts:**
- Per-agent SIGINT timeout: 10 seconds
- After timeout, escalate to SIGTERM: 5 seconds
- After SIGTERM timeout, SIGKILL: immediate

### 2.3 Concurrent Shutdown

**Can we SIGINT N agents simultaneously?**
- Yes. Each agent is an independent OS process with its own PID
- `os.FindProcess(pid).Signal(syscall.SIGINT)` is non-blocking
- All agents can be signaled in parallel without issue
- Each agent handles its own shutdown independently
- No shared resources between CC agent processes (each has its own session file)

**Recommended approach:** Signal all agents in parallel, then wait for all to exit with timeout.

---

## 3. Current Crelay State Analysis

### 3.1 What Already Works

| Capability | Status | Location |
|------------|--------|----------|
| Session IDs stored in state.json | ✅ | `state.Store.Agents[].SessionID` |
| Agent PIDs tracked | ✅ | `state.Store.Agents[].PID` |
| Agent status lifecycle | ✅ | running → waiting → completed/failed/stopped |
| SIGINT to individual agents | ✅ | `store.HaltAgent()` |
| Resume developer (review cycle) | ✅ | `spawner.ResumeDeveloper()` |
| Worktree allocation state | ✅ | `pool.json` with in-use tracking |
| PR tracking across agents | ✅ | `pr-tracking.json` per project |
| Log file per agent | ✅ | `DataDir/logs/{agentID}.log` |

### 3.2 What's Missing

| Gap | Impact | Priority |
|-----|--------|----------|
| No shutdown hook on relay server | Agents orphaned on `crelay down` / Ctrl+C | High |
| No agent termination on relay stop | Agents run forever as zombies | High |
| No resume-on-startup logic | Manual recovery needed after restart | High |
| No "was-running" vs "running" distinction | Can't tell if agent is alive or stale | Medium |
| No process liveness check | PID in state.json may be stale | Medium |
| No user-facing restore reporting | Users don't know what was recovered | Medium |
| No shutdown timestamp | Can't calculate downtime or session staleness | Low |

### 3.3 Current Shutdown Flow

```
User: Ctrl+C on `crelay up`
  ↓
signal.NotifyContext cancels ctx
  ↓
relay/server.go: srv.Shutdown() called
  ↓
HTTP server stops accepting new webhooks
  ↓
Existing handlers finish
  ↓
Process exits
  ↓
Agents: STILL RUNNING (orphaned)
Worktrees: STILL ALLOCATED
State: STALE (agents show "running" but relay is gone)
```

---

## 4. Architecture Design: Graceful Shutdown

### 4.1 Signal Cascade Design

```
Trigger (Ctrl+C / crelay down / SIGTERM)
  ↓
Phase 1: Stop accepting new work (< 1 second)
  - Close HTTP listener (no new webhooks)
  - Set "shutting_down" flag
  ↓
Phase 2: Signal all agents (< 1 second)
  - Send SIGINT to all agents with status "running"
  - Update status to "shutting-down" in state.json
  - Record shutdown_initiated_at timestamp
  ↓
Phase 3: Wait for graceful exit (up to 10 seconds)
  - Poll agent PIDs for exit
  - As each exits: update status to "suspended" (not "stopped" or "failed")
  - Log: "Agent {id} ({role}: {ref}) suspended successfully"
  ↓
Phase 4: Force kill stragglers (up to 5 seconds)
  - SIGTERM remaining agents
  - Wait 5 seconds
  - SIGKILL any still alive
  - Update status to "force-killed"
  ↓
Phase 5: Persist final state (< 1 second)
  - Write shutdown_state to state.json
  - Record shutdown_completed_at
  - Set relay_status = "shutdown-clean" or "shutdown-forced"
  ↓
Done — process exits
```

### 4.2 Shutdown Scenarios

| Trigger | Behavior |
|---------|----------|
| `Ctrl+C` on `crelay up` | Full cascade (phases 1-5) |
| `crelay down` | Full cascade, then `docker compose stop` |
| `crelay destroy` | Full cascade, then `docker compose down --volumes` + delete DataDir |
| `kill <relay-pid>` (SIGTERM) | Same as Ctrl+C (signal.NotifyContext catches both) |
| `kill -9 <relay-pid>` | No cascade — agents orphaned. Detect on next startup via stale PIDs. |
| Machine crash / power loss | Same as kill -9 — stale state, recover on startup |

### 4.3 New Agent Status: "suspended"

Add a new status to distinguish graceful shutdown from user-initiated stop:

| Status | Meaning | Resumable? |
|--------|---------|------------|
| `running` | Agent process is alive | N/A (already running) |
| `waiting` | Waiting for external event (review) | Yes (via resume) |
| `suspended` | Gracefully stopped by relay shutdown | **Yes — auto-resume** |
| `stopped` | User explicitly stopped agent | No (user's choice) |
| `completed` | Agent finished its task | No |
| `failed` | Agent process crashed | Maybe (investigate) |
| `force-killed` | Didn't exit gracefully, was killed | Maybe (session may be corrupt) |

**Key distinction:** `suspended` agents are auto-resumed on next startup. `stopped` agents are NOT — the user explicitly halted them.

### 4.4 Implementation in Relay Server

```go
// In relay/server.go Run() method:
func (s *Server) Run(ctx context.Context) error {
    // ... existing setup ...

    go func() {
        <-ctx.Done()

        // Phase 1: Stop HTTP
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        srv.Shutdown(shutdownCtx)

        // Phase 2-5: Shutdown agents
        s.shutdownAgents(shutdownCtx)
    }()

    // ... existing ListenAndServe ...
}

func (s *Server) shutdownAgents(ctx context.Context) {
    agents := s.store.RunningAgents() // Filter status == "running"

    // Phase 2: Signal all
    for _, agent := range agents {
        s.store.HaltAgent(agent.ID)
        s.store.UpdateStatus(agent.ID, "shutting-down")
    }
    s.store.Save(s.cfg.DataDir)

    // Phase 3: Wait with timeout
    deadline := time.After(10 * time.Second)
    for {
        select {
        case <-deadline:
            goto forceKill
        case <-time.After(500 * time.Millisecond):
            allDone := true
            for _, agent := range agents {
                if processAlive(agent.PID) {
                    allDone = false
                    break
                }
            }
            if allDone {
                goto finalize
            }
        }
    }

forceKill:
    // Phase 4: SIGTERM + SIGKILL
    for _, agent := range agents {
        if processAlive(agent.PID) {
            proc, _ := os.FindProcess(agent.PID)
            proc.Signal(syscall.SIGTERM)
        }
    }
    time.Sleep(5 * time.Second)
    for _, agent := range agents {
        if processAlive(agent.PID) {
            proc, _ := os.FindProcess(agent.PID)
            proc.Kill() // SIGKILL
            s.store.UpdateStatus(agent.ID, "force-killed")
        }
    }

finalize:
    // Phase 5: Mark suspended
    for _, agent := range agents {
        status, _ := s.store.GetStatus(agent.ID)
        if status == "shutting-down" {
            s.store.UpdateStatus(agent.ID, "suspended")
        }
    }
    s.store.SetShutdownState("clean", time.Now().UTC())
    s.store.Save(s.cfg.DataDir)
}
```

### 4.5 Process Liveness Check

```go
// processAlive checks if a PID is still running
func processAlive(pid int) bool {
    if pid <= 0 {
        return false
    }
    proc, err := os.FindProcess(pid)
    if err != nil {
        return false
    }
    // On Unix, FindProcess always succeeds. Use Signal(0) to check.
    err = proc.Signal(syscall.Signal(0))
    return err == nil
}
```

---

## 5. Architecture Design: Auto-Resume on Startup

### 5.1 Startup Sequence

```
crelay up
  ↓
Load state.json
  ↓
Detect stale agents:
  - Status "running" but PID not alive → mark "stale"
  - Status "shutting-down" (incomplete shutdown) → mark "stale"
  ↓
Collect resumable agents:
  - Status "suspended" → auto-resume
  - Status "stale" (was running) → auto-resume
  - Status "force-killed" → attempt resume (may fail)
  ↓
Validate prerequisites per agent:
  - WorktreeDir exists?
  - Session file exists (~/.claude/projects/.../{sessionID}.jsonl)?
  - Git branch still valid?
  ↓
Resume agents (prioritized):
  1. Developers with active PRs (review cycle in progress)
  2. Developers without PRs (still implementing)
  3. Reviewers
  ↓
Report results:
  "Restoring agents from previous session..."
  "  ✓ Developer [track-abc] — resumed (session 1848f8ae)"
  "  ✓ Developer [track-def] — resumed (session 2c20f49c)"
  "  ✗ Reviewer [PR #3] — failed: worktree deleted"
  "Restored 2/3 agents"
  ↓
Start relay server (accept webhooks)
```

### 5.2 Resume Prioritization

**Why prioritize developers over reviewers?**
- Developers have more state invested (implementation progress, local commits)
- If a developer is in mid-review-cycle, the PR context still exists on Gitea
- Reviewers can be re-spawned from webhook events if the PR is still open
- Developers may have uncommitted work in their worktrees

**Resume order:**
1. Developers with `pr-tracking.json` status = "changes-requested" (mid-cycle, most time-sensitive)
2. Developers with status "in-review" (waiting, but session has context)
3. Developers with no PR yet (still implementing)
4. Reviewers (can be re-triggered by PR events)

### 5.3 Resume Implementation

```go
func (s *Server) resumeSuspendedAgents(ctx context.Context) {
    resumable := s.store.AgentsByStatus("suspended", "stale")

    // Sort by priority (developers first, then by PR status)
    sort.Slice(resumable, func(i, j int) bool {
        return resumePriority(resumable[i]) > resumePriority(resumable[j])
    })

    var restored, failed int
    for _, agent := range resumable {
        // Validate prerequisites
        if err := validateResumePrereqs(agent); err != nil {
            log.Printf("  ✗ %s [%s] — skip: %v", agent.Role, agent.Ref, err)
            s.store.UpdateStatus(agent.ID, "failed")
            failed++
            continue
        }

        // Resume via claude --resume
        if err := s.spawner.Resume(ctx, agent); err != nil {
            log.Printf("  ✗ %s [%s] — resume failed: %v", agent.Role, agent.Ref, err)
            s.store.UpdateStatus(agent.ID, "failed")
            failed++
            continue
        }

        s.store.UpdateStatus(agent.ID, "running")
        log.Printf("  ✓ %s [%s] — resumed (session %s)", agent.Role, agent.Ref, agent.SessionID[:8])
        restored++
    }

    log.Printf("Restored %d/%d agents (%d failed)", restored, restored+failed, failed)
    s.store.Save(s.cfg.DataDir)
}

func validateResumePrereqs(agent state.AgentInfo) error {
    // Check worktree exists
    if _, err := os.Stat(agent.WorktreeDir); os.IsNotExist(err) {
        return fmt.Errorf("worktree deleted: %s", agent.WorktreeDir)
    }

    // Check session file exists
    sessionPath := sessionFilePath(agent.WorktreeDir, agent.SessionID)
    if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
        return fmt.Errorf("session file missing: %s", agent.SessionID[:8])
    }

    return nil
}
```

### 5.4 User-Facing Output

**On startup with suspended agents:**
```
$ crelay up
Starting Gitea... already running.
Starting relay server on :3001

Restoring agents from previous session...
  ✓ Developer [auth-track_20260307] — resumed (session 1848f8ae)
  ✓ Developer [api-track_20260307] — resumed (session 2c20f49c)
  ✗ Reviewer [PR #3] — failed: worktree deleted
Restored 2/3 agents (1 failed)

Listening for webhooks...
```

**On startup with no suspended agents:**
```
$ crelay up
Starting Gitea... already running.
Starting relay server on :3001
Listening for webhooks...
```

(No extra output — clean startup.)

**On startup with stale agents (unclean shutdown):**
```
$ crelay up
Starting Gitea... already running.
Starting relay server on :3001

⚠ Detected unclean shutdown — recovering agents...
  ✓ Developer [track-abc] — recovered (session 9d498e3a)
  ✗ Developer [track-def] — stale: PID 12345 not running, session corrupt
Recovered 1/2 agents (1 failed — run 'crelay agents' for details)

Listening for webhooks...
```

---

## 6. State Persistence Format

### 6.1 Recommendation: Extend state.json

**Do NOT create a separate `shutdown-state.json`.** Reasons:
- Two files means two sources of truth that can go out of sync
- Atomic writes are easier with one file
- The shutdown state is tightly coupled to agent state

### 6.2 Extended State Schema

```json
{
  "agents": [
    {
      "id": "1848f8ae-ccf6-4fcf-bcef-edecf3f86d3c",
      "role": "developer",
      "ref": "auth-track_20260307140000Z",
      "status": "suspended",
      "session_id": "1848f8ae-ccf6-4fcf-bcef-edecf3f86d3c",
      "pid": 0,
      "worktree_dir": "/path/to/worker-1",
      "log_file": "/path/to/logs/1848f8ae.log",
      "started_at": "2026-03-07T10:00:00Z",
      "updated_at": "2026-03-07T15:30:00Z",
      "suspended_at": "2026-03-07T15:30:00Z",
      "last_known_healthy": true
    }
  ],
  "shutdown": {
    "status": "clean",
    "initiated_at": "2026-03-07T15:30:00Z",
    "completed_at": "2026-03-07T15:30:02Z",
    "agents_suspended": 2,
    "agents_force_killed": 0
  }
}
```

### 6.3 New Fields

| Field | Type | Purpose |
|-------|------|---------|
| `agents[].suspended_at` | `time.Time` | When agent was suspended (null if not suspended) |
| `agents[].last_known_healthy` | `bool` | Was agent responsive when last checked? |
| `shutdown.status` | `string` | `"clean"`, `"forced"`, `"unknown"` (crash) |
| `shutdown.initiated_at` | `time.Time` | When shutdown started |
| `shutdown.completed_at` | `time.Time` | When shutdown finished |
| `shutdown.agents_suspended` | `int` | Count of gracefully suspended agents |
| `shutdown.agents_force_killed` | `int` | Count of force-killed agents |

### 6.4 Worktree State Preservation

**No additional worktree persistence needed.** The existing `pool.json` already tracks:
- Which worktrees are `in-use` vs `idle`
- Which `TrackID` is assigned to each worktree
- Which `AgentID` is using each worktree

On relay restart:
- Pool state is loaded from `pool.json`
- Worktrees marked `in-use` with a suspended agent are kept allocated
- If the agent's resume fails, the worktree is returned to idle

### 6.5 PR Tracking Across Restarts

**No changes needed.** `pr-tracking.json` already persists:
- Developer and reviewer agent IDs
- Session IDs for both
- Review cycle count
- PR status

On resume, the relay loads tracking state and resumes monitoring the PR for webhook events. The resumed developer agent continues its work; if a review event arrives, the relay handles it normally.

---

## 7. Error Taxonomy

### 7.1 Recoverable Errors

| Error | Detection | Recovery |
|-------|-----------|----------|
| Agent exited cleanly on SIGINT | PID gone, session file intact | Resume with `--resume` |
| Agent stale (relay crashed) | PID not alive, status "running" | Resume with `--resume` |
| Temporary network issue | API call failed mid-session | Resume — CC retries internally |
| Context window exceeded | CC compacts automatically | Resume — compaction handles it |

### 7.2 Terminal Errors (Cannot Auto-Resume)

| Error | Detection | Action |
|-------|-----------|--------|
| Session file deleted/corrupt | `.jsonl` missing or unparseable | Mark "failed", notify user |
| Worktree deleted | `WorktreeDir` doesn't exist | Mark "failed", offer re-spawn |
| Git branch force-pushed | Branch diverged from session state | Mark "failed", notify user |
| CC version incompatible | Resume fails with version error | Mark "failed", notify user |
| Auth token expired | CC fails to authenticate | Mark "failed", user must re-auth |

### 7.3 Ambiguous Errors (Attempt Resume, May Fail)

| Error | Detection | Action |
|-------|-----------|--------|
| Agent force-killed (SIGKILL) | status "force-killed" | Attempt resume; `.jsonl` may be incomplete |
| Files changed during downtime | `git status` shows differences | Attempt resume; agent re-reads state |
| Long downtime (days+) | `suspended_at` > 24h ago | Attempt resume; warn user about staleness |

### 7.4 Error Handling Decision Tree

```
On resume attempt:
  ├─ Session file exists?
  │   ├─ No → TERMINAL: mark "failed", log "session file missing"
  │   └─ Yes
  │       ├─ Worktree exists?
  │       │   ├─ No → TERMINAL: mark "failed", log "worktree deleted"
  │       │   └─ Yes
  │       │       ├─ `claude --resume` succeeds?
  │       │       │   ├─ No → TERMINAL: mark "failed", log exit code + stderr
  │       │       │   └─ Yes → RECOVERED: update PID, status = "running"
  │       │       └─ (process starts but exits quickly?)
  │       │           └─ TERMINAL: mark "failed", log "agent exited immediately"
```

---

## 8. Limitations and Risks

### 8.1 CC-Level Limitations

1. **No session expiry API**: We cannot programmatically check if a session is still valid without attempting to resume it. The only way to know is to try `--resume` and check the exit code.

2. **Context compaction is lossy**: When resumed after a long conversation, CC may have compacted earlier messages. The agent loses detailed context about early work but retains summaries. This is acceptable for most scenarios.

3. **No atomic session writes**: If CC is killed (SIGKILL) mid-write to the `.jsonl` file, the last entry may be truncated. CC handles this gracefully on resume (ignores incomplete trailing entry), but it's a data integrity risk.

4. **Session is directory-scoped**: The session file path is derived from the working directory. If the worktree path changes, the session cannot be found. This constrains worktree management — worktrees used by suspended agents must not be moved or renamed.

5. **No concurrent resume**: A session cannot be resumed by two processes simultaneously. If a stale PID check is wrong (race condition), two CC instances may conflict.

### 8.2 Crelay-Level Risks

1. **PID reuse**: Unix PIDs are recycled. A stale PID in state.json might match a different process. Mitigation: check process name or use process groups.

2. **State file corruption**: If the relay crashes during a `state.json` write, the file may be corrupt. Mitigation: write to a temp file, then rename (atomic on most filesystems).

3. **Worktree conflicts**: If a worktree has uncommitted changes from a force-killed agent, resuming may cause git conflicts. Mitigation: check `git status` before resume.

4. **Webhook replay**: During downtime, Gitea may fire webhooks that the relay misses. On restart, the relay won't know about PR events that occurred during shutdown. Mitigation: on startup, poll Gitea API for open PRs and reconcile state.

### 8.3 Mitigations

| Risk | Mitigation | Complexity |
|------|------------|------------|
| PID reuse | Check `/proc/{pid}/cmdline` (Linux) or `ps -p {pid} -o comm=` (macOS) | Low |
| State corruption | Atomic file writes (write temp + rename) | Low |
| Worktree conflicts | `git status --porcelain` check before resume | Low |
| Missed webhooks | Gitea API poll on startup for open PRs | Medium |
| Concurrent resume | Use file lock or check PID before resume | Low |

---

## 9. Proposed Implementation Tracks

Based on this research, I recommend splitting implementation into two tracks:

### Track 1: Graceful Shutdown (estimated: 6-8 tasks)

**Scope:**
- Add shutdown hook to relay server (`Run()` method)
- Implement signal cascade (SIGINT → wait → SIGTERM → SIGKILL)
- Add `processAlive()` helper
- Add "suspended" and "force-killed" agent statuses
- Extend `state.json` with shutdown metadata
- Update `crelay down` and `crelay destroy` to trigger cascade
- Add atomic file write for state.json
- Tests for shutdown sequence

**Dependencies:** None (can be implemented on current codebase)

**Conflicts:** Low risk with refactoring tracks — touches `relay/server.go` and `state/state.go`, but adds new functions rather than modifying existing ones.

### Track 2: Auto-Resume on Startup (estimated: 6-8 tasks)

**Scope:**
- Add startup resume logic to `crelay up`
- Implement prerequisite validation (worktree exists, session file exists)
- Add resume prioritization (developers > reviewers)
- Implement stale agent detection (PID alive check)
- Add user-facing restore reporting
- Handle resume failures gracefully
- Poll Gitea for missed webhook events during downtime (optional, medium complexity)
- Tests for resume flow

**Dependencies:** Track 1 (graceful shutdown) — needs the "suspended" status and shutdown metadata

**Conflicts:** Low risk — primarily adds new logic to `cli/up.go` and `relay/server.go`.

### Optional Track 3: Resilience Hardening

**Scope (lower priority, future):**
- Periodic health check for running agents (process liveness)
- Auto-restart failed agents
- Webhook event journal for replay on startup
- Session garbage collection (clean old session files)

**Dependencies:** Tracks 1 and 2

---

## Appendix A: CC CLI Flags Reference (Resume-Related)

```
--resume [value]           Resume by session ID or interactive picker
--continue, -c             Resume most recent session in current directory
--fork-session             Branch session history (new ID, same context)
--session-id <uuid>        Start new session with specific UUID
--no-session-persistence   Disable session file creation (--print only)
```

## Appendix B: Session File Location Formula

```
~/.claude/projects/-{sanitized_path}/{session_uuid}.jsonl

Where sanitized_path = working_directory
  .replace(/^\//, '')    // Remove leading slash
  .replace(/\//g, '-')   // Replace all slashes with dashes
```

Example:
```
Working dir: /Users/ben/dev/crelay-wt/worker-1
Session dir: ~/.claude/projects/-Users-ben-dev-crelay-wt-worker-1/
Session file: ~/.claude/projects/-Users-ben-dev-crelay-wt-worker-1/abc123.jsonl
```

## Appendix C: Exit Codes

| Exit Code | Meaning |
|-----------|---------|
| 0 | Clean exit (task complete or user quit) |
| 1 | General error |
| 130 | SIGINT (128 + 2) |
| 143 | SIGTERM (128 + 15) |
| 137 | SIGKILL (128 + 9) |
