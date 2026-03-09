# Specification: Conductor Skill Installation and Auto-Update

**Track ID:** skill-install-update_20260308231000Z
**Type:** Feature
**Created:** 2026-03-08T23:10:00Z
**Status:** Draft

## Summary

Add a mechanism for kiloforge to fetch the latest conductor skills from a GitHub repo, install them into projects, and optionally auto-update when new releases are available. Detects local modifications and asks for confirmation before overwriting.

## Context

Conductor skills (track-generator, developer, reviewer, etc.) are currently manually maintained in `~/.claude/skills/`. There's no versioning, no update mechanism, and no way to ensure all users have the same skill versions. Moving skills to a standalone GitHub repo enables versioned releases, and kiloforge can manage fetching and updating them.

## Codebase Analysis

- **No existing skill management** in kiloforge — skills are manually placed in `~/.claude/skills/`
- **Claude Code skills** live in `~/.claude/skills/<skill-name>/SKILL.md` (user-level) or can be project-level
- **`cli/init.go`** — natural place to offer skill installation on first setup
- **`cli/serve.go`** — relay daemon could check for updates periodically
- **GitHub Releases API** — `GET /repos/{owner}/{repo}/releases/latest` returns tag, assets, tarball URL — no auth required for public repos

## Prerequisites (External)

- **Conductor skills repo must exist on GitHub** — the user will set this up separately
- Repo should have tagged releases (e.g., `v1.0.0`, `v1.1.0`)
- Skills should be organized as directories: `<skill-name>/SKILL.md`
- A manifest file (e.g., `manifest.json`) listing available skills and their versions

## Acceptance Criteria

- [ ] `kf skills update` is the primary command — fetches latest release and installs/updates skills to `~/.claude/skills/`
- [ ] Before overwriting, detects locally modified skills (compares against last-installed checksums) and warns the user with a list of what will be overwritten
- [ ] User must confirm with `--force` or interactive prompt to overwrite modified skills
- [ ] Unmodified skills are updated silently
- [ ] `kf skills list` shows installed skills with their versions and modification status
- [ ] `kf skills --repo <owner/repo>` configures the source repo (stored in global config)
- [ ] Version tracking: installed version and per-file checksums are recorded so update checks are fast and modification detection works
- [ ] `kf init` offers to install skills if not present (interactive prompt)
- [ ] Auto-update option: configurable in `config.json` (`"auto_update_skills": true`)
- [ ] When auto-update is enabled, the relay daemon checks for updates periodically (e.g., daily) — this is a simple HTTP call to GitHub + file copy, no Claude Code agents or token usage involved. Notifies the user that an update is available; does NOT auto-install if local modifications are detected
- [ ] Skills are installed atomically (download to temp, then move) — no partial installs
- [ ] Works with public GitHub repos (no auth token required)
- [ ] Graceful degradation: if GitHub is unreachable, warn and continue with existing skills
- [ ] Dashboard shows a notification banner when skills are not installed (with an "Install Skills" button)
- [ ] Dashboard shows a notification banner when a skill update is available (with an "Update Skills" button)
- [ ] Dashboard "Install/Update Skills" button triggers the install/update via a backend API endpoint (same logic as CLI)
- [ ] If local modifications are detected, the dashboard shows which skills are modified and asks for confirmation before overwriting
- [ ] `/-/api/skills` endpoint returns current skill status: installed version, available version, modification status
- [ ] `POST /-/api/skills/update` endpoint triggers skill update (with optional `force` param)
- [ ] Both endpoints defined in OpenAPI spec (schema-first)

## Dependencies

- External: conductor skills repo must exist on GitHub with releases
- No internal track dependencies

## Out of Scope

- Private repo support (auth tokens) — can be added later
- Per-project skill versions (all projects share global skills for now)
- Skill authoring or publishing from kiloforge
- Migrating existing skills from `~/.claude/skills/` — manual backup is the user's responsibility

## Technical Notes

**Config additions:**
```json
{
  "skills_repo": "owner/conductor-skills",
  "skills_version": "v1.2.0",
  "auto_update_skills": false
}
```

**Installation flow:**
1. Query GitHub Releases API: `GET https://api.github.com/repos/{owner}/{repo}/releases/latest`
2. Compare `tag_name` with `config.skills_version`
3. If newer (or first install): download tarball via `tarball_url`
4. Extract to temp directory
5. Copy skill directories to `~/.claude/skills/`
6. Update `config.skills_version`

**CLI commands:**
```
kiloforge skills update               # check for newer version, show changes, ask for confirmation
kiloforge skills update --force       # update without confirmation (skip modification check)
kiloforge skills list                 # show installed skills + versions + modification status
kiloforge skills --repo owner/repo    # set the source repo
kiloforge skills --auto-update        # toggle auto-update on
kiloforge skills --no-auto-update     # toggle auto-update off
```

**Modification detection:**
- On install/update, store SHA-256 checksums of each installed file in `~/.kiloforge/skills-manifest.json`
- On next update, hash current files and compare against stored checksums
- Modified files are listed with a warning: "The following skills have local modifications that will be overwritten:"
- User confirms interactively, or uses `--force` to skip

**Auto-update flow (relay daemon — no agent/token usage):**
1. On startup, and then every 24h, check latest release tag (single HTTP GET to GitHub API)
2. If newer than installed and no local modifications: download tarball and install (pure file I/O)
3. If local modifications detected: log notice, do not install
4. This is entirely handled by kiloforge's Go code — no Claude Code invocation, no token consumption

**GitHub API (no auth for public repos):**
```
GET https://api.github.com/repos/{owner}/{repo}/releases/latest
Response: { "tag_name": "v1.2.0", "tarball_url": "https://...", ... }
```

Rate limit: 60 requests/hour for unauthenticated — more than sufficient for daily checks.

---

_Generated by conductor-track-generator from prompt: "Conductor skill installation and auto-update from GitHub repo"_
