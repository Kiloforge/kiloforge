# Kiloforge Data Format Specification

This document specifies the data formats used by Kiloforge for track management. All files live under `.agent/kf/` in the project root.

## Directory Layout

```
.agent/kf/
├── tracks.yaml              # Track registry (all tracks, one line per track)
├── product.md               # Product context
├── product-guidelines.md    # Product design guidelines
├── tech-stack.md            # Technology stack reference
├── index.md                 # Human-readable track listing
├── tracks/
│   ├── deps.yaml            # Dependency graph (adjacency list)
│   ├── conflicts.yaml       # Conflict risk pairs
│   ├── <track-id>/
│   │   └── track.yaml       # Per-track structured content
│   └── _archive/            # Archived track directories
└── bin/
    ├── kf-track             # CLI tool (bash)
    ├── kf-track-content     # Content management tool (python)
    └── tests/
        └── run_tests.sh     # Automated test suite
```

---

## Track IDs

Format: `{shortname}_{YYYYMMDDHHmmssZ}`

- `shortname`: kebab-case descriptive name (e.g., `agent-lifecycle-be`)
- `_`: literal underscore separator
- `YYYYMMDDHHmmssZ`: UTC timestamp of creation

Examples:
```
e2e-infra-mock-agent_20260309194830Z
admin-operations-ui_20260309173001Z
track-detail-view-be_20260309001726Z
```

For multiple tracks generated simultaneously, increment seconds to ensure uniqueness.

---

## tracks.yaml — Track Registry

The central registry of all tracks. One JSON object per line, keyed by track ID.

### Format

```
# Header comments (lines starting with #)
<track-id>: {"title":"...","status":"...","type":"...","created":"...","updated":"..."}
```

### Field Order (Canonical)

Fields MUST appear in this exact order:

1. `title` — Human-readable track title
2. `status` — Lifecycle state (see below)
3. `type` — Track type (see below)
4. `created` — ISO date (`YYYY-MM-DD`)
5. `updated` — ISO date (`YYYY-MM-DD`)
6. `archived_at` — ISO date (only when status is `archived`)
7. `archive_reason` — Free text (only when status is `archived`)

### Status Values

| Status | Symbol | Description |
|--------|--------|-------------|
| `pending` | `[ ]` | Created, not yet claimed |
| `in-progress` | `[~]` | Claimed by a developer agent |
| `completed` | `[x]` | All tasks done, merged |
| `archived` | `[a]` | Moved to `_archive/` |

### Type Values

- `feature` — New functionality
- `bug` — Bug fix
- `chore` — Maintenance, tooling, infrastructure
- `refactor` — Code restructuring without behavior change
- `research` — Investigation or spike

### Ordering

Lines are sorted **alphabetically by track ID**. This keeps diffs minimal and predictable.

### Example

```yaml
# Kiloforge Track Registry
#
# FORMAT: <track-id>: {"title":"...","status":"...","type":"...","created":"...","updated":"..."}
# STATUS: pending | in-progress | completed | archived
# ORDER:  Lines sorted alphabetically by track ID. JSON fields in canonical order.
# TOOL:   Use `kf-track` to manage entries. Do not edit by hand.
#

admin-ui_20260309173001Z: {"title":"Admin Operations UI","status":"completed","type":"feature","created":"2026-03-09","updated":"2026-03-09"}
agent-lifecycle-be_20260310030000Z: {"title":"Agent Lifecycle (Backend)","status":"in-progress","type":"feature","created":"2026-03-10","updated":"2026-03-10"}
```

---

## deps.yaml — Dependency Graph

An adjacency list expressing prerequisite relationships between tracks. A track cannot be claimed until all its dependencies are completed.

### Format

```yaml
# Header comments

<track-id>:
  - <dependency-track-id>
  - <dependency-track-id>

<track-id-with-no-deps>: []
```

### Rules

1. Only pending and in-progress tracks are listed. Completed tracks are pruned on cleanup.
2. Dependency lists within each entry are sorted alphabetically.
3. Track entries are sorted alphabetically by track ID.
4. Cycles are forbidden. If detected, the architect must restructure.
5. When a track is completed or archived, it is removed as both a key and a dependency value.

### Example

```yaml
# Track Dependency Graph

e2e-agent-lifecycle_20260309194834Z:
  - e2e-infra-mock-agent_20260309194830Z

e2e-infra-mock-agent_20260309194830Z: []

e2e-kanban-board_20260309194836Z:
  - e2e-infra-mock-agent_20260309194830Z
  - e2e-project-management_20260309194832Z
```

---

## conflicts.yaml — Conflict Risk Pairs

Records potential merge conflicts between concurrently active tracks. Each record is a strictly ordered pair of track IDs with risk metadata.

### Format

```
# Header comments
<lower-id>/<higher-id>: {"risk":"...","note":"...","added":"..."}
```

### Pair Key Ordering

The pair key is **strictly ordered**: the alphabetically lower ID comes first. This ensures only one record exists per pair.

Given tracks `z-feature` and `a-bugfix`, the key is always `a-bugfix/z-feature`.

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `risk` | string | `high`, `medium`, or `low` |
| `note` | string | Brief explanation of the conflict |
| `added` | string | ISO date when the pair was recorded |

### Rules

1. Pairs are added by architects at their discretion when generating tracks.
2. Pairs are auto-cleaned when either track is completed or archived.
3. Only active (pending/in-progress) tracks should have pairs.
4. Lines are sorted by pair key.

### Example

```yaml
# Track Conflict Risk Pairs
#
# Each line: <id-a>/<id-b>: {"risk":"high|medium|low","note":"...","added":"..."}

agent-lifecycle-be_20260310030000Z/agent-lifecycle-fe_20260310030001Z: {"risk":"low","note":"FE depends on BE API, no overlapping files","added":"2026-03-10"}
refactor-core_20260310060000Z/track-detail-view-be_20260309001726Z: {"risk":"high","note":"Both modify internal/core/service/track_service.go","added":"2026-03-10"}
```

---

## track.yaml — Per-Track Content

Each track has a `track.yaml` file at `.agent/kf/tracks/<track-id>/track.yaml` containing the full specification and implementation plan.

### Schema

```yaml
id: <track-id>
title: <string>
type: <feature|bug|chore|refactor|research>
status: <pending|in-progress|completed|archived>
created: <ISO datetime or date>
updated: <ISO datetime or date>
spec:
  summary: <string>
  context: |
    <multiline string>
  codebase_analysis: |
    <multiline string>
  acceptance_criteria:
    - <string>
    - <string>
  out_of_scope: |
    <multiline string>
  technical_notes: |
    <multiline string>
plan:
  - phase: <phase name>
    tasks:
      - text: <task description>
        done: false
      - text: <task description>
        done: true
  - phase: <phase name>
    tasks:
      - text: <task description>
        done: false
extra: {}
```

### Top-Level Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | yes | Track ID matching the directory name |
| `title` | yes | Human-readable title |
| `type` | yes | Track type |
| `status` | yes | Lifecycle status |
| `created` | yes | Creation timestamp |
| `updated` | yes | Last modification timestamp |
| `spec` | yes | Specification object |
| `plan` | yes | List of phases |
| `extra` | no | Arbitrary key-value pairs (default `{}`) |

### Spec Fields

| Field | Required | Description |
|-------|----------|-------------|
| `summary` | yes | 1-2 sentence summary of the track |
| `context` | no | Product/domain context |
| `codebase_analysis` | no | Key findings from code research |
| `acceptance_criteria` | no | List of acceptance criteria strings |
| `out_of_scope` | no | Explicit exclusions |
| `technical_notes` | no | Implementation approach and notes |

### Plan Structure

The plan is a list of phases. Each phase has a name and a list of tasks. Tasks are marked done as implementation progresses.

```yaml
plan:
  - phase: Setup
    tasks:
      - text: Install dependencies
        done: true
      - text: Configure project
        done: false
  - phase: Implementation
    tasks:
      - text: Build core feature
        done: false
```

Task addressing uses `<phase-index>.<task-index>` notation (1-based):
- `1.1` = first task in first phase
- `2.3` = third task in second phase

### Progress Calculation

- **Task completion**: count of `done: true` / total tasks
- **Phase completion**: a phase is complete when ALL its tasks are `done: true`
- **Track completion**: all phases complete

---

## CLI Tools

### kf-track (bash)

Primary CLI for registry, dependency, and conflict management.

```bash
# Registry operations
kf-track add <id> <title> [--type feature|bug|chore|refactor|research]
kf-track update <id> --status <status>
kf-track list [--status <status>] [--active]
kf-track show <id>
kf-track archive <id> [reason]

# Dependency operations
kf-track deps add <track-id> <dep-id>
kf-track deps remove <track-id> <dep-id>
kf-track deps list [track-id]
kf-track deps check <track-id>

# Conflict operations
kf-track conflicts add <id-a> <id-b> <risk> [note]
kf-track conflicts remove <id-a> <id-b>
kf-track conflicts list [track-id]
kf-track conflicts clean <track-id>
```

### kf-track-content (python)

Content management for per-track track.yaml files.

```bash
# Show content
kf-track-content show <id>                    # Full track.yaml
kf-track-content show <id> --section spec     # Spec section only
kf-track-content show <id> --section plan     # Plan section only

# Spec and plan shortcuts
kf-track-content spec <id>                    # Show spec
kf-track-content plan <id>                    # Show plan

# Progress
kf-track-content progress <id>               # Show completion stats

# Task management
kf-track-content task <id> <phase.task> --done    # Mark task complete
kf-track-content task <id> <phase.task> --undone  # Mark task incomplete
```

### Prerequisites

| Tool | Required By | Install |
|------|-------------|---------|
| `python3` | kf-track-content | `brew install python3` (macOS), `apt install python3` (Linux) |
| `jq` | kf-track | `brew install jq` (macOS), `apt install jq` (Linux) |
| `git` | kf-track | Usually pre-installed |

Prerequisites are checked at startup and clear install instructions are displayed if missing.

---

## Go SDK

The Go SDK at `backend/pkg/kf` provides programmatic access to all Kiloforge data formats.

```go
import "kiloforge/pkg/kf"
```

### Client API

```go
// Create a client
client := kf.NewClientFromProject("/path/to/project")

// Registry
tracks, _ := client.ListTracks()
active, _ := client.ListActiveTracks()
ready, _ := client.ListReadyTracks()  // Active with all deps satisfied
entry, _ := client.GetTrackEntry("my-track_20260310Z")
client.AddTrack(entry, []string{"dep-track-id"})
client.UpdateStatus("my-track_20260310Z", kf.StatusCompleted)
client.ArchiveTrack("my-track_20260310Z", "Done")

// Dependencies
satisfied, unmet, _ := client.CheckDeps("my-track_20260310Z")

// Conflicts
conflicts, _ := client.GetConflictsForTrack("my-track_20260310Z")
client.AddConflict("track-a", "track-b", "high", "Both modify same file")

// Track content
track, _ := client.GetTrack("my-track_20260310Z")
track.SetTaskDone(1, 2, true)  // Mark phase 1, task 2 as done
client.SaveTrack(track)

// Progress
stats, _ := client.GetTrackProgress("my-track_20260310Z")
fmt.Printf("%d/%d tasks complete (%d%%)\n", stats.CompletedTasks, stats.TotalTasks, stats.Percent)
```

### Low-Level API

For direct file I/O without the Client wrapper:

```go
// Registry
entries, _ := kf.ReadRegistryFile("tracks.yaml")
kf.WriteRegistryFile("tracks.yaml", entries)

// Dependencies
graph, _ := kf.ReadDepsFile("deps.yaml")
graph.AddDep("my-track", "prereq-track")
kf.WriteDepsFile("deps.yaml", graph)

// Conflicts
pairs, _ := kf.ReadConflictsFile("conflicts.yaml")
pair := kf.NewConflictPair("track-a", "track-b", "medium", "Shared files")
pairs = kf.AddOrUpdateConflict(pairs, pair)
kf.WriteConflictsFile("conflicts.yaml", pairs)

// Track content
track, _ := kf.ReadTrack("tracks/my-track/track.yaml")
track.SetTaskDone(1, 1, true)
kf.WriteTrack("tracks/my-track/track.yaml", track)
```

### Side Effects

`Client.UpdateStatus` and `Client.ArchiveTrack` automatically prune the completed/archived track from both `deps.yaml` and `conflicts.yaml`.

---

## Cross-Platform Support

All bash tools use a portable `sedi()` wrapper for `sed -i` compatibility across macOS (`sed -i ''`) and Linux (`sed -i`). Platform detection runs at startup:

```bash
case "$(uname -s)" in
  Darwin*)  _KF_PLATFORM="mac" ;;
  Linux*)   _KF_PLATFORM="linux" ;;
  MINGW*|MSYS*|CYGWIN*) _KF_PLATFORM="windows" ;;
esac
```

---

## Automated Testing

The test suite at `.agent/kf/bin/tests/run_tests.sh` provides 45 isolated tests covering:

- **kf-track**: Registry CRUD, status updates, archival
- **deps**: Add/remove/check/list dependencies
- **conflicts**: Add/remove/list/clean conflict pairs
- **kf-track-content**: Show, spec, plan, progress, task management
- **Integration**: Cross-tool workflows

Each test creates a temporary git repository, copies the tools, executes the test, and tears down. Tests run in CI on both Ubuntu and macOS.

```bash
# Run locally
.agent/kf/bin/tests/run_tests.sh

# CI (in .github/workflows/ci.yml)
# Runs on matrix: [ubuntu-latest, macos-latest]
```
