# Kiloforge Documentation

**1,000x Productivity.** Forge code at the speed of thought with AI agent swarms.

## Contents

### [Why I Built This](why-i-built-this.md)

The story behind Kiloforge — from first encountering agentic coding tools to building an orchestration platform that coordinates dozens of AI agents in parallel. Covers the origin, the "human mutex" moment, the philosophy of trusting agent processes, and what 1,000x productivity actually looks like.

### [Getting Started](getting-started.md)

Install Kiloforge, start the Cortex, register a project, and spawn the first agent. Covers the full onboarding flow: setup, track creation with the architect, the Command Deck walkthrough, and monitoring the Swarm.

### [Architecture Overview](architecture.md)

How the pieces fit together — CLI, Cortex control plane, and Command Deck. Covers the system diagram, communication protocols, codebase structure, key abstractions (tracks, agents, projects), skills system, notification bus, Swarm coordination, and data storage.

### [Skills Guide](skills.md)

The full skills catalog and how they work. Covers the architect → developer → reviewer pipeline, all skill categories (Core Workflow, Management, Review & Advisory, Setup & Onboarding, Infrastructure), installation, and skill validation.

### [Agents & Swarms](agents-and-swarms.md)

Deep dive into agent roles, lifecycle states, and Swarm coordination. Covers agent statuses and transitions, the suspension mechanism (grace periods, worker protection), graceful shutdown, notifications, Swarm capacity and worktree pooling, the queue service, merge serialization, and dispatch.

## Additional Resources

- [Main README](../README.md) — Complete command reference and quick start
- [OpenAPI Schema](../backend/api/openapi.yaml) — REST API specification
- [AsyncAPI Schema](../backend/api/asyncapi.yaml) — SSE and webhook event documentation
- [LICENSE](../LICENSE) — Apache License 2.0
