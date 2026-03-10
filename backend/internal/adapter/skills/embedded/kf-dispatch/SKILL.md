---
name: kf-dispatch
description: AI swarm dispatcher — analyzes project state and produces prescriptive worker assignments for idle developer agents.
metadata:
  argument-hint: "[--dry-run]"
---

# Kiloforge Dispatch

Swarm dispatcher that reviews project state (worktrees, active tracks, dependencies, conflict risks) and produces prescriptive worker assignments. Run this when you need to decide which idle workers should claim which tracks.

## Use this skill when

- You have idle developer worktrees and need to assign tracks
- You want to know what to work on next across the swarm
- The Kiloforger needs a dispatch plan for available workers

## Do not use this skill when

- You need to create new tracks (use `/kf-architect` instead)
- You want informational status only (use `/kf-status` instead)
- You are a developer worker ready to implement (use `/kf-developer <track-id>` instead)
- The project has no Kiloforge artifacts (use `/kf-setup` first)

---

## Instructions

### Step 1 — Resolve primary branch

```bash
PRIMARY_BRANCH=$(git show main:.agent/kf/config.yaml 2>/dev/null | grep '^primary_branch:' | awk '{print $2}')
PRIMARY_BRANCH="${PRIMARY_BRANCH:-main}"
```

### Step 2 — Scan worktree state

Identify all developer workers and classify them:

```bash
git worktree list
```

For each worktree whose folder name starts with `developer-`:

```bash
# Get the branch checked out in that worktree
git -C <worktree-path> branch --show-current
```

**Classify each worker:**

| Branch pattern | Classification | Meaning |
|---|---|---|
| `developer-*` (home branch) | **IDLE** | On home branch, not implementing a track |
| `feature/*`, `bug/*`, `chore/*`, `refactor/*` | **ACTIVE** | Currently implementing a track |
| Any other branch | **UNKNOWN** | Unusual state, flag for review |

Record the list of idle and active workers.

### Step 3 — Read track registry

Get the full project status from the primary branch:

```bash
.agent/kf/bin/kf-track status --ref ${PRIMARY_BRANCH}
```

Parse the output to build these lists:

- **AVAILABLE tracks** — status `pending` with all dependencies satisfied (labeled `AVAILABLE` in status output)
- **BLOCKED tracks** — status `pending` with unmet dependencies (labeled `BLOCKED`)
- **CLAIMED tracks** — status `in-progress` (labeled `CLAIMED`)
- **COMPLETED tracks** — status `completed`

If no tracks exist at all, skip to **Error State: No Tracks**.

### Step 4 — Analyze dependency graph for prioritization

For each AVAILABLE track, calculate a **priority score** using these heuristics (highest score = assign first):

#### 4a. Unblock factor (weight: 3x)

Count how many other tracks depend on this track (directly or transitively). Tracks that unblock many others get higher priority.

```bash
.agent/kf/bin/kf-track deps show --ref ${PRIMARY_BRANCH}
```

Scan the dependency graph: for each AVAILABLE track, count the number of BLOCKED tracks that list it (directly or indirectly) as a dependency. Multiply by 3.

#### 4b. Conflict avoidance (weight: 2x)

Check conflict risk pairs:

```bash
.agent/kf/bin/kf-track conflicts list --all --ref ${PRIMARY_BRANCH}
```

For each AVAILABLE track, check if it conflicts with any CLAIMED (in-progress) track. If so, **penalize** by -2 per active conflict. Two AVAILABLE tracks that conflict with each other should not be assigned to parallel workers — flag this.

#### 4c. Type diversity bonus (weight: 1x)

If multiple tracks are available, prefer a mix of types (feature, chore, bug, refactor) over assigning all of one type. Add +1 for tracks of types not already being worked on by active workers.

#### 4d. Task count tiebreaker

Among equally-scored tracks, prefer smaller tracks (fewer tasks) as they unblock the pipeline faster.

### Step 5 — Match workers to tracks

Apply the dispatch algorithm:

1. Sort AVAILABLE tracks by priority score (descending)
2. For each idle worker, assign the highest-priority unassigned track
3. Skip assignments where:
   - The track conflicts with another track already assigned in this dispatch round
   - The track conflicts with a CLAIMED track being worked by another active worker
4. If there are more idle workers than available tracks, the extra workers have no assignment

### Step 6 — Produce dispatch plan

Output the dispatch plan in this format:

```
================================================================================
                    KILOFORGE DISPATCH — SWARM ASSIGNMENTS
================================================================================

Workers: {idle_count} idle, {active_count} active, {total_count} total
Tracks:  {available_count} available, {blocked_count} blocked, {claimed_count} in-progress

--- ACTIVE WORKERS -----------------------------------------------------------

  developer-1  →  ACTIVE on feature/some-track (3/8 tasks done)
  developer-3  →  ACTIVE on chore/other-track (1/4 tasks done)

--- DISPATCH PLAN -------------------------------------------------------------

  developer-2  →  feature/high-priority-track
                   Priority: unblocks 3 tracks, no conflicts
                   Command: claude --worktree developer-2 -p "/kf-developer high-priority-track_20260310Z"

  developer-4  →  chore/cleanup-track
                   Priority: independent, small (2 tasks)
                   Command: claude --worktree developer-4 -p "/kf-developer cleanup-track_20260310Z"

--- UNASSIGNED ----------------------------------------------------------------

  (none)

--- DEFERRED (conflict risk) --------------------------------------------------

  bug/risky-track — conflicts with feature/high-priority-track (assigned above)
    Assign after developer-2 completes, or to a different worker later.

--- BLOCKED -------------------------------------------------------------------

  feature/dependent-track — waiting on: high-priority-track (AVAILABLE, assigned above)

================================================================================
```

If `--dry-run` was passed, add a note:

```
DRY RUN — no actions taken. Review the plan above and run the commands manually.
```

Otherwise, present the commands and ask the Kiloforger to confirm before executing.

### Step 7 — No-work recommendations

If there are idle workers but **no AVAILABLE tracks**, produce recommendations:

```
================================================================================
                    KILOFORGE DISPATCH — NO TRACKS AVAILABLE
================================================================================

Workers: {idle_count} idle, {active_count} active
Tracks:  0 available, {blocked_count} blocked, {claimed_count} in-progress

All available tracks are either claimed or blocked.

RECOMMENDATIONS:

  1. Wait for active workers to complete — this will unblock:
     {list of blocked tracks and what they're waiting on}

  2. Create new tracks with /kf-architect:
     {suggest areas based on completed track patterns or gaps}

  3. If tracks are stale (in-progress but no recent commits):
     Check worker status with: git -C <worktree-path> log --oneline -3

================================================================================
```

If there are **no tracks at all** (empty project):

```
================================================================================
                    KILOFORGE DISPATCH — NO TRACKS EXIST
================================================================================

No tracks found in the project. Create tracks first:

  /kf-architect <feature description>

================================================================================
```

If there are **no idle workers**:

```
================================================================================
                    KILOFORGE DISPATCH — ALL WORKERS BUSY
================================================================================

Workers: 0 idle, {active_count} active

  developer-1  →  feature/track-a (5/8 tasks)
  developer-2  →  chore/track-b (2/4 tasks)
  ...

All workers are actively implementing tracks. No dispatch needed.
To add capacity, create a new worktree:

  git worktree add developer-{N} main

================================================================================
```

---

## Error States

### Kiloforge Not Initialized

If `.agent/kf/tracks.yaml` is not found on the primary branch:

```
ERROR: Kiloforge not initialized.
Run /kf-setup to initialize Kiloforge for this project.
```

### No Git Worktrees

If `git worktree list` shows only one worktree (the main worktree) and no `developer-*` folders:

```
ERROR: No developer worktrees found.

Create worktrees for developer agents:
  git worktree add developer-1 main
  git worktree add developer-2 main
```

### Track CLI Not Found

If `.agent/kf/bin/kf-track` is not executable:

```
ERROR: Kiloforge CLI tools not found.
Run /kf-setup to initialize the project, or check that .agent/kf/bin/ is intact.
```
