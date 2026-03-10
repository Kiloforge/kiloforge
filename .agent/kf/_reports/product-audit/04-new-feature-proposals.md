# New Feature Proposals

Net-new capabilities with user stories, value propositions, and scope estimates.

---

## F1. Notification System

**User Story:** As a developer, I want to receive notifications when my agents complete, fail, or need attention, so I don't have to constantly monitor the dashboard.

**Value Proposition:** kiloforge agents run for minutes to hours. Without notifications, users must poll the dashboard or check terminals manually. This is the single biggest UX gap for a tool designed for autonomous agent operation.

**Scope:** L (4-6 tracks)

**Proposed Design:**
- Notification channels: desktop (macOS Notification Center), webhook (custom URL), Slack, terminal bell
- Configurable per-project and per-event-type
- Events: agent_completed, agent_failed, agent_escalated, agent_timeout, queue_empty, merge_conflict
- Backend: new NotificationService with channel adapters
- Frontend: notification preferences page, notification bell icon with recent alerts
- CLI: `kf notifications list|config` commands

**Impact:** 5/5 — Transforms kiloforge from "monitor it" to "fire and forget"

---

## F2. Dynamic Skill Loading

**User Story:** As a power user, I want to create and load custom skills without recompiling kiloforge, so I can customize agent behavior for my specific workflows.

**Value Proposition:** Currently all 15 skills are embedded at compile time. Custom workflows (security auditing, compliance review, specialized refactoring) require forking the repo. Dynamic loading turns kiloforge from a tool into a platform.

**Scope:** L (6-8 tracks)

**Proposed Design:**
- Skill discovery: scan `~/.claude/skills/kf-*` directories for SKILL.md files
- Skill registry: database table tracking installed skills with metadata
- Skill lifecycle: install (from git repo or local path), update, uninstall, enable/disable
- Skill marketplace: optional remote registry for community skills
- CLI: `kf skills install <repo-url>`, `kf skills create <name>`, `kf skills test <name>`
- API: CRUD endpoints for skill management
- Dashboard: skills management page with install/update/remove

**Impact:** 4/5 — Enables community ecosystem and custom workflows

---

## F3. Multi-User Support

**User Story:** As a team lead, I want multiple developers on my team to share a kiloforge instance, so we can coordinate agent workloads and share project configurations.

**Value Proposition:** kiloforge is currently single-user, local-only. Teams working on the same codebase must each run their own instance. Shared access would enable team-wide agent coordination, shared cost tracking, and collaborative review workflows.

**Scope:** XL (10+ tracks)

**Proposed Design:**
- Authentication: API key or OAuth with local identity provider
- Authorization: role-based (admin, developer, viewer)
- User management: `kf users add|remove|list`
- Per-user agent quotas and cost tracking
- Shared project registry with per-user permissions
- Audit logging: who did what and when
- Network access: bind to configurable interface (currently localhost-only)

**Impact:** 4/5 — Prerequisite for team adoption; high complexity

---

## F4. Agent Analytics Dashboard

**User Story:** As a developer, I want to see aggregate metrics about my agent usage — success rates, token costs, duration trends — so I can optimize my workflow and budget.

**Value Proposition:** kiloforge has rich per-agent data but no aggregate views. Users can't answer: "How much did my agents cost this week?", "What's my agent success rate?", or "Which skills/tracks take the longest?"

**Scope:** M (3-4 tracks)

**Proposed Design:**
- Backend: analytics service aggregating agent data by time period, project, skill, outcome
- API: `GET /-/api/analytics?period=7d&group_by=project`
- Dashboard: analytics page with charts — cost over time, success/failure rate, duration distribution, top projects by cost
- CLI: `kf analytics --period=30d --format=table`

**Impact:** 4/5 — High value for cost-conscious users

---

## F5. Webhook Extensibility

**User Story:** As a developer, I want kiloforge to send events to external services (Slack, Discord, custom URLs), so I can integrate agent workflows with my existing tools.

**Value Proposition:** kiloforge already has a rich SSE event bus with 11+ event types. Exposing these as outbound webhooks would enable integration with any external tool without custom code.

**Scope:** M (2-3 tracks)

**Proposed Design:**
- Webhook targets: configurable URLs with optional headers and authentication
- Event filtering: subscribe to specific event types per target
- Retry policy: configurable retries with exponential backoff
- CLI: `kf webhooks add|remove|list|test`
- API: CRUD for webhook configurations
- Dashboard: webhook management in settings

**Impact:** 4/5 — Enables rich ecosystem integration

---

## F6. Agent Scheduling and Automation

**User Story:** As a developer, I want to schedule recurring agent runs (e.g., nightly code review, weekly dependency updates), so routine tasks are fully automated.

**Value Proposition:** Currently agents must be manually spawned. Scheduling enables true "set it and forget it" automation for recurring tasks like: nightly review of open PRs, weekly dependency updates, daily test coverage reports.

**Scope:** M (3-4 tracks)

**Proposed Design:**
- Cron-like scheduling: `kf schedule add --cron "0 2 * * *" --skill kf-reviewer --project myapp`
- Schedule management: `kf schedule list|remove|pause|resume`
- Backend: SchedulerService with cron engine (e.g., robfig/cron)
- API: CRUD for schedules
- Dashboard: schedule management page
- Execution history: link scheduled runs to agent instances

**Impact:** 3/5 — High value for mature users; niche for beginners

---

## F7. GitHub Forge Support

**User Story:** As a developer who uses GitHub, I want kiloforge to work with GitHub repos and PRs, so I can use kiloforge without switching to Gitea.

**Value Proposition:** kiloforge currently requires Gitea as the local forge. Many developers already use GitHub and would prefer kiloforge to integrate directly, either as an alternative forge or by bridging operations to GitHub.

**Scope:** XL (8+ tracks)

**Proposed Design:**
- GitHub adapter implementing the same port interfaces as the Gitea adapter
- Configuration: `forge: github` vs `forge: gitea` per project
- PR lifecycle: create, review, merge via GitHub API
- Webhook: GitHub webhook event handling
- Branch protection: respect GitHub branch rules
- Auth: GitHub Personal Access Token or GitHub App

**Impact:** 4/5 — Removes the biggest adoption barrier for GitHub-centric teams

---

## F8. CLI Interactive Mode (TUI)

**User Story:** As a developer working in the terminal, I want a rich terminal UI for monitoring agents and managing kiloforge, so I don't need to switch to a browser.

**Value Proposition:** The dashboard is browser-based. Developers who prefer terminal workflows must use individual CLI commands. A TUI (using Bubble Tea or similar) would provide a dashboard-like experience in the terminal.

**Scope:** L (6-8 tracks)

**Proposed Design:**
- Library: charmbracelet/bubbletea (Go TUI framework)
- Views: agent list with live updates, agent log viewer, project status, queue monitor
- Navigation: keyboard-driven with vim-like keybindings
- Real-time: SSE or polling for live updates
- CLI: `kf tui` or `kf dashboard --tui`

**Impact:** 3/5 — High value for terminal-centric users
