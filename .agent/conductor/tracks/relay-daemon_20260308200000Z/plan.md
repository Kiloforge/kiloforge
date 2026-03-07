# Implementation Plan: Relay Server Daemon Mode

**Track ID:** relay-daemon_20260308200000Z

## Phase 1: PID File Management (4 tasks)

### Task 1.1: Define PID file port interface
- [x] Create `internal/core/port/pidfile.go` with `PIDManager` interface

### Task 1.2: Write PID file adapter tests
- [x] Create `internal/adapter/pidfile/pidfile_test.go`

### Task 1.3: Implement PID file adapter
- [x] Create `internal/adapter/pidfile/pidfile.go` implementing `PIDManager`

### Task 1.4: Verify Phase 1
- [x] Run `go test ./internal/adapter/pidfile/...` — all pass

## Phase 2: Internal Serve Command (4 tasks)

### Task 2.1: Create hidden `serve` command
- [x] Add `internal/adapter/cli/serve.go` with hidden Cobra command

### Task 2.2: Add graceful shutdown with PID cleanup
- [x] PID file write on start, SIGTERM/SIGINT handler, cleanup on exit

### Task 2.3: Add log file output
- [x] Log output to `$DataDir/relay.log` with timestamps

### Task 2.4: Verify Phase 2
- [x] `go build ./...` compiles

## Phase 3: Daemon Spawning in `up` (4 tasks)

### Task 3.1: Refactor `up` to spawn daemon
- [x] Replace blocking server with daemon spawn via `crelay serve`

### Task 3.2: Add relay-already-running detection
- [x] PID file check, stale cleanup, skip if running

### Task 3.3: Update `down` to stop relay
- [x] SIGTERM → wait 5s → SIGKILL → remove stale PID

### Task 3.4: Update `destroy` to stop relay
- [x] Stop relay daemon before Gitea teardown

## Phase 4: Status & Polish (3 tasks)

### Task 4.1: Update `status` command
- [x] Show relay daemon state with PID and port

### Task 4.2: Update `init` command
- [x] Spawn relay daemon after Gitea setup

### Task 4.3: Verify Phase 4
- [x] Full test suite passes (17 packages)

---

**Total: 4 phases, 15 tasks — all complete**
