---
name: kf-compact-archive
description: Remove archived and completed track directories from the working tree while preserving recovery via git history with rich metadata tracking
---

# Compact Archive

Remove archived (`_archive/`) and completed (`[x]`) track directories from the working tree while preserving access via git history. Records rich metadata about each compaction point for future recovery.

## Use this skill when

- The `_archive/` directory has accumulated track folders and you want to reclaim working tree space
- Completed `[x]` tracks have accumulated in the index and you want to clean them up
- The user says "compact archive" or "compact"
- After a bulk archive, when the user wants to clean up archived directories

## Do not use this skill when

- Kiloforge is not initialized (use `/kf-setup` first)
- There is no `_archive/` directory AND no `[x]` completed tracks in the index
- The user wants to archive completed tracks without compacting — use `/kf-bulk-archive` instead

## Instructions

### Step 1: Identify compactable tracks

Scan **both** sources of completed/archived tracks:

1. **Archived tracks**: List all track directories in `.agent/conductor/tracks/_archive/` (if the directory exists)
2. **Completed index tracks**: Parse `.agent/conductor/tracks.md` and find all rows marked `[x]` in the main table (above the `## Archived Tracks` section) that have a corresponding directory in `.agent/conductor/tracks/{trackId}/`

If both sources are empty, report "Nothing to compact" and stop.

### Step 2: Record compaction point

```bash
HASH=$(git rev-parse HEAD)
```

### Step 3: Gather metadata from all compactable tracks

For each track directory — from both `_archive/` and from the main `tracks/` dir (for `[x]` completed tracks) — read its `metadata.json` to collect:

- **Status**: `complete`, `superseded`, `deprecated`, `dropped`, or other non-complete status
- **Created timestamp**: from `metadata.json` `created` field (or parse track ID datetime suffix as fallback)
- **Completed timestamp**: from `metadata.json` `completedAt` or `updated` field (only for completed tracks)

Compute summary stats across ALL compactable tracks (both sources combined):

| Stat | Description |
| ---- | ----------- |
| `completed_count` | Tracks with status `complete` |
| `uncompleted_count` | Tracks with any other status (superseded, deprecated, dropped, etc.) |
| `first_created` | Earliest created ISO timestamp across all compacted tracks |
| `last_created` | Latest created ISO timestamp across all compacted tracks |
| `first_completed` | Earliest completed ISO timestamp across completed tracks (or `---` if none) |
| `last_completed` | Latest completed ISO timestamp across completed tracks (or `---` if none) |

### Step 4: Update archive-compactions.md

Create or append to `.agent/conductor/archive-compactions.md`.

**If creating the file for the first time**, use this format:

```markdown
# Archive Compaction Points

Archived track data can be recovered by checking out the commit before each compaction point.

## Source: `.agent/conductor/tracks.md`
## Archive: `.agent/conductor/tracks/_archive/`

If the tracks index or archive folder location is ever changed, declare the new paths below
and start a new compaction table under that declaration.

| Commit | Date | Completed | Uncompleted | First Created | Last Created | First Completed | Last Completed |
| ------ | ---- | --------- | ----------- | ------------- | ------------ | --------------- | -------------- |
| {HASH} | {YYYY-MM-DDTHH:MM:SSZ} | {completed_count} | {uncompleted_count} | {first_created} | {last_created} | {first_completed} | {last_completed} |
```

**If the file already exists**, append a new row to the **current** table (the one under the most recent Source/Archive declaration). Do NOT create a new table unless the paths have changed.

**If the Source or Archive paths have changed**, append a new section:

```markdown
## Source: `{new_tracks_md_path}`
## Archive: `{new_archive_path}`

| Commit | Date | Completed | Uncompleted | First Created | Last Created | First Completed | Last Completed |
| ------ | ---- | --------- | ----------- | ------------- | ------------ | --------------- | -------------- |
| {HASH} | {YYYY-MM-DDTHH:MM:SSZ} | ... |
```

### Step 5: Delete compacted track directories

Remove directories from **both** sources:

```bash
# Remove archived tracks
rm -rf .agent/conductor/tracks/_archive/

# Remove completed [x] track directories from the main tracks dir
rm -rf .agent/conductor/tracks/{trackId}/  # for each [x] track identified in Step 1
```

**Important:** Only delete directories for tracks that were identified in Step 1. Do NOT delete directories for `[ ]`, `[~]`, or `[future]` tracks.

### Step 6: Clean up tracks.md

Update `.agent/conductor/tracks.md`:

1. **Remove all `[x]` completed rows** from the main index table. Only `[ ]` (pending), `[~]` (in progress), and `[future]` rows should remain.
2. **Remove all content under the `## Archived Tracks` section**. This includes all batch archive entries added by `/kf-bulk-archive`. The section header itself can be kept (empty) or removed — either is fine.

Both the `[x]` index entries and the "## Archived Tracks" batch listings are preserved in `archive-compactions.md` and recoverable via git history, so keeping them in `tracks.md` would just be orphaned references to directories that no longer exist.

### Step 7: Commit

```bash
git add .agent/conductor/tracks/ .agent/conductor/tracks.md .agent/conductor/archive-compactions.md
git commit -m "chore: compact archive ({completed_count} completed, {uncompleted_count} uncompleted — recover from {HASH})"
```

### Step 8: Report

```
================================================================================
                     COMPACT ARCHIVE COMPLETE
================================================================================
Commit before compaction:  {HASH}
Tracks removed:            {total_count}
  From _archive/:          {archive_count}
  From index [x]:          {index_completed_count}
  Completed:               {completed_count}
  Uncompleted:             {uncompleted_count}
Date range (created):      {first_created} — {last_created}
Date range (completed):    {first_completed} — {last_completed}

Recovery commands:
  git show {HASH}:.agent/conductor/tracks.md
  git ls-tree {HASH} .agent/conductor/tracks/_archive/
  git ls-tree {HASH} .agent/conductor/tracks/
  git show {HASH}:.agent/conductor/tracks/{trackId}/spec.md
================================================================================
```

## Recovery Reference

To recover compacted tracks, use the commit hash from the compactions table:

```bash
# Recover the full tracks.md index at that point (includes [x] entries)
git show {HASH}:.agent/conductor/tracks.md

# List all archived tracks at that point
git ls-tree {HASH} .agent/conductor/tracks/_archive/

# List all track directories (including completed ones) at that point
git ls-tree {HASH} .agent/conductor/tracks/

# Recover a specific track's files
git show {HASH}:.agent/conductor/tracks/{trackId}/spec.md
git show {HASH}:.agent/conductor/tracks/{trackId}/metadata.json
git show {HASH}:.agent/conductor/tracks/_archive/{trackId}/spec.md
```
