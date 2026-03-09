# E2E Test Tracks — Dependency Order

## Overview

12 tracks for comprehensive E2E testing using Playwright and a mock agent binary. All tracks depend on the foundation track which sets up test infrastructure.

## Dependency Graph

```
e2e-infra-mock-agent (Track 1 — FOUNDATION)
├── e2e-health-preflight-onboarding (Track 2)
├── e2e-project-management (Track 3)
│   ├── e2e-track-management (Track 4)
│   ├── e2e-kanban-board (Track 7)
│   └── e2e-git-origin-sync (Track 11)
├── e2e-agent-lifecycle (Track 5)
├── e2e-interactive-terminal (Track 6)
├── e2e-sse-realtime (Track 8)
├── e2e-merge-lock (Track 9)
├── e2e-distributed-tracing (Track 10)
└── e2e-quota-usage (Track 12)
```

## Execution Waves

Tracks within the same wave can be developed in parallel.

### Wave 1 (must complete first)
| Track ID | Title |
|----------|-------|
| `e2e-infra-mock-agent_20260309194830Z` | E2E Test Infrastructure and Mock Agent Binary |

### Wave 2 (depends only on Wave 1)
| Track ID | Title |
|----------|-------|
| `e2e-health-preflight-onboarding_20260309194831Z` | Health Check, Preflight Validation, and Onboarding Flow |
| `e2e-project-management_20260309194832Z` | Project Management — Add, Remove, Setup, and Sync Status |
| `e2e-agent-lifecycle_20260309194834Z` | Agent Lifecycle — Spawn, Monitor, Stop, Resume, Delete |
| `e2e-interactive-terminal_20260309194835Z` | Interactive Agent Terminal via WebSocket |
| `e2e-sse-realtime_20260309194837Z` | Real-Time SSE Updates — Connection, Events, and Reconnection |
| `e2e-merge-lock_20260309194838Z` | Merge Lock Coordination — Acquire, Heartbeat, Release, Conflict |
| `e2e-distributed-tracing_20260309194839Z` | Distributed Tracing — Trace List, Detail, and Timeline |
| `e2e-quota-usage_20260309194841Z` | Quota and Token Usage — Display, Rate Limits, and Cost Estimates |

### Wave 3 (depends on Wave 1 + `e2e-project-management`)
| Track ID | Title |
|----------|-------|
| `e2e-track-management_20260309194833Z` | Track Management — List, Detail, Generate, and Delete |
| `e2e-kanban-board_20260309194836Z` | Kanban Board — View, Move, Sync, and Column Transitions |
| `e2e-git-origin-sync_20260309194840Z` | Git Origin Sync — Push, Pull, Status, and Error Handling |

## Key Constraints

- **Mock agent only** — no real Claude CLI. Use the mock agent binary from Track 1.
- **Playwright MCP** — developer agents should use the Playwright skill to verify tests as they build them.
- **Three test categories per track** — happy path, edge cases, and expected failures.
- **Wave 3 needs a project** — Tracks 4, 7, 11 require a project to exist, so they depend on Track 3's test helpers for project seeding.
