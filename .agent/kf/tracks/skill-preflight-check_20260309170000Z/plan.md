# Implementation Plan: Pre-flight Skill Validation Before Agent Spawning

**Track ID:** skill-preflight-check_20260309170000Z

## Phase 1: Skill Checker

- [x] Task 1.1: Add `RequiredSkill` type and `CheckRequired()` function in `backend/internal/adapter/skills/checker.go` — checks both global and local dirs for `SKILL.md` presence
- [x] Task 1.2: Add `ErrSkillsMissing` typed error in `backend/internal/adapter/agent/spawner.go` — returned when required skills are not found
- [x] Task 1.3: Add `ValidateSkills()` to spawner — maps agent role to required skills, calls `CheckRequired()`, returns `ErrSkillsMissing` with details
- [x] Task 1.4: Unit tests for `CheckRequired()` — test global found, local found, both missing, mixed scenarios

## Phase 2: CLI Integration

- [x] Task 2.1: Add `promptSkillInstall()` in `backend/internal/adapter/cli/implement.go` — interactive prompt offering global/local/abort options
- [x] Task 2.2: Wire pre-flight check into `kf implement` — call `ValidateSkills()` before `SpawnDeveloper()`, on `ErrSkillsMissing` run prompt, retry validation after install
- [x] Task 2.3: N/A — no CLI reviewer entrypoint exists; reviewer is spawned via webhook (handled in Phase 3)
- [x] Task 2.4: Test that `kf implement` blocks when skills are missing (integration test or manual verification)

## Phase 3: REST API Integration

- [x] Task 3.1: Add skill validation to REST agent spawn handlers — return 412 Precondition Failed with JSON listing missing skills
- [x] Task 3.2: Update OpenAPI spec with 412 response schema for agent spawn endpoints

## Phase 4: Verification

- [x] Task 4.1: Verify `go test ./...` passes
- [x] Task 4.2: Verify `kf implement` with skills installed works unchanged
- [x] Task 4.3: Verify `kf implement` without skills prompts and installs correctly
