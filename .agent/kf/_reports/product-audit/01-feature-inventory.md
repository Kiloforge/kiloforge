# Feature Inventory

## Maturity Scale

| Rating | Label | Meaning |
|--------|-------|---------|
| 5 | Mature | Production-quality, well-tested, good UX, documented |
| 4 | Solid | Works well, minor polish needed |
| 3 | Functional | Works but has noticeable gaps |
| 2 | Basic | Minimum viable, needs significant work |
| 1 | Stub | Exists but incomplete or broken |

---

## 1. CLI Commands

| Command | Category | Maturity | Notes |
|---------|----------|----------|-------|
| `init` | Lifecycle | 5 | First-time setup with prereq checks, Docker compose, Gitea provisioning |
| `up` | Lifecycle | 5 | Daily start with health checks |
| `down` | Lifecycle | 5 | Clean shutdown |
| `destroy` | Lifecycle | 4 | Nuclear reset; could use --force confirmation UX |
| `status` | Lifecycle | 5 | Shows system health, Gitea status, running agents |
| `version` | Lifecycle | 4 | VCS stamping; worktree edge case handled via Makefile |
| `add` | Projects | 4 | Registers project, mirrors to Gitea, sets up webhooks |
| `projects` | Projects | 4 | Lists registered projects |
| `push` | Projects | 4 | Pushes to Gitea mirror |
| `sync` | Projects | 4 | Git sync with origin |
| `agents` | Agents | 4 | Lists active/recent agents |
| `logs` | Agents | 4 | Streams agent logs |
| `stop` | Agents | 4 | Stops running agent |
| `attach` | Agents | 5 | WebSocket terminal attachment to running agent |
| `escalated` | Agents | 3 | Lists escalated (permission-blocked) agents; niche UX |
| `cost` | Agents | 4 | Token usage and cost tracking |
| `implement` | Orchestration | 4 | Spawns agent to implement a track |
| `pool` | Orchestration | 4 | Worker pool management for parallel agents |
| `dashboard` | Orchestration | 5 | Opens dashboard in browser |
| `serve` | Orchestration | 5 | Starts REST API server |
| `skills` | Skills | 4 | Skill management (update, list) |

**CLI Summary:** 21 commands, average maturity 4.3/5. Strong foundation. Main gaps: no `--help` examples in most commands, no shell completion generation, no `--json` output flag for scriptability.

---

## 2. REST API

| Category | Endpoints | Maturity | Notes |
|----------|-----------|----------|-------|
| System | 5 (health, preflight, config, status, ssh-keys) | 5 | Comprehensive health + preflight checks |
| Agents | 8 (list, spawn, get, delete, log, stop, resume, consent) | 4 | Missing: bulk operations, filtering on list |
| Projects | 12 (list, add, remove, push, pull, sync, diff, branches, metadata, setup, settings) | 4 | Most complete category; missing pagination |
| Tracks | 4 (list, generate, get, delete) | 4 | Missing: update, bulk operations |
| Board | 3 (get, move card, sync) | 4 | Kanban operations; works well for single-board |
| Queue | 4 (status, start, stop, settings) | 4 | Dependency-aware scheduling |
| Locks | 4 (list, acquire, heartbeat, release) | 5 | TTL + heartbeat pattern; well-designed |
| Traces | 2 (list, get) | 3 | No filtering, no pagination, no search |
| Skills | 2 (status, update) | 3 | Minimal CRUD; no individual skill management |
| Consent | 2 (get, record) | 3 | Agent permission consent; narrow scope |
| Admin | 1 (run operation) | 3 | Single generic endpoint; not RESTful |
| Quota | 1 (get) | 4 | Token/cost aggregation |

**API Summary:** 49 operations across 43 paths. Schema-first (OpenAPI 3.1) with code generation. Consistent error format. Main gaps:
- **No pagination on any list endpoint** (critical for scale)
- **No filtering/sorting query parameters** (agents by status, traces by time range)
- **No bulk operations** (stop all agents, delete multiple tracks)
- **Rate limiting defined but not verified** (429 responses in spec, implementation unclear)

---

## 3. Dashboard (Frontend)

| Page | Components | Maturity | Notes |
|------|------------|----------|-------|
| Overview | Stats, AgentGrid, Projects, QueuePanel, Tracks, Traces | 5 | Comprehensive overview; real-time updates via SSE |
| AgentHistory | Filterable table, histogram | 4 | Good filtering; could use pagination for large histories |
| AgentDetail | Metadata, DiffView, LogViewer, AgentTerminal | 5 | Rich detail view with WebSocket terminal |
| ProjectPage | Kanban board, settings, info tabs | 4 | Board works well; settings tab could be richer |
| TrackDetail | Spec/plan display | 3 | Read-only display; no editing, no progress visualization |
| TracePage | Span timeline visualization | 3 | Basic span view; no filtering, no zoom |

**Shared Infrastructure:**
- **28 custom hooks** — Comprehensive data fetching layer
- **SSE integration** — Real-time updates for agents, projects, tracks, board
- **WebSocket** — Interactive agent terminal
- **CSS Modules** — Scoped styling throughout
- **TanStack Query** — Consistent data fetching and caching

**Dashboard Summary:** 5 pages, 63 component files, React 19. Strong real-time foundation. Main gaps:
- **No responsive/mobile design** — Desktop-only layout
- **No keyboard shortcuts** — Mouse-dependent navigation
- **No dark/light theme toggle** (or auto-detect)
- **No bulk selection/operations** in list views
- **TrackDetail is read-only** — No inline editing of specs or plans
- **No search/command palette** — No quick navigation between entities

---

## 4. Embedded Skills

| Skill | Purpose | Maturity | Notes |
|-------|---------|----------|-------|
| kf-architect | Research codebase, create tracks with specs/plans | 5 | Core workflow skill; comprehensive |
| kf-developer | Implement tracks in worktree workflow | 5 | Full lifecycle: validate → implement → merge |
| kf-reviewer | PR review against track spec | 4 | Works well; review cycle limit (5) |
| kf-implement | Execute tasks from a track plan (single-branch) | 4 | Simpler alternative to kf-developer |
| kf-new-track | Create a single track with spec and plan | 4 | Standalone track creation |
| kf-manage | Archive, restore, delete, rename tracks | 4 | Has resources/scripts (most complete skill) |
| kf-setup | Initialize project with kiloforge artifacts | 4 | First-time onboarding |
| kf-status | Display project status and next actions | 3 | Informational only |
| kf-report | Generate project timeline and velocity reports | 4 | Outputs to _reports directory |
| kf-product-advisor | Product strategy and recommendations | 3 | Advisory role; less tested than core skills |
| kf-validate | Validate kiloforge artifacts | 4 | Completeness and consistency checks |
| kf-revert | Git-aware undo by work unit | 3 | Powerful but risky; needs more guardrails |
| kf-bulk-archive | Archive all completed tracks | 4 | Batch operation |
| kf-compact-archive | Remove archived track dirs, preserve git history | 4 | Space cleanup |
| kf-parallel | DEPRECATED — redirects to kf-architect/kf-developer | 1 | Should be removed |

**Skills Summary:** 15 skills (1 deprecated), average maturity 3.8/5. Core workflow (architect → developer → reviewer) is very mature. Utility skills are adequate. Main gaps:
- **No skill versioning** — All skills update atomically; can't pin versions per project
- **No skill testing framework** — No way to validate skill behavior in CI
- **No third-party skill loading** — All skills are embedded at compile time
- **kf-parallel should be removed** — Deprecated but still ships

---

## 5. Orchestration Pipeline

| Component | Maturity | Notes |
|-----------|----------|-------|
| Agent spawning | 5 | Supports interactive and non-interactive modes, mock agent for testing |
| Work queue | 5 | Dependency-aware scheduling, priority, start/stop controls |
| Merge lock | 5 | Dual-mode (HTTP with TTL/heartbeat, mkdir fallback), crash recovery |
| PR lifecycle | 4 | Create, review, merge via Gitea; lacks GitHub support |
| OpenTelemetry tracing | 4 | Span creation across agent lifecycle; needs better visualization |
| SSE event bus | 5 | 11+ event types, reconnect handling, project scoping |
| WebSocket terminal | 5 | Real-time agent interaction |
| Cleanup service | 4 | Automatic resource cleanup |
| Git sync | 4 | Bidirectional sync with origin |

**Orchestration Summary:** Mature core pipeline. Main gaps:
- **No retry/recovery for failed agents** — If an agent crashes, manual restart required
- **No agent timeout enforcement** — Runaway agents can consume resources indefinitely
- **No GitHub PR support** — Only Gitea PRs; limits adoption for teams using GitHub
- **No webhook extensibility** — Can't send events to external systems (Slack, Discord, etc.)
