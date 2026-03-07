# Command Reference

## `crelay init`

Initialize Gitea and start the relay server. This is the primary entry point — one command to start everything.

**Synopsis:**
```bash
crelay init [--gitea-port PORT] [--relay-port PORT] [--repo NAME] [--data-dir PATH]
```

**What it does (in order):**
1. Creates data directory (`~/.crelay/`)
2. Starts Gitea Docker container (or restarts if stopped)
3. Waits for Gitea to become healthy (up to 60s)
4. Creates admin user `conductor` via `gitea admin user create`
5. Creates API access token
6. Creates repository matching current directory name
7. Adds `gitea` git remote to project
8. Pushes `main` branch to Gitea
9. Registers webhooks for PR events
10. Saves configuration to `~/.crelay/config.json`
11. Starts relay server (blocks until Ctrl+C)

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--gitea-port` | `3000` | Port for Gitea web UI and API |
| `--relay-port` | `3001` | Port for webhook relay server |
| `--repo` | Directory name | Repository name in Gitea |
| `--data-dir` | `~/.crelay` | Where to store config, state, logs, and Gitea data |

**Idempotent:** Safe to run multiple times. Skips steps that are already done (container exists, user exists, repo exists).

---

## `crelay status`

Display the current status of all components.

**Synopsis:**
```bash
crelay status
```

**Output:**
```
Conductor Relay Status
======================
Gitea:       running (http://localhost:3000)
Relay:       running (http://localhost:3001)
Project:     /Users/you/dev/my-project
Repository:  conductor/my-project
Data:        /Users/you/.crelay
Agents:      2 active
```

**Checks performed:**
- Docker container status via `docker inspect`
- Relay health via `curl http://localhost:PORT/health`
- Agent count from state file

---

## `crelay agents`

List all tracked agents.

**Synopsis:**
```bash
crelay agents [--json]
```

**Output (table):**
```
ID        ROLE       TRACK/PR    STATUS    SESSION   STARTED
a1b2c3d4  developer  auth_track  running   e5f6g7h8  14:23:01
i9j0k1l2  reviewer   PR #3       completed m3n4o5p6  14:25:12
```

**Output (JSON):**
```json
[
  {
    "id": "a1b2c3d4-...",
    "role": "developer",
    "ref": "auth_track",
    "status": "running",
    "session_id": "e5f6g7h8-...",
    "pid": 12345,
    "worktree_dir": "/Users/you/dev/my-project",
    "log_file": "/Users/you/.crelay/logs/a1b2c3d4-....log",
    "started_at": "2026-03-07T14:23:01Z",
    "updated_at": "2026-03-07T14:23:01Z"
  }
]
```

**Agent statuses:**

| Status | Meaning |
|--------|---------|
| `running` | Agent process is alive and working |
| `waiting` | Agent is alive but waiting for input (e.g., review feedback) |
| `halted` | Agent was stopped via `attach` (session preserved) |
| `stopped` | Agent was stopped via `stop` |
| `completed` | Agent finished successfully |
| `failed` | Agent process exited with error |

---

## `crelay logs <agent-id>`

View log output from an agent.

**Synopsis:**
```bash
crelay logs <agent-id> [-f]
```

**Arguments:**
- `agent-id` — Full or prefix of the agent UUID (minimum 4 characters)

**Flags:**

| Flag | Description |
|------|-------------|
| `-f`, `--follow` | Follow mode — stream new output as it arrives |

**Logs format:** Raw stream-json output from Claude, one JSON object per line. Each line contains tool calls, text output, progress events, etc.

---

## `crelay attach <agent-id>`

Halt a running agent and provide the command to resume it interactively.

**Synopsis:**
```bash
crelay attach <agent-id>
```

**What it does:**
1. Looks up the agent by ID (prefix match)
2. If the agent is running, sends SIGINT to halt it
3. Updates agent status to `halted`
4. Prints the `claude --resume <session-id>` command

**Use cases:**
- Agent is waiting for input (review feedback, merge approval)
- Agent is stuck and needs manual guidance
- You want to inspect what the agent is doing interactively

**After attaching:** The agent's full conversation history is preserved. When you run `claude --resume`, you pick up exactly where the agent left off.

---

## `crelay stop <agent-id>`

Stop a running agent without providing resume instructions.

**Synopsis:**
```bash
crelay stop <agent-id>
```

**What it does:**
1. Sends SIGINT to the agent process
2. Updates status to `stopped`
3. Prints the session ID for later reference

**Difference from `attach`:** `stop` is for when you want to terminate an agent. `attach` is for when you want to take over.

---

## `crelay destroy`

Tear down the Gitea instance and clean up.

**Synopsis:**
```bash
crelay destroy [--data]
```

**What it does:**
1. Stops the Gitea Docker container
2. Removes the container
3. Removes the `gitea` git remote from the project

**Flags:**

| Flag | Description |
|------|-------------|
| `--data` | Also delete `~/.crelay/` (config, state, logs, Gitea volumes) |

**Note:** Running agents are NOT stopped by `destroy`. Stop them first with `crelay stop`.

---

## Relay HTTP API

The relay server exposes a minimal HTTP API (primarily for webhook consumption, but useful for scripting).

### `GET /health`

Health check.

```bash
curl http://localhost:3001/health
# {"status":"ok"}
```

### `GET /api/agents`

List all agents as JSON.

```bash
curl http://localhost:3001/api/agents
# [{"id":"...","role":"reviewer",...}]
```

### `POST /webhook`

Receives Gitea webhook payloads. Not intended for manual use — Gitea sends these automatically.

**Handled events:**
- `pull_request` (actions: `opened`, `reopened`, `synchronize`)
- `pull_request_review`
- `pull_request_comment`
