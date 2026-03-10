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

This agent is expected to run in a worktree whose folder name starts with `worker-` (e.g., `worker-1`, `worker-2`, `worker-3`). The corresponding branch name matches the folder name.

### Step 0 — Verify worktree identity and resolve primary branch

```bash
git branch --show-current
git rev-parse --git-common-dir 2>/dev/null
git rev-parse --git-dir 2>/dev/null
git worktree list
```

- The current branch should match `worker-*` (this is the **home branch**)
- If not on a `worker-*` branch, warn but continue
- Record the **main worktree path** from `git worktree list` — needed for merge operations
- Record the **home branch** (the `worker-*` branch) — to return to after merge

**Resolve the primary branch** from `.agent/kf/config.yaml`:

```bash
PRIMARY_BRANCH=$(git show main:.agent/kf/config.yaml 2>/dev/null | grep '^primary_branch:' | awk '{print $2}')
PRIMARY_BRANCH="${PRIMARY_BRANCH:-main}"
echo "Primary branch: $PRIMARY_BRANCH"
```

Record `PRIMARY_BRANCH` for all subsequent operations. If `config.yaml` doesn't exist or has no `primary_branch`, default to `main`.

**IMPORTANT: Your home branch is almost always stale.** Other architects and developers merge to the primary branch continuously. Before doing ANY validation or track lookups, you MUST sync first:

```bash
git reset --hard ${PRIMARY_BRANCH}
```

This ensures your local working tree has the latest tracks.yaml, deps.yaml, and track directories. Without this, `kf-track get`, `kf-track list`, and `kf-track deps check` will operate on stale data and may report tracks as "not found" even though they exist on the primary branch.

**All track state reads should come from the primary branch.** Either sync first (preferred) or use `--ref ${PRIMARY_BRANCH}` on commands that support it.

---

## Phase 1: Validation

### Step 0b — Sync with primary branch

Before any validation, sync the working tree to the latest primary branch state:

```bash
git reset --hard ${PRIMARY_BRANCH}
```

This is mandatory — skip this and you risk operating on stale track data.

### Step 1 — Parse track ID

If no argument was provided:

```
ERROR: Track ID required.

Usage: /kf-developer <track-id>

To see available tracks, run `.agent/kf/bin/kf-track list` or /kf-architect to create new ones.
```

**HALT.**

### Step 2 — Verify Kiloforge is initialized

Check these files exist (read from primary branch):
```bash
git show ${PRIMARY_BRANCH}:.agent/kf/product.md > /dev/null 2>&1
git show ${PRIMARY_BRANCH}:.agent/kf/workflow.md > /dev/null 2>&1
git show ${PRIMARY_BRANCH}:.agent/kf/tracks.yaml > /dev/null 2>&1
```

If missing: Display error and suggest `/kf-setup`. **HALT.**

### Step 3 — Validate track exists and is claimable

1. **Check track exists and get its status:**
   ```bash
   .agent/kf/bin/kf-track get {trackId}
   ```
   If not found (exits non-zero):
   ```
   ERROR: Track not found — {trackId}

   Available tracks:
   {output from `.agent/kf/bin/kf-track list --active`}
   ```
   **HALT.**

   If track status is `completed`:
   ```
   ERROR: Track already complete — {trackId}

   This track has already been implemented and marked complete on ${PRIMARY_BRANCH}.
   ```
   **HALT.**

   If track is marked `in-progress`, check if another worker has it.

2. **Check if another worker has claimed it:**
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

3. **Check dependency graph — all prerequisites must be completed:**

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
1. Create branch {type}/{trackId} from ${PRIMARY_BRANCH}
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

The `worker-*` home branch is a dead/marker branch. Its only purpose is recording the point at which this worker last synced with the primary branch. Sync it now so the marker reflects where we're starting from, then branch off:

```bash
# Sync home branch to primary branch (updates the marker)
git reset --hard ${PRIMARY_BRANCH}

# Create implementation branch from primary branch
git checkout -b {type}/{trackId} ${PRIMARY_BRANCH}
```

Branch naming: `{type}/{trackId}` where type comes from metadata (e.g., `feature/auth_20250115100000Z`).

> **Note:** The implementation branch is created from `${PRIMARY_BRANCH}`, not from the home branch. The `git reset --hard` just before serves as a timestamp marker — it records when this worker last synced, which can be useful for diagnosing staleness.

### Step 6 — Load workflow configuration

Read `.agent/kf/workflow.md` and parse:
- Verification commands (e.g., `make test`, `make e2e`)
- TDD strictness level
- Commit strategy

### Step 7 — Load track context

Load track context via CLI (now from the working tree, which is based on the primary branch):
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
  --base ${PRIMARY_BRANCH} \
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
  --base ${PRIMARY_BRANCH} \
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

The merge lock uses the shared `kf-merge-lock` helper script, which handles dual-mode (HTTP/mkdir) acquisition, heartbeat, and release automatically. See `.agent/kf/bin/kf-merge-lock help` for full details.

**CRITICAL: NEVER force-remove another worker's lock.** Do not `rm -rf` the lock directory or force-release an HTTP lock held by another worker. The lock exists to coordinate merges — removing it risks corrupting the merge of the worker that holds it. If the lock appears stale, report it and wait for user instructions. Only the lock holder or the user may release it.

**Default (auto-merge enabled):** Use blocking acquire with timeout:

```bash
.agent/kf/bin/kf-merge-lock acquire --timeout 300
```

**With `--disable-auto-merge`:** Try once (non-blocking). If held, report and **HALT**:

```bash
.agent/kf/bin/kf-merge-lock acquire --timeout 0
```

If acquire fails, report lock status and wait for user to say "merge" to retry.

**After acquire, start heartbeat in background:**

```bash
while true; do .agent/kf/bin/kf-merge-lock heartbeat; sleep 30; done &
HEARTBEAT_PID=$!
```

**From this point: release the lock on ANY failure:**

```bash
kill $HEARTBEAT_PID 2>/dev/null; wait $HEARTBEAT_PID 2>/dev/null
.agent/kf/bin/kf-merge-lock release
```

#### 11b. Rebase onto latest primary branch

```bash
if ! git rebase ${PRIMARY_BRANCH}; then
  echo "Rebase conflict detected — resolving track state files..."
fi
```

**On conflict — simplified resolution for track state files:**

Track state files (`tracks.yaml`, `deps.yaml`, `conflicts.yaml`) are append/update structures where the primary branch's version is always ground truth (it reflects all other workers' completions). Accept theirs, then re-apply your changes via CLI:

```bash
# Accept primary branch's version of all track state files
git checkout --theirs .agent/kf/tracks.yaml .agent/kf/tracks/deps.yaml .agent/kf/tracks/conflicts.yaml 2>/dev/null
git add .agent/kf/tracks.yaml .agent/kf/tracks/deps.yaml .agent/kf/tracks/conflicts.yaml 2>/dev/null

# Continue the rebase (repeat if multiple conflicting commits)
git rebase --continue
```

After rebase completes, re-apply the developer's track state changes:

```bash
# Re-mark track complete (updates tracks.yaml, prunes deps.yaml and conflicts.yaml)
.agent/kf/bin/kf-track update {trackId} --status completed

# Amend the last commit with re-applied changes
git add .agent/kf/tracks.yaml .agent/kf/tracks/deps.yaml .agent/kf/tracks/conflicts.yaml
git commit --amend --no-edit
```

If a **non-state file** conflicts (e.g., source code files), that is a genuine conflict — release lock (`kf-merge-lock release`), report, and **HALT**.

If rebase still fails after resolution: release lock (`kf-merge-lock release`), report, **HALT**.

#### 11c. Post-rebase verification

Run the full verification suite from `workflow.md` (e.g., `make test`, `make e2e`).

On failure: release lock (`kf-merge-lock release`), report, **HALT**.

#### 11d. Fast-forward merge into primary branch

```bash
if git -C {primary-branch-worktree-path} merge {type}/{trackId} --ff-only; then
  kill $HEARTBEAT_PID 2>/dev/null; wait $HEARTBEAT_PID 2>/dev/null
  .agent/kf/bin/kf-merge-lock release
  echo "MERGE SUCCEEDED — lock released"
else
  kill $HEARTBEAT_PID 2>/dev/null; wait $HEARTBEAT_PID 2>/dev/null
  .agent/kf/bin/kf-merge-lock release
  echo "MERGE FAILED — lock released"
  exit 1
fi
```

On failure: lock released. Report and **HALT**.

#### 11e. Cleanup — return to home branch

```bash
# Verify merge
git -C {primary-branch-worktree-path} log --oneline -3

# Return to developer home branch FIRST (can't delete current branch)
git checkout {worker-home-branch}

# Delete implementation branch (safe — it's been merged and we've switched away)
git branch -d {type}/{trackId}

# If --with-review was used: clean up remote branch and close PR
# GitHub: gh pr close {pr-number} (if not auto-closed) && git push ${REMOTE_NAME} --delete {type}/{trackId}
# Gitea: tea pr close {pr-number} && git push ${REMOTE_NAME} --delete {type}/{trackId}

# Sync home branch to primary branch (updates the marker to post-merge state)
git reset --hard ${PRIMARY_BRANCH}
```

Report:

```
================================================================================
                         MERGE COMPLETE
================================================================================
Track:       {trackId} - {title}
Merged into: ${PRIMARY_BRANCH}
Branch:      {type}/{trackId} (deleted)
Home branch: {worker-home-branch} (synced to ${PRIMARY_BRANCH})

Developer is ready for next track.
================================================================================
```

---

## Error Handling Summary

| Error                      | Action                                                   |
|----------------------------|----------------------------------------------------------|
| No track ID provided       | Display usage, **HALT**                                  |
| Track not found            | List available tracks from primary branch, **HALT**      |
| Track already complete     | Notify, **HALT**                                         |
| Track already claimed      | Show claiming worker/branch, **HALT**                    |
| Track missing spec/plan    | Suggest regeneration, **HALT**                           |
| Kiloforge not initialized  | Suggest `/kf-setup`, **HALT**                     |
| Verification failure       | Report details, offer fix/retry/wait                     |
| Merge lock held            | Report (`kf-merge-lock status`), wait for other worker |
| Rebase conflict (state files) | Accept theirs, continue rebase, re-apply via CLI     |
| Rebase conflict (source code) | Release lock, report, **HALT**                       |
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

The merge lock is managed by the shared `.agent/kf/bin/kf-merge-lock` helper, which supports dual-mode acquisition:

1. **HTTP mode** — Preferred when kiloforge orchestrator is running. Uses TTL (120s), heartbeat (every 30s), and server-side long-poll for `--auto-merge`. Crash recovery via automatic TTL expiry.
2. **mkdir mode** — Fallback when orchestrator is unreachable. Uses `$(git rev-parse --git-common-dir)/merge.lock` directory. PID-based stale detection with auto-cleanup.

Detection is automatic. Run `kf-merge-lock status` to inspect current lock state.

## Critical Rules

1. **ALWAYS validate before implementing** — never start work on an invalid or claimed track
2. **ALWAYS read track state from the primary branch** — resolve from `.agent/kf/config.yaml`, default `main`. Use `git show ${PRIMARY_BRANCH}:<path>`, not local working tree
3. **NEVER push to remote unless `--with-review`** — without review flag, all branches are local only
4. **Auto-merge is the default** — only pause for explicit "merge" command when `--disable-auto-merge` is provided
5. **ALWAYS verify after rebase** — full verification after rebase, before merge
6. **ALWAYS use --ff-only** — clean fast-forward merges only
7. **ONE merge at a time** — enforce via cross-worktree merge lock (HTTP preferred, mkdir fallback)
8. **HALT on any failure** — do not continue past errors without user input
9. **Follow workflow.md** — all TDD, commit, and verification rules apply
10. **Return to home branch** — always checkout back to `worker-*` branch after merge
11. **Clean up remote on merge** — if `--with-review`, delete remote branch and close PR after merge
12. **ALWAYS send heartbeat** — start heartbeat after lock acquire, stop after release
13. **NEVER force-remove another worker's lock** — if the merge lock is held, HALT and wait for user instructions. Do not `rm -rf` the lock directory or force-release HTTP locks held by others.
