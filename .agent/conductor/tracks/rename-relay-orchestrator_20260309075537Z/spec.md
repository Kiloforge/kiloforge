# Specification: Rename Relay Server to Orchestrator

**Track ID:** rename-relay-orchestrator_20260309075537Z
**Type:** Refactor
**Created:** 2026-03-09T07:55:37Z
**Status:** Draft

## Summary

Rename the "relay server" to "orchestrator" throughout the codebase. The server's role has evolved from a simple webhook relay into a full development orchestration engine — managing agents, worktree pools, board state, tracing, locks, and the dashboard. The name should reflect what it actually does.

## Context

The relay server was originally conceived as a webhook relay between Gitea and Claude Code agents. It has since grown into:
- Agent lifecycle management (spawn, monitor, halt, resume)
- Worktree pool coordination
- PR review cycle orchestration
- Board state management (native kanban)
- Scoped lock service
- OpenTelemetry trace collection
- Dashboard server with real-time SSE
- Quota tracking and cost reporting

"Relay" no longer describes this component. "Orchestrator" accurately reflects its role as the central coordinator of the kiloforge development workflow.

## Codebase Analysis

**Blast radius: ~30 files, 128 references**

| Category | References | Files |
|----------|-----------|-------|
| Go config struct/fields (`RelayPort`) | 5 | 4 (config.go, defaults.go, flags_adapter.go, merger.go) |
| CLI user-facing messages | 38 | 8 (serve.go, daemon.go, init.go, up.go, down.go, destroy.go, status.go, dashboard.go, add.go) |
| REST server comments/logging | 5 | 1 (server.go) |
| Gitea client (webhook relay port) | 2 | 1 (client.go) |
| Compose template | 7 | 2 (template.go, template_test.go) |
| AsyncAPI spec | 1 | 1 (asyncapi.yaml) |
| README | 6 | 1 |
| Log filename (`relay.log`) | 2 | 2 (serve.go, daemon.go) |
| Env var (`KF_RELAY_URL`) | planned | skills + config |

### Key Rename Mappings

| Before | After |
|--------|-------|
| `RelayPort` (struct field) | `OrchestratorPort` |
| `relay_port` (JSON config) | `orchestrator_port` |
| `WithRelayPort()` (flag option) | `WithOrchestratorPort()` |
| `relay.log` (log file) | `orchestrator.log` |
| `[relay]` (log prefix) | `[orchestrator]` |
| "relay daemon" (CLI messages) | "orchestrator" |
| "relay server" (docs) | "orchestrator" |
| `KF_RELAY_URL` (env var) | `KF_ORCH_URL` |
| `relayPort` (local vars) | `orchPort` |
| `relayRunning` / `relayPID` | `orchRunning` / `orchPID` |
| "Relay Server" (README) | "Orchestrator" |

### Naming Convention

- **Full name in docs/UI:** "orchestrator"
- **Short form in code vars:** `orch` (e.g., `orchPort`, `OrchestratorPort`)
- **Env var:** `KF_ORCH_URL` (short, matches convention of `KF_*` being brief)
- **Log file:** `orchestrator.log`

## Acceptance Criteria

- [ ] Config struct field renamed from `RelayPort` to `OrchestratorPort`
- [ ] JSON config key renamed from `relay_port` to `orchestrator_port`
- [ ] Flag option renamed from `WithRelayPort` to `WithOrchestratorPort`
- [ ] All CLI user-facing messages say "orchestrator" instead of "relay"
- [ ] Log prefix changed from `[relay]` to `[orchestrator]`
- [ ] Log filename changed from `relay.log` to `orchestrator.log`
- [ ] REST server comments updated
- [ ] Gitea client parameter renamed from `relayPort` to `orchPort`
- [ ] Compose template field renamed
- [ ] AsyncAPI spec description updated
- [ ] README updated
- [ ] Env var references updated to `KF_ORCH_URL`
- [ ] All tests pass after rename
- [ ] `make build` succeeds

## Dependencies

- **rebrand-historical-records_20260309063900Z** — Should ideally complete first to avoid double-touching conductor track files. But not a hard blocker since this track focuses on Go code + docs, not track metadata.

## Blockers

None.

## Conflict Risk

- **kf-skills-source_20260309063859Z** — LOW. Skills reference `KF_RELAY_URL` which this track renames to `KF_ORCH_URL`. If skills track completes first, the skill files will need updating. If this track completes first, the skills track should use `KF_ORCH_URL` directly. Either ordering works — just one extra find-replace.
- **rebrand-historical-records_20260309063900Z** — LOW. Both touch `.agent/conductor/` docs but different content ("crelay" vs "relay").

## Out of Scope

- The `rest/` package directory name stays — it describes the transport protocol, not the role
- The Cobra subcommand `serve` stays — it's a verb, not a role name
- Renaming the `daemon` package — it describes the process mode, not the role
- Historical track IDs containing "relay" — track IDs are immutable timestamps

## Technical Notes

- **Config migration:** Users with existing `~/.kiloforge/config.json` containing `relay_port` will need the config reader to handle the old key gracefully. Add a simple fallback: if `orchestrator_port` is 0 and `relay_port` exists, use `relay_port`. Or just document the breaking change (this is pre-release software).
- **Log file rename:** Existing `relay.log` files are not migrated — new logs go to `orchestrator.log`
- **Compose template:** The `RelayPort` template field renames to `OrchestratorPort` — the compose YAML itself doesn't use the word "relay" (it's just a port number)

---

_Generated by conductor-track-generator from prompt: "Rename relay server to orchestrator"_
