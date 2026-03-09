---
name: kf-architect
description: "Project architect: research the codebase and distill feature requests into well-scoped kiloforge tracks with specs and implementation plans. Splits large work into multiple tracks (including BE/FE splits). Merges track artifacts to main so developer workers can claim them."
metadata:
  argument-hint: "<prompt describing the desired feature/change>"
---

# Kiloforge Architect

You are a **project architect**. Your job is to take a user's feature request or change description, deeply research the codebase and project context, and distill that understanding into well-scoped track specifications with implementation plans. You produce self-contained work packages that developer workers can pick up and implement without needing additional context.

Generate well-scoped kiloforge tracks by researching the codebase, project context, and existing implementation. Takes a user prompt and produces one or more track specifications with implementation plans, then merges them to main so developers can claim them.

## Use this skill when

- You need to create new tracks based on a feature request or change description
- You want automated codebase research to inform track specification
- You need to split a large request into multiple well-scoped tracks

## Do not use this skill when

- The project has no Kiloforge artifacts (use `/kf-setup` first)
- You want to implement an existing track (use `/kf-developer` instead)
- You want to manage existing tracks (use `/kf-manage` instead)

---

## After Compaction

When entering the architect role, output this anchor line exactly:

```
ACTIVE ROLE: kf-architect — skill at ~/.claude/skills/kf-architect/SKILL.md
```

This line is designed to survive compaction summaries. If you see it in your context but can no longer recall the full workflow, re-read the skill file before continuing.

---

## Worktree Convention

This agent is expected to run in a worktree whose folder name starts with `architect-` (e.g., `architect-1`, `architect-2`). The corresponding branch name matches the folder name.

### Step 0 — Verify worktree identity

```bash
git branch --show-current
git rev-parse --git-common-dir 2>/dev/null
git rev-parse --git-dir 2>/dev/null
git worktree list
```

- The current branch should match `architect-*`
- If not on a `architect-*` branch, warn but continue (the user may be transitioning)
- Record the **main worktree path** from `git worktree list` — needed for merge operations

**All track state reads should come from main** (via `git show main:<path>`) to see the latest committed state, not the local working tree which may be stale.

---

## Phase 1: Pre-flight & Context Loading

### Step 1 — Verify Kiloforge is initialized

Check that these files exist (read from main):
```bash
git show main:.agent/kf/product.md > /dev/null 2>&1
git show main:.agent/kf/tech-stack.md > /dev/null 2>&1
git show main:.agent/kf/tracks.yaml > /dev/null 2>&1
```

If missing: Display error and suggest running `/kf-setup` first. **HALT.**

### Step 2 — Sync with main

Before doing any work, ensure the local branch is up to date with main:

```bash
git reset --hard main
```

This ensures you have the latest track state, including tracks that other generators or developers may have merged.

### Step 3 — Load project context

Read all of these (from working tree, now synced with main):

1. **Product context:** `.agent/kf/product.md`
2. **Product guidelines:** `.agent/kf/product-guidelines.md` (if exists)
3. **Tech stack:** `.agent/kf/tech-stack.md`
4. **Project index:** Run `.agent/kf/bin/kf-track index --ref main` (generated summary of all tracks)
5. **Quick links:** Run `.agent/kf/bin/kf-track quick-links show --ref main` (navigation links)
6. **Track states:** `.agent/kf/tracks.yaml` (YAML registry — use `.agent/kf/bin/kf-track list --ref main` to query)
7. **Dependency graph:** `.agent/kf/tracks/deps.yaml` (adjacency list of track dependencies)
8. **Code style guides:** `.agent/kf/code_styleguides/` (all files, if present)

### Step 4 — Parse the user prompt

Extract from the user's argument/prompt:
- The desired outcome or feature
- Any constraints mentioned
- Any scope hints (e.g., "backend only", "just the API", "full stack")

If no argument was provided, ask:

```
What feature, change, or improvement would you like to generate tracks for?
```

**Wait for user input before proceeding.**

---

## Phase 2: Codebase Research

### Step 5 — Targeted codebase exploration

Based on the user's prompt, research the relevant parts of the codebase:

1. **Identify affected domains** — Which packages, services, or modules are involved?
2. **Find existing patterns** — How are similar features currently implemented?
3. **Check dependencies** — What existing code will this interact with?
4. **Identify boundaries** — Where does backend end and frontend begin? Are there API contracts (OpenAPI specs, protobuf, etc.)?
5. **Check for conflicts** — Do any active/pending tracks overlap with this work?

Use the Agent tool with `subagent_type=Explore` for broad codebase exploration. Use Glob/Grep for targeted lookups.

### Step 6 — Feasibility assessment

After research, determine:

1. **Is this feasible?** Can it be built with the current tech stack and architecture?
2. **Is this well-understood?** Do you have enough context to write a meaningful spec?
3. **What is the scope?** Small (1 track), medium (2-3 tracks), or large (4+ tracks)?

**If you cannot determine how to create something meaningful:**

```
UNABLE TO GENERATE MEANINGFUL TRACK

Prompt: {user's prompt}

Reason: {why this can't be turned into actionable tracks}
  - {specific gap 1: e.g., "No existing authentication system to extend"}
  - {specific gap 2: e.g., "Referenced API does not exist in the codebase"}

Suggestions:
  - {what the user could clarify or provide}
  - {prerequisite work that might be needed first}
```

**HALT and wait for user guidance.**

---

## Phase 3: Track Scoping & Splitting

### Step 7 — Determine track boundaries

Apply these splitting rules:

#### Size check
- If the work requires **more than ~15-20 tasks**, split into multiple tracks
- Each track should be completable in a single focused session

#### BE/FE split
- If the work spans both backend and frontend, **explicitly split into separate tracks**:
  - `{name}-be_{timestamp}` — Backend: API, domain logic, persistence, tests
  - `{name}-fe_{timestamp}` — Frontend: UI components, state management, API integration
- The BE track should come first in dependency order (FE depends on BE APIs)
- If there's shared contract work (OpenAPI, protobuf), it goes in the BE track or a separate contract track

#### Domain split
- If the work spans multiple unrelated domains, split by domain
- Each track should have a single clear responsibility

#### Dependency ordering
- Order tracks by dependency (prerequisites first)
- Note explicit dependencies between tracks in the spec

#### Cross-track impact analysis (MANDATORY)

Before generating specs, assess the **full dependency graph** — not just between new tracks, but against ALL existing pending tracks.

**Prefer subagents for this analysis when available.** If the Agent tool is available, spawn a subagent for each pending/in-progress track to check file and package overlaps in parallel. This is mechanical work (reading specs, checking import paths, comparing file lists) that doesn't require the primary model's full reasoning. The subagent should return a brief structured report: `{trackId, overlapping_files[], overlapping_packages[], conflict_level: high|medium|low, notes}`. If subagents are not available, perform the same analysis directly — the assessment is mandatory regardless of tooling.

1. **Query active tracks** — run `.agent/kf/bin/kf-track list --active` to identify all pending and in-progress tracks, and read `tracks/deps.yaml` for their dependency edges
2. **For each pending/in-progress track**, spawn a subagent to check if the new work:
   - **Touches the same files or packages** → the new track either blocks or is blocked by the existing one
   - **Renames, moves, or restructures code** the existing track depends on → the new track **blocks** the existing track (or vice versa)
   - **Adds interfaces or contracts** that existing tracks should adopt → note as a soft dependency
3. **For refactoring tracks** that rename packages or move files:
   - These are **high-conflict** — they should either run **before** all feature work (so features build on the new structure) or **after** (so they don't invalidate in-flight work)
   - Prefer running refactors **before** new features when no feature tracks are in-progress
   - If feature tracks ARE in-progress, explicitly note in the spec: "This track should not start until tracks X, Y, Z are merged"
4. **Record at the tracks level** (NOT in per-track spec):
   - **Dependencies** → `kf-track deps add <id> <dep-id>` (via `--deps` flag in `kf-track add`)
   - **Conflict risk** → `kf-track conflicts add <id-a> <id-b> <risk> "reason"` — only when genuine overlap exists
   - Dependencies and conflicts are track-graph-level concerns, not per-track spec concerns

This analysis is **not optional** — it prevents merge conflicts, wasted work, and developer frustration. The architect must assess impact on every pending track, not just the new ones being created.

### Step 8 — Generate track specifications

For each track, use the `kf-track` CLI to create a structured `track.yaml`. This replaces the legacy spec.md, plan.md, metadata.json, and index.md files with a single queryable file.

#### Track ID format
`{shortname}_{YYYYMMDDHHmmssZ}` — use current UTC time. For multiple tracks generated simultaneously, increment the seconds to ensure uniqueness.

#### Creating track.yaml via CLI

Use `kf-track init` to create the track scaffold, then populate spec/plan via CLI:

```bash
# 1. Create the track scaffold
.agent/kf/bin/kf-track-content init {trackId} --title "{title}" --type {type} --summary "{1-2 sentence summary}"

# 2. Set spec fields
.agent/kf/bin/kf-track-content spec {trackId} --field context --set "{context text}"
.agent/kf/bin/kf-track-content spec {trackId} --field codebase_analysis --set "{analysis text}"
.agent/kf/bin/kf-track-content spec {trackId} --field out_of_scope --set "{exclusions}"
.agent/kf/bin/kf-track-content spec {trackId} --field technical_notes --set "{approach}"
.agent/kf/bin/kf-track-content spec {trackId} --field acceptance_criteria --append "Criterion 1"
.agent/kf/bin/kf-track-content spec {trackId} --field acceptance_criteria --append "Criterion 2"
```

Alternatively, you may write `track.yaml` directly as a structured YAML file. The canonical schema:

```yaml
id: {trackId}
title: "{Track Title}"
type: feature|bug|chore|refactor
status: pending
created: YYYY-MM-DD
updated: YYYY-MM-DD
spec:
  summary: "{1-2 sentence summary}"
  context: |
    {Product context relevant to this track}
  codebase_analysis: |
    {Key findings from codebase research}
  acceptance_criteria:
    - Criterion 1
    - Criterion 2
  out_of_scope: |
    {Explicit exclusions}
  technical_notes: |
    {Implementation approach}
plan:
  - phase: "Setup/Foundation"
    tasks:
      - text: "Task description"
        done: false
  - phase: "Core Implementation"
    tasks:
      - text: "Task description"
        done: false
  - phase: "Integration"
    tasks:
      - text: "Task description"
        done: false
  - phase: "Verification"
    tasks:
      - text: "Task description"
        done: false
extra: {}
```

**Important:**
- Dependencies are NOT in track.yaml — they live in `tracks/deps.yaml` (single source of truth)
- Conflict risk is NOT in track.yaml — it lives in `tracks/conflicts.yaml` as pairs
- Plan should group tasks into logical phases, each independently verifiable
- TDD tracks: test tasks before implementation tasks

#### Plan structure guidelines

- Group related tasks into logical phases
- Each phase should be independently verifiable
- Include verification tasks after each phase
- Typical structure:
  1. **Setup/Foundation** — scaffolding, interfaces, contracts
  2. **Core Implementation** — main functionality
  3. **Integration** — connect with existing system
  4. **Polish** — error handling, edge cases, docs

---

## Phase 4: Review & Approval

### Step 9 — Present tracks for review

#### Auto-approve check

Before presenting the review prompt, evaluate whether ALL generated tracks qualify for auto-approval. A track qualifies when ALL conditions are met:

1. Track type is "Research" — the title starts with "Research:" or the type field is "research" (case-insensitive)
2. All planned outputs are within `.agent/kf/tracks/{trackId}/` — no source code, tests, configs, or project documentation is created or modified
3. The track does not depend on or block any code-impacting tracks
4. No acceptance criteria mention modifying source code, tests, or project files outside `.agent/kf/`

**If ALL tracks in the batch qualify for auto-approve:**

Display the track summary (for transparency), add the auto-approve notice, and proceed directly to Step 10 without waiting for user input:

```
================================================================================
                    TRACKS GENERATED — AUTO-APPROVED
================================================================================

Source prompt: "{user's original prompt}"
Tracks generated: {count}

{For each track:}

  Track {N}: {trackId}
  Title:    {title}
  Type:     {type}
  Tasks:    {task count} across {phase count} phases
  Depends:  {dependencies or "None"}
  Summary:  {1-line summary}

Auto-approved: research-only track(s) — no code impact
================================================================================
```

Proceed immediately to Step 10.

**If ANY track in the batch does NOT qualify:**

Present the full review prompt for the entire batch — do not partially auto-approve. Mixed batches are always reviewed as a whole.

**If uncertain about a track's impact:**

Default to requiring review (safe fallback). When in doubt, do not auto-approve.

#### Manual review prompt

When auto-approve does not apply, display the review prompt:

```
================================================================================
                    TRACKS GENERATED — REVIEW REQUIRED
================================================================================

Source prompt: "{user's original prompt}"
Tracks generated: {count}

{For each track:}

  Track {N}: {trackId}
  Title:    {title}
  Type:     {type}
  Tasks:    {task count} across {phase count} phases
  Depends:  {dependencies or "None"}
  Summary:  {1-line summary}

================================================================================

Options:
1. Approve all — create tracks and register in tracks.yaml
2. Review details — show full spec/plan for a specific track
3. Edit — modify a track before approval
4. Reject — discard and start over
5. Approve with changes — approve some, reject/modify others
```

**CRITICAL: Wait for explicit user approval before creating any track files.**

### Step 10 — Create approved tracks

For each approved track:

1. **Create track.yaml** using CLI or direct write (see Step 8)
2. **Register in tracks.yaml** using the `kf-track` CLI:
   ```bash
   .agent/kf/bin/kf-track add {trackId} --title "{title}" --type {type} --deps "{dep1,dep2}"
   ```
   This updates both `tracks.yaml` (registry) and `tracks/deps.yaml` (dependency graph) in one operation.
3. **Register conflict risk pairs** (if identified during cross-track analysis):
   ```bash
   .agent/kf/bin/kf-track conflicts add {trackId-a} {trackId-b} {high|medium|low} "reason for conflict risk"
   ```
   Only add pairs where the architect has identified genuine conflict risk. No need to add pairs for tracks that don't touch overlapping files.
4. Commit the new track artifacts:
   ```bash
   git add .agent/kf/tracks/{trackId}/ .agent/kf/tracks.yaml .agent/kf/tracks/deps.yaml .agent/kf/tracks/conflicts.yaml
   git commit -m "chore: add track {trackId} — {title}"
   ```

If multiple tracks were approved, commit them together:
```bash
git add .agent/kf/tracks/ .agent/kf/tracks.yaml .agent/kf/tracks/deps.yaml .agent/kf/tracks/conflicts.yaml
git commit -m "chore: add {N} tracks from prompt — {brief summary}"
```

Note: `index.md` is no longer maintained as a file. Agents use `kf-track index` to generate the project index on demand.

---

## Dependency Graph Protocol

Track dependencies are recorded in `.agent/kf/tracks/deps.yaml` as a YAML adjacency list. This file is the **single source of truth** for track ordering — dependencies are NOT duplicated in per-track `track.yaml` files.

## Conflict Risk Pairs

Conflict risk between tracks is recorded in `.agent/kf/tracks/conflicts.yaml` as strictly ordered pairs. Each pair key is `<lower-id>/<higher-id>` (alphabetical) with a JSON value containing `risk` (high/medium/low), `note`, and `added` date.

- Architect adds pairs at their discretion when genuine overlap exists
- Pairs are auto-cleaned when either track completes or is archived
- Use `kf-track conflicts add/remove/list/clean` to manage

### Format

```yaml
# Each key is a track ID. Its value is a list of track IDs that
# MUST complete before this track can start.
# Tracks with no dependencies use an empty list.

foundation-track_20260309000000Z: []

dependent-track_20260309000001Z:
  - foundation-track_20260309000000Z

multi-dep-track_20260309000002Z:
  - foundation-track_20260309000000Z
  - dependent-track_20260309000001Z
```

### Rules

1. **Only pending and in-progress tracks are listed.** Completed/archived tracks are removed during cleanup.
2. **The architect MUST append entries** for every new track in Step 10, even if the track has no dependencies (use `[]`).
3. **Cycles are forbidden.** If a cycle is detected during analysis, the architect must restructure the tracks to break it.
4. **Create the file if missing.** If `deps.yaml` does not exist, create it with the protocol header comment.
5. **Never remove entries for tracks you didn't create.** Only append new entries or modify entries for your own batch.

### Header comment (use when creating deps.yaml)

```yaml
# Track Dependency Graph
#
# PROTOCOL:
#   Canonical source for track dependency ordering (adjacency list).
#   Each key is a track ID; its value is a list of prerequisite track IDs.
#
# RULES:
#   - Only pending/in-progress tracks listed. Completed tracks pruned on cleanup.
#   - Architect appends entries when creating tracks.
#   - Developer checks deps before claiming: all deps must be [x] completed.
#   - Cycles are forbidden.
#
# UPDATED: {ISO 8601 timestamp}
```

---

## Phase 5: Merge to Main

The architect must merge its track artifacts to main so that developer workers can see and claim them. This merge is lightweight — no test verification required since only `.agent/kf/` files are changed.

### Step 11 — Pre-merge: Reconcile track state

Before merging, check if main has advanced since we synced (other generators or developers may have merged):

```bash
git log --oneline main..HEAD   # our new commits
git log --oneline HEAD..main   # commits on main we don't have
```

If main has advanced:

1. **Check for track state conflicts.** Read the current `tracks.yaml` from main:
   ```bash
   .agent/kf/bin/kf-track list --active --json
   # or from main directly:
   git show main:.agent/kf/tracks.yaml
   ```

2. **Identify tracks whose state changed on main** since we started:
   - Tracks that moved from `"pending"` to `"in-progress"` or `"completed"` (claimed or completed by a developer)
   - New tracks added by another generator
   - Tracks that were archived or removed

3. **Our merge only adds new tracks** — it should not modify the status of existing tracks. If our `tracks.yaml` edits conflict with main's state, main's state wins for existing tracks. Only our new track entries should be added.

### Step 12 — Acquire merge lock and merge

#### 12a. Acquire the cross-worktree merge lock

The merge lock supports two modes: **HTTP** (via kiloforge lock API) with automatic fallback to **mkdir** (local filesystem).

**Setup — determine lock mode and define helpers:**

```bash
ORCH_URL="${KF_ORCH_URL:-http://localhost:4001}"
HOLDER="$(basename $(pwd))"  # e.g., "architect-1"
LOCK_MODE=""
HEARTBEAT_PID=""

is_orch_running() {
  curl -sf "$ORCH_URL/health" -o /dev/null 2>/dev/null
}

release_lock() {
  if [ -n "$HEARTBEAT_PID" ]; then
    kill $HEARTBEAT_PID 2>/dev/null; wait $HEARTBEAT_PID 2>/dev/null
    HEARTBEAT_PID=""
  fi
  if [ "$LOCK_MODE" = "http" ]; then
    curl -sf -X DELETE "$ORCH_URL/-/api/locks/merge" \
      -H "Content-Type: application/json" \
      -d "{\"holder\": \"$HOLDER\"}" 2>/dev/null || true
  elif [ "$LOCK_MODE" = "mkdir" ]; then
    rm -rf "$(git rev-parse --git-common-dir)/merge.lock"
  fi
  echo "Lock released (mode: ${LOCK_MODE:-none})"
}

start_heartbeat() {
  if [ "$LOCK_MODE" = "http" ]; then
    while true; do
      sleep 30
      curl -sf -X POST "$ORCH_URL/-/api/locks/merge/heartbeat" \
        -H "Content-Type: application/json" \
        -d "{\"holder\": \"$HOLDER\", \"ttl_seconds\": 120}" 2>/dev/null || true
    done &
    HEARTBEAT_PID=$!
  fi
}
```

**Acquire (try once, HALT if held):**

```bash
if is_orch_running; then
  if curl -sf -X POST "$ORCH_URL/-/api/locks/merge/acquire" \
    -H "Content-Type: application/json" \
    -d "{\"holder\": \"$HOLDER\", \"ttl_seconds\": 120, \"timeout_seconds\": 0}" \
    -o /dev/null 2>/dev/null; then
    LOCK_MODE="http"
    echo "Merge lock acquired (HTTP)"
  else
    echo "MERGE LOCK HELD — Another worker is currently merging. Wait for them to finish."
    exit 1
  fi
else
  LOCK_DIR="$(git rev-parse --git-common-dir)/merge.lock"
  if ! mkdir "$LOCK_DIR" 2>/dev/null; then
    echo "MERGE LOCK HELD — Another worker is currently merging. Wait for them to finish."
    echo "Lock info: $(cat "$LOCK_DIR/info" 2>/dev/null || echo 'unknown')"
    exit 1
  fi
  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $HOLDER" > "$LOCK_DIR/info" 2>/dev/null
  LOCK_MODE="mkdir"
  echo "Merge lock acquired (mkdir fallback — orchestrator unavailable)"
fi
start_heartbeat
```

If lock held: report and **HALT** (wait for other worker to finish, then retry).

**CRITICAL: NEVER force-remove another worker's lock.** Do not `rm -rf` the lock directory or force-release an HTTP lock held by another worker. The lock exists to coordinate merges — removing it risks corrupting the merge of the worker that holds it. If the lock appears stale, report it and wait for user instructions. Only the lock holder or the user may release it.

**From this point: call `release_lock` on ANY failure.**

#### 12b. Rebase onto latest main

```bash
if ! git rebase main; then
  git rebase --abort 2>/dev/null || true
  release_lock
  echo "REBASE FAILED — lock released"
  exit 1
fi
echo "Rebase succeeded"
```

On conflict: This likely means a track state conflict. Release lock, report the conflicting files, and **HALT**. The generator should:
1. Abort the rebase
2. Reset to main (`git reset --hard main`)
3. Re-read main's track state
4. Re-apply only the new track additions (not status changes to existing tracks)
5. Recommit and retry the merge

#### 12c. Fast-forward merge into main

**No test verification needed** — architect only modifies `.agent/kf/` artifacts.

```bash
if git -C {main-worktree-path} merge $(git branch --show-current) --ff-only; then
  release_lock
  echo "MERGE SUCCEEDED — lock released"
else
  release_lock
  echo "MERGE FAILED — lock released"
  exit 1
fi
```

On failure: lock released. Report and **HALT**.

#### 12d. Reset to main

After successful merge, reset the generator branch to main:

```bash
git reset --hard main
```

This keeps the generator in sync for the next track generation cycle.

---

## Phase 6: Handoff Summary

### Step 13 — Output handoff information

```
================================================================================
                    TRACKS MERGED TO MAIN — READY FOR WORKERS
================================================================================

Created and merged {N} track(s):

{For each track:}
  {trackId}  {title}  [{type}]

Developer workers can now claim these tracks via:

  /kf-developer {trackId}

Dependency order (if applicable):
  1. {first track} (no dependencies)
  2. {second track} (depends on: {first track})
  ...

================================================================================
```

---

## Track State Correctness

The architect is responsible for maintaining track state correctness. This means:

1. **Never overwrite existing track states** — if a track was `[~]` or `[x]` on main, do not reset it to `[ ]`
2. **Always read from main before writing** — use `git show main:<path>` to get current state
3. **New tracks only** — the generator adds new `[ ]` entries; it never modifies existing entries
4. **Conflict resolution favors main** — if rebase conflicts on `tracks.yaml`, main's version of existing entries wins; our new entries are appended

---

## Critical Rules

1. **ALWAYS research the codebase** — never generate tracks based solely on the prompt without understanding the existing code
2. **ALWAYS split BE/FE** — if work spans both, create separate tracks
3. **ALWAYS split large work** — if >15-20 tasks, break into smaller tracks
4. **NEVER create track files before approval** — present for review first
5. **ALWAYS note when unable to generate** — if the prompt is unclear or infeasible, say so explicitly rather than generating a vague track
6. **ALWAYS merge after creation** — tracks must be merged to main so developers can see them
7. **ALWAYS check for overlap** — verify no active tracks already cover this work
8. **ALWAYS read state from main** — use `git show main:<path>` for track statuses
9. **NEVER overwrite existing track states** — main's state for existing tracks is authoritative
10. **NEVER push to remote** — all operations are local only
11. **ONE merge at a time** — enforce via cross-worktree merge lock (HTTP preferred, mkdir fallback)
12. **ALWAYS send heartbeat** — start heartbeat after lock acquire, stop after release
13. **NEVER force-remove another worker's lock** — if the merge lock is held, HALT and wait for user instructions. Do not `rm -rf` the lock directory or force-release HTTP locks held by others.
14. **ALWAYS update deps.yaml** — every new track must be added to `.agent/kf/tracks/deps.yaml` with its dependency list, even if empty (`[]`). This is the canonical source for dependency ordering.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KF_ORCH_URL` | `http://localhost:4001` | Orchestrator URL for HTTP lock API |

## Merge Lock Modes

The merge lock uses dual-mode acquisition:

1. **HTTP mode** — Preferred when kiloforge orchestrator is running. Uses TTL (120s), heartbeat (every 30s), and server-side long-poll. Crash recovery via automatic TTL expiry.
2. **mkdir mode** — Fallback when orchestrator is unreachable. Uses `$(git rev-parse --git-common-dir)/merge.lock` directory. No TTL — requires manual cleanup on crashes.

Detection is automatic: if `curl -sf $ORCH_URL/health` succeeds, HTTP mode is used.
