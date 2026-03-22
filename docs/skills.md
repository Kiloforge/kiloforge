# Skills Guide

Skills are structured slash commands that agents use to perform specific workflows. Each skill is a markdown file that defines a role, a set of rules, and a step-by-step workflow. When an agent invokes a skill (e.g., `/kf-developer`), the skill's full prompt is loaded into the agent's context, guiding it through the workflow.

## How Skills Work

Skills are installed as markdown files in `~/.claude/skills/`. The Cortex validates that required skills are available before spawning any agent. The Kiloforger can manage skills via the CLI:

```bash
kf skills list     # List installed skills with versions
kf skills update   # Update to the latest skill release
```

Skills are distributed as part of the Kiloforge release. They are embedded in the binary and installed to the Kiloforger's machine on first run or update.

## Supported Agent Tools

Kiloforge skills use the `SKILL.md` format supported by multiple AI coding agents. Each tool has its own directory convention for global and project-scoped skill installation.

### Skill Directory Paths

| Agent Tool | Global Path | Project-Scoped Path |
|------------|-------------|---------------------|
| **Claude Code** | `~/.claude/skills/` | `.claude/skills/` |
| **OpenCode** | `~/.config/opencode/skills/` | `.opencode/skills/` |
| **Amp** (Sourcegraph) | `~/.config/amp/skills/` | `.agents/skills/` |
| **Codex** (OpenAI) | `~/.agents/skills/` | `.agents/skills/` |
| **Antigravity** (Google) | `~/.gemini/antigravity/skills/` | `.agent/skills/` |

> **Note:** OpenCode and Amp fall back to reading `~/.claude/skills/`, so installing skills globally for Claude Code also makes them available to those tools.

### Global Installation

Install skills globally so they are available across all projects on your machine. The install scripts handle skills, `~/.kf/bin/` CLI tools, and `~/.kf/.venv/` Python venv setup automatically.

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/Kiloforge/kiloforge-skills/main/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/Kiloforge/kiloforge-skills/main/install.ps1 | iex
```

The install scripts detect all supported agent tools on your machine and install skills to each tool's global directory. They also set up `~/.kf/bin/` CLI tools and `~/.kf/.venv/` Python venv.

**Updating:** Run `/kf-update` inside Claude Code, or re-run the install script above.

### Project-Scoped Installation

Install skills within a repository so all team members share the same skill versions. Project-scoped skills take precedence over global skills when both exist. For project-scoped installs, use a git clone approach with the correct skills directory for your agent tool:

```bash
# Claude Code
SKILLS_DIR=.claude/skills \
  && rm -rf "$SKILLS_DIR"/kf-* \
  && git clone --depth 1 https://github.com/Kiloforge/kiloforge-skills.git /tmp/kf-skills \
  && cp -r /tmp/kf-skills/kf-* "$SKILLS_DIR/" \
  && rm -rf /tmp/kf-skills

# OpenCode
SKILLS_DIR=.opencode/skills \
  && rm -rf "$SKILLS_DIR"/kf-* \
  && git clone --depth 1 https://github.com/Kiloforge/kiloforge-skills.git /tmp/kf-skills \
  && cp -r /tmp/kf-skills/kf-* "$SKILLS_DIR/" \
  && rm -rf /tmp/kf-skills

# Amp / Codex (shared path)
SKILLS_DIR=.agents/skills \
  && rm -rf "$SKILLS_DIR"/kf-* \
  && git clone --depth 1 https://github.com/Kiloforge/kiloforge-skills.git /tmp/kf-skills \
  && cp -r /tmp/kf-skills/kf-* "$SKILLS_DIR/" \
  && rm -rf /tmp/kf-skills

# Antigravity
SKILLS_DIR=.agent/skills \
  && rm -rf "$SKILLS_DIR"/kf-* \
  && git clone --depth 1 https://github.com/Kiloforge/kiloforge-skills.git /tmp/kf-skills \
  && cp -r /tmp/kf-skills/kf-* "$SKILLS_DIR/" \
  && rm -rf /tmp/kf-skills
```

Add the chosen skills directory to your `.gitignore` if you prefer not to commit skills into the repository.

## The Core Pipeline

The primary development workflow follows the **architect → developer** pipeline:

```
Kiloforger's intent
       │
       ▼
  /kf-architect          ← Researches codebase, creates tracks
       │
       ▼
  /kf-developer          ← Claims track, implements in worktree, merges
       │
       ▼
  Merged to main
```

### 1. Architect (`/kf-architect`)

The architect is the entry point for new work. Given a feature request or description, it:

- Researches the codebase to understand the existing architecture
- Designs the implementation approach
- Creates one or more tracks with full specs (acceptance criteria, context, codebase analysis)
- Creates phased implementation plans with granular tasks
- Splits large work into multiple tracks, including frontend/backend splits
- Sets up dependency ordering between tracks
- Identifies potential conflict pairs with other active tracks
- Merges track artifacts to main so developer agents can claim them

**Usage:**
```bash
claude -p "/kf-architect Add user authentication with OAuth2 support"
```

### 2. Developer (`/kf-developer`)

The developer is the core implementation agent. Given a track ID, it:

- Validates the track is available and all dependencies are met
- Creates an implementation branch from main
- Implements each task in the plan sequentially
- Follows TDD workflow if configured (red → green → refactor)
- Commits after each task using conventional commits
- Runs verification commands (tests, build, lint) at phase completion
- Acquires the merge lock, rebases onto main, verifies, and fast-forward merges
- Cleans up the implementation branch

Developers run autonomously in pooled worktrees. They never auto-suspend — they continue working even when the Kiloforger isn't watching.

**Usage:**
```bash
kf implement <track-id>
# or directly:
claude -p "/kf-developer <track-id>"
```

### 3. Status & Dispatch (`/kf-status`)

The status command displays project progress and includes dispatch recommendations for idle developer agents. It:

- Shows overall progress, active tracks, and blocked dependencies
- Scans the track registry for available tracks
- Checks dependency satisfaction
- Produces prescriptive assignments for idle worktrees
- Considers conflict pairs to avoid parallel work on conflicting tracks

## Skill Catalog

### Core Workflow

| Skill | Purpose |
|-------|---------|
| `kf-architect` | Research codebase, design tracks with specs and plans |
| `kf-developer` | Claim and implement a track in a worktree |
| `kf-status` | Display project status and dispatch recommendations |
| `kf-bin` | Reference hub for `~/.kf/bin/` CLI tools |

### Management

| Skill | Purpose |
|-------|---------|
| `kf-manage` | Track lifecycle: archive, compact, restore, delete, rename, cleanup |
| `kf-revert` | Git-aware undo by logical work unit (track, phase, or task) |
| `kf-conflict-resolver` | Resolve git merge conflicts during push/pull sync |

### Review & Advisory

| Skill | Purpose |
|-------|---------|
| `kf-advisor-product` | Product strategy: design, branding, feature prioritization, competitive analysis |
| `kf-advisor-reliability` | Audit testing, linting, type safety, CI gates, and dependency security |

Advisory skills are interactive — they produce reports and recommendations for the Kiloforger to review and act on. They auto-suspend after a grace period when the Kiloforger disconnects.

### Setup & Onboarding

| Skill | Purpose |
|-------|---------|
| `kf-setup` | Initialize project with Kiloforge artifacts (product, tech stack, workflow, styles) |
| `kf-getting-started` | Interactive bootstrapper for new projects |

### Infrastructure

| Skill | Purpose |
|-------|---------|
| `kf-validate` | Check Kiloforge artifacts for completeness and consistency |
| `kf-repair` | Diagnose and fix track registry, dependency graph, and data integrity issues |
| `kf-report` | Generate timeline, velocity, SLOC, and cost estimate reports |

## Skill Validation

Before spawning an agent, the Cortex validates that all required skills are installed and their checksums match the expected versions. If skills are missing or outdated:

1. The Cortex reports which skills need updating
2. The Kiloforger runs `kf skills update` to install the latest versions
3. The spawn request is retried

This ensures all agents operate with consistent, known-good skill definitions.
