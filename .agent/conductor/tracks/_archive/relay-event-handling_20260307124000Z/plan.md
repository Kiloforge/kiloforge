# Implementation Plan: Relay Event Handling for Issues and PRs with Multi-Project Routing

**Track ID:** relay-event-handling_20260307124000Z

## Phase 1: Issue Event Support

### Task 1.1: Add issue events to webhook registration [x]
### Task 1.2: Implement issue event handlers [x]
### Task 1.3: Implement issue comment handler [x]
### Task 1.4: Register new event types in webhook dispatcher [x]

### Verification 1
- [x] Issue events handled: opened, edited, closed, label_updated, assigned
- [x] Issue comments handled
- [x] Webhook registration includes new events
- [x] Tests pass

## Phase 2: Multi-Project Routing

### Task 2.1: Inject project registry into relay server [x]
### Task 2.2: Update all event handlers for project context [x]
### Task 2.3: Update health endpoint [x]

### Verification 2
- [x] Events routed to correct project via repo name
- [x] Unknown repos logged and ignored
- [x] All handlers include project context in logs
- [x] Health endpoint reports project count
- [x] Tests pass

## Phase 3: Relay Lifecycle Integration

### Task 3.1: Integrate relay into `kf up` [x]
### Task 3.2: Integrate relay into `kf init` [x]
### Task 3.3: Update call sites for new NewServer signature [x]
### Task 3.4: Update README and docs [x]

### Verification 3
- [x] `kf up` starts Gitea + relay
- [x] `kf init` ends with relay running
- [x] Relay logs events from registered projects
- [x] README documents event handling
- [x] Build and tests pass
