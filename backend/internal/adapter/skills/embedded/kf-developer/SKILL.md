---
name: kf-developer
description: Receive a track ID, validate it is an active unclaimed track, then implement it following the kiloforge workflow. Worker role in the track generation => approval => push to worker pipeline.
metadata:
  argument-hint: "<track-id> [--disable-auto-merge] [--with-review]"
---

# Kiloforge Developer

Implement a kiloforge track in a parallel worktree workflow. Receives a track ID, validates it is available for work, then executes the full implementation cycle: branch, implement, verify, and merge.

## Use this skill when

- A track has been generated and approved via `/kf-architect`
- You have been assigned a specific track ID to implement
- You are a developer worker in a parallel worktree setup

## Do not use this skill when

- You need to create new tracks (use `/kf-architect` instead)
- The project has no Kiloforge artifacts (use `/kf-setup` first)
- You are working in a single-branch workflow (use `/kf-implement` instead)

---

## After Compaction

When entering the developer role, output this anchor line exactly:

```
ACTIVE ROLE: kf-developer — track {trackId} — skill at ~/.claude/skills/kf-developer/SKILL.md
```

This line is designed to survive compaction summaries. If you see it in your context but can no longer recall the full workflow, re-read the skill file before continuing. For project-specific values, re-read only what you need:

- Verification commands: `.agent/kf/workflow.md`
- Track list/statuses: `.agent/kf/bin/kf-track list`
- Track progress: `.agent/kf/bin/kf-track-content progress {trackId}`
- Main worktree path: `git worktree list`

---

## Worktree Convention

This agent is expected to run in a worktree whose folder name starts with `developer-` (e.g., `developer-1`, `developer-2`, `developer-3`). The corresponding branch name matches the folder name.

### Step 0 — Verify worktree identity

```bash
git branch --show-current
git rev-parse --git-common-dir 2>/dev/null
git rev-parse --git-dir 2>/dev/null
git worktree list
```

- The current branch should match `developer-*` (this is the **home branch**)
- If not on a `developer-*` branch, warn but continue
- Record the **main worktree path** from `git worktree list` — needed for merge operations
- Record the **home branch** (the `developer-*` branch) — to return to after merge

**All track state reads should come from main** (via `git show main:<path>`) to see the latest committed state, not the local working tree which may be stale.

---

## Phase 1: Validation

### Step 1 — Parse track ID

If no argument was provided:

```
ERROR: Track ID required.

Usage: /kf-developer <track-id>

To see available tracks, run `.agent/kf/bin/kf-track list` or /kf-architect to create new ones.
```

**HALT.**

### Step 2 — Verify Kiloforge is initialized

Check these files exist (read from main):
```bash
git show main:.agent/kf/product.md > /dev/null 2>&1
git show main:.agent/kf/workflow.md > /dev/null 2>&1
git show main:.agent/kf/tracks.yaml > /dev/null 2>&1
```

If missing: Display error and suggest `/kf-setup`. **HALT.**

### Step 3 — Validate track exists and is claimable

1. **Check track exists on main:**
   ```bash
   git show main:.agent/kf/tracks/{trackId}/track.yaml > /dev/null 2>&1
   ```
   If not found:
   ```
   ERROR: Track not found — {trackId}

   The track .agent/kf/tracks/{trackId}/ does not exist on main.
   This track ID may be incorrect, or the track may not have been merged to main yet.

   Available tracks (from main):
   {output from `.agent/kf/bin/kf-track list --active`}
   ```
   **HALT.**

2. **Check track status on main** using `kf-track`:
   ```bash
   .agent/kf/bin/kf-track get {trackId}
   ```

   - If track status is `completed`:
     ```
     ERROR: Track already complete — {trackId}

     This track has already been implemented and marked complete on main.
     ```
     **HALT.**

   - If track is marked `in-progress`, check if another worker has it:

3. **Check if another worker has claimed it:**
   ```bash
   git worktree list
   git branch --list 'feature/*' 'bug/*' 'chore/*' 'refactor/*'
   ```

   Look for a branch matching `*/{trackId}`. If found:
   ```
   ERROR: Track already claimed — {trackId}

   Branch {type}/{trackId} already exists, indicating another worker is implementing this track.

   Worktree: {worktree path if identifiable}
   Branch:   {branch name}

   Choose a different track or coordinate with the other worker.
   ```
   **HALT.**

4. **Check track has required files on main:**
   ```bash
   git show main:.agent/kf/tracks/{trackId}/track.yaml > /dev/null 2>&1
   ```
   If missing:
   ```
   ERROR: Track incomplete — {trackId}

   Missing track.yaml on main.
   This track may need to be regenerated via /kf-architect.
   ```
   **HALT.**

5. **Check dependency graph — all prerequisites must be completed:**
   ```bash
   git show main:.agent/kf/tracks/deps.yaml 2>/dev/null
   ```

   Run the dependency check:
   ```bash
   .agent/kf/bin/kf-track deps check {trackId}
   ```

   If the command exits non-zero (BLOCKED), it will list unmet dependencies:
   ```
   ERROR: Dependencies not met — {trackId}

   {output from kf-track deps check}

   Wait for these tracks to complete, or ask the architect to restructure dependencies.
   ```
   **HALT.**

   If `deps.yaml` does not exist, skip this check (backwards compatibility).

### Step 4 — Enter developer mode

```
================================================================================
                    KILOFORGE DEVELOPER — TRACK VALIDATED
================================================================================

Track:    {trackId}
Title:    {title from track.yaml}
Type:     {type}
Tasks:    {total tasks from track.yaml plan}
Phases:   {total phases}

Beginning implementation:
1. Create branch {type}/{trackId} from main
2. Implement all tasks following the plan
3. Verify and prepare for merge
================================================================================
```

**Proceed immediately to Phase 2 (Setup).**

Output the compaction anchor:
```
ACTIVE ROLE: kf-developer — track {trackId} — skill at ~/.claude/skills/kf-developer/SKILL.md
```

---

## Phase 2: Setup

### Step 5 — Sync home branch and create implementation branch

The `developer-*` home branch is a dead/marker branch. Its only purpose is recording the point at which this worker last synced with main. Sync it now so the marker reflects where we're starting from, then branch off main:

```bash
# Sync home branch to main (updates the marker)
git reset --hard main

# Create implementation branch from main
git checkout -b {type}/{trackId} main
```

Branch naming: `{type}/{trackId}` where type comes from metadata (e.g., `feature/auth_20250115100000Z`).

> **Note:** The implementation branch is created from `main`, not from the home branch. The `git reset --hard main` just before serves as a timestamp marker — it records when this worker last synced, which can be useful for diagnosing staleness.

### Step 6 — Load workflow configuration

Read `.agent/kf/workflow.md` and parse:
- Verification commands (e.g., `make test`, `make e2e`)
- TDD strictness level
- Commit strategy

### Step 7 — Load track context

Load track context via CLI (now from the working tree, which is based on main):
```bash
# Full track content
.agent/kf/bin/kf-track-content show {trackId}

# Or section by section for large tracks:
.agent/kf/bin/kf-track-content show {trackId} --section spec
.agent/kf/bin/kf-track-content show {trackId} --section plan
.agent/kf/bin/kf-track-content progress {trackId}

# Check conflict risk with other active tracks
.agent/kf/bin/kf-track conflicts list {trackId}
```

Also read project context:
- `.agent/kf/product.md`
- `.agent/kf/tech-stack.md`
- `.agent/kf/code_styleguides/` (if present)

---

## Phase 3: Implementation

### Step 8 — Execute the plan

Follow the exact same implementation workflow as `/kf-implement`:

- Execute each task in the plan sequentially
- Follow TDD workflow if configured in `workflow.md`
- Commit after each task completion using the commit strategy from `workflow.md`
- Update task completion via CLI: `.agent/kf/bin/kf-track-content task {trackId} <phase>.<task> --done`
- Check progress: `.agent/kf/bin/kf-track-content progress {trackId}`
- Run phase verification at the end of each phase
- **Do NOT pause between phases** — proceed continuously through all phases without waiting for user approval

### Step 9 — Mark track complete

After all tasks are done, update all tracking files and commit:

1. **Update track status** using `kf-track` (updates `tracks.yaml`, prunes `deps.yaml`, and cleans `conflicts.yaml` automatically):
   ```bash
   .agent/kf/bin/kf-track update {trackId} --status completed
   ```
2. Verify all tasks are marked done: `.agent/kf/bin/kf-track-content progress {trackId}`

```bash
git add .agent/kf/tracks.yaml .agent/kf/tracks/deps.yaml .agent/kf/tracks/conflicts.yaml .agent/kf/tracks/{trackId}/
git commit -m "chore: mark track {trackId} complete"
```

---

## Phase 4: Review (only with `--with-review`)

This entire phase is **skipped** unless `--with-review` was provided. Without it, proceed directly to Phase 5 (Merge).

### Step 10r — Discover own session ID

Find this agent's session ID so it can be posted on the PR for later `claude --resume`:

```bash
PROJECT_DIR=$(echo "$PWD" | sed 's|/|-|g; s|^-||')
SESSION_ID=$(ls -t ~/.claude/projects/-${PROJECT_DIR}/*.jsonl 2>/dev/null | head -1 | xargs basename | sed 's/.jsonl//')
echo "Session ID: $SESSION_ID"
```

Record the session ID for use in PR comments.

### Step 10r.1 — Determine remote and platform

Resolve the git remote and PR platform:

1. **Remote name:** Use env var `KF_REMOTE` if set, otherwise `origin`
2. **PR platform:** Use env var `KF_PR_PLATFORM` if set (`github` or `gitea`), otherwise auto-detect:
   ```bash
   REMOTE_URL=$(git remote get-url ${REMOTE_NAME})
   ```
   - If URL contains `github.com` → `github` (use `gh` CLI)
   - Otherwise → `gitea` (use `tea` CLI or raw API)

### Step 10r.2 — Push branch and create PR

```bash
git push ${REMOTE_NAME} {type}/{trackId}
```

Create PR with session ID embedded:

**GitHub:**
```bash
gh pr create \
  --base main \
  --head {type}/{trackId} \
  --title "{type}: {track title} ({trackId})" \
  --body "$(cat <<'EOF'
## Track

- **Track ID:** {trackId}
- **Type:** {type}
- **Tasks:** {completed}/{total}

## Developer Session

\`\`\`
DEVELOPER_SESSION={session-id}
DEVELOPER_WORKTREE={worktree-folder-name}
RESUME_CMD=claude --resume {session-id}
\`\`\`

---
_Created by kf-developer with --with-review_
EOF
)"
```

**Gitea:**
```bash
tea pr create \
  --base main \
  --head {type}/{trackId} \
  --title "{type}: {track title} ({trackId})" \
  --description "..." # same body as above
```

Record the PR number/URL.

### Step 10r.3 — Wait for review

```
================================================================================
                    PR CREATED — WAITING FOR REVIEW
================================================================================
Track:      {trackId} - {title}
Branch:     {type}/{trackId}
PR:         {pr-url}
Session:    {session-id}

To trigger a reviewer:
  claude --worktree reviewer-1 -p "/kf-reviewer {pr-url}"

Or in an existing reviewer worktree:
  /kf-reviewer {pr-url}

This agent is WAITING. It will resume when review feedback is provided.
Say "review complete" or paste review feedback to continue.
================================================================================
```

**CRITICAL: HALT and wait for user input.** The agent stays alive, preserving full context. It will be unblocked when:
- The user types feedback directly
- The user says "review complete" or "approved"
- A script or the reviewer agent sends input to this terminal

### Step 10r.4 — Process review feedback

When unblocked, determine the review outcome:

1. **Read PR review status:**

   **GitHub:**
   ```bash
   gh pr view {pr-number} --json reviews,comments
   ```

   **Gitea:**
   ```bash
   tea pr view {pr-number}
   ```

2. **If approved (no changes requested):** Proceed to Phase 5 (Merge). The merge process will also clean up the remote branch after merge.

3. **If changes requested:** Read the review comments, then:
   - Address each comment (fix code, reply to comments)
   - Commit fixes
   - Push updates:
     ```bash
     git push ${REMOTE_NAME} {type}/{trackId}
     ```
   - Reply to PR comments explaining fixes (via `gh pr comment` or `tea pr comment`)
   - Return to Step 10r.3 (wait for next review round)

### Step 10r.5 — Review cycle limit

Track the number of review iterations. If the review cycle exceeds **5 iterations** without approval:

```
================================================================================
                    REVIEW CYCLE LIMIT REACHED
================================================================================
Track:      {trackId} - {title}
PR:         {pr-url}
Iterations: 5/5

The review process has reached its maximum iteration count.
Manual intervention required.
================================================================================
```

**HALT and wait for user guidance.**

---

## Phase 5: Merge

### Step 10 — Report completion and merge (or wait)

By default, auto-merge is enabled — proceed directly to the merge sequence after implementation completes.

If `--disable-auto-merge` **was** provided (and `--with-review` was not used or review is already approved):

```
================================================================================
                    TRACK COMPLETE — READY TO MERGE
================================================================================
Track:      {trackId} - {title}
Branch:     {type}/{trackId}
Tasks:      {completed}/{total}

Ready to merge. Say "merge" to begin the lock -> rebase -> verify -> merge sequence.
================================================================================
```

**Wait for explicit "merge" command before proceeding.**

If `--disable-auto-merge` was **not** provided (default): skip the pause and proceed directly to the merge sequence.

If `--with-review` **was** provided and review is approved: proceed directly to the merge sequence (review approval implies merge authorization).

### Step 11 — Merge sequence

When the user says "merge" (or immediately if auto-merge is enabled (default) / post-review-approval), execute the full merge protocol:

#### 11a. Acquire merge lock

The merge lock supports two modes: **HTTP** (via kiloforge lock API) with automatic fallback to **mkdir** (local filesystem). The HTTP mode provides TTL-based crash recovery and server-side long-poll for `--auto-merge`.

**Setup — determine lock mode and define helpers:**

```bash
ORCH_URL="${KF_ORCH_URL:-http://localhost:4001}"
HOLDER="$(basename $(pwd))"  # e.g., "developer-1"
LOCK_MODE=""
HEARTBEAT_PID=""

# Check if orchestrator lock API is available
is_orch_running() {
  curl -sf "$ORCH_URL/health" -o /dev/null 2>/dev/null
}

# Release lock (call on ANY failure after acquire)
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

# Start heartbeat in background (HTTP mode only)
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

**CRITICAL: NEVER force-remove another worker's lock.** Do not `rm -rf` the lock directory or force-release an HTTP lock held by another worker. The lock exists to coordinate merges — removing it risks corrupting the merge of the worker that holds it. If the lock appears stale, report it and wait for user instructions. Only the lock holder or the user may release it.

**Default (auto-merge enabled):** Use blocking acquire. HTTP mode uses server-side long-poll (timeout_seconds: 300). mkdir mode uses a polling loop:

```bash
if is_orch_running; then
  # HTTP mode — blocking with server-side long-poll
  if curl -sf -X POST "$ORCH_URL/-/api/locks/merge/acquire" \
    -H "Content-Type: application/json" \
    -d "{\"holder\": \"$HOLDER\", \"ttl_seconds\": 120, \"timeout_seconds\": 300}" \
    --max-time 310 -o /dev/null 2>/dev/null; then
    LOCK_MODE="http"
    echo "Merge lock acquired (HTTP)"
  else
    echo "MERGE LOCK TIMEOUT — could not acquire after 300s"
    exit 1
  fi
else
  # mkdir fallback — polling loop
  LOCK_DIR="$(git rev-parse --git-common-dir)/merge.lock"
  ATTEMPT=0
  while ! mkdir "$LOCK_DIR" 2>/dev/null; do
    ATTEMPT=$((ATTEMPT + 1))
    echo "MERGE LOCK HELD — waiting for lock... (attempt $ATTEMPT)"
    echo "Lock info: $(cat "$LOCK_DIR/info" 2>/dev/null || echo 'unknown')"
    sleep 10
  done
  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $HOLDER" > "$LOCK_DIR/info" 2>/dev/null
  LOCK_MODE="mkdir"
  echo "Merge lock acquired (mkdir fallback) after $ATTEMPT retries"
fi
start_heartbeat
```

Run this as a single bash command with an appropriate timeout (e.g., 300 seconds). The HTTP long-poll replaces the sleep loop for faster acquisition.

**With `--disable-auto-merge`:** Try once. If the lock is held, report and **HALT** — wait for the user to say "merge" to retry.

```bash
if is_orch_running; then
  # HTTP mode — non-blocking (timeout_seconds: 0)
  if curl -sf -X POST "$ORCH_URL/-/api/locks/merge/acquire" \
    -H "Content-Type: application/json" \
    -d "{\"holder\": \"$HOLDER\", \"ttl_seconds\": 120, \"timeout_seconds\": 0}" \
    -o /dev/null 2>/dev/null; then
    LOCK_MODE="http"
    echo "Merge lock acquired (HTTP)"
  else
    echo "MERGE LOCK HELD — Another worker is currently merging."
    echo "Say 'merge' to retry."
    exit 1
  fi
else
  # mkdir fallback
  LOCK_DIR="$(git rev-parse --git-common-dir)/merge.lock"
  if ! mkdir "$LOCK_DIR" 2>/dev/null; then
    echo "MERGE LOCK HELD — Another worker is currently merging."
    echo "Lock info: $(cat "$LOCK_DIR/info" 2>/dev/null || echo 'unknown')"
    echo "Say 'merge' to retry."
    exit 1
  fi
  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $HOLDER" > "$LOCK_DIR/info" 2>/dev/null
  LOCK_MODE="mkdir"
  echo "Merge lock acquired (mkdir fallback — orchestrator unavailable)"
fi
start_heartbeat
```

**From this point: call `release_lock` on ANY failure.**

#### 11b. Rebase onto latest main

```bash
if ! git rebase main; then
  git rebase --abort 2>/dev/null || true
  release_lock
  echo "REBASE FAILED — lock released"
  exit 1
fi
echo "Rebase succeeded"
```

On conflict: lock released automatically. Report and **HALT**.

#### 11c. Post-rebase verification

Run the full verification suite from `workflow.md` (e.g., `make test`, `make e2e`).

On failure: call `release_lock`, report, **HALT**.

#### 11d. Fast-forward merge into main

```bash
if git -C {main-worktree-path} merge {type}/{trackId} --ff-only; then
  release_lock
  echo "MERGE SUCCEEDED — lock released"
else
  release_lock
  echo "MERGE FAILED — lock released"
  exit 1
fi
```

On failure: lock released. Report and **HALT**.

#### 11e. Cleanup — return to home branch

```bash
# Verify merge
git -C {main-worktree-path} log --oneline -3

# Delete implementation branch (safe — it's been merged)
git branch -d {type}/{trackId}

# If --with-review was used: clean up remote branch and close PR
# GitHub: gh pr close {pr-number} (if not auto-closed) && git push ${REMOTE_NAME} --delete {type}/{trackId}
# Gitea: tea pr close {pr-number} && git push ${REMOTE_NAME} --delete {type}/{trackId}

# Return to developer home branch
git checkout {developer-home-branch}

# Sync home branch to main (updates the marker to post-merge state)
git reset --hard main
```

Report:

```
================================================================================
                         MERGE COMPLETE
================================================================================
Track:       {trackId} - {title}
Merged into: main
Branch:      {type}/{trackId} (deleted)
Home branch: {developer-home-branch} (synced to main)

Developer is ready for next track.
================================================================================
```

---

## Error Handling Summary

| Error                      | Action                                                   |
|----------------------------|----------------------------------------------------------|
| No track ID provided       | Display usage, **HALT**                                  |
| Track not found on main    | List available tracks from main, **HALT**                |
| Track already complete     | Notify, **HALT**                                         |
| Track already claimed      | Show claiming worker/branch, **HALT**                    |
| Track missing spec/plan    | Suggest regeneration, **HALT**                           |
| Kiloforge not initialized  | Suggest `/kf-setup`, **HALT**                     |
| Verification failure       | Report details, offer fix/retry/wait                     |
| Merge lock held            | Report, wait for other worker                            |
| Rebase conflict            | Abort rebase, release lock, report, **HALT**             |
| Post-rebase verify failure | Release lock, report, offer fix/retry/abort              |
| Merge not fast-forwardable | Release lock, offer re-rebase or abort                   |

---

## Flags Summary

| Flag | Effect |
|------|--------|
| (none) | Default: implement, auto-merge (poll merge lock if held) |
| `--disable-auto-merge` | Pause after implementation; wait for explicit "merge" command |
| `--with-review` | After implementation: push, create PR, wait for review, then merge. Review approval implies merge authorization |
| `--disable-auto-merge --with-review` | Review cycle runs, and after approval pause for explicit "merge" command |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KF_ORCH_URL` | `http://localhost:4001` | Orchestrator URL for HTTP lock API |

## Merge Lock Modes

The merge lock uses dual-mode acquisition:

1. **HTTP mode** — Preferred when kiloforge orchestrator is running. Uses TTL (120s), heartbeat (every 30s), and server-side long-poll for `--auto-merge`. Crash recovery via automatic TTL expiry.
2. **mkdir mode** — Fallback when orchestrator is unreachable. Uses `$(git rev-parse --git-common-dir)/merge.lock` directory. No TTL — requires manual cleanup on crashes.

Detection is automatic: if `curl -sf $ORCH_URL/health` succeeds, HTTP mode is used.

## Critical Rules

1. **ALWAYS validate before implementing** — never start work on an invalid or claimed track
2. **ALWAYS read track state from main** — use `git show main:<path>`, not local working tree
3. **NEVER push to remote unless `--with-review`** — without review flag, all branches are local only
4. **Auto-merge is the default** — only pause for explicit "merge" command when `--disable-auto-merge` is provided
5. **ALWAYS verify after rebase** — full verification after rebase, before merge
6. **ALWAYS use --ff-only** — clean fast-forward merges only
7. **ONE merge at a time** — enforce via cross-worktree merge lock (HTTP preferred, mkdir fallback)
8. **HALT on any failure** — do not continue past errors without user input
9. **Follow workflow.md** — all TDD, commit, and verification rules apply
10. **Return to home branch** — always checkout back to `developer-*` branch after merge
11. **Clean up remote on merge** — if `--with-review`, delete remote branch and close PR after merge
12. **ALWAYS send heartbeat** — start heartbeat after lock acquire, stop after release
13. **NEVER force-remove another worker's lock** — if the merge lock is held, HALT and wait for user instructions. Do not `rm -rf` the lock directory or force-release HTTP locks held by others.
