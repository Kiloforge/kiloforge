# Implementation Plan: Track Generator Auto-Approve for Non-Code Tracks

**Track ID:** track-gen-auto-approve_20260309220000Z

## Phase 1: Skill Update

- [ ] Task 1.1: Update `~/.claude/skills/conductor-track-generator/SKILL.md` Phase 4 (Step 9) — add auto-approve check logic before the review prompt
- [ ] Task 1.2: Add auto-approve criteria definition to the skill (research type, no code output, conductor-only artifacts)
- [ ] Task 1.3: Add "uncertain = require review" safe fallback rule
- [ ] Task 1.4: Update Step 9 output format — add "Auto-approved" notice line when auto-approving
- [ ] Task 1.5: Add rule: mixed batches (research + code) always require full review

## Phase 2: Embedded Source Update

- [ ] Task 2.1: If an embedded skill source copy exists in the repo, apply the same changes there
- [ ] Task 2.2: Commit skill changes

## Phase 3: Verification

- [ ] Task 3.1: Test by generating a research track — verify it auto-approves without prompting
- [ ] Task 3.2: Test by generating a code track — verify it still requires review
- [ ] Task 3.3: Test mixed batch — verify full review is required
