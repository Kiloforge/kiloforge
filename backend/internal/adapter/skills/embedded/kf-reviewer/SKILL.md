---
name: kf-reviewer
description: Review a PR created by a kf-developer agent. Fetch the diff, review against track spec and project standards, approve or request changes. Pairs with kf-developer --with-review.
metadata:
  argument-hint: "<pr-url-or-number> [--max-iterations N]"
---

# Kiloforge Reviewer

Review a pull request created by a kf-developer with `--with-review`. Fetches the PR diff, reviews it against the track specification, project standards, and code quality criteria, then approves or requests changes via the PR platform.

## Use this skill when

- A developer agent has created a PR and is waiting for review
- You are running in a reviewer worktree (e.g., `reviewer-1`)
- You want automated code review against project standards

## Do not use this skill when

- There is no PR to review
- You want to implement a track (use `/kf-developer` instead)
- The PR was not created by the kiloforge workflow

---

## After Compaction

Output this anchor line exactly:

```
ACTIVE ROLE: kf-reviewer — PR {pr-url} — skill at ~/.claude/skills/kf-reviewer/SKILL.md
```

If you see this after compaction, re-read the skill file before continuing.

---

## Worktree Convention

This agent is expected to run in a worktree whose folder name starts with `reviewer-` (e.g., `reviewer-1`). The corresponding branch name matches the folder name.

The reviewer works from its own worktree but reads code via the PR diff and by checking out the PR branch locally for deeper inspection when needed.

---

## Phase 0: Sync with primary branch

Before any track lookups or validation, sync to the latest state. Your worktree may be stale — other developers and architects merge continuously:

```bash
git reset --hard main
```

This ensures `kf-track-content show`, `kf-track get`, and file reads see the latest track specs and project context.

---

## Phase 1: PR Intake

### Step 1 — Parse arguments

- **Required:** PR URL or number
- **Optional:** `--max-iterations N` (default: 5) — max review rounds before halting

If no argument:
```
ERROR: PR reference required.

Usage: /kf-reviewer <pr-url-or-number> [--max-iterations N]
```
**HALT.**

### Step 2 — Determine PR platform

Same logic as kf-developer:
1. **`KF_PR_PLATFORM`** env var if set (`github` or `gitea`)
2. Auto-detect from remote URL
3. **`KF_REMOTE`** env var or `origin` for remote name

### Step 3 — Fetch PR details

**GitHub:**
```bash
gh pr view {pr-number} --json title,body,headRefName,baseRefName,files,additions,deletions,commits
```

**Gitea:**
```bash
tea pr view {pr-number}
```

Extract:
- **Track ID** — from PR title or body
- **Developer session ID** — from PR body (`DEVELOPER_SESSION=...`)
- **Developer worktree** — from PR body (`DEVELOPER_WORKTREE=...`)
- **Branch name** — the head ref
- **Files changed** — list of modified files

### Step 4 — Load review context

Read project standards and track spec to review against:

1. **Track spec** (via CLI):
   ```bash
   .agent/kf/bin/kf-track-content show {trackId} --section spec
   ```
2. **Track plan:**
   ```bash
   .agent/kf/bin/kf-track-content show {trackId} --section plan
   ```
3. **Product context:** `.agent/kf/product.md`
4. **Product guidelines:** `.agent/kf/product-guidelines.md` (if exists)
5. **Tech stack:** `.agent/kf/tech-stack.md`
6. **Workflow rules:** `.agent/kf/workflow.md`
7. **Code style guides:** `.agent/kf/code_styleguides/` (if present)

Output the compaction anchor:
```
ACTIVE ROLE: kf-reviewer — PR {pr-url} — skill at ~/.claude/skills/kf-reviewer/SKILL.md
```

---

## Phase 2: Code Review

### Step 5 — Fetch the diff

**GitHub:**
```bash
gh pr diff {pr-number}
```

**Gitea:**
```bash
tea pr diff {pr-number}
```

For large diffs, also fetch the PR branch locally for deeper inspection:
```bash
git fetch ${REMOTE_NAME} {head-branch}
git log --oneline main..FETCH_HEAD
```

### Step 6 — Review the changes

Evaluate the PR against these criteria:

#### Correctness
- Does the code implement what the track spec requires?
- Are all acceptance criteria from the spec addressed?
- Are there logic errors, off-by-one errors, or missing edge cases?

#### Architecture & Patterns
- Does it follow the project's architectural patterns (from tech-stack.md, code style guides)?
- Is the code in the right domain/package/module?
- Are dependencies appropriate (no circular deps, proper layering)?

#### Code Quality
- Clear naming, reasonable function sizes?
- No dead code, no commented-out blocks?
- Proper error handling?

#### Testing
- Are tests included if required by workflow.md?
- Do tests cover the key behaviors?
- TDD compliance if workflow.md requires it?

#### Security
- No hardcoded secrets, no SQL injection, no XSS?
- Input validation at system boundaries?

#### Track Completeness
- Are all tasks in the plan marked complete?
- Are kiloforge artifacts (tracks.yaml, track.yaml) properly updated?

### Step 7 — Determine review outcome

Based on the review, choose one of:

1. **APPROVE** — code meets all criteria, no issues found
2. **REQUEST CHANGES** — specific issues that must be fixed
3. **COMMENT** — minor suggestions that don't block merge (still counts as approval)

---

## Phase 3: Submit Review

### Step 8 — Post review to PR

#### If APPROVE:

**GitHub:**
```bash
gh pr review {pr-number} --approve --body "$(cat <<'EOF'
## Review: Approved

{Brief summary of what was reviewed}

All acceptance criteria met. Code quality, architecture, and testing look good.

---
_Reviewed by kf-reviewer_
EOF
)"
```

**Gitea:** Post approval comment via API.

#### If REQUEST CHANGES:

**GitHub:**
```bash
gh pr review {pr-number} --request-changes --body "$(cat <<'EOF'
## Review: Changes Requested

### Issues Found

{For each issue:}
#### {issue-number}. {Brief title}
- **File:** `{file-path}:{line}`
- **Severity:** {blocking|suggestion}
- **Issue:** {description}
- **Suggestion:** {how to fix}

### Summary

{N} blocking issue(s) found. Please address and push updates.

---
_Reviewed by kf-reviewer (iteration {N}/{max})_
EOF
)"
```

Also post inline comments on specific lines where possible:

**GitHub:**
```bash
gh api repos/{owner}/{repo}/pulls/{pr-number}/comments \
  -f body="{comment}" \
  -f path="{file}" \
  -f line={line} \
  -f side="RIGHT" \
  -f commit_id="{head-sha}"
```

---

## Phase 4: Post-Review

### Step 9 — Report and wait

#### If APPROVED:

```
================================================================================
                    REVIEW COMPLETE — APPROVED
================================================================================
PR:         {pr-url}
Track:      {trackId}
Iteration:  {N}/{max}
Verdict:    APPROVED

The developer agent can now proceed to merge.

To resume the developer:
  claude --resume {developer-session-id}
  Then type: "review complete — approved"

Or in the developer's terminal, type: "review complete — approved"
================================================================================
```

**HALT.** The reviewer's job is done for this PR (unless the developer pushes more changes and the reviewer is asked to review again).

#### If CHANGES REQUESTED:

```
================================================================================
                    REVIEW COMPLETE — CHANGES REQUESTED
================================================================================
PR:         {pr-url}
Track:      {trackId}
Iteration:  {N}/{max}
Issues:     {count} blocking

Waiting for developer to push fixes.

To notify the developer:
  In the developer's terminal, type: "review feedback — changes requested"

Say "re-review" when the developer has pushed updates.
================================================================================
```

**HALT and wait.** The reviewer stays alive, preserving its understanding of the PR and the issues it found. When unblocked:

### Step 10 — Re-review cycle

When the user says "re-review" or indicates the developer has pushed fixes:

1. Increment iteration counter
2. Check iteration limit (`--max-iterations`):
   - If limit reached:
     ```
     ================================================================================
                         REVIEW CYCLE LIMIT REACHED
     ================================================================================
     PR:         {pr-url}
     Iterations: {max}/{max}

     Maximum review iterations reached without approval.
     Manual intervention required.
     ================================================================================
     ```
     **HALT.**

3. Fetch the updated diff:
   ```bash
   gh pr diff {pr-number}
   ```

4. **Focus the re-review on previously raised issues:**
   - Check if each previously flagged issue has been addressed
   - Look for any regressions introduced by the fixes
   - Note any new issues in the changed code
   - Do NOT re-review unchanged code unless a fix impacted it

5. Return to Step 7 (determine outcome) and Step 8 (post review)

---

## Error Handling

| Error | Action |
|-------|--------|
| PR not found | Display error, **HALT** |
| Track ID not parseable from PR | Ask user for track ID |
| Platform CLI not available (`gh`/`tea`) | Display install instructions, **HALT** |
| Diff too large to review | Review file-by-file, summarize |
| Developer session ID not in PR body | Output manual resume instructions |

---

## Critical Rules

1. **NEVER modify code** — the reviewer only reads and comments, never pushes changes
2. **ALWAYS review against the track spec** — acceptance criteria are the primary standard
3. **ALWAYS post reviews via the PR platform** — not just terminal output
4. **Be specific** — every change request must cite file, line, and suggested fix
5. **Focus re-reviews** — only check previously raised issues and their fix impact
6. **Respect iteration limits** — halt after max iterations
7. **Include developer session info** — always output the resume command for the developer
8. **Stay alive between reviews** — preserve context across review iterations
