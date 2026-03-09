---
name: kf-status
description: Display project status, active tracks, and next actions
metadata:
  argument-hint: "[--ref <branch>]"
---

# Kiloforge Status

Display the current status of the Kiloforge project, including overall progress, active tracks, and next actions.

## Use this skill when

- User asks for project status, progress, or overview
- You need to see what tracks are active, pending, or ready to start

## Do not use this skill when

- You need to create or modify tracks (use `/kf-architect` or `/kf-developer`)
- The project has no Kiloforge artifacts (use `/kf-setup` first)

## Instructions

### Step 1 — Resolve primary branch

Read the primary branch from config:

```bash
PRIMARY_BRANCH=$(git show main:.agent/kf/config.yaml 2>/dev/null | grep '^primary_branch:' | awk '{print $2}')
PRIMARY_BRANCH="${PRIMARY_BRANCH:-main}"
```

### Step 2 — Run the status command

The `kf-track status` command generates the full factual status report:

```bash
.agent/kf/bin/kf-track status --ref ${PRIMARY_BRANCH}
```

This outputs all **data** sections:
- **Overall Progress** — track counts, task counts, progress bar
- **Active Tracks** — table with per-track task completion, deps count, and enriched status labels:
  - `CLAIMED` — in-progress tracks (actively being worked by a developer)
  - `AVAILABLE` — pending tracks with all dependencies satisfied (ready to claim)
  - `BLOCKED` — pending tracks with unmet dependencies
  - **Deps column** — shows `N/M met` or `no deps` per track
- **Current Focus** — claimed tracks with next pending task
- **Ready to Start** — pending tracks with all dependencies satisfied
- **Conflict Risk** — active conflict pairs from `conflicts.yaml`, showing risk level and notes (only appears when active pairs exist)
- **Blocked** — pending tracks with unmet dependencies and their current statuses

### Step 3 — Assess and recommend

The CLI output is purely factual. After presenting it, add your **assessment** — the parts that require judgment:

1. **Prioritization** — If multiple tracks are ready, recommend which to start first and why (e.g., "e2e-infra should go first — 11 tracks depend on it" or "these two are independent and can run in parallel across worktrees")

2. **Bottleneck analysis** — If many tracks are blocked on the same dependency, call it out (e.g., "e2e-infra is the critical path — completing it unblocks 11 tracks")

3. **Parallelism opportunities** — Identify ready tracks that are independent and can be worked simultaneously by different developer agents

4. **Recommendations** — Based on the state:
   - No in-progress tracks + idle workers → suggest starting ready tracks
   - No pending tracks → suggest `/kf-architect` to create new work
   - Many completed tracks not archived → suggest `/kf-bulk-archive`
   - In-progress tracks with low progress → note they may be stalled

### Single track detail

For a specific track, use:

```bash
.agent/kf/bin/kf-track show {trackId} --ref ${PRIMARY_BRANCH}
.agent/kf/bin/kf-track progress {trackId} --ref ${PRIMARY_BRANCH}
```

## Error States

### Kiloforge Not Initialized

If `kf-track status` fails with "No tracks.yaml found":

```
ERROR: Kiloforge not initialized.
Run /kf-setup to initialize Kiloforge for this project.
```

### No Tracks

If the output shows 0 total tracks:

```
Kiloforge is set up but no tracks have been created yet.
Run /kf-architect to create tracks from a feature request.
```
