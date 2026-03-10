---
name: kf-getting-started
description: Interactive project bootstrapper with platform-aware defaults.
  Guides users through project creation decisions, generates a blueprint,
  scaffolds the project, and invokes kf-setup with pre-populated answers.
metadata:
  argument-hint: "[--resume]"
---

# Getting Started

Bootstrap a new project from scratch. This skill asks high-level questions a non-technical founder or early-stage developer can answer, translates those answers into concrete technical decisions with strong opinionated defaults, scaffolds the project, and hands off to `kf-setup`.

## Use this skill when

- Starting a brand-new (greenfield) project from an empty or near-empty directory
- The user doesn't yet know their tech stack and needs guided recommendations
- You want to scaffold a project structure before running `/kf-setup`

## Do not use this skill when

- The project already has code and just needs Kiloforge initialized (use `/kf-setup`)
- The user already knows their tech stack and project structure
- The directory contains an existing application (brownfield — use `/kf-setup` directly)

---

## Pre-flight Checks

1. **Check for existing state** — look for `.agent/kf/getting-started-state.json`:
   - If found with `"status": "in_progress"` and `--resume` was passed (or offer resume): load state and skip completed sections
   - If found with `"status": "complete"`: inform the user setup already ran, suggest `/kf-setup` if Kiloforge artifacts are missing

2. **Check if already initialized** — if `.agent/kf/product.md` exists:
   - Warn that Kiloforge is already initialized
   - Ask if they want to continue (will overwrite) or abort

3. **Verify greenfield** — check the project directory:
   - If `go.mod`, `package.json`, `requirements.txt`, `Cargo.toml`, or `src/` exist: warn this looks like an existing project and suggest `/kf-setup` instead
   - If user insists, continue anyway

4. **Initialize state file** — create `.agent/kf/getting-started-state.json`:
   ```json
   {
     "status": "in_progress",
     "current_section": "identity",
     "current_question": 1,
     "completed_sections": [],
     "answers": {},
     "files_created": [],
     "started_at": "ISO_TIMESTAMP",
     "last_updated": "ISO_TIMESTAMP"
   }
   ```

---

## Interactive Q&A Protocol

**CRITICAL RULES:**

- Ask **ONE question per turn** — wait for the user's response before proceeding
- Offer 2-4 suggested answers plus a "Type your own" option
- Update `getting-started-state.json` after each answered question
- Use the user's previous answers to inform smart defaults for later questions
- If the user types "skip", use the default and move on

---

## Section 1: Project Identity (4 questions)

**Q1: Project Name**

```
What is your project name?

Suggested:
1. [Infer from current directory name]
2. Type your own
```

Save as `answers.project_name`.

**Q2: Project Description**

```
Describe your project in one sentence — what does it do?

Suggested:
1. A web application that [does X]
2. A CLI tool for [doing Y]
3. A mobile app that [does Z]
4. Type your own
```

Save as `answers.description`.

**Q3: Problem Statement**

```
What problem does this project solve? Who feels this pain today?

Suggested:
1. Users struggle to [pain point] because [reason]
2. There's no good way to [need] without [workaround]
3. Type your own
```

Save as `answers.problem`.

**Q4: Target Users**

```
Who are the primary users?

Suggested:
1. Developers building [X]
2. End users who need [Y]
3. Internal teams managing [Z]
4. Type your own
```

Save as `answers.target_users`.

After Q4, mark section complete: add `"identity"` to `completed_sections`, set `current_section` to `"platform"`.

---

## Section 2: Platform & Target Discovery (3-4 questions)

**Q1: Platform Type**

```
What platform are you building for?

1. Web application (browser-based)
2. Mobile app (phone/tablet)
3. Desktop application (Windows/Mac/Linux)
4. CLI tool (terminal/command-line)
5. Server / API only (no UI)
6. Type your own
```

Save as `answers.platform`. This drives conditional branching below.

**Q2: Target OS (conditional)**

Only ask if platform is `mobile` or `desktop`:

_If mobile:_
```
Which mobile platforms?

1. Both Android and iOS (cross-platform)
2. Android only
3. iOS only
```

_If desktop:_
```
Which desktop platforms?

1. All (Windows, Mac, Linux)
2. Mac only
3. Windows and Linux
4. Type your own
```

Save as `answers.target_os`. Skip if platform is `web`, `cli`, or `server`.

**Q3: Form Factor / Scope (conditional)**

_If web:_
```
What type of web application?

1. Full-stack (frontend + backend + database)
2. Frontend SPA with external API
3. Server-rendered (SSR/SSG)
4. Static site / landing page
```

_If server:_
```
What type of server application?

1. REST API
2. GraphQL API
3. gRPC service
4. Background worker / queue processor
```

Save as `answers.form_factor`.

**Q4: Scale Expectations**

```
What scale do you expect initially?

1. Personal / hobby project
2. Small team (< 10 users)
3. Production (100+ users)
4. High-scale (10K+ users)
```

Save as `answers.scale`.

After this section, mark complete: add `"platform"` to `completed_sections`, set `current_section` to `"tech_stack"`.

---

## Section 3: Tech Stack Recommendations (3-5 questions)

Present **smart defaults based on platform choice**. Use the platform-to-stack mapping table (see below) to pre-select recommendations.

**Q1: Primary Language**

```
Based on your [platform] project, I recommend:

→ [Default language from mapping table]

Alternatives:
1. Accept recommendation: [default]
2. [Alternative 1]
3. [Alternative 2]
4. Type your own
```

Save as `answers.language`.

**Q2: Framework**

```
For [language] on [platform], I recommend:

→ [Default framework from mapping table]

Alternatives:
1. Accept recommendation: [default]
2. [Alternative 1]
3. [Alternative 2]
4. None / standard library only
5. Type your own
```

Save as `answers.framework`.

**Q3: Database (if applicable)**

Skip if platform is `cli` and user hasn't indicated data persistence needs.

```
What database fits your needs?

→ [Default from mapping: PostgreSQL for production, SQLite for tools/small]

1. Accept recommendation: [default]
2. PostgreSQL
3. SQLite
4. MongoDB
5. None / stateless
6. Type your own
```

Save as `answers.database`.

**Q4: Package Manager / Build System (auto-resolved)**

Auto-resolve based on language choice — inform the user but don't ask unless ambiguous:

- Go → `go mod` + `Makefile`
- TypeScript/JavaScript → ask: `npm`, `pnpm`, `bun`, or `yarn`
- Rust → `cargo`
- Python → ask: `uv`, `pip` + `pyproject.toml`, or `poetry`

Save as `answers.package_manager`.

**Q5: Infrastructure (optional)**

```
Where do you plan to deploy? (Press enter to skip — you can decide later)

1. Docker / self-hosted
2. AWS (Lambda, ECS, etc.)
3. Vercel / Netlify
4. Fly.io / Railway
5. Not decided yet
```

Save as `answers.infrastructure`.

After this section, mark complete: add `"tech_stack"` to `completed_sections`, set `current_section` to `"architecture"`.

---

## Section 4: Architecture Preferences (2-3 questions)

**Q1: Project Structure**

```
What project structure pattern?

→ Recommended for [language]: [default from mapping]

1. Monorepo (frontend + backend in one repo)
2. Single-concern (one app per repo)
3. Accept recommendation
```

Save as `answers.project_structure`.

**Q2: API Style (if applicable)**

Skip if `cli` or no backend component.

```
What API style?

→ Recommended: REST (simple, well-understood)

1. REST
2. GraphQL
3. gRPC
4. Type your own
```

Save as `answers.api_style`.

**Q3: Architecture Pattern**

```
What architecture pattern?

→ Recommended for [scale]: [default]

1. Clean Architecture (ports & adapters) — best for production apps
2. Simple layered (handler → service → repository)
3. Flat / minimal structure — best for small tools and scripts
```

Save as `answers.architecture_pattern`.

After this section, mark complete: add `"architecture"` to `completed_sections`, set `current_section` to `"quality"`.

---

## Section 5: Quality & Workflow Defaults (2-3 questions)

Provide strong defaults that match production-quality standards.

**Q1: Testing Strategy**

```
What testing approach?

→ Recommended: Strict TDD with layered tests

1. Strict TDD — tests before implementation, red-green-refactor (recommended)
2. Moderate — tests encouraged, not blocking
3. Flexible — tests for complex logic only
```

Save as `answers.testing_strategy`.

**Q2: Commit Conventions**

```
What commit style?

→ Recommended: Conventional Commits

1. Conventional Commits (feat:, fix:, refactor:, etc.) (recommended)
2. Descriptive messages, no strict format
3. Type your own
```

Save as `answers.commit_style`.

**Q3: Code Review Policy**

```
What code review policy?

1. Required for all changes
2. Required for non-trivial changes
3. Optional / self-review OK (recommended for solo projects)
```

Save as `answers.code_review`.

After this section, mark complete: add `"quality"` to `completed_sections`, set `current_section` to `"review"`.

---

## Section 6: Blueprint Review (1 question)

Compile all answers into a summary and present for approval.

**Display the blueprint:**

```
================================================================================
                        PROJECT BLUEPRINT
================================================================================

Project:        {project_name}
Description:    {description}
Problem:        {problem}
Users:          {target_users}

Platform:       {platform} ({target_os if applicable})
Form Factor:    {form_factor}
Scale:          {scale}

Language:       {language}
Framework:      {framework}
Database:       {database}
Build System:   {package_manager}
Infrastructure: {infrastructure}

Structure:      {project_structure}
API Style:      {api_style}
Architecture:   {architecture_pattern}

Testing:        {testing_strategy}
Commits:        {commit_style}
Code Review:    {code_review}
================================================================================
```

**Q1: Approve Blueprint**

```
Does this look right?

1. Yes, proceed with scaffolding
2. Let me change something (specify what to update)
3. Start over
```

If the user wants changes, ask which field to update, accept the new value, re-display the blueprint, and ask again. Repeat until approved.

After approval, write the blueprint to `.agent/kf/project-blueprint.yaml`:

```yaml
project:
  name: "{project_name}"
  description: "{description}"
  problem: "{problem}"
  target_users: "{target_users}"

platform:
  type: "{platform}"
  target_os: "{target_os}"
  form_factor: "{form_factor}"
  scale: "{scale}"

tech_stack:
  language: "{language}"
  framework: "{framework}"
  database: "{database}"
  package_manager: "{package_manager}"
  infrastructure: "{infrastructure}"

architecture:
  project_structure: "{project_structure}"
  api_style: "{api_style}"
  pattern: "{architecture_pattern}"

quality:
  testing: "{testing_strategy}"
  commits: "{commit_style}"
  code_review: "{code_review}"

generated_at: "ISO_TIMESTAMP"
```

Mark section complete: add `"review"` to `completed_sections`, set `current_section` to `"scaffold"`.

---

## Section 7: Project Scaffolding

Generate the project directory structure and config files based on the blueprint. Do NOT generate business logic — only scaffolding.

### Scaffolding by Language

**Go:**
```
{project_name}/
├── go.mod                    # module {project_name}
├── Makefile                  # build, test, lint targets
├── .gitignore                # Go defaults
├── README.md                 # project name + description
├── cmd/
│   └── {project_name}/
│       └── main.go           # minimal main with TODO
├── internal/                 # if Clean Architecture
│   ├── core/
│   │   ├── domain/
│   │   ├── port/
│   │   └── service/
│   └── adapter/
└── pkg/                      # if simple layered
```

**TypeScript / JavaScript:**
```
{project_name}/
├── package.json              # name, scripts, dependencies
├── tsconfig.json             # strict mode
├── .gitignore                # node_modules, dist
├── README.md
├── src/
│   └── index.ts              # minimal entry point
└── tests/
    └── .gitkeep
```

_If React frontend:_
```
{project_name}/
├── package.json
├── tsconfig.json
├── vite.config.ts
├── index.html
├── .gitignore
├── README.md
├── src/
│   ├── main.tsx
│   ├── App.tsx
│   └── components/
└── tests/
```

**Rust:**
```
{project_name}/
├── Cargo.toml
├── .gitignore
├── README.md
└── src/
    └── main.rs               # or lib.rs
```

**Python:**
```
{project_name}/
├── pyproject.toml
├── .gitignore
├── README.md
├── src/
│   └── {project_name}/
│       └── __init__.py
└── tests/
    └── __init__.py
```

### Common Files (all languages)

- `.gitignore` — language-appropriate defaults
- `README.md` — project name, description, and getting started placeholder
- `Makefile` — if applicable (Go, C/C++, or user prefers Make)

### Scaffolding Rules

1. Create all directories and files
2. Initialize version control: `git init` (if not already a git repo)
3. Initialize package manager: `go mod init`, `npm init -y`, `cargo init`, etc.
4. Do NOT install dependencies yet — just declare them in config files
5. Do NOT generate any business logic, only structural scaffolding
6. Add each created file to `getting-started-state.json` → `files_created`

After scaffolding, mark complete: add `"scaffold"` to `completed_sections`, set `current_section` to `"handoff"`.

---

## Section 8: Handoff to kf-setup

Auto-invoke `/kf-setup` with pre-populated context from the blueprint. The goal is to skip or fast-track kf-setup's Q&A by providing answers derived from the blueprint.

### Pre-populate kf-setup Answers

Map blueprint fields to kf-setup sections:

| kf-setup Section | Pre-populated From |
|---|---|
| Product Definition | `project.name`, `project.description`, `project.problem`, `project.target_users` |
| Product Guidelines | Defaults based on `quality.*` and `platform.type` |
| Tech Stack | `tech_stack.*` (language, framework, database, infrastructure) |
| Workflow | `quality.testing` → TDD policy, `quality.commits` → commit strategy, `quality.code_review` → review policy |
| Code Style Guides | Auto-select based on `tech_stack.language` |

### Handoff Process

1. Inform the user:
   ```
   Project scaffolded! Now initializing Kiloforge with your blueprint answers.
   I'll run /kf-setup and pre-fill answers from your blueprint — you can still
   adjust anything during setup.
   ```

2. Create `.agent/kf/` directory if it doesn't exist

3. Write pre-populated artifacts directly (same as kf-setup output):
   - `.agent/kf/product.md` — from blueprint project section
   - `.agent/kf/product-guidelines.md` — from blueprint quality + sensible defaults
   - `.agent/kf/tech-stack.md` — from blueprint tech_stack section
   - `.agent/kf/workflow.md` — from blueprint quality section
   - `.agent/kf/code_styleguides/` — auto-selected based on language

4. Create `.agent/kf/tracks.yaml`:
   ```yaml
   tracks: {}
   ```

5. Update `getting-started-state.json`:
   - Set `status` to `"complete"`
   - Update `files_created` with all generated files
   - Set `last_updated`

6. Display completion summary:
   ```
   ================================================================================
                     PROJECT BOOTSTRAPPED SUCCESSFULLY
   ================================================================================

   Project:     {project_name}
   Platform:    {platform}
   Stack:       {language} + {framework}
   Location:    {current directory}

   Scaffolded files:
   {list of files_created}

   Kiloforge artifacts:
   - .agent/kf/product.md
   - .agent/kf/product-guidelines.md
   - .agent/kf/tech-stack.md
   - .agent/kf/workflow.md
   - .agent/kf/code_styleguides/{language}.md
   - .agent/kf/tracks.yaml

   Next steps:
   1. Review generated files and customize as needed
   2. Run /kf-architect to plan your first feature tracks
   3. Run /kf-developer <track-id> to start implementing
   ================================================================================
   ```

---

## State Management

The state file `.agent/kf/getting-started-state.json` tracks progress through all sections:

```json
{
  "status": "in_progress|complete",
  "current_section": "identity|platform|tech_stack|architecture|quality|review|scaffold|handoff",
  "current_question": 1,
  "completed_sections": ["identity", "platform", ...],
  "answers": {
    "project_name": "...",
    "description": "...",
    "problem": "...",
    "target_users": "...",
    "platform": "...",
    "target_os": "...",
    "form_factor": "...",
    "scale": "...",
    "language": "...",
    "framework": "...",
    "database": "...",
    "package_manager": "...",
    "infrastructure": "...",
    "project_structure": "...",
    "api_style": "...",
    "architecture_pattern": "...",
    "testing_strategy": "...",
    "commit_style": "...",
    "code_review": "..."
  },
  "files_created": [],
  "started_at": "ISO_TIMESTAMP",
  "last_updated": "ISO_TIMESTAMP"
}
```

**Update rules:**
- After each answered question: update `answers`, `current_question`, `last_updated`
- After each completed section: add to `completed_sections`, advance `current_section`, reset `current_question` to 1
- After each file creation: append to `files_created`
- On completion: set `status` to `"complete"`

---

## Resume Handling

If `--resume` is passed or state file exists with `"status": "in_progress"`:

1. Load `getting-started-state.json`
2. Display resume summary:
   ```
   Resuming project setup from section: {current_section}, question {current_question}

   Completed sections: {completed_sections}
   Answers so far:
   - Project: {project_name}
   - Platform: {platform}
   ...
   ```
3. Skip completed sections
4. Resume from `current_section` and `current_question`
5. Verify previously created files still exist — if any are missing, offer to regenerate

---

## Error Handling

- **File write fails:** Halt and report the error. Do not update state.
- **User cancels (Ctrl+C or "cancel"):** Save current state for future `--resume`.
- **State file corrupted:** Offer to start fresh or attempt partial recovery from `completed_sections`.
- **Directory not empty warning:** Inform user, list existing files, ask to proceed or abort.
- **kf-setup already ran:** If `.agent/kf/product.md` exists, warn and ask before overwriting.

---

## Platform-to-Stack Mapping Tables

### Web Application

| Component | Default | Alternatives |
|---|---|---|
| Language | TypeScript | JavaScript, Go, Python |
| Frontend | React + Vite | Next.js, Vue, Svelte |
| Backend | Go (net/http) | Node.js (Express), Python (FastAPI) |
| Database | PostgreSQL | SQLite, MongoDB |
| Testing | Vitest + Go test | Jest, Playwright |
| Build | Makefile + Vite | Turbo, nx |
| Structure | Monorepo | Separate repos |
| Architecture | Clean Architecture | Layered |

### Mobile — Cross-Platform

| Component | Default | Alternatives |
|---|---|---|
| Language | TypeScript | Dart |
| Framework | React Native + Expo | Flutter |
| State | Zustand | Redux, MobX |
| Testing | Jest + Detox | Maestro |
| Build | EAS Build | Fastlane |

### Mobile — Native Android

| Component | Default | Alternatives |
|---|---|---|
| Language | Kotlin | Java |
| Framework | Jetpack Compose | XML Views |
| Architecture | MVVM + Clean | MVI |
| Testing | JUnit + Espresso | Robolectric |
| Build | Gradle (Kotlin DSL) | — |

### Mobile — Native iOS

| Component | Default | Alternatives |
|---|---|---|
| Language | Swift | Objective-C |
| Framework | SwiftUI | UIKit |
| Architecture | MVVM + Clean | TCA (Composable Architecture) |
| Testing | XCTest | Quick/Nimble |
| Build | Xcode + SPM | — |

### Desktop — Cross-Platform

| Component | Default | Alternatives |
|---|---|---|
| Language | TypeScript | Rust |
| Framework | Electron | Tauri |
| UI | React | Svelte |
| Testing | Playwright | — |
| Build | electron-builder | tauri-cli |

### CLI Tool

| Component | Default | Alternatives |
|---|---|---|
| Language | Go | Rust, Python |
| Framework | Cobra | clap (Rust), click (Python) |
| Testing | Go test (table-driven) | — |
| Build | Makefile + goreleaser | cargo, setuptools |
| Structure | cmd/ + internal/ | flat |
| Architecture | Flat / minimal | Clean (if complex) |

### Server / API Only

| Component | Default | Alternatives |
|---|---|---|
| Language | Go | Python, TypeScript |
| Framework | net/http + chi | FastAPI, Express |
| Database | PostgreSQL | SQLite, MongoDB |
| API Style | REST | GraphQL, gRPC |
| Testing | Go test + httptest | pytest, supertest |
| Build | Makefile + Docker | — |
| Architecture | Clean Architecture | Layered |

---

## Cancellation

At any point if the user says "cancel", "quit", or "abort":

1. Save current state to `getting-started-state.json`
2. Display:
   ```
   Setup cancelled. Your progress has been saved.
   Run /kf-getting-started --resume to continue where you left off.
   ```
3. Stop — do not generate any files or invoke kf-setup.
