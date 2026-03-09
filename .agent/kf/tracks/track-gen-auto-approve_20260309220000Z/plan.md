# Implementation Plan: Track Generator Auto-Approve for Non-Code Tracks

**Track ID:** track-gen-auto-approve_20260309220000Z

## Phase 1: Skill Update

- [x] Task 1.1: Update `backend/internal/adapter/skills/embedded/kf-track-generator/SKILL.md` Phase 4 (Step 9) — add auto-approve check logic before the review prompt
- [x] Task 1.2: Add auto-approve criteria definition to the skill (research type, no code output, conductor-only artifacts)
- [x] Task 1.3: Add "uncertain = require review" safe fallback rule
- [x] Task 1.4: Update Step 9 output format — add "Auto-approved" notice line when auto-approving
- [x] Task 1.5: Add rule: mixed batches (research + code) always require full review

## Phase 2: Verification

- [x] Task 2.1: Commit skill changes
- [x] Task 2.2: Test by generating a research track — verify it auto-approves without prompting
- [x] Task 2.3: Test by generating a code track — verify it still requires review
- [x] Task 2.4: Test mixed batch — verify full review is required
