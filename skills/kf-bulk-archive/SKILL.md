---
name: kf-bulk-archive
description: Archive all completed tracks by moving their directories to _archive and updating tracks.md
---

# Bulk Archive

Move all completed track directories into `_archive/` and update `tracks.md` in a single operation.

## Use this skill when

- All (or many) active tracks are marked `[x]` complete and need archiving
- The user says "bulk archive" or "archive completed tracks"
- Cleaning up the active tracks table after a round of parallel work

## Do not use this skill when

- Kiloforge is not initialized (use `/kf-setup` first)
- There are no completed tracks to archive
- The user wants to compact (delete) archived directories — use `/kf-compact-archive` instead

## Instructions

### Step 1: Identify completed tracks

Read `.agent/conductor/tracks.md` and find all rows marked `[x]` in the active table (above the `## Archived` section).

If none are found, report "No completed tracks to archive" and stop.

### Step 2: Move directories

```bash
mkdir -p .agent/conductor/tracks/_archive/
```

For each completed track:
```bash
mv .agent/conductor/tracks/{trackId}/ .agent/conductor/tracks/_archive/{trackId}/
```

### Step 3: Update tracks.md

1. Remove the `[x]` rows from the active table
2. Add a new batch archive entry under `## Archived Tracks`:

```markdown
### Batch Archive {YYYY-MM-DDTHH:MM:SSZ} ({count} tracks)

All active tracks completed and archived at {YYYY-MM-DDTHH:MM:SSZ}.

| Track ID | Title | Reason |
| -------- | ----- | ------ |
| {trackId} | {title} | Completed |
...
```

Place the new batch entry **above** any previous archive batches so the most recent is first.

### Step 4: Commit

```bash
git add .agent/conductor/tracks.md .agent/conductor/tracks/
git commit -m "chore: bulk archive {count} completed tracks"
```

### Step 5: Report

```
================================================================================
                      BULK ARCHIVE COMPLETE
================================================================================
Tracks archived:  {count}
Commit:           {hash}
Active tracks:    {remaining} remaining

Want to compact the archive? Run /kf-compact-archive
================================================================================
```
