---
name: kf-implement
description: Execute tasks from a track's implementation plan following TDD workflow
metadata:
  argument-hint: "[track-id] [--task X.Y] [--phase N]"
---

# Implement Track

Execute tasks from a track's implementation plan, following the workflow rules defined in `.agent/kf/workflow.md`.

## Use this skill when

- Working on implement track tasks or workflows
- Needing guidance, best practices, or checklists for implement track

## Do not use this skill when

- The task is unrelated to implement track
- You need a different domain or tool outside this scope

## Instructions

- Clarify goals, constraints, and required inputs.
- Apply relevant best practices and validate outcomes.
- Provide actionable steps and verification.

## Pre-flight Checks

1. **Sync with primary branch** — your working tree may be stale:
   ```bash
   git reset --hard main
   ```
   This ensures you see the latest tracks, dependencies, and project state. Without this, `kf-track get` and `kf-track list` may report tracks as "not found" even if they exist on main.

2. Verify Kiloforge is initialized:
   - Check `.agent/kf/product.md` exists
   - Check `.agent/kf/workflow.md` exists
   - Check `.agent/kf/tracks.yaml` exists
   - If missing: Display error and suggest running `/kf-setup` first

2. Load workflow configuration:
   - Read `.agent/kf/workflow.md`
   - Parse TDD strictness level
   - Parse commit strategy
   - Parse verification checkpoint rules

## Track Selection

### If argument provided:

- Validate track exists:
  ```bash
  .agent/kf/bin/kf-track get {argument}
  ```
- If not found: Search for partial matches, suggest corrections

### If no argument:

1. List available tracks:
   ```bash
   .agent/kf/bin/kf-track list --active
   ```
2. Display selection menu:

   ```
   Select a track to implement:

   In Progress:
   1. [~] auth_20250115100000Z - User Authentication

   Pending:
   2. [ ] nav-fix_20250114081500Z - Navigation Bug Fix
   3. [ ] dashboard_20250113140000Z - Dashboard Feature

   Enter number or track ID:
   ```

## Context Loading

Load all relevant context for implementation:

1. Track documents (via CLI):
   ```bash
   .agent/kf/bin/kf-track-content show {trackId} --section spec
   .agent/kf/bin/kf-track-content show {trackId} --section plan
   .agent/kf/bin/kf-track-content progress {trackId}
   ```

2. Project context:
   - `.agent/kf/product.md` - Product understanding
   - `.agent/kf/tech-stack.md` - Technical constraints
   - `.agent/kf/workflow.md` - Process rules

3. Code style (if exists):
   - `.agent/kf/code_styleguides/{language}.md`

## Track Status Update

Update track to in-progress:

```bash
.agent/kf/bin/kf-track update {trackId} --status in-progress
```

## Task Execution Loop

For each incomplete task (check via `.agent/kf/bin/kf-track-content progress {trackId}`):

### 1. Task Identification

Find the next incomplete task:
```bash
.agent/kf/bin/kf-track-content progress {trackId}
```

Look for the first task not yet marked done.

### 2. Task Start

Announce: "Starting Task X.Y: {description}"

### 3. TDD Workflow (if TDD enabled in workflow.md)

**Red Phase - Write Failing Test:**

```
Following TDD workflow for Task X.Y...

Step 1: Writing failing test
```

- Create test file if needed
- Write test(s) for the task functionality
- Run tests to confirm they fail
- If tests pass unexpectedly: HALT, investigate

**Green Phase - Implement:**

```
Step 2: Implementing minimal code to pass test
```

- Write minimum code to make test pass
- Run tests to confirm they pass
- If tests fail: Debug and fix

**Refactor Phase:**

```
Step 3: Refactoring while keeping tests green
```

- Clean up code
- Run tests to ensure still passing

### 4. Non-TDD Workflow (if TDD not strict)

- Implement the task directly
- Run any existing tests
- Manual verification as needed

### 5. Task Completion

**Commit changes** (following commit strategy from workflow.md):

```bash
git add -A
git commit -m "{commit_prefix}: {task description} ({trackId})"
```

**Mark task done via CLI:**

```bash
.agent/kf/bin/kf-track-content task {trackId} X.Y --done
```

Commit the track update:

```bash
git add .agent/kf/tracks/{trackId}/
git commit -m "chore: mark task X.Y complete ({trackId})"
```

### 6. Phase Completion Check

After each task, check if phase is complete:

```bash
.agent/kf/bin/kf-track-content progress {trackId}
```

If all tasks in current phase are done:

**Run phase verification:**

```
Phase {N} complete. Running verification...
```

- Execute verification commands from workflow.md
- Run full test suite

**Report and wait for approval:**

```
Phase {N} Verification Results:
- All phase tasks: Complete
- Tests: {passing/failing}
- Verification: {pass/fail}

Approve to continue to Phase {N+1}?
1. Yes, continue
2. No, there are issues to fix
3. Pause implementation
```

**CRITICAL: Wait for explicit user approval before proceeding to next phase.**

## Error Handling During Implementation

### On Tool Failure

```
ERROR: {tool} failed with: {error message}

Options:
1. Retry the operation
2. Skip this task and continue
3. Pause implementation
4. Revert current task changes
```

- HALT and present options
- Do NOT automatically continue

### On Test Failure

```
TESTS FAILING after Task X.Y

Failed tests:
- {test name}: {failure reason}

Options:
1. Attempt to fix
2. Rollback task changes
3. Pause for manual intervention
```

### On Git Failure

```
GIT ERROR: {error message}

This may indicate:
- Uncommitted changes from outside Kiloforge
- Merge conflicts
- Permission issues

Options:
1. Show git status
2. Attempt to resolve
3. Pause for manual intervention
```

## Track Completion

When all phases and tasks are complete:

### 1. Final Verification

```
All tasks complete. Running final verification...
```

- Run full test suite
- Check all acceptance criteria from spec
- Generate verification report

### 2. Update Track Status

```bash
.agent/kf/bin/kf-track update {trackId} --status completed
```

Verify completion:
```bash
.agent/kf/bin/kf-track-content progress {trackId}
```

### 3. Documentation Sync Offer

```
Track complete! Would you like to sync documentation?

This will update:
- .agent/kf/product.md (if new features added)
- .agent/kf/tech-stack.md (if new dependencies added)
- README.md (if applicable)

1. Yes, sync documentation
2. No, skip
```

### 4. Cleanup Offer

```
Track {trackId} is complete.

Cleanup options:
1. Archive - Move to .agent/kf/tracks/_archive/
2. Delete - Remove track directory
3. Keep - Leave as-is
```

### 5. Completion Summary

```
Track Complete: {track title}

Summary:
- Track ID: {trackId}
- Phases completed: {N}/{N}
- Tasks completed: {M}/{M}
- Commits created: {count}
- Tests: All passing

Next steps:
- Run /kf-status to see project progress
- Run /kf-new-track for next feature
```

## Resumption

If implementation is paused and resumed:

1. Check current progress:
   ```bash
   .agent/kf/bin/kf-track-content progress {trackId}
   ```
2. Find current incomplete task
3. Ask user:

   ```
   Resuming track: {title}

   Last task in progress: Task {X.Y}: {description}

   Options:
   1. Continue from where we left off
   2. Restart current task
   3. Show progress summary first
   ```

## Critical Rules

1. **NEVER skip verification checkpoints** - Always wait for user approval between phases
2. **STOP on any failure** - Do not attempt to continue past errors
3. **Follow workflow.md strictly** - TDD, commit strategy, and verification rules are mandatory
4. **Use CLI tools for track updates** - Use `kf-track` and `kf-track-content` for all status and progress updates
5. **Commit frequently** - Each task completion should be committed
6. **Use kf-track-content for progress** - Never parse plan files manually; use `kf-track-content progress` instead
