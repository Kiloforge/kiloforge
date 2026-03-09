# Implementation Plan: Rebrand Historical Conductor Records

**Track ID:** rebrand-historical-records_20260309063900Z

## Phase 0: Pre-flight — Reconcile with Existing kf State

- [ ] Task 0.1: Read current conductor state files (`product.md`, `tech-stack.md`, `index.md`, `tracks.md`, `workflow.md`) — these already use kf/kiloforge naming from prior setup. Note their current content to avoid overwriting.
- [ ] Task 0.2: Inventory all "kiloforge" references across `.agent/conductor/` — categorize by file type (archived tracks, active tracks, registry files) and replacement type (CLI command, product name, env var, path)

## Phase 1: Archived Track Metadata Titles

- [ ] Task 1.1: Update metadata.json title in `add-project-command_20260307122000Z` — "kf add" → "kf add"
- [ ] Task 1.2: Update metadata.json title in `implement-command_20260307125001Z` — "kf implement" → "kf implement"
- [ ] Task 1.3: Update metadata.json title in `fix-add-remote-url_20260307130000Z` — "kf add" → "kf add"
- [ ] Task 1.4: Update metadata.json title in `impl-conductor-lock-migration_20260308150001Z` — "kiloforge Lock API" → "Kiloforge Lock API"
- [ ] Task 1.5: Search all other archived metadata.json files for "kiloforge" and update

## Phase 2: Archived Track Spec and Plan Content

- [ ] Task 2.1: Bulk replace "kiloforge" CLI references with "kf" in all archived spec.md files
- [ ] Task 2.2: Bulk replace "kiloforge" CLI references with "kf" in all archived plan.md files
- [ ] Task 2.3: Replace `~/.kiloforge/` path references with `~/.kiloforge/` in archived files
- [ ] Task 2.4: Replace `KF_*` env var references with `KF_*` in archived files
- [ ] Task 2.5: Replace "kiloforge" product name references with "kiloforge" in archived files

## Phase 3: Active Track Spec and Plan Content

- [ ] Task 3.1: Bulk replace "kiloforge" CLI references with "kf" in all active (non-archived) spec.md files
- [ ] Task 3.2: Bulk replace "kiloforge" CLI references with "kf" in all active plan.md files
- [ ] Task 3.3: Replace `~/.kiloforge/` path references with `~/.kiloforge/` in active files
- [ ] Task 3.4: Replace `KF_*` env var references with `KF_*` in active files
- [ ] Task 3.5: Replace "kiloforge" product name references with "kiloforge" in active files

## Phase 4: Registry Files

- [ ] Task 4.1: Update any remaining "kiloforge" references in `.agent/conductor/tracks.md` completed entry titles
- [ ] Task 4.2: Update any remaining "kiloforge" references in `.agent/conductor/index.md` track listings

## Phase 5: Verification

- [ ] Task 5.1: Run case-insensitive grep for "kiloforge" across all `.agent/conductor/` files — should return zero matches
- [ ] Task 5.2: Verify all metadata.json files are valid JSON
- [ ] Task 5.3: Verify track IDs are unchanged in all files
- [ ] Task 5.4: Verify existing kf-era conductor files (`product.md`, `tech-stack.md`, `workflow.md`) are unchanged — no regressions from the merge
