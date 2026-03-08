# Conductor - Kiloforge

Navigation hub for project context.

## Quick Links

- [Product Definition](./product.md)
- [Product Guidelines](./product-guidelines.md)
- [Tech Stack](./tech-stack.md)
- [Workflow](./workflow.md)
- [Tracks](./tracks.md)
- [Go Style Guide](./code_styleguides/go.md)

## Active Tracks

- **research-docker-compose-init_20260307120000Z** — Research: Docker Compose for Init with v1/v2 CLI Compatibility
- **research-global-gitea-multiproject_20260307120001Z** — Research: Global Gitea Server for Multi-Project Coordination
- **init-docker-compose_20260307120100Z** — Implement Init Command with Docker Compose and Global Gitea
- **refactor-config-port-adapter_20260307121000Z** — Refactor Config to Port/Adapter Pattern with Layered Resolution
- **add-project-command_20260307122000Z** — Implement 'kf add' Command for Project Registration
- **down-destroy-commands_20260307123000Z** — Add 'up', 'down' Commands and Refactor 'destroy'
- **relay-event-handling_20260307124000Z** — Relay Event Handling for Issues and PRs with Multi-Project Routing
- **init-ssh-and-auth_20260307125500Z** — SSH Key Auto-Registration and Randomized Admin Password
- **worktree-pool_20260307125000Z** — Worktree Pool Management
- **implement-command_20260307125001Z** — 'kf implement' Command — Spawn Developer Agent
- **review-cycle-relay_20260307125002Z** — Developer-Reviewer Relay Cycle
- **merge-cleanup_20260307125003Z** — PR Merge, Worktree Cleanup, and Agent Teardown
- **fix-add-remote-url_20260307130000Z** — Fix 'kf add' to Accept Remote URLs

## Pending Tracks

- **refactor-clean-arch_20260307140000Z** — Restructure Packages into Clean Architecture Layout
- **refactor-domain-ports_20260307140001Z** — Extract Domain Types, Port Interfaces, and Service Layer
- **test-coverage-alignment_20260307140002Z** — Test Coverage Alignment with Style Guide
- **research-cc-quota-monitoring_20260307150000Z** — Research: Claude Code Quota Monitoring and Graceful Degradation
- **research-graceful-shutdown-recovery_20260307150001Z** — Research: Graceful Agent Shutdown, Session Persistence, and Recovery
- **impl-quota-tracker_20260307160000Z** — CC Stream-JSON Parser and Centralized Quota Tracker
- **impl-quota-aware-agents_20260307160001Z** — Quota-Aware Agent Management and Cost Reporting
- **impl-graceful-shutdown-recovery_20260307160002Z** — Graceful Agent Shutdown and Auto-Recovery on Restart
- **impl-webui-server_20260308140000Z** — Web UI Server with Real-Time Agent Monitoring
- **impl-webui-integration_20260308140001Z** — Web UI CLI Integration and Gitea Links
- **impl-lock-service_20260308150000Z** — HTTP-Based Scoped Lock Service in Relay Server
- **impl-conductor-lock-migration_20260308150001Z** — Migrate Conductor Skills to Use Kiloforge Lock API
- **impl-unified-server_20260308160000Z** — Unified Server with Reverse Proxy to Gitea
- **impl-gitea-issue-api_20260308170000Z** — Extend Gitea Client with Issue, Label, and Project Board APIs
- **impl-track-board-sync_20260308170001Z** — Track-to-Gitea Board Sync Service
- **impl-board-webhook-sync_20260308170002Z** — Webhook-Driven Board State Synchronization
- **board-agent-lifecycle_20260308180000Z** — Board-Driven Agent Lifecycle Control
- **monorepo-restructure_20260308180001Z** — Restructure as Monorepo with Backend/Frontend Split
- **react-dashboard_20260308180002Z** — React Dashboard for Real-Time Agent Monitoring
- **live-status-badges_20260308190000Z** — Universal Live Agent Status Badges
- **relay-daemon_20260308200000Z** — Relay Server Daemon Mode
- **openapi-codegen_20260308200001Z** — OpenAPI Code Generation for Relay API
- **schema-first-guidance_20260308200002Z** — Schema-First API Development Guidance and Standards
- **fix-route-conflict_20260308200100Z** — Fix Route Pattern Conflict Panic on Startup
- **testing-audit-e2e_20260308201000Z** — Testing Audit — Integration, E2E, and Smoke Tests
- **remove-password-persistence_20260308210000Z** — Remove Password from Config Persistence
- **build-embed-guidance_20260308210001Z** — Build and Embed Pattern Guidance
- **readme-prerequisites_20260308210002Z** — Update README Prerequisites to Docker + Claude Code
- **add-local-ssh-identity_20260308220000Z** — Add Command SSH Identity and Repo Name Improvements
- **project-scoped-dashboard_20260308220001Z** — Project-Scoped Dashboard Tracks
- **origin-sync-command_20260308220002Z** — Origin Sync Command
- **prereq-check-init_20260308230000Z** — Prerequisite Check During Init
- **skill-install-update_20260308231000Z** — Conductor Skill Installation and Auto-Update
- **research-otel-task-tracing_20260308233000Z** — Research: OpenTelemetry for Task-Level Tracing and Token Metrics
- **impl-otel-task-tracing_20260308233001Z** — OpenTelemetry Task-Level Tracing and Token Metrics
- **fix-init-build-bugs_20260308234000Z** — Fix Init Ctrl+C, Build Failure Propagation, and VCS Stamping
- **research-native-track-board_20260308235000Z** — Research: Native Track Board in Kiloforge Dashboard
- **impl-native-track-board_20260308235001Z** — Native Track Board with Dashboard Kanban and Agent Lifecycle
- **fix-init-password-display_20260308235500Z** — Fix Init Password Display
- **rebrand-kiloforge_20260309055250Z** — Rebrand crelay to Kiloforge (CLI: kf)
- **track-lifecycle-tracing_20260309062329Z** — Track Lifecycle Tracing with OTel
- **kf-skills-source_20260309063859Z** — Kiloforge-Branded Skill Source Artifacts
- **rebrand-historical-records_20260309063900Z** — Rebrand Historical Conductor Records
- **rename-relay-orchestrator_20260309075537Z** — Rename Relay Server to Orchestrator
- **fix-password-display-v3_20260309083826Z** — Fix Init Password Display (Root Cause)
- **project-manage-api_20260309084650Z** — Project Add/Remove REST API
- **project-manage-ui_20260309084651Z** — Project Add/Remove Dashboard UI
- **sse-event-bus_20260309091500Z** — SSE Event Bus Infrastructure
- **sse-entity-subscriptions_20260309091501Z** — SSE Entity Subscriptions
- **ssh-key-selection_20260309100000Z** — Interactive SSH Key Selection for Add Command
- **quota-reframe-be_20260309103000Z** — Reframe Quota System — Tokens and Rate Limits (Backend)
- **quota-reframe-fe_20260309103001Z** — Reframe Quota System — Tokens and Rate Limits (Frontend)
- **model-selection_20260309110000Z** — Configurable Model Selection with Opus Default
- **agent-completion-callback_20260309112000Z** — Agent Completion Callback and Dry-Run Mode
- **licensing_20260309113000Z** — Add Apache 2.0 License and Upstream Attribution
- **fix-buildvcs-worktree_20260309114000Z** — Fix VCS Stamping in Git Worktrees
- **fix-project-mgr-wiring_20260309120000Z** — Fix Project Manager Wiring in REST Server
- **ssh-key-selection-ui_20260309120001Z** — SSH Key Selection in Project Add UI
- **gitea-proxy-authn_20260309123000Z** — Passwordless Gitea Login via Reverse Proxy Authentication
- **dashboard-root-routing-be_20260309130000Z** — Dashboard Root Routing and Kiloforge Rebrand Defaults (Backend)
- **dashboard-root-routing-fe_20260309130001Z** — Dashboard Root Routing (Frontend)
- **tracing-default-on-be_20260309133000Z** — Enable Tracing by Default with Config API (Backend)
- **tracing-default-on-fe_20260309133001Z** — Tracing Toggle in Dashboard UI (Frontend)
- **sqlite-storage-core_20260309140000Z** — SQLite Storage Layer — Core Schema and Migration
- **origin-sync-api_20260309143000Z** — Origin Push/Pull REST API with Remote Branch Targeting
- **origin-sync-ui_20260309143001Z** — Origin Push/Pull Dashboard UI
- **interactive-agent-be_20260309150000Z** — Interactive Agent Sessions via WebSocket (Backend)
- **interactive-agent-fe_20260309150001Z** — Interactive Agent Terminal in Dashboard (Frontend)
- **release-process_20260309153000Z** — Release Process with GoReleaser, GitHub Actions, and Homebrew
- **dashboard-track-gen_20260309160000Z** — Dashboard-Driven Track Generation with Interactive Agent
- **fix-spa-and-init-output_20260309163000Z** — Fix SPA Asset MIME Type, Init Output URLs, and Password Display
- **skill-preflight-check_20260309170000Z** — Pre-flight Skill Validation Before Agent Spawning
- **fix-test-ci-pr-tracking_20260309173000Z** — Fix CI Test Failure — PR Tracking Store References
- **admin-operations-ui_20260309173001Z** — Admin Operations UI — Bulk Archive, Compact, and Report from Dashboard
- **skill-ref-migration_20260309173002Z** — Migrate All Skill References from conductor-* to kf-*

## Getting Started

Run `/conductor-track-generator` to generate tracks from a prompt, or `/conductor-new-track` to create a single track manually.
