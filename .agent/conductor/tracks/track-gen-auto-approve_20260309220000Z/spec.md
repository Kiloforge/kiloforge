# Specification: Track Generator Auto-Approve for Non-Code Tracks

**Track ID:** track-gen-auto-approve_20260309220000Z
**Type:** Chore
**Created:** 2026-03-09T22:00:00Z
**Status:** Draft

## Summary

Update the `kf-track-generator` skill so that tracks which do not impact code can skip the user review/approval step and be auto-approved. This applies to research tracks that produce reference documents (e.g., `.agent/conductor/tracks/{id}/research.md`). It does NOT apply to project documentation changes (README, docs/, etc.) which affect the repo.

## Context

Currently, the track generator always presents tracks for user review before creating files and merging. This is the correct default for code-impacting tracks. However, research tracks (type: "Research") only produce internal conductor documents — they don't touch source code, tests, configs, or project docs. Requiring manual approval for these adds unnecessary friction.

## Codebase Analysis

### Current review flow (SKILL.md Phase 4, Steps 9-10)

The track generator presents all tracks in a summary block with options (Approve/Review/Edit/Reject/Approve with changes) and waits for explicit user approval before writing any files. This is controlled by:

```
**CRITICAL: Wait for explicit user approval before creating any track files.**
```

### Track types

The spec template includes a `Type` field: `Feature | Bug | Chore | Refactor`. Research tracks are typically typed as `Chore` or use a custom "Research" type prefix in the title. The distinction is:

- **Code-impacting tracks** — Feature, Bug, Refactor, Chore (when touching code)
- **Non-code tracks** — Research tracks that produce only `.agent/conductor/` artifacts (specs, research docs, design notes)

### Auto-approve criteria

A track qualifies for auto-approve when ALL of these are true:

1. Track type is "Research" (or title starts with "Research:")
2. The plan contains NO tasks that modify source code, tests, configs, or project documentation
3. All outputs are within `.agent/conductor/tracks/{trackId}/` (internal conductor artifacts)
4. The track has no dependencies on code changes and no blockers against code tracks

### What is NOT auto-approved

- Tracks that modify any file outside `.agent/conductor/`
- Tracks that update project documentation (README, docs/, CHANGELOG, etc.)
- Tracks that change configs, CI/CD, or build files
- Any track where the generator is uncertain about impact

### Skill file location

Skills are embedded in the binary at `backend/internal/adapter/skills/embedded/` and installed at runtime to either `~/.claude/skills/kf-*` (global) or `<repo>/.claude/skills/kf-*` (local) based on user preference. The file to modify:

- **`backend/internal/adapter/skills/embedded/kf-track-generator/SKILL.md`** — embedded source that gets installed to the user's chosen location

## Acceptance Criteria

- [ ] In-repo track generator skill (`backend/internal/adapter/skills/embedded/kf-track-generator/SKILL.md`) updated with auto-approve logic in Phase 4 (Review & Approval)
- [ ] When generating research-only tracks, the generator auto-approves and skips the review prompt
- [ ] Auto-approve output includes a clear notice: "Auto-approved (non-code research track)"
- [ ] Mixed batches (research + code tracks) still require review for the full batch — no partial auto-approve
- [ ] The generator still presents the track summary (for transparency) but proceeds without waiting for input
- [ ] Code-impacting tracks are NEVER auto-approved — the existing review flow is preserved
- [ ] Project documentation tracks (README, docs/) are NOT auto-approved
- [ ] Skill file changes committed

## Dependencies

None.

## Blockers

None.

## Conflict Risk

- LOW — only modifies the skill SKILL.md file, which no other track touches.

## Out of Scope

- Changing the developer or reviewer skills
- Adding auto-approve for other track types
- Dashboard UI for track approval (that's a separate concern)
- Renaming the skill from `conductor-track-generator` to `kf-track-generator` (covered by `kf-skills-source`)

## Technical Notes

### Skill modification

In SKILL.md, Phase 4 (Review & Approval), Step 9 should be updated to:

```markdown
### Step 9 — Present tracks for review

**Auto-approve check:** Before presenting the review prompt, evaluate whether ALL generated tracks qualify for auto-approval:

A track qualifies for auto-approve when ALL conditions are met:
1. Track type is "Research" (title starts with "Research:" or type field is "research")
2. All planned outputs are within `.agent/conductor/tracks/{trackId}/` — no source code, tests, configs, or project docs
3. The track does not depend on or block any code-impacting tracks

If ALL tracks in the batch qualify:
- Still display the track summary (for transparency)
- Add notice: "Auto-approved: research-only track(s) — no code impact"
- Skip the approval prompt — proceed directly to Step 10 (Create tracks)

If ANY track in the batch does NOT qualify:
- Present the full review prompt as before (all tracks reviewed together)
- Do not partially auto-approve — the batch is reviewed as a whole

If uncertain about a track's impact:
- Default to requiring review (safe fallback)
```

### Detection heuristic

The generator already knows the track type and plan contents at generation time. The auto-approve check is a simple classification:

```
is_research_only(track):
  - type contains "research" (case-insensitive)
  - OR title starts with "Research:"
  - AND plan tasks only reference creating/writing files in .agent/conductor/
  - AND no acceptance criteria mention modifying source code
  - AND out_of_scope includes code changes (or plan explicitly states no code)
```

---

_Generated by conductor-track-generator from prompt: "auto-approve non-code research tracks in track generator"_
