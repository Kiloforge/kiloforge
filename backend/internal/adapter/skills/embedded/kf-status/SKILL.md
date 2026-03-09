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

The `kf-track status` command generates the full project status report automatically:

```bash
.agent/kf/bin/kf-track status --ref ${PRIMARY_BRANCH}
```

This outputs:
- Project name and overall progress (track counts, progress bar)
- Active tracks table with per-track task completion
- Current focus (in-progress tracks with next task)
- Ready-to-start tracks (pending with all dependencies met)
- Next actions (what to start or continue)

### Step 3 — Present and assess

Display the command output to the user. If the agent is in a worktree that may be out of sync, always use `--ref ${PRIMARY_BRANCH}` to read from the authoritative branch.

Add any assessment or recommendations based on the output:
- If tracks are blocked, explain what they're waiting on
- If many tracks are ready, suggest prioritization
- If no tracks are pending, suggest `/kf-architect` to create new ones

### Single track detail

For a specific track, use:

```bash
.agent/kf/bin/kf-track-content show {trackId}
.agent/kf/bin/kf-track-content progress {trackId}
```

Or with `--ref` to read from a branch:

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
