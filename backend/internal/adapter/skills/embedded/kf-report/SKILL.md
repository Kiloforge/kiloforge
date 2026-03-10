---
name: kf-report
description: Generate comprehensive project timeline, velocity, SLOC, track
  summary, and cost estimate reports from kiloforge tracks and git history.
  Outputs markdown reports to .agent/kf/_reports/.
metadata:
  argument-hint: "[--timeline] [--velocity] [--tracks] [--phases] [--sloc] [--costs] [--full] [--detailed] [--since YYYY-MM-DD] [--until YYYY-MM-DD]"
  allowed-tools: Read Glob Grep Bash Write
---

# Kiloforge Project Report

Generate strict, reproducible project reports in **markdown format**: timeline with daily activity, development velocity, track summary, project phases, SLOC analysis, and multi-model cost estimates. All data is collected from git history and kiloforge track artifacts.

Reports are written to `.agent/kf/_reports/` as markdown files.

## Use this skill when

- The user wants a project timeline, velocity report, SLOC count, cost estimate, or track summary
- The user asks "how is the project going", "show me progress", "what's the velocity"
- The user needs a retrospective or status snapshot for stakeholders

## Do not use this skill when

- The task is about implementing or managing tracks (use kf-implement, kf-manage)
- The user wants real-time status of the current task (use kf-status)

## Pre-flight Checks

1. **Sync with primary branch** — your working tree may be stale:
   ```bash
   git reset --hard main
   ```
   This ensures you see the latest track statuses, completed/archived tracks, and project metadata. Without this, reports may show outdated data or miss recently completed tracks.

2. Verify Kiloforge is initialized:
   - Check `.agent/kf/product.md` exists
   - Check `.agent/kf/tracks.yaml` exists
   - If missing: Display error and suggest running `/kf-setup` first

3. Verify git repository:
   - Run `git rev-parse --is-inside-work-tree`
   - If not a git repo: Display error — git history is required for timeline and velocity

4. Determine date range:
   - If `--since` provided: use that date
   - If `--until` provided: use that date
   - Default: from first commit to today
   - Store as `$SINCE` and `$UNTIL` (YYYY-MM-DD format)

5. Detect compacted archives:
   - Check if `.agent/kf/archive-compactions.md` exists
   - If it exists, read it and parse the compaction table(s):
     - Extract each row: `Commit`, `Date`, `Completed`, `Uncompleted`, `First Created`, `Last Created`, `First Completed`, `Last Completed`
     - Store as `$COMPACTION_POINTS` array for use in subsequent sections
   - For each compaction point, recover track metadata from git history:
     ```bash
     git ls-tree --name-only {COMMIT_SHA} .agent/kf/tracks/_archive/
     ```
   - For each track directory found, recover its metadata:
     ```bash
     git show {COMMIT_SHA}:.agent/kf/tracks/_archive/{trackId}/metadata.json
     ```
   - Store recovered metadata in `$COMPACTED_TRACKS` array with fields: trackId, status, type, created, updated, tasks.total, tasks.completed, compaction_commit
   - Also recover the tracks.yaml index at that point for track titles:
     ```bash
     git show {COMMIT_SHA}:.agent/kf/tracks.yaml
     ```

5. Determine report sections:
   - `--full`: generate ALL sections
   - No flags (bare invocation): generate ALL sections (equivalent to `--full`)
   - One or more section flags provided: generate ONLY the flagged sections

6. Determine detail level:
   - Default: **summary mode** — show counts, aggregates, and top-N highlights only
   - `--detailed`: **detailed mode** — expand full lists of every track, every daily log entry, every file

7. Ensure output directory exists:
   ```bash
   mkdir -p .agent/kf/_reports
   ```

8. Read product name from `.agent/kf/product.md` (the `## Project Name` field).

---

## Output Format

All reports are rendered as **GitHub-flavored markdown** and written to a file:

```
.agent/kf/_reports/{YYYY-MM-DD}-full-report.md
```

If a section-specific flag is used instead of `--full`, use the section name:

```
.agent/kf/_reports/{YYYY-MM-DD}-timeline.md
.agent/kf/_reports/{YYYY-MM-DD}-velocity.md
.agent/kf/_reports/{YYYY-MM-DD}-sloc.md
.agent/kf/_reports/{YYYY-MM-DD}-costs.md
```

After writing the file, display the report content to the user AND confirm the file path.

---

## SECTION 1: Project Timeline (--timeline)

### Data Collection Procedure

**Step 1: Get daily commit activity**

```bash
git log --all --format='%ad|%s' --date=format:'%Y-%m-%d %H:%M'
```

Use Python to aggregate per day: commit count, time span (earliest–latest), key activity subjects.

**Step 2: Identify track completion events**

```bash
git log --all --format='%ad' --date=format:'%Y-%m-%d' --grep='mark track' | sort | uniq -c | sort -k2
```

**Step 3: Read track metadata for completion dates**

For each track in `.agent/kf/tracks/` and `_archive/`, plus `$COMPACTED_TRACKS` from pre-flight step 4:
- Extract `created`, `updated`, `status`
- Deduplicate by trackId

**Step 4: Classify key activity per day**

For each day, pick up to 3 representative activities:
- Prefer `feat:` and `fix:` commit subjects
- Fall back to `docs:` or `chore:` if no feature/fix commits
- Truncate each to ~60 chars

### Output Template

```markdown
## Project Timeline

| | |
|---|---|
| **Duration** | {$SINCE} – {$UNTIL} ({N} calendar days, {M} active days) |
| **Commits** | {total} total |
| **Tracks** | ~{lifetime} lifetime ({completed} completed, {pending} pending) |
| **Codebase** | {go_sloc} lines Go · {ts_sloc} lines TS/TSX · {migrations} SQL migrations |

### Daily Activity

| Date | Commits | Span | Tracks Completed | Key Activity |
|------|--------:|------|:----------------:|--------------|
| {date} ({day}) | {N} | {HH:MM}–{HH:MM} | {N or —} | {description} |
...
```

**Rules:**
- Show ALL active days (no top-N filtering in default mode)
- Gap days: show a single row with `—` in all columns and *(gap)* in Span
- Bold track completion counts >= 10
- Key Activity: semicolon-separated list of up to 3 short descriptions

---

## SECTION 2: Development Velocity (--velocity)

### Data Collection Procedure

**Step 1: Commits per day** — `git log --all --format='%ad' --date=format:'%Y-%m-%d' | sort | uniq -c | sort -k2`

**Step 2: Group by ISO week** — aggregate weekly commit totals

**Step 3: Compute track completion rates** — from "mark track" grep, count per day and per period

**Step 4: Identify velocity periods** — compare single-worker vs parallel-worker phases by checking when commit volume and track completion rate shift significantly

### Output Template

```markdown
## Velocity Progression

| Period | Commits | Description |
|--------|--------:|-------------|
| Week {N} ({date range}) | {N} | {summary} |
...

| Metric | Value |
|--------|-------|
| Commits/day (active) | ~{N} avg |
| Peak | {N} commits on {date} ({context}) |
| Single-worker track rate ({date range}) | ~{N} tracks/day avg |
| Parallel-worker track rate ({date range}) | ~{N} tracks/day avg **({X}x speedup)** |
```

**Rules:**
- Always show weekly buckets
- Always show commits/day average and peak
- If parallel worktree usage is detected (multiple worktree directories or high commit concurrency), compute and show the speedup factor
- Bold the speedup multiplier

---

## SECTION 3: Project Phases (--phases)

### Data Collection Procedure

Analyze the commit log to identify natural development phases by looking at:
- Shifts in commit message patterns (docs → feat → fix → chore)
- Velocity inflection points
- Track completion clustering
- Significant architectural changes mentioned in commit subjects

Group into phases with: name, date range, bullet-point highlights, commit count.

### Output Template

```markdown
## Project Phases

### Phase {N}: {Name} — *{date range}*
- {highlight}
- {highlight}
- {N} commits

...
```

---

## SECTION 4: Track Summary (--tracks)

### Data Collection Procedure

**Step 1:** Read `.agent/kf/tracks.yaml` and parse status field from JSON values: `completed`, `in-progress`, `pending`, `archived`

**Step 2:** Count on-disk archived tracks: `ls -d .agent/kf/tracks/_archive/*/ 2>/dev/null | wc -l`

**Step 3:** Count compacted tracks from `$COMPACTION_POINTS` (pre-flight step 4)

**Step 4:** For active/pending tracks, query progress via CLI:
```bash
.agent/kf/bin/kf-track list --active --json
.agent/kf/bin/kf-track-content progress {trackId}
```

**Step 5:** Detect blockers — check dependency graph via CLI:
```bash
.agent/kf/bin/kf-track deps list
```
Also scan track specs for `BLOCKED:`, `depends on`, `dependency` keywords:
```bash
.agent/kf/bin/kf-track-content show {trackId} --section spec
```

### Output Template

```markdown
## Track Summary

### Counts

| Category | Count |
|----------|------:|
| Completed (current on-disk) | {N} |
| Pending (not started) | {N} |
| Future (planned) | {N} |
| Compacted (git-only) — completed | {N} |
| Compacted (git-only) — uncompleted | {N} |
| **Lifetime total** | **~{N}** |

> **Compaction point:** `{SHORT_SHA}` ({date})
> Tracks spanning: {first_created} to {last_created}

### Pending Tracks

1. **{trackId}** — {title}
...

### Future Tracks

1. **{trackId}** — {title}
...

### Blockers

{If none:} No blockers identified.
{If found:} List each blocker with track ID and description.
```

**Detailed mode** (`--detailed`): expand completed/archived/compacted sections with individual track listings.

---

## SECTION 5: SLOC Analysis (--sloc)

### Data Collection Procedure

**Use `scc` for SLOC counting.** Try these methods in order:

**1. Docker (preferred — no local install needed):**

```bash
docker run --rm -v "$(pwd):/code:ro" boyter/scc:latest \
    --exclude-dir=node_modules,vendor,.git,.agent,rest/gen,storagev1 \
    --not-match='wire_gen\.go|\.gen\.|\.pb\.' \
    /code
```

**2. Local binary (if Docker unavailable or fails):**

```bash
which scc && scc --exclude-dir=node_modules,vendor,.git,.agent,rest/gen,storagev1 \
    --not-match='wire_gen\.go|\.gen\.|\.pb\.' \
    {project_root}
```

Both give: language, files, code, comments, blanks, lines, complexity, COCOMO estimate.

**3. Fallback (if neither Docker nor local scc available):** Use `git ls-files` + per-file `grep -c` counting (see classification rules below). Warn the user to install scc.

**Classification rules for fallback only:**

EXCLUDE: `*.md`, `*.txt`, config JSON/YAML/TOML, lock files, generated code (`*.gen.*`, `*.pb.go`), vendored deps, `.agent/*`, test fixtures, assets, tool config.

TEST: `*_test.go`, `*.test.ts`, `*.spec.*`, `test_*.py`, etc.

FUNCTIONAL: everything else with a code extension.

### Output Template

```markdown
## SLOC Report

> **Tool:** scc
> **Excludes:** {exclusion list}

| Language | Files | Code | Comments | Blanks | Lines | Complexity |
|----------|------:|-----:|---------:|-------:|------:|-----------:|
| {lang} | {N} | {N} | {N} | {N} | {N} | {N} |
...
| **TOTAL** | **{N}** | **{N}** | **{N}** | **{N}** | **{N}** | **{N}** |

**Processed:** {N} MB

### Breakdown

| Category | SLOC | Share |
|----------|-----:|------:|
| Backend ({lang}) | {N} | {pct}% |
| Frontend ({lang}) | {N} | {pct}% |
| Config/Infra | {N} | {pct}% |
| Schema/DB | {N} | {pct}% |
| Other | {N} | {pct}% |
```

**Rules:**
- Sort languages by Code descending
- Bold the TOTAL row
- Breakdown groups: Backend = Go/Python/Rust/Java, Frontend = TS/TSX/JS/JSX/CSS/HTML, Config/Infra = YAML/Shell/Makefile/Dockerfile, Schema/DB = SQL/Protobuf, Other = everything else

---

## SECTION 6: Cost Estimates (--costs)

### Data Collection Procedure

**Model 1: COCOMO** — from `scc` output (already computed with SLOC). If `scc` unavailable, compute manually:
- KLOC = total_code_lines / 1000
- Effort = 2.4 * KLOC^1.05 (person-months)
- Cost = Effort * $15,000/person-month (industry average loaded cost)
- Schedule = 2.5 * Effort^0.38 (months)
- People = Effort / Schedule

**Model 2: Function Point Analysis**

Analyze the system to identify:
- **External Inputs (EI)**: data entering the system (API endpoints accepting data, CLI commands, webhook receivers, file uploads). Weight: 4 per EI.
- **External Outputs (EO)**: data/reports leaving the system (notifications, metrics export, streaming output, downloads, rendered views). Weight: 5 per EO.
- **External Inquiries (EQ)**: input→output queries (listing endpoints with filters, search, status checks, graph queries). Weight: 4 per EQ.
- **Internal Logical Files (ILF)**: maintained data stores (database tables/collections, KV stores, in-memory caches). Weight: 10 per ILF.
- **External Interface Files (EIF)**: external systems referenced (databases, message brokers, cloud APIs, third-party services, container runtimes). Weight: 7 per EIF.

Identify these by reading `product.md`, `tech-stack.md`, track titles, and scanning commit subjects for integration keywords.

Compute:
- UFP = sum of (count * weight) for each category
- GSC total: rate 14 General System Characteristics 0-5 each based on system complexity (data communications, distributed processing, performance, config complexity, transaction rate, online entry, end-user efficiency, online update, complex processing, reusability, installation ease, operational ease, multiple sites, facilitate change)
- VAF = 0.65 + (0.01 * GSC_total)
- AFP = UFP * VAF
- Cost range: AFP * $500 (low), AFP * $1000 (mid), AFP * $1500 (high)

**Model 3: Parametric (SLOC-based)**

- Use total SLOC from scc
- Productivity range: 10-20 SLOC/hr (varies by language complexity)
- Hours = SLOC / productivity
- Cost = Hours * hourly_rate ($75 low, $112.50 mid, $150 high)

**Model 4: Effort by Analogy**

Based on the system's architectural scope (from product.md and tech-stack.md), estimate what comparable systems cost:
- Identify the key architectural components (e.g., "distributed pipeline + messaging + gRPC + web UI + CLI + container orchestration")
- Provide freelance/agency and in-house team estimates based on industry benchmarks

**Model 5: AI-Assisted Actual Cost**

- Active dev time: count of active days from git log
- Calendar time: first commit to last commit
- Estimated API cost: rough estimate based on commit volume and session patterns
- Human involvement: characterize the development model (solo + AI, team + AI, etc.)

### Output Template

```markdown
## Cost Estimates

### COCOMO (organic model, via scc)

| Metric | Value |
|--------|-------|
| Estimated Cost | ${N} |
| Schedule Effort | {N} months |
| People Required | {N} |

### Function Point Analysis

| Component | Count | Weight | Total |
|-----------|------:|-------:|------:|
| External Inputs (EI) | {N} | x 4 | {N} |
| External Outputs (EO) | {N} | x 5 | {N} |
| External Inquiries (EQ) | {N} | x 4 | {N} |
| Internal Logical Files (ILF) | {N} | x 10 | {N} |
| External Interface Files (EIF) | {N} | x 7 | {N} |
| **Unadjusted Function Points** | | | **{N}** |

| Metric | Value |
|--------|-------|
| Value Adjustment Factor | {N} (GSC: {N}/70) |
| Adjusted Function Points | {N} |

| Rate | Estimate |
|------|----------|
| Low ($500/FP) | ${N} |
| Mid ($1,000/FP) | ${N} |
| High ($1,500/FP) | ${N} |

### Parametric (SLOC-based)

| Metric | Value |
|--------|-------|
| SLOC | {N} |
| Productivity range | 10–20 SLOC/hr |
| Effort | {N} – {N} hours |
| Cost @ $75–150/hr | ${N} – ${N} |

### Effort by Analogy

> Comparable scope: {architectural description}

| Context | Estimate |
|---------|----------|
| Freelance/agency | ${N} – ${N} |
| In-house team ({duration}) | ${N} – ${N} |

### AI-Assisted Actual Cost

| Metric | Value |
|--------|-------|
| Active dev time | {N} days ({N} calendar days) |
| Estimated API cost | ~${N} – ${N} |
| Human time | {description} |

### Aggregate Cost Summary

| Model | Low | Mid | High |
|-------|----:|----:|-----:|
| COCOMO | — | ${N} | — |
| Function Point Analysis | ${N} | ${N} | ${N} |
| Parametric (SLOC) | ${N} | ${N} | ${N} |
| Effort by Analogy | ${N} | ${N} | ${N} |
| **Cross-model range** | **${low_of_lows}** | **${avg_of_mids}** | **${high_of_highs}** |

| Aggregate Metric | Value |
|------------------|-------|
| Median estimate | ~${N} |
| Geometric mean | ~${N} |
| Actual (AI-assisted) | ~${N} – ${N} |
| **Efficiency factor** | **~{N}x – {N}x cost reduction vs median** |
```

**Aggregate computation rules:**
- Cross-model low = minimum of all model lows (exclude COCOMO which has no low/high)
- Cross-model high = maximum of all model highs (use COCOMO as its own high)
- Cross-model mid = average of all model midpoints
- Median = median of the four model midpoints
- Geometric mean = (product of four model midpoints)^(1/4)
- Efficiency factor = median / actual_cost_range

---

## SECTION 7: Full Report (--full or default)

When `--full` is specified or no section flags are given, generate ALL sections in this order:

1. **Header** — H1 with project name, blockquote with generation date and period
2. **Timeline** (Section 1)
3. **Velocity** (Section 2)
4. **Project Phases** (Section 3)
5. **Track Summary** (Section 4)
6. **SLOC Report** (Section 5)
7. **Cost Estimates** (Section 6)
8. **Summary** — final section with key metrics table

### Full Report Header

```markdown
# Project Report: {Project Name}

> **Generated:** {YYYY-MM-DD}
> **Period:** {$SINCE} – {$UNTIL}

---
```

### Full Report Summary (footer)

```markdown
---

## Summary

Built in **{N} calendar days** with **{N} commits** across **{N} active days**.

| Metric | Value |
|--------|-------|
| SLOC | {N} ({primary languages}) |
| Files | {N} |
| Tracks (lifetime) | ~{N} |
| Tracks completed | {N} |
| Tracks pending | {N} |
| SQL migrations | {N} |
| Compaction points | {N} ({description}) |
| Peak velocity | {N} commits/day, {N} tracks/day ({context}) |
```

### Output File

Write the complete report to:

```
.agent/kf/_reports/{YYYY-MM-DD}-full-report.md
```

If a report for today already exists, overwrite it (reports are regenerated snapshots, not append-only).

---

## Error States

### Kiloforge Not Initialized

Display error and suggest: `Run /kf-setup to initialize Kiloforge for this project.`

### No Git History

Display error: `This report requires git commit history. Ensure you are in a git repository with at least one commit.`

### No Tracks Found

Display warning: `No tracks found in .agent/kf/tracks/. Track Summary and Phases sections will be empty. Timeline, Velocity, SLOC, and Costs can still be generated.`

### scc Not Available

Try Docker first: `docker run --rm -v "$(pwd):/code:ro" boyter/scc:latest /code`

If Docker also fails, display warning: `scc not found. Install with: brew install scc — or run via Docker: docker pull boyter/scc. Falling back to manual SLOC counting (less accurate).`

Then use the manual git ls-files + grep fallback for SLOC, and compute COCOMO manually.

---

## Performance Notes

- For repos with > 1000 commits, batch git log operations rather than per-file queries
- Use Python one-liners via `python3 -c "..."` for aggregation that would be complex in awk/sed
- SLOC: prefer `scc` — it is fast, accurate, and gives COCOMO for free
- **Compacted archive recovery**: `git show` calls are expensive. Recover ALL needed files (track.yaml) per compacted track in a single batch during pre-flight, then reuse across sections. Do NOT re-run `git show` per section.
- Cache intermediate results: if generating a full report, reuse git log data across sections rather than re-querying

## Critical Rules

1. **Output markdown** — All reports use GitHub-flavored markdown with tables, headers, blockquotes, and bold emphasis. No ASCII box-drawing.
2. **Write to file** — Always write the report to `.agent/kf/_reports/` AND display it to the user.
3. **Follow data collection procedures** — Do not improvise; use the specified git commands and aggregation methods.
4. **Use `scc` for SLOC** — Prefer `scc` over manual counting. Only fall back to manual if `scc` is unavailable.
5. **Include all 5 cost models** — COCOMO, FPA, Parametric, Analogy, and AI-Assisted. Always include the Aggregate Cost Summary.
6. **Always show the date range** — Every section must state its period.
7. **Round percentages to integers** — No decimal places in percentage displays.
8. **Sort tables by most relevant metric** — SLOC by code descending, tracks by status then date.
9. **Handle missing data gracefully** — If a section has no data, show a warning note, do not skip the section.
10. **Account for compacted archives** — Always check for and include compacted track data in totals.
11. **Always merge reports to the primary branch** — Follow the Merge Protocol below so reports are visible to all worktrees.

---

## Merge Protocol

After writing report files, merge them to the primary branch so all worktrees can access the reports. Reports only touch `.agent/kf/_reports/` — no post-rebase verification is needed.

### Step 1 — Resolve primary branch and record home branch

```bash
PRIMARY_BRANCH=$(git show main:.agent/kf/config.yaml 2>/dev/null | grep '^primary_branch:' | awk '{print $2}')
PRIMARY_BRANCH="${PRIMARY_BRANCH:-main}"
HOME_BRANCH=$(git branch --show-current)
MAIN_WORKTREE=$(git worktree list | grep -E '\['"$PRIMARY_BRANCH"'\]' | awk '{print $1}')
```

### Step 2 — Create a temporary branch from the primary branch

```bash
REPORT_BRANCH="report/$(date -u +%Y%m%d-%H%M%SZ)"
git checkout -b "$REPORT_BRANCH" "$PRIMARY_BRANCH"
```

### Step 3 — Write reports (existing logic)

Execute the report generation sections as described above. All report files are written to `.agent/kf/_reports/`.

### Step 4 — Commit report files

```bash
git add .agent/kf/_reports/
git commit -m "chore: add project report $(date -u +%Y-%m-%d)"
```

If there are no changes to commit (report identical to existing), skip the merge protocol and clean up:
```bash
git checkout "$HOME_BRANCH"
git branch -d "$REPORT_BRANCH"
```

### Step 5 — Acquire merge lock

Use the shared merge lock helper with a blocking timeout:

```bash
.agent/kf/bin/kf-merge-lock acquire --timeout 300
```

Start heartbeat after acquisition:
```bash
while true; do .agent/kf/bin/kf-merge-lock heartbeat; sleep 30; done &
HEARTBEAT_PID=$!
```

**From this point, release the lock on ANY failure:**
```bash
kill $HEARTBEAT_PID 2>/dev/null; wait $HEARTBEAT_PID 2>/dev/null
.agent/kf/bin/kf-merge-lock release
```

### Step 6 — Rebase onto latest primary branch

```bash
git rebase "$PRIMARY_BRANCH"
```

Report files rarely conflict. If a conflict occurs in `.agent/kf/_reports/`, accept ours (the new report overwrites the old):

```bash
git checkout --ours .agent/kf/_reports/
git add .agent/kf/_reports/
git rebase --continue
```

If any non-report file conflicts: release lock, report, and **HALT**.

### Step 7 — Fast-forward merge into the primary branch

```bash
git -C "$MAIN_WORKTREE" merge "$REPORT_BRANCH" --ff-only
```

On success:
```bash
kill $HEARTBEAT_PID 2>/dev/null; wait $HEARTBEAT_PID 2>/dev/null
.agent/kf/bin/kf-merge-lock release
```

On failure: release lock, report, and **HALT**.

### Step 8 — Cleanup and return to home branch

```bash
git checkout "$HOME_BRANCH"
git branch -d "$REPORT_BRANCH"
git reset --hard "$PRIMARY_BRANCH"
```

Report the file path to the user and confirm the merge.
