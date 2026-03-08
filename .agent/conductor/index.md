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
- **agent-list-monitoring-ui_20260309180000Z** — Agent List with Role/Track Links and Monitoring View
- **agent-random-names_20260309183000Z** — Random Human-Friendly Agent Names
- **goose-migrations_20260309184000Z** — Migrate to Goose for Database Schema Migrations
- **agent-permissions-consent_20260309190000Z** — Agent Permissions Flag and User Consent
- **tanstack-query-migration_20260309193000Z** — Migrate Frontend Data Fetching to TanStack Query
- **claude-auth-check_20260309200000Z** — Claude CLI Authentication Check Before Agent Spawning
- **guided-tour-be_20260309203000Z** — Guided Tour State API and Demo Seed Data (Backend)
- **guided-tour-fe_20260309203001Z** — Guided Tour Overlay with Simulated Onboarding Flow (Frontend)
- **fix-stale-frontend-build_20260309210000Z** — Fix Stale Frontend Build — Rebuild dist with Correct Base Path
- **fix-add-empty-repo-cleanup_20260309213000Z** — Fix kf add for Empty Repos and Add Rollback on Failure
- **track-gen-auto-approve_20260309220000Z** — Track Generator Auto-Approve for Non-Code Tracks
- **fix-tour-api-mismatch_20260309223000Z** — Fix Tour API Request Body Mismatch
- **error-toast-notifications_20260309223001Z** — Error Toast Notifications in Dashboard
- **fix-gitea-push-auth_20260309224000Z** — Fix Gitea Push Authentication — Embed Credentials in Remote URL
- **frontend-test-infra_20260309225000Z** — Frontend Test Infrastructure and Makefile Integration
- **fix-dashboard-interactive-agent_20260309230000Z** — Fix Interactive Agent Wiring in Dashboard Command
- **fix-project-sse-subscription_20260309230001Z** — Wire Project SSE Events in Dashboard Frontend
- **tracing-always-on-be_20260309231000Z** — Remove Optional Tracing — Always-On (Backend)
- **tracing-always-on-fe_20260309231001Z** — Remove Tracing Toggle UI (Frontend)
- **fix-serve-interactive-agent_20260309232000Z** — Fix Interactive Agent Wiring in Serve Command (kf up)
- **fix-skills-412-prompt_20260309233000Z** — Handle 412 Skills Missing with Install Prompt Instead of Error
- **embedded-skills-default_20260309234000Z** — Embedded Skills as Default — Remove Repo Dependency
- **tour-ux-improvements_20260309235000Z** — Tour UX Improvements — State Transitions, Demo Data, and Example URL
- **fix-track-gen-project-init_20260309235500Z** — Auto-Initialize Conductor Artifacts Before Track Generation
- **gitignore-frontend-dist_20260310000000Z** — Stop Committing Frontend Dist — Build Before Backend Instead
- **fix-nested-claude-env_20260310001000Z** — Fix Nested Claude Session Detection — Unset CLAUDECODE Env Var
- **fix-spawner-verbose-flag_20260310003000Z** — Fix Agent Spawner — Add --verbose Flag for stream-json
- **setup-prereq-be_20260310004000Z** — Setup Prerequisite Check Before Agent Spawning (Backend)
- **setup-prereq-fe_20260310004001Z** — Setup Prerequisite Check — Dashboard UI (Frontend)
- **cli-sqlite-migration_20260310005000Z** — Migrate CLI Commands from JSON Files to SQLite
- **refactor-cli-thin-adapters_20260310010000Z** — Refactor CLI Commands to Thin Adapters with Shared Service Layer
- **fix-board-sync-ux-be_20260310012000Z** — Fix Board Sync and SSE Events (Backend)
- **fix-board-sync-ux-fe_20260310012001Z** — Fix Board UX — SSE Handler, Sync Button, Empty Columns (Frontend)
- **fix-project-delete-refresh_20260310013000Z** — Fix Project Delete — Refresh List and Close Modal
- **agent-display-ttl-be_20260310014000Z** — Agent Display TTL and History API (Backend)
- **agent-display-ttl-fe_20260310014001Z** — Agent Display TTL and History Page (Frontend)
- **setup-gate-be_20260310234710Z** — Gate All Agent Endpoints Behind Setup Check (Backend)
- **setup-gate-fe_20260310234711Z** — Disable Agent Actions Until Setup Complete (Frontend)

## Getting Started

Run `/kf-track-generator` to generate tracks from a prompt, or `/kf-new-track` to create a single track manually.
