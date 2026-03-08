# Implementation Plan: Kiloforge-Branded Skill Source Artifacts

**Track ID:** kf-skills-source_20260309063859Z

## Phase 1: Directory Setup and Bulk Copy

- [ ] Task 1.1: Create `skills/` directory at repo root
- [ ] Task 1.2: Copy all 14 conductor skills from `~/.claude/skills/conductor-*/` to `skills/kf-*/` with renamed directories
- [ ] Task 1.3: Copy `conductor-manage/resources/` subdirectory to `kf-manage/resources/`
- [ ] Task 1.4: Verify all 14 skill directories exist with SKILL.md files

## Phase 2: YAML Frontmatter Rebrand

- [ ] Task 2.1: Update `name:` field in all 14 SKILL.md files from `conductor-*` to `kf-*`
- [ ] Task 2.2: Update `description:` fields that reference "conductor" to use "kiloforge" or "kf"
- [ ] Task 2.3: Update `metadata.argument-hint:` fields where present
- [ ] Task 2.4: Verify YAML frontmatter is valid in all files

## Phase 3: Content Rebrand — Environment Variables and API References

- [ ] Task 3.1: Replace `CRELAY_RELAY_URL` with `KF_RELAY_URL` in `kf-developer/SKILL.md`
- [ ] Task 3.2: Replace `CRELAY_RELAY_URL` with `KF_RELAY_URL` in `kf-track-generator/SKILL.md`
- [ ] Task 3.3: Replace "crelay lock API" / "crelay relay" with "kiloforge lock API" / "kiloforge relay" in both files
- [ ] Task 3.4: Search all skill files for any remaining `CRELAY_` or `crelay` references and update

## Phase 4: Content Rebrand — Slash Commands and Cross-References

- [ ] Task 4.1: Replace all `/conductor-*` slash command references with `/kf-*` across all 14 SKILL.md files
- [ ] Task 4.2: Update cross-skill references (e.g., "use `/conductor-setup` first" → "use `/kf-setup` first")
- [ ] Task 4.3: Update `conductor-manage/resources/implementation-playbook.md` → `kf-manage/resources/implementation-playbook.md` content
- [ ] Task 4.4: Replace "conductor" role references with "kiloforge" where appropriate in skill descriptions

## Phase 5: Verification

- [ ] Task 5.1: Grep all files under `skills/` for remaining "conductor-" prefix references — should be zero (except historical context in descriptions where appropriate)
- [ ] Task 5.2: Grep all files under `skills/` for remaining "CRELAY_" references — should be zero
- [ ] Task 5.3: Grep all files under `skills/` for remaining "crelay" references — should be zero
- [ ] Task 5.4: Verify each SKILL.md has valid YAML frontmatter with correct `name:` matching directory name
- [ ] Task 5.5: Verify directory count is 14 with no missing skills
