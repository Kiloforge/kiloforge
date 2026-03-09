# Conductor Legacy Reference

This document explains the old "Conductor" skill system that was used before Kiloforge adopted the `kf-*` skill naming and `.agent/kf/` directory structure. Agents may encounter references to Conductor in compacted conversation summaries, git history, or archived track records.

## What was Conductor?

Conductor was the original name for the track-based project management system now called Kiloforge. It used:

- **Directory:** `.agent/conductor/` (now `.agent/kf/`)
- **Skills:** `conductor-*` prefix (now `kf-*`)
- **Track registry:** `tracks.md` — a Markdown table (now `tracks.yaml` — YAML with JSON lines)

## Directory Mapping

| Old Path (Conductor) | New Path (Kiloforge) |
|----------------------|---------------------|
| `.agent/conductor/product.md` | `.agent/kf/product.md` |
| `.agent/conductor/product-guidelines.md` | `.agent/kf/product-guidelines.md` |
| `.agent/conductor/tech-stack.md` | `.agent/kf/tech-stack.md` |
| `.agent/conductor/workflow.md` | `.agent/kf/workflow.md` |
| `.agent/conductor/index.md` | `.agent/kf/index.md` |
| `.agent/conductor/tracks.md` | `.agent/kf/tracks.yaml` (new format) |
| `.agent/conductor/tracks/{id}/` | `.agent/kf/tracks/{id}/` |
| `.agent/conductor/tracks/_archive/` | `.agent/kf/tracks/_archive/` |
| `.agent/conductor/code_styleguides/` | `.agent/kf/code_styleguides/` |

## Skill Mapping

| Old Skill (Conductor) | New Skill (Kiloforge) |
|-----------------------|----------------------|
| `conductor-setup` | `kf-setup` |
| `conductor-architect` / `conductor-track-generator` | `kf-architect` |
| `conductor-developer` | `kf-developer` |
| `conductor-reviewer` | `kf-reviewer` |
| `conductor-implement` | `kf-implement` |
| `conductor-manage` | `kf-manage` |
| `conductor-status` | `kf-status` |

## Old tracks.md Format (Markdown Table)

The old `tracks.md` used a Markdown table for the track registry:

```markdown
| Status | Track ID | Title | Created | Updated |
| ------ | -------- | ----- | ------- | ------- |
| [x] | some-track_20260308Z | Track Title | 2026-03-08 | 2026-03-08 |
| [ ] | pending-track_20260309Z | Another Track | 2026-03-09 | 2026-03-09 |
| [~] | wip-track_20260309Z | In Progress Track | 2026-03-09 | 2026-03-09 |
```

Status markers: `[x]` = completed, `[~]` = in-progress, `[ ]` = pending.

Archives were appended as a separate section with batch headers.

## New tracks.yaml Format (YAML + JSON)

The new format uses one track per line with JSON metadata:

```yaml
some-track_20260308Z: {"title":"Track Title","status":"completed","type":"feature","created":"2026-03-08","updated":"2026-03-08"}
pending-track_20260309Z: {"title":"Another Track","status":"pending","type":"feature","created":"2026-03-09","updated":"2026-03-09"}
```

Managed via the `kf-track` CLI tool at `.agent/kf/bin/kf-track`.

## What to Do When You Encounter Conductor References

1. **In compacted summaries:** The compacted text may reference `.agent/conductor/` paths or `conductor-*` skills. Map them to the new paths using the tables above.

2. **In git history:** Commits before the migration will reference `.agent/conductor/`. This is expected — the history is preserved as-is.

3. **In archived tracks:** Track specs in `.agent/kf/tracks/_archive/` may reference conductor skills or paths in their text. The content is correct for its era — do not modify archived records.

4. **In CLAUDE.md or skill files:** If you find stale conductor references, update them to kf equivalents.

## Legacy tracks-legacy.md

A copy of the final `tracks.md` before migration is preserved at `.agent/kf/tracks-legacy.md` for reference. This file is read-only and should not be modified.
