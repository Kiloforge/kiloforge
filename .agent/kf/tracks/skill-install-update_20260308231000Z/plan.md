# Implementation Plan: Conductor Skill Installation and Auto-Update

[x] Phase 1: Config and GitHub Client (3 tasks)

[x] Task 1.1: Add skills config fields
- **File:** `backend/internal/adapter/config/config.go`
- Add `SkillsRepo string` (`json:"skills_repo,omitempty"`)
- Add `SkillsVersion string` (`json:"skills_version,omitempty"`)
- Add `AutoUpdateSkills bool` (`json:"auto_update_skills,omitempty"`)
- Update merger to handle new fields

[x] Task 1.2: Create GitHub release checker
- **File:** `backend/internal/adapter/skills/github.go`
- `LatestRelease(repo string) (Release, error)` — calls GitHub Releases API
- `Release` struct: `TagName`, `TarballURL`, `PublishedAt`
- No auth required (public repos)
- Respect rate limits, timeout after 10s

[x] Task 1.3: Test GitHub release checker
- **File:** `backend/internal/adapter/skills/github_test.go`
- Test JSON parsing of GitHub release response
- Test error handling (404, rate limited, network timeout)
- Use httptest server for unit tests (no real GitHub calls)

[x] Phase 2: Skill Installer (4 tasks)

[x] Task 2.1: Create skill installer with checksum tracking
- **File:** `backend/internal/adapter/skills/installer.go`
- `Install(tarballURL, destDir string) ([]InstalledSkill, error)` — download, extract, copy skills
- Download tarball to temp file
- Extract and identify skill directories (contain SKILL.md)
- Copy atomically: extract to temp dir, then rename to final location
- After install, compute and store SHA-256 checksums in `~/.kiloforge/skills-manifest.json`
- Return list of installed skills

[x] Task 2.2: Create modification detector
- **File:** `backend/internal/adapter/skills/manifest.go`
- `SkillsManifest` struct: maps `skill-name/file-path` → SHA-256 checksum
- `DetectModified(skillsDir string, manifest SkillsManifest) []ModifiedSkill`
- Compares current file hashes against stored manifest
- `ModifiedSkill` struct: `Name`, `Files []string` (list of changed files)

[x] Task 2.3: Create skill lister
- **File:** `backend/internal/adapter/skills/installer.go`
- `ListInstalled(skillsDir string, manifest SkillsManifest) []InstalledSkill`
- `InstalledSkill` struct: `Name`, `Path`, `Modified bool`

[x] Task 2.4: Add version comparison
- **File:** `backend/internal/adapter/skills/version.go`
- `IsNewer(current, latest string) bool` — semver comparison
- Handle `v` prefix (`v1.2.0` vs `1.2.0`)

[x] Task 2.5: Test installer and modification detection
- **File:** `backend/internal/adapter/skills/installer_test.go`
- Test tarball extraction with a synthetic tarball
- Test atomic install (temp → rename)
- Test checksum computation and storage
- Test modification detection (modified file vs unmodified)
- Test version comparison

[x] Phase 3: CLI Commands (4 tasks)

[x] Task 3.1: Create skills command group
- **File:** `backend/internal/adapter/cli/skills.go`
- `kf skills update` — check for newer version, detect modifications, ask for confirmation, install
- `kf skills update --force` — skip modification check / confirmation
- `kf skills list` — show installed skills with version and modification status
- Register in root.go

[x] Task 3.2: Add --repo and auto-update flags
- **File:** `backend/internal/adapter/cli/skills.go`
- `kf skills --repo owner/repo` — set source repo in config
- `kf skills --auto-update` / `--no-auto-update` — toggle config flag

[x] Task 3.3: Offer skills install during init
- **File:** `backend/internal/adapter/cli/init.go`
- After successful init, if no skills installed and `skills_repo` is configured, prompt to install
- Non-interactive: skip if stdin is not a terminal

[x] Task 3.4: Test CLI commands
- **File:** `backend/internal/adapter/cli/skills_test.go`
- Test list command with mock skills directory
- Test repo config persistence

[x] Phase 4: Auto-Update in Daemon (3 tasks)

[x] Task 4.1: Add periodic update check to relay daemon
- **File:** `backend/internal/adapter/cli/serve.go` or new `skills/updater.go`
- On startup: check if auto-update enabled, run update check
- Schedule check every 24h via `time.Ticker`
- Notify that update is available; do not auto-install if modifications detected
- Log to relay.log

[x] Task 4.2: Test auto-update logic
- Test that checker respects `auto_update_skills` config flag
- Test that checker skips when version is current
- Test graceful degradation when GitHub is unreachable

[x] Task 4.3: Run full test suite
- `make test` — all pass
- `make test-smoke` — smoke tests pass

[x] Phase 5: Dashboard API — Skills Endpoints (3 tasks)

[x] Task 5.1: Add skills endpoints to OpenAPI spec
- **File:** `backend/api/openapi.yaml`
- `GET /-/api/skills` — returns `{ installed_version, available_version, skills: [{ name, modified, files }], update_available }`
- `POST /-/api/skills/update` — triggers install/update, accepts `{ force: bool }` body
- Define `SkillStatus`, `SkillDetail`, `SkillUpdateRequest`, `SkillUpdateResponse` schemas
- Run `oapi-codegen` to regenerate server/types

[x] Task 5.2: Implement skills API handler
- **File:** `backend/internal/adapter/rest/skills_handler.go`
- Implement generated strict handler interface for both endpoints
- `GET` handler: calls installer's `ListInstalled` + GitHub checker's `LatestRelease`, returns combined status
- `POST` handler: calls installer's `Install` with force flag, returns result
- Wire into server registration

[x] Task 5.3: Test skills API endpoints
- **File:** `backend/internal/adapter/rest/skills_handler_test.go`
- Test GET returns correct status (no skills, up-to-date, update available, modified)
- Test POST triggers update and returns result
- Test POST without force when modifications exist returns error with modified file list

[x] Phase 6: Dashboard UI — Skill Notifications (3 tasks)

[x] Task 6.1: Add skills status hook
- **File:** `frontend/src/hooks/useSkillsStatus.ts`
- Fetch `/-/api/skills` on mount and on interval (every 60s)
- Expose `{ installed, updateAvailable, modified, loading }` state

[x] Task 6.2: Add skill notification banner component
- **File:** `frontend/src/components/SkillsBanner.tsx`
- If skills not installed: warning banner with "Install Skills" button
- If update available: info banner with "Update Skills" button
- If local modifications detected on update: show list of modified skills, ask for confirmation before proceeding
- Calls `POST /-/api/skills/update` with appropriate force flag
- Shows progress/result feedback

[x] Task 6.3: Mount banner in dashboard layout
- **File:** `frontend/src/App.tsx` (or layout component)
- Render `SkillsBanner` at top of dashboard, above project sections
- Banner dismisses after successful install/update

[x] Phase 7: Documentation (1 task)

[x] Task 7.1: Update README with skills commands
- Add `kf skills` section to README command reference
- Document initial setup: `kf skills --repo owner/conductor-skills && kiloforge skills update`
- Mention dashboard skill management as alternative to CLI
