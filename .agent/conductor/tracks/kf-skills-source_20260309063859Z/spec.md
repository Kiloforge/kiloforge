# Specification: Kiloforge-Branded Skill Source Artifacts

**Track ID:** kf-skills-source_20260309063859Z
**Type:** Chore
**Created:** 2026-03-09T06:38:59Z
**Status:** Draft

## Summary

Copy the 14 conductor skills into the repo as kiloforge-branded source artifacts under `skills/`. Rebrand all references from "conductor-" prefix to "kf-" prefix, replace "kiloforge" with "kiloforge"/"kf", and update env var references from `KF_*` to `KF_*`. These source files become the canonical distribution artifacts that the existing skill installer packages and deploys.

## Context

The conductor skills currently live only in `~/.claude/skills/conductor-*/` as local user files. There is no source-of-truth in the repo. The existing skill installer (`backend/internal/adapter/skills/`) already supports downloading skills from GitHub tarballs and installing them to `~/.claude/skills/`. By placing the rebranded skills in the repo, they become:

1. **Version-controlled** — changes are tracked, reviewed, and committed
2. **Distributable** — the installer can package them from the repo
3. **Consistently branded** — all "conductor-" prefixes become "kf-", matching the CLI rename

### Current Skills (14 total)

| Old Name | New Name |
|----------|----------|
| `conductor-bulk-archive` | `kf-bulk-archive` |
| `conductor-compact-archive` | `kf-compact-archive` |
| `conductor-developer` | `kf-developer` |
| `conductor-implement` | `kf-implement` |
| `conductor-manage` | `kf-manage` |
| `conductor-new-track` | `kf-new-track` |
| `conductor-parallel` | `kf-parallel` |
| `conductor-report` | `kf-report` |
| `conductor-revert` | `kf-revert` |
| `conductor-reviewer` | `kf-reviewer` |
| `conductor-setup` | `kf-setup` |
| `conductor-status-private` | `kf-status` |
| `conductor-track-generator` | `kf-track-generator` |
| `conductor-validate` | `kf-validate` |

### Rebrand Mappings

| Before | After |
|--------|-------|
| `conductor-` prefix in dir/skill names | `kf-` prefix |
| `/conductor-developer` (slash command) | `/kf-developer` |
| `KF_RELAY_URL` | `KF_RELAY_URL` |
| "kiloforge" in descriptions/docs | "kiloforge" or "kf" |
| "kiloforge lock API" | "kiloforge lock API" |
| "kiloforge relay" | "kiloforge relay" |

## Codebase Analysis

**Skill source location:** `~/.claude/skills/conductor-*/SKILL.md` (14 skills)
- Only 2 skills reference "kiloforge" in content: `conductor-developer` and `conductor-track-generator` (merge lock sections)
- All 14 use `conductor-` prefix in YAML frontmatter `name:` field and directory names
- 1 skill (`conductor-manage`) has a `resources/` subdirectory

**Existing installer code:** `backend/internal/adapter/skills/`
- `installer.go` — Downloads tarballs, extracts skills by finding `SKILL.md` files
- `manifest.go` — Tracks installed skill checksums for modification detection
- `github.go` — GitHub release tarball URL construction
- The installer already uses `kf-skills-*` temp file prefixes (post-rebrand)

**Target location in repo:** `skills/` (top-level directory)
- Mirrors the structure that the installer expects: `skills/{name}/SKILL.md`
- The installer's `findSkills()` function discovers directories containing `SKILL.md`

## Acceptance Criteria

- [ ] `skills/` directory exists at repo root with all 14 rebranded skills
- [ ] Each skill directory is named `kf-{function}` (e.g., `skills/kf-developer/SKILL.md`)
- [ ] YAML frontmatter `name:` field matches directory name in every SKILL.md
- [ ] All `KF_RELAY_URL` references replaced with `KF_RELAY_URL`
- [ ] All "kiloforge" text references replaced with "kiloforge" or "kf" as appropriate
- [ ] All `/conductor-*` slash command references updated to `/kf-*`
- [ ] Cross-skill references updated (e.g., "use `/conductor-setup` first" → "use `/kf-setup` first")
- [ ] `conductor-manage/resources/` subdirectory copied to `kf-manage/resources/`
- [ ] Skill descriptions updated for trigger accuracy (so Claude invokes `/kf-*` not `/conductor-*`)

## Dependencies

None.

## Blockers

None.

## Conflict Risk

None — creates a new `skills/` directory, no overlap with existing files.

## Out of Scope

- Modifying the installer to read from the local `skills/` directory (separate track)
- Removing the original `~/.claude/skills/conductor-*` skills (user manages those)
- Updating CLAUDE.md or settings to register the new skill names (separate concern)
- Automated skill deployment from repo to `~/.claude/skills/` (the installer already handles this)

## Technical Notes

- **Copy, don't move** — the original conductor skills remain in `~/.claude/skills/` untouched
- **Mechanical rename** — this is primarily find-replace work across 14 SKILL.md files + 1 resource file
- **YAML frontmatter** — the `name:` field in each SKILL.md must exactly match the directory name for Claude to register the skill correctly
- **Slash command triggers** — skill descriptions contain trigger patterns (e.g., "Use when the user runs `/conductor-developer`"). These must be updated to `/kf-*` for correct invocation.
- **argument-hint metadata** — some skills have `metadata.argument-hint:` in frontmatter that references command syntax; update these too

---

_Generated by conductor-track-generator from prompt: "Create kiloforge-branded skill source artifacts in the repo"_
