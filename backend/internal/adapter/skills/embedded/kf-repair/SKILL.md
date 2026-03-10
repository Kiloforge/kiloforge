---
name: kf-repair
description: Audit Kiloforge system integrity (track registry, dependency graph, conflict pairs, worktree state, merge lock, cross-references) and perform repairs directly or escalate to /kf-architect for larger issues.
allowed-tools: Read Glob Grep Bash
metadata:
  argument-hint: "[--full | --registry | --deps | --conflicts | --worktrees] [--fix]"
  model: sonnet
---

# Kiloforge System Repair

Systematic health audit and repair agent for Kiloforge project management artifacts. Diagnoses integrity issues across track registry, dependency graph, conflict pairs, worktree/branch state, merge lock, and configuration — then applies safe repairs or escalates structural issues.

## Use this skill when

- Track commands fail unexpectedly or return stale data
- After interrupted operations (killed agents, failed merges, aborted rebases)
- Multiple agents report conflicting track states
- Dependency graph seems wrong (tracks blocked that shouldn't be, or unblocked prematurely)
- Merge lock appears stuck or stale
- Periodic health check before starting a batch of work
- After bulk operations (archive, compact, mass status changes)

## Do not use this skill when

- You need to audit codebase reliability (use `/kf-reliability`)
- You need to review track content quality (use `/kf-reviewer`)
- You need to create new tracks (use `/kf-architect`)
- The project has no Kiloforge artifacts (use `/kf-setup` first)

---

## After Compaction

When entering the repair role, output this anchor line exactly:

```
ACTIVE ROLE: kf-repair — skill at ~/.claude/skills/kf-repair/SKILL.md
```

---

## Arguments

| Flag | Effect |
|------|--------|
| (none) | Full audit across all dimensions |
| `--full` | Full audit (same as no flags) |
| `--registry` | Audit only track registry integrity |
| `--deps` | Audit only dependency graph integrity |
| `--conflicts` | Audit only conflict pairs integrity |
| `--worktrees` | Audit only worktree and branch state |
| `--fix` | Apply safe repairs automatically without prompting |

Multiple dimension flags can be combined: `--registry --deps` audits both.

---

## Introduction (no arguments)

When invoked without arguments or with `--help`, display:

```
================================================================================
                       KILOFORGE SYSTEM REPAIR
================================================================================

I audit Kiloforge system integrity and repair issues across 8 dimensions:

  1. Track Registry     — tracks.yaml entry validity
  2. Track Directories  — track.yaml files match registry
  3. Dependency Graph   — deps.yaml references, cycles, stale entries
  4. Conflict Pairs     — conflicts.yaml ordering, active references
  5. Worktree State     — branches match tracks, orphaned branches
  6. Merge Lock         — stale lock detection
  7. Configuration      — config.yaml validity, primary branch exists
  8. Cross-References   — consistency across all data files

Usage:
  /kf-repair                     Full audit (all dimensions)
  /kf-repair --registry --deps   Audit specific dimensions
  /kf-repair --fix               Auto-apply safe repairs

I fix safe issues directly (prune stale deps, re-sort files, release
stale locks) and escalate structural problems to /kf-architect.
================================================================================
```

Then proceed to the full audit.

---

## Phase 1: Pre-flight

### Step 1 — Resolve primary branch

```bash
PRIMARY_BRANCH=$(git show main:.agent/kf/config.yaml 2>/dev/null | grep '^primary_branch:' | awk '{print $2}')
PRIMARY_BRANCH="${PRIMARY_BRANCH:-main}"
echo "Primary branch: $PRIMARY_BRANCH"
```

### Step 2 — Verify Kiloforge is initialized

Check these files exist:
```bash
git show ${PRIMARY_BRANCH}:.agent/kf/product.md > /dev/null 2>&1
git show ${PRIMARY_BRANCH}:.agent/kf/tracks.yaml > /dev/null 2>&1
```

If missing: suggest `/kf-setup` and **HALT**.

### Step 3 — Load data files

Read from the primary branch to ensure freshness:

```bash
git show ${PRIMARY_BRANCH}:.agent/kf/tracks.yaml > /tmp/kf-repair-tracks.yaml
git show ${PRIMARY_BRANCH}:.agent/kf/tracks/deps.yaml > /tmp/kf-repair-deps.yaml 2>/dev/null
git show ${PRIMARY_BRANCH}:.agent/kf/tracks/conflicts.yaml > /tmp/kf-repair-conflicts.yaml 2>/dev/null
git show ${PRIMARY_BRANCH}:.agent/kf/config.yaml > /tmp/kf-repair-config.yaml 2>/dev/null
```

### Step 4 — Determine audit scope

Parse arguments to decide which dimensions to audit. Default is all dimensions.

---

## Phase 2: Audit Dimensions

For each check, record:
- **Status**: PASS / WARN / FAIL
- **Severity**: CRITICAL / HIGH / MEDIUM / LOW
- **Finding**: What was found
- **Repair**: What can be done (auto-fix or escalate)

### Dimension 1: Track Registry Integrity

Validate `tracks.yaml` entries:

```bash
.agent/kf/bin/kf-track list --all --ref ${PRIMARY_BRANCH}
```

Check each entry for:

| Check | Severity | Description |
|-------|----------|-------------|
| 1.1 Required fields | HIGH | Every entry has `title`, `status`, `type`, `created` |
| 1.2 Valid status | HIGH | Status is one of: `pending`, `in-progress`, `completed`, `archived` |
| 1.3 Valid type | MEDIUM | Type is one of: `feature`, `bug`, `chore`, `refactor` |
| 1.4 No duplicates | HIGH | No duplicate track IDs |
| 1.5 Alphabetical sort | LOW | Entries sorted alphabetically by ID |
| 1.6 Date format | LOW | `created` and `updated` fields are valid YYYY-MM-DD |

**Repair actions:**
- 1.2/1.3: Report invalid values, suggest correction — **escalate**
- 1.4: Report duplicates — **escalate** (manual resolution needed)
- 1.5: Re-sort entries — **auto-fix**
- 1.6: Report invalid dates — **auto-fix** if clearly parseable

### Dimension 2: Track Directory Consistency

```bash
# List all track directories
ls .agent/kf/tracks/*/track.yaml 2>/dev/null

# Cross-reference with registry
.agent/kf/bin/kf-track list --all --ref ${PRIMARY_BRANCH}
```

| Check | Severity | Description |
|-------|----------|-------------|
| 2.1 Directory exists | HIGH | Every pending/in-progress track has a directory with `track.yaml` |
| 2.2 No orphans | MEDIUM | No track directories without a registry entry (except `_archive`) |
| 2.3 Status match | HIGH | `track.yaml` status matches `tracks.yaml` status |
| 2.4 Required sections | MEDIUM | Active tracks have spec and plan sections in `track.yaml` |

**Repair actions:**
- 2.1: Missing directory for active track — **escalate** (track may need spec regeneration)
- 2.2: Orphaned directory — **auto-fix** (delete if no useful content, or register)
- 2.3: Status mismatch — **auto-fix** (sync track.yaml to match tracks.yaml as source of truth)

### Dimension 3: Dependency Graph Integrity

```bash
.agent/kf/bin/kf-track deps list --ref ${PRIMARY_BRANCH}
```

| Check | Severity | Description |
|-------|----------|-------------|
| 3.1 Valid references | HIGH | All track IDs in deps.yaml exist in tracks.yaml |
| 3.2 No stale entries | MEDIUM | No completed/archived tracks referenced as dependencies |
| 3.3 No self-deps | HIGH | No track depends on itself |
| 3.4 No cycles | CRITICAL | Dependency graph is a DAG (no circular dependencies) |
| 3.5 Alphabetical sort | LOW | Entries sorted alphabetically |

**Cycle detection approach:**
```bash
# Use kf-track deps check on each pending/in-progress track
for track in $(kf-track list --active --format ids); do
  kf-track deps check "$track" 2>&1
done
```

**Repair actions:**
- 3.1: Remove references to non-existent tracks — **auto-fix**
- 3.2: Prune completed/archived tracks from deps — **auto-fix**
- 3.3: Remove self-dependencies — **auto-fix**
- 3.4: Report cycle — **escalate** (architect must restructure)
- 3.5: Re-sort — **auto-fix**

### Dimension 4: Conflict Pairs Integrity

```bash
.agent/kf/bin/kf-track conflicts list --ref ${PRIMARY_BRANCH} 2>/dev/null
```

| Check | Severity | Description |
|-------|----------|-------------|
| 4.1 Valid references | HIGH | Both tracks in each pair exist in tracks.yaml |
| 4.2 Strict ordering | MEDIUM | Pair keys are alphabetically ordered (a < b) |
| 4.3 Active tracks only | MEDIUM | No completed/archived tracks in conflict pairs |
| 4.4 Valid risk levels | LOW | Risk level is one of: `low`, `medium`, `high` |
| 4.5 No duplicate pairs | MEDIUM | No duplicate conflict pair entries |

**Repair actions:**
- 4.1: Remove pairs referencing non-existent tracks — **auto-fix**
- 4.2: Re-order pairs — **auto-fix**
- 4.3: Prune completed/archived tracks from pairs — **auto-fix**
- 4.5: Deduplicate — **auto-fix**

### Dimension 5: Worktree and Branch State

```bash
git worktree list
git branch --list 'feature/*' 'bug/*' 'chore/*' 'refactor/*'
.agent/kf/bin/kf-merge-lock status 2>/dev/null
```

| Check | Severity | Description |
|-------|----------|-------------|
| 5.1 Branch-track match | MEDIUM | Implementation branches (`feature/*`, etc.) match active track IDs |
| 5.2 Orphaned branches | LOW | No implementation branches for completed/non-existent tracks |
| 5.3 Merge lock | HIGH | Merge lock is not stale (PID alive for mkdir, TTL valid for HTTP) |
| 5.4 Primary branch exists | CRITICAL | The configured primary branch exists as a git branch |
| 5.5 Worktree health | MEDIUM | All worktrees in `git worktree list` are valid (paths exist) |

**Repair actions:**
- 5.2: Report orphaned branches — suggest `git branch -d` — **prompt user**
- 5.3: Release stale lock — **auto-fix** (only if PID is dead / TTL expired)
- 5.4: Invalid primary branch — **escalate** (config error)
- 5.5: Prune dead worktrees — `git worktree prune` — **auto-fix**

### Dimension 6: Configuration Integrity

```bash
cat .agent/kf/config.yaml
```

| Check | Severity | Description |
|-------|----------|-------------|
| 6.1 Config exists | MEDIUM | `config.yaml` exists |
| 6.2 Primary branch valid | HIGH | `primary_branch` value is a real git branch |
| 6.3 Required fields | MEDIUM | Has `project_name` and `primary_branch` |

### Dimension 7: Merge Lock State

```bash
.agent/kf/bin/kf-merge-lock status
```

| Check | Severity | Description |
|-------|----------|-------------|
| 7.1 Lock not stale | HIGH | If locked, the holding process is still alive |
| 7.2 Lock dir clean | LOW | No leftover lock artifacts from crashed processes |

**Repair actions:**
- 7.1: Stale lock (dead PID) — release via `kf-merge-lock release` — **auto-fix**
- 7.1: Lock held by live process — **do NOT touch** (report and move on)

### Dimension 8: Cross-Reference Consistency

This dimension validates relationships across all data files:

| Check | Severity | Description |
|-------|----------|-------------|
| 8.1 Deps reference active tracks | HIGH | Every track in deps.yaml is pending or in-progress in tracks.yaml |
| 8.2 Conflicts reference active tracks | HIGH | Every track in conflicts.yaml is pending or in-progress |
| 8.3 Track dirs match registry | HIGH | Set of track directories equals set of active track IDs |
| 8.4 In-progress has branch | MEDIUM | Every in-progress track has a matching implementation branch |

These are aggregate checks that combine findings from earlier dimensions. No separate repair — repairs come from the individual dimensions.

---

## Phase 3: Repair Procedures

### Safe auto-fixes (applied with `--fix` or after user confirmation)

| Repair | Dimensions | Action |
|--------|-----------|--------|
| Prune stale deps | 3.1, 3.2, 3.3 | Remove entries from deps.yaml referencing completed/archived/non-existent tracks or self-references |
| Prune stale conflicts | 4.1, 4.3 | Remove entries from conflicts.yaml referencing completed/archived/non-existent tracks |
| Re-sort files | 1.5, 3.5 | Sort tracks.yaml and deps.yaml entries alphabetically |
| Release stale lock | 7.1 | Release merge lock if holding PID is dead |
| Prune dead worktrees | 5.5 | Run `git worktree prune` |
| Sync status | 2.3 | Update track.yaml status to match tracks.yaml |
| Reorder conflict pairs | 4.2 | Ensure pair keys are alphabetically ordered |

**Implementation approach for repairs:**

```bash
# Use kf-track CLI for status updates
.agent/kf/bin/kf-track update {trackId} --status {correct-status}

# Use kf-track deps prune for stale dependency cleanup
# (If CLI supports it — otherwise edit deps.yaml directly)

# Use kf-merge-lock release for stale locks
.agent/kf/bin/kf-merge-lock release

# Use git worktree prune for dead worktrees
git worktree prune
```

When `--fix` is NOT passed, list all proposed repairs and ask:
```
The following repairs are available:
  1. [AUTO-FIX] Prune 3 stale entries from deps.yaml
  2. [AUTO-FIX] Release stale merge lock (PID 12345 is dead)
  3. [ESCALATE] Track foo_123Z is missing its directory — needs /kf-architect

Apply auto-fixes? (y/n)
```

When `--fix` IS passed, apply all auto-fixes silently and only report escalations.

### Escalation rules

Escalate to `/kf-architect` when:
- A track is missing its directory entirely (needs spec/plan regeneration)
- Dependency cycle detected (needs restructuring)
- Duplicate track IDs found (needs manual dedup)
- Primary branch doesn't exist (config needs fixing)
- Source code conflict during merge (needs manual resolution)

Format escalation as:
```
ESCALATE: {issue description}
  Recommended action: /kf-architect to {specific fix}
```

---

## Phase 4: Summary Report

Output a terminal summary after all checks:

```
================================================================================
                       KF SYSTEM HEALTH REPORT
================================================================================

Dimension                  Checks  Pass  Warn  Fail
─────────────────────────  ──────  ────  ────  ────
1. Track Registry              6     5     1     0
2. Track Directories           4     4     0     0
3. Dependency Graph            5     4     0     1
4. Conflict Pairs              5     5     0     0
5. Worktree State              5     4     1     0
6. Configuration               3     3     0     0
7. Merge Lock                  2     2     0     0
8. Cross-References            4     3     1     0
─────────────────────────  ──────  ────  ────  ────
Total                         34    30     3     1

Critical Findings:
  [FAIL] 3.2 deps.yaml references completed track 'foo_123Z' — PRUNED
  [WARN] 1.5 tracks.yaml not alphabetically sorted — FIXED
  [WARN] 5.2 Orphaned branch 'feature/old_track' — suggest deletion
  [WARN] 8.1 deps.yaml references non-active track — PRUNED

Repairs Applied: 2
Escalations: 0

Overall: HEALTHY (with minor repairs applied)
================================================================================
```

Status summary line:
- **HEALTHY** — 0 FAIL, 0-3 WARN
- **DEGRADED** — 1-3 FAIL or 4+ WARN
- **CRITICAL** — 4+ FAIL or any CRITICAL severity FAIL

---

## Error Handling

| Error | Action |
|-------|--------|
| Kiloforge not initialized | Suggest `/kf-setup`, **HALT** |
| Cannot read primary branch | Report git error, **HALT** |
| CLI tools missing | Report missing tool path, **HALT** |
| Repair fails | Report error, skip repair, continue audit |
| Lock held by live process | Report holder info, do NOT release, continue |
