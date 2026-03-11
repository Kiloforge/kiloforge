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

Install skills globally so they are available across all projects on your machine.

**Unix (macOS / Linux):**

```bash
# Claude Code (also works for OpenCode and Amp via fallback)
SKILLS_DIR=~/.claude/skills \
  && rm -rf "$SKILLS_DIR"/kf-* \
  && git clone --depth 1 https://github.com/benbaldavis/kiloforge-skills.git /tmp/kf-skills \
  && cp -r /tmp/kf-skills/kf-* "$SKILLS_DIR/" \
  && rm -rf /tmp/kf-skills

# Codex
SKILLS_DIR=~/.agents/skills \
  && rm -rf "$SKILLS_DIR"/kf-* \
  && git clone --depth 1 https://github.com/benbaldavis/kiloforge-skills.git /tmp/kf-skills \
  && cp -r /tmp/kf-skills/kf-* "$SKILLS_DIR/" \
  && rm -rf /tmp/kf-skills

# Antigravity
SKILLS_DIR=~/.gemini/antigravity/skills \
  && rm -rf "$SKILLS_DIR"/kf-* \
  && git clone --depth 1 https://github.com/benbaldavis/kiloforge-skills.git /tmp/kf-skills \
  && cp -r /tmp/kf-skills/kf-* "$SKILLS_DIR/" \
  && rm -rf /tmp/kf-skills
```

**Windows (PowerShell):**

```powershell
# Claude Code (also works for OpenCode and Amp via fallback)
$d = "$HOME\.claude\skills"
Remove-Item "$d\kf-*" -Recurse -Force -ErrorAction SilentlyContinue
git clone --depth 1 https://github.com/benbaldavis/kiloforge-skills.git $env:TEMP\kf-skills
Copy-Item "$env:TEMP\kf-skills\kf-*" "$d\" -Recurse
Remove-Item "$env:TEMP\kf-skills" -Recurse -Force

# Codex
$d = "$HOME\.agents\skills"
Remove-Item "$d\kf-*" -Recurse -Force -ErrorAction SilentlyContinue
git clone --depth 1 https://github.com/benbaldavis/kiloforge-skills.git $env:TEMP\kf-skills
Copy-Item "$env:TEMP\kf-skills\kf-*" "$d\" -Recurse
Remove-Item "$env:TEMP\kf-skills" -Recurse -Force

# Antigravity
$d = "$HOME\.gemini\antigravity\skills"
Remove-Item "$d\kf-*" -Recurse -Force -ErrorAction SilentlyContinue
git clone --depth 1 https://github.com/benbaldavis/kiloforge-skills.git $env:TEMP\kf-skills
Copy-Item "$env:TEMP\kf-skills\kf-*" "$d\" -Recurse
Remove-Item "$env:TEMP\kf-skills" -Recurse -Force
```

The `rm -rf "$SKILLS_DIR"/kf-*` cleanup step removes stale skill directories before copying, ensuring renamed or deleted skills don't linger.

### Project-Scoped Installation

Install skills within a repository so all team members share the same skill versions. Project-scoped skills take precedence over global skills when both exist.

```bash
# Claude Code
SKILLS_DIR=.claude/skills \
  && rm -rf "$SKILLS_DIR"/kf-* \
  && git clone --depth 1 https://github.com/benbaldavis/kiloforge-skills.git /tmp/kf-skills \
  && cp -r /tmp/kf-skills/kf-* "$SKILLS_DIR/" \
  && rm -rf /tmp/kf-skills

# OpenCode
SKILLS_DIR=.opencode/skills \
  && rm -rf "$SKILLS_DIR"/kf-* \
  && git clone --depth 1 https://github.com/benbaldavis/kiloforge-skills.git /tmp/kf-skills \
  && cp -r /tmp/kf-skills/kf-* "$SKILLS_DIR/" \
  && rm -rf /tmp/kf-skills

# Amp / Codex (shared path)
SKILLS_DIR=.agents/skills \
  && rm -rf "$SKILLS_DIR"/kf-* \
  && git clone --depth 1 https://github.com/benbaldavis/kiloforge-skills.git /tmp/kf-skills \
  && cp -r /tmp/kf-skills/kf-* "$SKILLS_DIR/" \
  && rm -rf /tmp/kf-skills

# Antigravity
SKILLS_DIR=.agent/skills \
  && rm -rf "$SKILLS_DIR"/kf-* \
  && git clone --depth 1 https://github.com/benbaldavis/kiloforge-skills.git /tmp/kf-skills \
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

### 3. Dispatch (`/kf-dispatch`)

The dispatcher analyzes project state and produces worker assignments for idle developer agents. It:

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
| `kf-implement` | Execute tasks from a track plan (single-branch workflow) |
| `kf-dispatch` | Analyze state and assign work to idle agents |

### Management

| Skill | Purpose |
|-------|---------|
| `kf-manage` | Track lifecycle: archive, restore, delete, rename, cleanup |
| `kf-bulk-archive` | Archive all completed tracks at once |
| `kf-compact-archive` | Remove archived track directories while preserving git history |
| `kf-revert` | Git-aware undo by logical work unit (track, phase, or task) |
| `kf-new-track` | Create a single new track with spec and plan |
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
| `kf-interactive` | General-purpose interactive agent session |

### Infrastructure

| Skill | Purpose |
|-------|---------|
| `kf-validate` | Check Kiloforge artifacts for completeness and consistency |
| `kf-repair` | Diagnose and fix track registry, dependency graph, and data integrity issues |
| `kf-report` | Generate timeline, velocity, SLOC, and cost estimate reports |
| `kf-data-guardian` | Corruption detection heuristics (internal, not user-invocable) |
| `kf-status` | Display project status, active tracks, and next actions |

## Skill Validation

Before spawning an agent, the Cortex validates that all required skills are installed and their checksums match the expected versions. If skills are missing or outdated:

1. The Cortex reports which skills need updating
2. The Kiloforger runs `kf skills update` to install the latest versions
3. The spawn request is retried

This ensures all agents operate with consistent, known-good skill definitions.
