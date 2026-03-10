---
name: kf-conflict-resolver
description: Resolve git merge conflicts during push or pull sync operations. Fetches remote changes, merges, resolves conflicts (or escalates), and completes the sync.
metadata:
  argument-hint: "<direction> <remote-branch> [--ssh-key-path <path>]"
---

# Kiloforge Conflict Resolver

Resolve git merge conflicts that arise during push or pull sync operations. Fetches remote changes, attempts a merge, resolves conflicts intelligently, and completes the sync. Escalates to the user when conflicts are ambiguous or high-risk.

## Use this skill when

- A push failed with a non-fast-forward rejection (diverged history)
- A pull detected diverged branches (409 response)
- The user or system has spawned a conflict resolver agent

## Do not use this skill when

- There is no actual conflict (push/pull succeeded normally)
- The user wants manual control over the merge process
- The conflict involves binary files (escalate immediately)

---

## Arguments

This skill receives context from the spawning system:

| Argument | Required | Description |
|----------|----------|-------------|
| `direction` | Yes | `push` or `pull` — determines the sync direction |
| `remote-branch` | Yes | The remote branch to sync with (e.g., `main`) |
| `--ssh-key-path` | No | Path to SSH key for authenticated push/pull |

If direction or remote-branch is missing, **HALT** with usage instructions.

---

## Phase 1: Validate Environment

### Step 1 — Parse arguments

Extract direction and remote-branch from the arguments. Validate:

- `direction` must be `push` or `pull`
- `remote-branch` must be a non-empty string

If invalid:
```
ERROR: Invalid arguments.

Usage: /kf-conflict-resolver <push|pull> <remote-branch> [--ssh-key-path <path>]

  direction:     "push" or "pull"
  remote-branch: the remote branch name (e.g., "main")
```
**HALT.**

### Step 2 — Verify git state

```bash
# Confirm we're in a git repository
git rev-parse --is-inside-work-tree

# Check current branch
git branch --show-current

# Check for uncommitted changes
git status --porcelain
```

If there are uncommitted changes, auto-commit them first:
```bash
git add -A
git commit -m "wip: auto-save before conflict resolution"
```

### Step 3 — Verify remote is reachable

```bash
git ls-remote --exit-code origin 2>&1
```

If the remote is unreachable:
```
ERROR: Cannot reach remote "origin".

Check your network connection and SSH key configuration.
```
**HALT.**

Report:
```
================================================================================
                  CONFLICT RESOLVER — ENVIRONMENT VALIDATED
================================================================================
Direction:     {direction}
Remote branch: {remote-branch}
Local branch:  {current-branch}
Remote:        origin
================================================================================
```

---

## Phase 2: Fetch and Merge

### Step 4 — Fetch latest remote state

```bash
git fetch origin {remote-branch}
```

If fetch fails, report the error and **HALT**.

### Step 5 — Check divergence

```bash
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse origin/{remote-branch})
BASE=$(git merge-base HEAD origin/{remote-branch})

echo "Local:  $LOCAL"
echo "Remote: $REMOTE"
echo "Base:   $BASE"
```

Determine the state:

| Local vs Base | Remote vs Base | State |
|---------------|----------------|-------|
| Same | Same | **Already in sync** — nothing to do |
| Ahead | Same | **Local ahead** — for push: just push; for pull: nothing to do |
| Same | Ahead | **Remote ahead** — for push: rebase and push; for pull: fast-forward merge |
| Ahead | Ahead | **Diverged** — merge required |

If already in sync or no conflict:
```
No conflict detected — branches are compatible.
```
For push direction, push directly. For pull direction, fast-forward merge. **Done.**

### Step 6 — Attempt merge

```bash
git merge origin/{remote-branch} --no-edit
```

If the merge succeeds with no conflicts:
```
Auto-merge succeeded — no manual conflict resolution needed.
```
Proceed to Phase 4 (Complete Sync).

If the merge has conflicts, proceed to Phase 3.

---

## Phase 3: Resolve Conflicts

### Step 7 — Inventory conflicts

```bash
git diff --name-only --diff-filter=U
```

List all conflicted files. Categorize them:

| Category | Action |
|----------|--------|
| Binary files | **Escalate immediately** — cannot auto-resolve |
| Generated files (e.g., `go.sum`, `package-lock.json`) | Regenerate after resolving source files |
| Track state files (`.agent/kf/tracks.yaml`, `deps.yaml`, `conflicts.yaml`) | Accept remote version, re-apply local track updates |
| Source code files | Analyze and resolve intelligently |

### Step 8 — Resolve track state files

Track state files are append/update structures. The remote version is ground truth:

```bash
# Accept remote version for all track state files
for f in .agent/kf/tracks.yaml .agent/kf/tracks/deps.yaml .agent/kf/tracks/conflicts.yaml; do
  if git diff --name-only --diff-filter=U | grep -q "$f"; then
    git checkout --theirs "$f"
    git add "$f"
    echo "Resolved (accept remote): $f"
  fi
done
```

### Step 9 — Resolve source code conflicts

For each remaining conflicted file:

1. **Read the file** to see the conflict markers (`<<<<<<<`, `=======`, `>>>>>>>`)
2. **Read the surrounding context** to understand what each side changed
3. **Assess complexity:**
   - **Simple** — One side adds code, the other modifies nearby code. Both changes are independent.
   - **Moderate** — Both sides modify the same function or block, but changes are logically separable.
   - **Complex** — Both sides modify the same lines with incompatible logic, or the correct resolution requires understanding business logic.

4. **For simple and moderate conflicts:**
   - Merge both changes, preserving the intent of each side
   - Remove all conflict markers
   - Ensure the result is syntactically valid
   - `git add` the resolved file

5. **For complex conflicts — ESCALATE:**
   ```
   ================================================================================
                     CONFLICT ESCALATION — HUMAN REVIEW NEEDED
   ================================================================================
   File:    {file-path}
   Reason:  {why this conflict is too risky to auto-resolve}

   Local changes (ours):
   {summary of what the local side changed}

   Remote changes (theirs):
   {summary of what the remote side changed}

   Recommendation:
   {suggested resolution approach}
   ================================================================================
   ```
   **HALT and wait for user guidance.**

### Step 10 — Regenerate generated files

If any generated files were conflicted:

```bash
# Go dependencies
go mod tidy

# Node dependencies (if applicable)
npm install  # regenerates package-lock.json
```

`git add` the regenerated files.

### Step 11 — Complete the merge

After all conflicts are resolved:

```bash
# Verify no remaining conflicts
REMAINING=$(git diff --name-only --diff-filter=U)
if [ -n "$REMAINING" ]; then
  echo "ERROR: Unresolved conflicts remain:"
  echo "$REMAINING"
  # HALT
fi

git commit --no-edit
```

---

## Phase 4: Complete Sync

### Step 12 — Verify build

Run a quick build check to ensure the merge result is valid:

```bash
make build
```

If the build fails:
```
WARNING: Build failed after conflict resolution. The merge may have introduced errors.
```
Show the build errors and offer to fix or escalate. Do not push/complete if the build is broken.

### Step 13 — Complete the sync

**For push direction:**
```bash
git push origin HEAD:{remote-branch}
```

If push fails (another conflict occurred in the meantime):
```
Push failed — remote has new changes since our fetch. Re-running from Phase 2.
```
Return to Step 4 (max 3 retries, then escalate).

**For pull direction:**
The merge commit is already on the local branch. Pull is complete.

---

## Phase 5: Report

### Step 14 — Summary

```
================================================================================
                    CONFLICT RESOLUTION COMPLETE
================================================================================
Direction:       {direction}
Remote branch:   {remote-branch}
Files resolved:  {count}
Method:          {auto-merge | manual resolution | mixed}

Resolved files:
{list of resolved files with resolution method}

Status: Sync complete.
================================================================================
```

---

## Error Handling

| Error | Action |
|-------|--------|
| Missing arguments | Display usage, **HALT** |
| Remote unreachable | Report network error, **HALT** |
| Fetch failed | Report error, **HALT** |
| Binary file conflict | Escalate to user, **HALT** |
| Complex source conflict | Escalate with context, **HALT** |
| Build fails post-merge | Report errors, offer fix or escalate |
| Push fails after resolution | Retry from fetch (max 3), then escalate |
| Merge already in progress | `git merge --abort`, then restart |

## Safety Principles

1. **Conservative by default** — When in doubt, escalate rather than guess
2. **Never force-push** — All pushes are normal pushes; if rejected, re-fetch and re-merge
3. **Preserve both sides** — When resolving, include changes from both sides unless they truly conflict
4. **Build verification** — Always verify the merge result builds before pushing
5. **Transparent reporting** — Show exactly what was resolved and how at every step
