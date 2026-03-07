# Implementation Plan: Add 'up', 'down' Commands and Refactor 'destroy'

**Track ID:** down-destroy-commands_20260307123000Z

## Phase 1: Compose Runner and New Commands

### Task 1.1: Add Stop method to compose runner
- Add `Stop(ctx, composeDir) error` to `internal/compose/runner.go`
- Runs `docker compose stop` — stops containers without removing them
- Tests: verify correct command is built

### Task 1.2: Implement `crelay up` command
- Create `internal/cli/up.go`
- Loads config — errors if not found ("run 'crelay init' first")
- Detects compose runner
- Checks if Gitea is already running (via API) — if so, print and exit
- Calls `runner.Up()` → waits for Gitea ready → prints URL
- Flags: none (ports/dirs come from saved config)
- Register in root.go

### Task 1.3: Implement `crelay down` command
- Create `internal/cli/down.go`
- Loads config, detects compose runner, calls `runner.Stop()`
- Prints success message with restart hint (`crelay up`)
- If Gitea is already stopped, print "Gitea is not running" and exit cleanly
- Register in root.go

### Task 1.4: Update `crelay init` success message
- Change the post-init message to reference `crelay down` / `crelay up` for the stop/start cycle
- Remove "(coming soon)" references if still present

### Verification 1
- [ ] `crelay up` starts Gitea when initialized
- [ ] `crelay up` errors when not initialized
- [ ] `crelay down` stops Gitea without data loss
- [ ] `crelay up` restarts after `down`
- [ ] Tests pass

## Phase 2: Destroy Refactor and Docs

### Task 2.1: Rewrite `crelay destroy` with confirmation
- Rewrite `internal/cli/destroy.go`
- Remove `--data` flag (destroy always deletes everything)
- Add `--force` flag to skip confirmation
- Without `--force`: print critical warning, prompt for "yes" via stdin
- Steps: compose down --volumes → remove data directory → print done
- If config can't be loaded (already destroyed), print "nothing to destroy" and exit

### Task 2.2: Update README and docs
- Add `up` and `down` to Commands section with examples
- Update `destroy` documentation with confirmation behavior
- Update `docs/commands.md` and `docs/getting-started.md`
- Update Architecture/Workflow sections if they reference `init` for start/stop

### Task 2.3: Final verification
- `go build ./...` succeeds
- `go test ./...` passes
- Full cycle: init → down → up → down → destroy --force

### Verification 2
- [ ] `crelay destroy` shows warning and requires confirmation
- [ ] `crelay destroy --force` skips prompt
- [ ] All docs updated with up/down/destroy
- [ ] Build and tests pass
