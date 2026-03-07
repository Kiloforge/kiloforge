# Implementation Plan: Relay Event Handling for Issues and PRs with Multi-Project Routing

**Track ID:** relay-event-handling_20260307124000Z

## Phase 1: Issue Event Support

### Task 1.1: Add issue events to webhook registration
- Update `CreateWebhook()` in `internal/gitea/client.go`
- Add `"issues"` and `"issue_comment"` to the events array
- Tests: verify event list includes new events

### Task 1.2: Implement issue event handlers
- Add `handleIssues()` to `internal/relay/server.go`
- Handle actions: `opened`, `edited`, `closed`, `label_updated`, `assigned`
- Extract: issue number, title, action, labels, assignee
- Log structured output for each action
- Tests: verify handler extracts fields correctly from sample payloads

### Task 1.3: Implement issue comment handler
- Add `handleIssueComment()` to `internal/relay/server.go`
- Handle action: `created`
- Extract: issue number, comment body (truncated), author
- Log structured output
- Tests: verify handler with sample payload

### Task 1.4: Register new event types in webhook dispatcher
- Update the `switch event` block in `handleWebhook()` to dispatch `"issues"` and `"issue_comment"`
- Tests: verify dispatch routes correctly

### Verification 1
- [ ] Issue events handled: opened, edited, closed, label_updated, assigned
- [ ] Issue comments handled
- [ ] Webhook registration includes new events
- [ ] Tests pass

## Phase 2: Multi-Project Routing

### Task 2.1: Inject project registry into relay server
- Change `NewServer()` signature: replace `repoName string` param with project registry
- Server loads registry and holds a reference for lookups
- Add `resolveProject(payload)` method that extracts `repository.name` and looks up in registry
- If project not found, log warning and return
- Tests: verify routing with mock registry

### Task 2.2: Update all event handlers for project context
- Each handler calls `resolveProject()` first
- Include project slug in all log output: `[relay] [project-slug] event: details`
- Pass project context to handlers (project dir, slug, repo name)
- Update existing PR handlers to use project context instead of `s.repoName`
- Remove `repoName` field from Server struct
- Tests: verify project context appears in handler logic

### Task 2.3: Update health endpoint
- `/health` returns project count from registry
- Example: `{"status": "ok", "projects": 3}`
- Tests: verify health response

### Verification 2
- [ ] Events routed to correct project via repo name
- [ ] Unknown repos logged and ignored
- [ ] All handlers include project context in logs
- [ ] Health endpoint reports project count
- [ ] Tests pass

## Phase 3: Relay Lifecycle Integration

### Task 3.1: Integrate relay into `crelay up`
- After Gitea is started and ready, start the relay server (blocking, foreground)
- Relay loads global config + project registry
- Ctrl+C stops the relay; Gitea stays running (compose)
- Print relay URL and registered project count on startup
- If no projects registered, print hint to use `crelay add`

### Task 3.2: Integrate relay into `crelay init`
- After first-time Gitea setup, start the relay (same as `up`)
- `init` flow: setup Gitea → configure admin → save config → start relay (blocking)

### Task 3.3: Update call sites for new NewServer signature
- Update any code that constructs a Server to use the new signature (registry instead of repoName)
- Ensure `crelay up` and `crelay init` both construct the server the same way

### Task 3.4: Update README and docs
- Document relay behavior: starts with `up`/`init`, runs in foreground
- Document supported events (issues, PRs) with log output examples
- Update architecture diagram to show multi-project event flow
- Update `docs/commands.md`

### Verification 3
- [ ] `crelay up` starts Gitea + relay
- [ ] `crelay init` ends with relay running
- [ ] Relay logs events from registered projects
- [ ] README documents event handling
- [ ] Build and tests pass
