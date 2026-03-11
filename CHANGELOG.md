# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0-alpha.1]

### Added

- **CLI commands:** `kf init`, `up`, `down`, `destroy`, `add`, `projects`, `status`, `implement`, `agents`, `logs`, `stop`, `attach`, `escalated`, `cost`, `dashboard`, `sync`, `push`, `serve`, `skills`, `version`
- **Local Gitea orchestration:** Docker-managed Gitea instance for multi-project coordination
- **Agent lifecycle:** Spawn, monitor, and stop Claude Code developer agents
- **Worktree pool:** Automatic git worktree management for parallel agent work
- **Developer-reviewer relay:** Automated PR creation and review cycles
- **Merge cleanup:** PR merge, worktree cleanup, and agent teardown
- **Real-time dashboard:** React-based web UI with agent monitoring, trace timeline, and track board
- **SSE event bus:** Server-sent events for live status updates
- **OpenTelemetry tracing:** Task-level tracing with span persistence in SQLite
- **SQLite storage:** Persistent storage layer replacing flat-file JSON stores
- **Quota tracking:** Token usage monitoring and rate limit enforcement
- **Model selection:** Configurable model selection with Opus default
- **Origin sync:** Push/pull to upstream remotes with branch targeting
- **SSH key management:** Interactive SSH key selection for project registration
- **Reverse proxy auth:** Passwordless Gitea login via reverse proxy headers
- **OpenAPI codegen:** Schema-first API development with oapi-codegen
- **Completion callbacks:** Agent completion notifications with dry-run mode
- **Configurable tracing:** Enable/disable tracing via config API and dashboard toggle
- **Apache 2.0 license**

### Infrastructure

- GoReleaser for cross-platform binary distribution (darwin/linux/windows × amd64/arm64)
- GitHub Actions CI/CD with tag-triggered releases
- Install script for macOS/Linux (`install.sh`)
