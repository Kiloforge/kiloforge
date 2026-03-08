# Implementation Plan: Embedded Skills as Default — Remove Repo Dependency

**Track ID:** embedded-skills-default_20260309234000Z

## Phase 1: Auto-Install During Init

- [ ] Task 1.1: Create `installEmbeddedSkills(cfg)` function in `skills.go` — iterates required skills, calls `InstallEmbedded()` for each, skips if hash matches
- [ ] Task 1.2: Replace `offerSkillsInstall(ctx, cfg)` call in `init.go` with `installEmbeddedSkills(cfg)` — mandatory, no prompt
- [ ] Task 1.3: Keep `offerSkillsInstall` for backward compat but make it fall back to embedded when no repo

## Phase 2: Update Skills API

- [ ] Task 2.1: Update `GetSkillsStatus` in `api_handler.go` — return embedded skill list when no `SkillsRepo` configured
- [ ] Task 2.2: Update `UpdateSkills` in `api_handler.go` — re-extract from embedded when no repo instead of returning 400

## Phase 3: Update Frontend

- [ ] Task 3.1: Update `SkillsBanner.tsx` — remove `if (!status.repo) return null` guard, show status based on skills array
- [ ] Task 3.2: Update skill-related type definitions if needed

## Phase 4: Verification

- [ ] Task 4.1: `make test` passes
- [ ] Task 4.2: Fresh `kf init` installs skills automatically
- [ ] Task 4.3: Dashboard shows skill status without repo configured
- [ ] Task 4.4: Interactive agent spawn succeeds without manual skill setup
