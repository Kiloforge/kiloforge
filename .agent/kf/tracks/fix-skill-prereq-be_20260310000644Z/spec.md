# Specification: Fix Skill Prerequisite Chain — Rename and Required List (Backend)

**Track ID:** fix-skill-prereq-be_20260310000644Z
**Type:** Bug
**Created:** 2026-03-10T00:06:44Z
**Status:** Draft

## Summary

The skills prerequisite check is broken in multiple ways: the embedded `kf-track-generator` skill was renamed to `kf-architect` globally but not in the repo, `RequiredSkillsForRole("interactive")` still references the old name, and `kf-setup` is not listed as a required skill despite being a prerequisite for setup. This means the preflight `skills_ok` field may incorrectly report skills as missing (looking for `kf-track-generator` which no longer exists at that name globally), and there's no validation that the setup skill is installed before attempting setup.

## Context

The skill rename from `kf-track-generator` to `kf-architect` was done in user-level skills (`~/.claude/skills/`) and project-level skills (`skills/`) but NOT in:
- The embedded skills directory (`backend/internal/adapter/skills/embedded/kf-track-generator/`)
- The `RequiredSkillsForRole` mapping in `checker.go` (line 75)

Additionally, `kf-setup` is a prerequisite for the setup step itself — skills must be installed mechanically before the `/kf-setup` agent can run. But `kf-setup` is not in any required skills list, so the backend never validates its presence.

## Codebase Analysis

### `checker.go` (`backend/internal/adapter/skills/checker.go`)

- **Line 75:** `RequiredSkillsForRole("interactive")` returns `{Name: "kf-track-generator"}` — should be `kf-architect`
- **Line 63-79:** Role mapping doesn't include `kf-setup` for any role
- **`skillExists()`** (line 169): Looks for `SKILL.md` in `{dir}/{name}/SKILL.md`

### Embedded skills (`backend/internal/adapter/skills/embedded/`)

- `kf-track-generator/SKILL.md` exists — needs renaming to `kf-architect/SKILL.md`
- `kf-setup/SKILL.md` exists — already embedded, just not in the required list
- 14 skills total embedded

### `GetPreflight` (`api_handler.go` line 144)

- Uses `skills.RequiredSkillsForRole("interactive")` to check skills
- Returns `skills_ok: false` if any required skills missing
- Currently checks for `kf-track-generator` which won't be found at `~/.claude/skills/kf-track-generator/` since it was renamed to `kf-architect`

### `checker_test.go`

- `TestRequiredSkillsForRole` verifies interactive role returns `kf-track-generator` — needs updating
- `TestListEmbedded` checks for `kf-track-generator` in embedded list — needs updating

### Other references

- `api_handler.go` line 1339, 340, 1461: All call `checkSkillsForRole("interactive", ...)` — these will check for the wrong skill name
- `StartProjectSetup` (line 1696): Calls `checkSkillsForRole("interactive", ...)` before spawning setup agent — this should also verify `kf-setup` specifically

## Acceptance Criteria

- [ ] Embedded skill directory renamed: `embedded/kf-track-generator/` → `embedded/kf-architect/`
- [ ] Embedded `kf-architect/SKILL.md` content updated to match the current `kf-architect` skill (frontmatter name, title, etc.)
- [ ] `RequiredSkillsForRole("interactive")` returns `kf-architect` instead of `kf-track-generator`
- [ ] New role or cross-cutting requirement: `kf-setup` is validated before setup agent spawning
- [ ] `GetPreflight` preflight response correctly reports `skills_ok` based on renamed skill
- [ ] `StartProjectSetup` validates `kf-setup` skill is installed before spawning setup agent
- [ ] `checker_test.go` updated for new skill names
- [ ] `make generate` produces no diff
- [ ] `make test` passes

## Dependencies

None.

## Blockers

None.

## Conflict Risk

- LOW — modifies `checker.go`, `checker_test.go`, embedded skill directory, and a few lines in `api_handler.go`. No pending tracks touch these files.

## Out of Scope

- Frontend changes (separate track: `fix-skill-prereq-fe_20260310000645Z`)
- Renaming other embedded skills that reference `track-generator` in their content (cosmetic)
- Updating the `kf-parallel` embedded skill deprecation message

## Technical Notes

### Embedded skill rename

```bash
mv backend/internal/adapter/skills/embedded/kf-track-generator backend/internal/adapter/skills/embedded/kf-architect
```

Then update `kf-architect/SKILL.md` content — copy from the current project-level `skills/kf-architect/SKILL.md` or the user-level `~/.claude/skills/kf-architect/SKILL.md`.

### RequiredSkillsForRole update

```go
case "interactive":
    return []RequiredSkill{
        {Name: "kf-architect", Reason: "required for track generation"},
    }
```

### kf-setup validation

Option A — Add to a "setup" role:
```go
case "setup":
    return []RequiredSkill{
        {Name: "kf-setup", Reason: "required for project setup"},
    }
```

Then in `StartProjectSetup`, call `checkSkillsForRole("setup", proj.ProjectDir)`.

Option B — Add `kf-setup` to the "interactive" role (since setup runs as interactive agent):
```go
case "interactive":
    return []RequiredSkill{
        {Name: "kf-architect", Reason: "required for track generation"},
        {Name: "kf-setup", Reason: "required for project setup"},
    }
```

Option A is cleaner — setup is a distinct operation, not all interactive agents need `kf-setup`.

### Preflight skills_ok

The `GetPreflight` handler (line 161) calls `RequiredSkillsForRole("interactive")` — after the rename this will correctly check for `kf-architect`. No code change needed beyond the role mapping fix.

However, preflight should also report `kf-setup` status since the frontend needs to know if skills are ready for the full chain. Consider checking all required skills across roles, or adding a "setup" role check to preflight.

### Test updates

- `TestRequiredSkillsForRole`: Update expected name from `kf-track-generator` to `kf-architect`, add test for "setup" role
- `TestListEmbedded`: Update expected name from `kf-track-generator` to `kf-architect`

---

_Generated by kf-architect from prompt: "Fix skill prerequisite chain — rename and required list"_
