# Implementation Plan: Fix Skill Prerequisite Chain — Rename and Required List (Backend)

**Track ID:** fix-skill-prereq-be_20260310000644Z

## Phase 1: Embedded Skill Rename

- [ ] Task 1.1: Rename `embedded/kf-track-generator/` directory to `embedded/kf-architect/`
- [ ] Task 1.2: Update `kf-architect/SKILL.md` content to match current `skills/kf-architect/SKILL.md` from the repo
- [ ] Task 1.3: Update `kf-parallel/SKILL.md` embedded content if it references `kf-track-generator`

## Phase 2: Required Skills Mapping

- [ ] Task 2.1: Update `RequiredSkillsForRole("interactive")` to return `kf-architect` instead of `kf-track-generator`
- [ ] Task 2.2: Add `RequiredSkillsForRole("setup")` returning `kf-setup`
- [ ] Task 2.3: Update `StartProjectSetup` handler to call `checkSkillsForRole("setup", proj.ProjectDir)` instead of `checkSkillsForRole("interactive", ...)`
- [ ] Task 2.4: Update `GetPreflight` to also check "setup" role skills so `skills_ok` reflects full prerequisite chain

## Phase 3: Tests

- [ ] Task 3.1: Update `TestRequiredSkillsForRole` — change expected interactive skill to `kf-architect`, add "setup" role test
- [ ] Task 3.2: Update `TestListEmbedded` — change expected name from `kf-track-generator` to `kf-architect`
- [ ] Task 3.3: Update any other tests referencing `kf-track-generator`

## Phase 4: Verification

- [ ] Task 4.1: `make generate` produces no diff
- [ ] Task 4.2: `make test` passes
