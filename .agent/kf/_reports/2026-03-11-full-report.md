# Project Report: Kiloforge

> **Generated:** 2026-03-11
> **Period:** 2026-03-07 – 2026-03-11

---

## Project Timeline

| | |
|---|---|
| **Duration** | 2026-03-07 – 2026-03-11 (5 calendar days, 5 active days) |
| **Commits** | 1,552 total |
| **Tracks** | ~348 lifetime (126 completed on-disk, 199 archived, 2 pending) |
| **Codebase** | 39,652 lines Go · 21,170 lines TS/TSX · 7 SQL migrations |

### Daily Activity

| Date | Commits | Span | Tracks Completed | Key Activity |
|------|--------:|------|:----------------:|--------------|
| 2026-03-07 (Sat) | 95 | 17:03–23:59 | **14** | feat(agent): quota tracker with thread-safe aggregation; feat(agent): stream-json parser for CC output; fix: named Docker volume and unauthenticated health check |
| 2026-03-08 (Sun) | 292 | 00:00–23:49 | **58** | feat(ui): Kiloforge favicon; feat(routing): dashboard at /, Gitea at /gitea/; fix(test): update gitea client paths |
| 2026-03-09 (Mon) | 317 | 00:03–21:53 | **76** | feat(fe): diff viewer into AgentCard; feat(fe): diff viewer components — FileList, FileDiff; fix: WebSocket session context inheritance |
| 2026-03-10 (Tue) | 560 | 05:05–23:59 | **126** | feat(history): recovery status filters; feat(detail-page): recovery info and replace action; fix(ci): skip TestResumeDeveloper when CLI unavailable |
| 2026-03-11 (Wed) | 288 | 00:00–08:58 | **74** | feat(db): persist MirrorDir in SQLite; feat(cli): update destroy prompt for external mirrors; fix(agent): wire sessionIDCallback in SpawnInteractive |

---

## Velocity Progression

| Period | Commits | Description |
|--------|--------:|-------------|
| Week 10 (2026-03-07 – 2026-03-08) | 387 | Project bootstrap through initial dashboard; 72 tracks completed |
| Week 11 (2026-03-09 – 2026-03-11) | 1,165 | Parallel worktree scaling, feature blitz, reliability hardening; 276 tracks completed |

| Metric | Value |
|--------|-------|
| Commits/day (active) | ~310 avg |
| Peak | 560 commits on 2026-03-10 (parallel worktree swarm peak) |
| Week 10 track rate | ~36 tracks/day avg |
| Week 11 track rate | ~92 tracks/day avg **(2.6x speedup)** |
| Peak track completions | **126** tracks on 2026-03-10 |

---

## Project Phases

### Phase 1: Bootstrap & Architecture — *2026-03-07*
- Project inception: conductor-relay renamed to crelay/kiloforge
- Docker Compose integration, Gitea setup, init/destroy commands
- Config refactored to port/adapter pattern
- Agent spawning and stream-JSON parsing
- 95 commits

### Phase 2: Dashboard & Core Features — *2026-03-08*
- React dashboard with Kanban board, agent cards, SSE real-time updates
- Routing, favicon, theme, track generation UI
- Agent lifecycle management (spawn, stop, completion callbacks)
- Worktree build fixes, CI pipeline setup
- 292 commits

### Phase 3: Feature Blitz & E2E Testing — *2026-03-09*
- Diff viewer, interactive terminal, error toasts, admin operations
- 12 end-to-end test tracks implemented
- Agent permissions, consent dialogs, random names
- Embedded skills, WebSocket relay
- 317 commits

### Phase 4: Parallel Scaling & Polish — *2026-03-10*
- Peak velocity: 560 commits, 126 tracks completed in single day
- Analytics, agent recovery, work stash, branch diff view
- Dashboard theme alignment, design guide
- Architecture review, CI reliability hardening
- SDK adoption, board layout improvements
- 560 commits

### Phase 5: Hardening & Release Prep — *2026-03-11*
- User mirror (BE/FE), project creation, advisor hub
- CI green track, lint enforcement, documentation overhaul
- Cortex rebrand, branding cleanup
- Agent interrupt, idle suspend, diff panel features queued
- 288 commits

---

## Track Summary

### Counts

| Category | Count |
|----------|------:|
| Completed (current on-disk) | 126 |
| Archived (on-disk) | 199 |
| Pending (not started) | 2 |
| **Lifetime total** | **~348** |

### Pending Tracks

1. **fix-agent-resume-session-id-be_20260312000000Z** — Fix agent resume — capture real Claude session ID from SDK
2. **user-mirror-fe_20260310225405Z** — Surface mirror directory in Command Deck UI

### Blockers

No blockers identified.

---

## SLOC Report

> **Tool:** scc
> **Excludes:** node_modules, vendor, .git, .agent, rest/gen, storagev1, dist

| Language | Files | Code | Comments | Blanks | Lines | Complexity |
|----------|------:|-----:|---------:|-------:|------:|-----------:|
| Go | 281 | 39,652 | 2,643 | 6,285 | 48,580 | 9,679 |
| TypeScript | 209 | 21,170 | 980 | 2,971 | 25,121 | 1,747 |
| CSS | 59 | 6,636 | 74 | 1,075 | 7,785 | 0 |
| Markdown | 45 | 8,968 | 0 | 3,539 | 12,507 | 0 |
| YAML | 9 | 3,919 | 5 | 217 | 4,141 | 0 |
| SQL | 7 | 181 | 16 | 19 | 216 | 0 |
| JSON | 4 | 102 | 0 | 4 | 106 | 0 |
| Shell | 2 | 125 | 15 | 34 | 174 | 9 |
| Makefile | 1 | 100 | 6 | 21 | 127 | 45 |
| JavaScript | 1 | 31 | 3 | 1 | 35 | 0 |
| HTML | 1 | 15 | 0 | 0 | 15 | 0 |
| License | 1 | 160 | 0 | 31 | 191 | 0 |
| **TOTAL** | **620** | **81,059** | **3,742** | **14,197** | **98,998** | **11,480** |

**Processed:** 2.84 MB

### Breakdown

| Category | SLOC | Share |
|----------|-----:|------:|
| Backend (Go) | 39,652 | 49% |
| Frontend (TS/CSS/HTML) | 27,852 | 34% |
| Config/Infra (YAML/Shell/Makefile) | 4,144 | 5% |
| Schema/DB (SQL) | 181 | 0% |
| Docs (Markdown) | 8,968 | 11% |
| Other (JSON/JS/License) | 293 | 0% |

---

## Cost Estimates

### COCOMO (organic model, via scc)

| Metric | Value |
|--------|-------|
| Estimated Cost | $2,727,951 |
| Schedule Effort | 20.14 months |
| People Required | 12.03 |

### Function Point Analysis

| Component | Count | Weight | Total |
|-----------|------:|-------:|------:|
| External Inputs (EI) | 22 | x 4 | 88 |
| External Outputs (EO) | 12 | x 5 | 60 |
| External Inquiries (EQ) | 18 | x 4 | 72 |
| Internal Logical Files (ILF) | 7 | x 10 | 70 |
| External Interface Files (EIF) | 5 | x 7 | 35 |
| **Unadjusted Function Points** | | | **325** |

> **EI (22):** CLI commands (init, destroy, add, status, up, down, spawn, stop, resume, board), REST API endpoints (agents CRUD, projects CRUD, tracks, board, notifications, analytics, advisor launch), webhook receiver, file upload
> **EO (12):** SSE event streams, agent output streaming, diff views, notification toasts, analytics charts, board state, CSV/report export, health endpoint, quota metrics
> **EQ (18):** Agent list/detail/history, project list/detail, track list/status, board columns, notification list, analytics queries, advisor results, log tailing, migration status, config queries
> **ILF (7):** SQLite tables — agents, projects, tracks, work_queue, notifications, reliability_events, migrations
> **EIF (5):** Docker/Gitea API, Claude Code CLI, filesystem/git, SSH keys, OpenAPI schema

| Metric | Value |
|--------|-------|
| Value Adjustment Factor | 1.08 (GSC: 43/70) |
| Adjusted Function Points | 351 |

> **GSC Breakdown (43/70):** Data communications (5), Distributed processing (3), Performance (3), Config complexity (4), Transaction rate (4), Online entry (4), End-user efficiency (4), Online update (3), Complex processing (3), Reusability (3), Installation ease (2), Operational ease (2), Multiple sites (1), Facilitate change (2)

| Rate | Estimate |
|------|----------|
| Low ($500/FP) | $175,500 |
| Mid ($1,000/FP) | $351,000 |
| High ($1,500/FP) | $526,500 |

### Parametric (SLOC-based)

| Metric | Value |
|--------|-------|
| SLOC | 81,059 (functional code) |
| Productivity range | 10–20 SLOC/hr |
| Effort | 4,053 – 8,106 hours |
| Cost @ $75–150/hr | $303,975 – $1,215,900 |

### Effort by Analogy

> Comparable scope: Local dev platform with Go backend (REST + SSE + WebSocket), React dashboard, Docker orchestration, embedded Gitea git forge, SQLite persistence, CLI tooling, AI agent spawning and lifecycle management, real-time observability

| Context | Estimate |
|---------|----------|
| Freelance/agency | $200,000 – $500,000 |
| In-house team (5–6 months) | $400,000 – $800,000 |

### AI-Assisted Actual Cost

| Metric | Value |
|--------|-------|
| Active dev time | 5 days (5 calendar days) |
| Estimated API cost | ~$200 – $500 |
| Human time | Solo developer + Claude Code agent swarm; human provides direction and review |

### Aggregate Cost Summary

| Model | Low | Mid | High |
|-------|----:|----:|-----:|
| COCOMO | — | $2,727,951 | — |
| Function Point Analysis | $175,500 | $351,000 | $526,500 |
| Parametric (SLOC) | $303,975 | $759,938 | $1,215,900 |
| Effort by Analogy | $200,000 | $600,000 | $800,000 |
| **Cross-model range** | **$175,500** | **$1,109,722** | **$2,727,951** |

| Aggregate Metric | Value |
|------------------|-------|
| Median estimate | ~$680,000 |
| Geometric mean | ~$764,000 |
| Actual (AI-assisted) | ~$200 – $500 |
| **Efficiency factor** | **~1,360x – 3,400x cost reduction vs median** |

---

## Summary

Built in **5 calendar days** with **1,552 commits** across **5 active days**.

| Metric | Value |
|--------|-------|
| SLOC | 81,059 (Go, TypeScript, CSS) |
| Files | 620 |
| Tracks (lifetime) | ~348 |
| Tracks completed | 126 (on-disk) + 199 archived |
| Tracks pending | 2 |
| SQL migrations | 7 |
| Compaction points | 0 |
| Peak velocity | 560 commits/day, 126 tracks/day (2026-03-10, parallel swarm) |
