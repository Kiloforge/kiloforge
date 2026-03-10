# Agents & Swarms

This page covers the agent lifecycle, roles, status transitions, suspension and resume, the notification system, and Swarm coordination.

## Agent Roles

Every agent has a role that determines its behavior, suspension policy, and interaction model.

| Role | Type | Description |
|------|------|-------------|
| `developer` | Worker | Implements tracks autonomously in pooled worktrees. Never auto-suspends. |
| `reviewer` | Worker | Reviews PRs against track specs. Never auto-suspends. |
| `architect` | Interactive | Researches codebases and creates tracks. Auto-suspends on disconnect. |
| `advisor-product` | Interactive | Product strategy, branding, and competitive analysis. Auto-suspends on disconnect. |
| `advisor-reliability` | Interactive | Audits testing, linting, type safety, and CI. Auto-suspends on disconnect. |
| `interactive` | Interactive | General-purpose interactive session. Auto-suspends on disconnect. |

**Worker roles** (developer, reviewer) run autonomously without a browser connection. They are spawned via `kf implement` or directly and continue working until their task is complete. They never auto-suspend — even if the Kiloforger closes the Command Deck.

**Interactive roles** (architect, advisor, interactive) require the Kiloforger's input. They communicate via WebSocket through the Command Deck's interactive terminal. They auto-suspend after a grace period when the Kiloforger disconnects.

## Agent Lifecycle

### Statuses

| Status | Terminal? | Description |
|--------|-----------|-------------|
| `running` | No | Agent is actively executing (processing a turn) |
| `waiting` | No | Agent has finished its turn and is waiting for input or next task |
| `halted` | No | Agent has been paused (e.g., board demotion, pre-attach) |
| `suspending` | No | Agent is in the process of being suspended (shutdown phase) |
| `suspended` | No | Agent has been suspended due to idle disconnect or graceful shutdown |
| `stopped` | Yes | Agent was manually stopped by the Kiloforger via `kf stop` |
| `completed` | Yes | Agent finished its work successfully |
| `failed` | Yes | Agent encountered a fatal error |
| `force-killed` | Yes | Agent was force-killed during shutdown (SIGKILL after timeout) |
| `resume-failed` | Yes | Resume attempt failed (missing session or worktree) |
| `replaced` | Yes | Agent was replaced by another instance |

### Status Transitions

```
                    ┌─────────┐
                    │ running │◄──────────────────────┐
                    └────┬────┘                       │
                         │                            │
                    turn complete                  resume
                         │                            │
                    ┌────▼────┐                  ┌────┴──────┐
                    │ waiting │                  │ suspended │
                    └────┬────┘                  └───────────┘
                         │                            ▲
              ┌──────────┼──────────┐                 │
              │          │          │            idle timeout /
         kf stop    completes    fails         graceful shutdown
              │          │          │                 │
         ┌────▼───┐ ┌────▼─────┐ ┌─▼────┐    ┌──────┴────┐
         │stopped │ │completed │ │failed│    │suspending │
         └────────┘ └──────────┘ └──────┘    └───────────┘
```

### Active vs Terminal

- **Active** statuses (`running`, `waiting`): The agent process is alive and may continue working.
- **Terminal** statuses (`stopped`, `completed`, `failed`, `force-killed`, `resume-failed`, `replaced`): The agent process has exited. The session may still be resumable via `kf attach`.

## Suspension

The Cortex automatically suspends interactive agents that lose their WebSocket connection. This conserves resources when the Kiloforger navigates away from the Command Deck.

### How It Works

1. The Kiloforger opens the Command Deck and connects to an interactive agent via WebSocket
2. The agent processes the Kiloforger's input and sends structured responses
3. The Kiloforger closes the browser tab or navigates away — the WebSocket disconnects
4. The `ConnectionSuspender` starts a grace period timer (default: 30 seconds)
5. If the Kiloforger reconnects within the grace period, the timer is cancelled
6. If the timer expires, the agent is suspended:
   - Status changes to `suspending`, then `suspended`
   - The Claude Code session is saved
   - The agent process is terminated gracefully

### Worker Protection

Worker roles (`developer`, `reviewer`) are **never auto-suspended**, regardless of WebSocket connection state. They run autonomously and don't need a browser connection. This is critical — a developer agent mid-implementation must not be interrupted because the Kiloforger closed a browser tab.

### Grace Period Configuration

The grace period is configurable in `~/.kiloforge/config.json`:

```json
{
  "idle_suspend_seconds": 30
}
```

Set to `0` to disable auto-suspension entirely.

### Resume

Suspended agents can be resumed in two ways:

1. **`kf attach <agent-id>`** — Prints the `claude --resume <session-id>` command to resume the agent's Claude session interactively
2. **Command Deck** — Click "Resume" on a suspended agent's card

The agent picks up exactly where it left off, with full session context preserved.

## Graceful Shutdown

When the Kiloforger runs `kf down`, the Cortex performs a three-phase shutdown:

1. **Phase 1 — SIGINT:** Send SIGINT to all `running`/`waiting` agents. Status changes to `suspending`.
2. **Phase 2 — Wait:** Wait up to a timeout for agents to exit gracefully. Exited agents are marked `suspended`.
3. **Phase 3 — SIGKILL:** Force-kill any agents that haven't exited. These are marked `force-killed`.

On the next `kf up`, suspended agents are available for resume. Force-killed agents may have lost their most recent work but the session is still available for manual recovery.

## Notifications

The notification system alerts the Kiloforger when agents need attention.

### When Notifications Fire

| Trigger | Condition | Title |
|---------|-----------|-------|
| Interactive turn end | An interactive agent finishes its turn and waits for input | "{agent_name} needs your attention" |
| Worker turn end | A worker agent's log shows `turn_end` (detected by watcher every 2s) | "{agent_name} needs your attention" |
| Turn start | Agent begins processing new input | Notification auto-dismissed |
| Terminal status | Agent reaches stopped/completed/failed/etc. | All notifications cleaned up |

### Deduplication

Only one active notification exists per agent at a time. If an agent already has an unacknowledged notification, creating another is a no-op.

### Delivery

Notifications are delivered to the Command Deck via SSE events:
- `notification_created` — new notification with ID, agent ID, title, and body
- `notification_dismissed` — notification cleared (agent resumed or reached terminal status)

### API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `GET /api/notifications` | GET | List active (unacknowledged) notifications. Optional `?agent_id=` filter. |
| `POST /api/notifications/{id}/acknowledge` | POST | Mark a notification as acknowledged |

## Swarm Coordination

The Swarm is the collective of all Claude Code agents managed by the Cortex for a given project.

### Capacity

The maximum number of concurrent agents is controlled by `max_swarm_size` (default: 3):

```json
{
  "max_swarm_size": 3
}
```

The Cortex enforces this limit. When the Swarm is at capacity:
- `kf implement` rejects new spawn requests with a capacity error
- Interactive agent spawns from the Command Deck are blocked
- Capacity changes are published as `capacity_changed` SSE events

The Kiloforger can check capacity via:
```bash
kf agents    # shows active agent count
kf pool      # shows worktree utilization
```

Or in the Command Deck's Swarm panel.

### Worktree Pool

Each developer agent runs in an isolated git worktree. The pool:

- Maintains worktree slots for the project (one per potential developer agent)
- When `kf implement` is called: acquires an idle worktree, resets to main, creates an implementation branch
- When an agent completes: returns the worktree to the idle pool
- When an agent is interrupted: auto-commits work and creates a stash branch for recovery
- Pool state is persisted in `pool.json`

### Queue Service

The queue service automates track scheduling with dependency awareness:

1. **Scan** — Reads `tracks.yaml` and `deps.yaml` to find tracks ready for implementation
2. **Order** — Topologically sorts ready tracks by their dependency graph
3. **Enqueue** — Adds tracks to the queue, respecting `max_swarm_size`
4. **Spawn** — Acquires a worktree and spawns a developer agent for each queued track
5. **Monitor** — Watches agent lifecycle events; when an agent completes, returns its worktree and spawns the next track
6. **Capacity check** — Defers to the global Swarm capacity checker before spawning

The queue can be controlled via the API:
- `POST /api/queue/start` — Start automatic queue processing
- `POST /api/queue/stop` — Pause the queue
- `PUT /api/queue/settings` — Adjust max workers

### Merge Serialization

When multiple developer agents complete their tracks concurrently, they must merge to main one at a time. The merge lock ensures serialization:

1. Agent acquires the merge lock (HTTP mode preferred, mkdir fallback)
2. Agent rebases onto latest main
3. Agent runs full verification suite (tests, build, lint)
4. Agent fast-forward merges into main
5. Agent releases the merge lock

If the lock is held by another agent, the requesting agent waits (with configurable timeout). The lock has a TTL of 120 seconds with heartbeat every 30 seconds — if an agent crashes while holding the lock, it expires automatically.

See the [Architecture Overview](architecture.md) for more on the merge lock.

### Dispatch

The `/kf-dispatch` skill analyzes project state and produces worker assignments:

- Identifies idle worktrees in the pool
- Scans the track registry for pending tracks with satisfied dependencies
- Considers conflict pairs to avoid scheduling conflicting tracks in parallel
- Produces prescriptive assignments: "worker-1 should implement track X, worker-2 should implement track Y"

This is useful when the Kiloforger has multiple tracks ready and wants the Swarm to work through them efficiently.
