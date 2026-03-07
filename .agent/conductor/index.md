# Conductor - crelay

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
- **add-project-command_20260307122000Z** — Implement 'crelay add' Command for Project Registration
- **down-destroy-commands_20260307123000Z** — Add 'up', 'down' Commands and Refactor 'destroy'
- **relay-event-handling_20260307124000Z** — Relay Event Handling for Issues and PRs with Multi-Project Routing
- **init-ssh-and-auth_20260307125500Z** — SSH Key Auto-Registration and Randomized Admin Password
- **worktree-pool_20260307125000Z** — Worktree Pool Management
- **implement-command_20260307125001Z** — 'crelay implement' Command — Spawn Developer Agent
- **review-cycle-relay_20260307125002Z** — Developer-Reviewer Relay Cycle
- **merge-cleanup_20260307125003Z** — PR Merge, Worktree Cleanup, and Agent Teardown
- **fix-add-remote-url_20260307130000Z** — Fix 'crelay add' to Accept Remote URLs

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
- **impl-conductor-lock-migration_20260308150001Z** — Migrate Conductor Skills to Use crelay Lock API
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

## Getting Started

Run `/conductor-track-generator` to generate tracks from a prompt, or `/conductor-new-track` to create a single track manually.
