# Implementation Plan: Relay Server Daemon Mode

**Track ID:** relay-daemon_20260308200000Z

## Phase 1: PID File Management (4 tasks)

### Task 1.1: Define PID file port interface
Create `internal/core/port/pidfile.go` with `PIDManager` interface: `Write(pid int) error`, `Read() (int, error)`, `Remove() error`, `IsRunning() (bool, int, error)`.

### Task 1.2: Write PID file adapter tests
Create `internal/adapter/pidfile/pidfile_test.go` — test write/read/remove/stale detection.

### Task 1.3: Implement PID file adapter
Create `internal/adapter/pidfile/pidfile.go` implementing `PIDManager`. Uses `$DataDir/relay.pid`. Stale detection via `os.FindProcess` + signal 0.

### Task 1.4: Verify Phase 1
Run `go test ./internal/adapter/pidfile/...` — all pass.

## Phase 2: Internal Serve Command (4 tasks)

### Task 2.1: Create hidden `serve` command
Add `internal/adapter/cli/serve.go` with a hidden Cobra command `serve` that runs the relay server in the foreground (same logic as current `up` server startup). This is the entry point for the daemon process.

### Task 2.2: Add graceful shutdown with PID cleanup
The `serve` command writes PID file on start, installs SIGTERM/SIGINT handler that: drains HTTP server, saves agent state, removes PID file, exits cleanly.

### Task 2.3: Add log file output
The `serve` command opens `$DataDir/relay.log` for append and redirects its logger output there. Include timestamps.

### Task 2.4: Verify Phase 2
Run `go build ./...` — compiles. Manual: `crelay serve` starts and writes PID, Ctrl+C cleans up.

## Phase 3: Daemon Spawning in `up` (4 tasks)

### Task 3.1: Refactor `up` to spawn daemon
Replace the blocking `server.Run(ctx)` call in `up.go` with: check PID → spawn `crelay serve` as detached background process → wait briefly for PID file → report success.

### Task 3.2: Add relay-already-running detection
If PID file exists and process is alive, print "Relay already running (PID X)" and skip. If stale, clean up and respawn.

### Task 3.3: Update `down` to stop relay
Add relay stop logic to `down.go`: read PID file → send SIGTERM → wait up to 5s for process exit → force SIGKILL if needed → remove stale PID.

### Task 3.4: Update `destroy` to stop relay
Add relay stop logic before Gitea teardown in `destroy.go`.

## Phase 4: Status & Polish (3 tasks)

### Task 4.1: Update `status` command
Show relay daemon state: "Relay: running (PID 12345) on :3001" or "Relay: stopped".

### Task 4.2: Update `init` command
After Gitea init, also start the relay daemon (same logic as `up`).

### Task 4.3: Verify Phase 4
Run full test suite. Manual: `crelay init` → verify both Gitea and relay start. `crelay down` → both stop. `crelay status` → shows correct state.

---

**Total: 4 phases, 15 tasks**
