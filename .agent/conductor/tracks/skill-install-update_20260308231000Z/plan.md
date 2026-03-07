# Implementation Plan: Conductor Skill Installation and Auto-Update

## Phase 1: Config and GitHub Client (3 tasks)

### Task 1.1: Add skills config fields
- **File:** `backend/internal/adapter/config/config.go`
- Add `SkillsRepo string` (`json:"skills_repo,omitempty"`)
- Add `SkillsVersion string` (`json:"skills_version,omitempty"`)
- Add `AutoUpdateSkills bool` (`json:"auto_update_skills,omitempty"`)
- Update merger to handle new fields

### Task 1.2: Create GitHub release checker
- **File:** `backend/internal/adapter/skills/github.go`
- `LatestRelease(repo string) (Release, error)` — calls GitHub Releases API
- `Release` struct: `TagName`, `TarballURL`, `PublishedAt`
- No auth required (public repos)
- Respect rate limits, timeout after 10s

### Task 1.3: Test GitHub release checker
- **File:** `backend/internal/adapter/skills/github_test.go`
- Test JSON parsing of GitHub release response
- Test error handling (404, rate limited, network timeout)
- Use httptest server for unit tests (no real GitHub calls)

## Phase 2: Skill Installer (4 tasks)

### Task 2.1: Create skill installer with checksum tracking
- **File:** `backend/internal/adapter/skills/installer.go`
- `Install(tarballURL, destDir string) ([]InstalledSkill, error)` — download, extract, copy skills
- Download tarball to temp file
- Extract and identify skill directories (contain SKILL.md)
- Copy atomically: extract to temp dir, then rename to final location
- After install, compute and store SHA-256 checksums in `~/.crelay/skills-manifest.json`
- Return list of installed skills

### Task 2.2: Create modification detector
- **File:** `backend/internal/adapter/skills/manifest.go`
- `SkillsManifest` struct: maps `skill-name/file-path` → SHA-256 checksum
- `DetectModified(skillsDir string, manifest SkillsManifest) []ModifiedSkill`
- Compares current file hashes against stored manifest
- `ModifiedSkill` struct: `Name`, `Files []string` (list of changed files)

### Task 2.3: Create skill lister
- **File:** `backend/internal/adapter/skills/installer.go`
- `ListInstalled(skillsDir string, manifest SkillsManifest) []InstalledSkill`
- `InstalledSkill` struct: `Name`, `Path`, `Modified bool`

### Task 2.4: Add version comparison
- **File:** `backend/internal/adapter/skills/version.go`
- `IsNewer(current, latest string) bool` — semver comparison
- Handle `v` prefix (`v1.2.0` vs `1.2.0`)

### Task 2.5: Test installer and modification detection
- **File:** `backend/internal/adapter/skills/installer_test.go`
- Test tarball extraction with a synthetic tarball
- Test atomic install (temp → rename)
- Test checksum computation and storage
- Test modification detection (modified file vs unmodified)
- Test version comparison

## Phase 3: CLI Commands (4 tasks)

### Task 3.1: Create skills command group
- **File:** `backend/internal/adapter/cli/skills.go`
- `crelay skills update` — check for newer version, detect modifications, ask for confirmation, install
- `crelay skills update --force` — skip modification check / confirmation
- `crelay skills list` — show installed skills with version and modification status
- Register in root.go

### Task 3.2: Add --repo and auto-update flags
- **File:** `backend/internal/adapter/cli/skills.go`
- `crelay skills --repo owner/repo` — set source repo in config
- `crelay skills --auto-update` / `--no-auto-update` — toggle config flag

### Task 3.3: Offer skills install during init
- **File:** `backend/internal/adapter/cli/init.go`
- After successful init, if no skills installed and `skills_repo` is configured, prompt to install
- Non-interactive: skip if stdin is not a terminal

### Task 3.4: Test CLI commands
- **File:** `backend/internal/adapter/cli/skills_test.go`
- Test list command with mock skills directory
- Test repo config persistence

## Phase 4: Auto-Update in Daemon (3 tasks)

### Task 4.1: Add periodic update check to relay daemon
- **File:** `backend/internal/adapter/cli/serve.go` or new `skills/updater.go`
- On startup: check if auto-update enabled, run update check
- Schedule check every 24h via `time.Ticker`
- Notify that update is available; do not auto-install if modifications detected
- Log to relay.log

### Task 4.2: Test auto-update logic
- Test that checker respects `auto_update_skills` config flag
- Test that checker skips when version is current
- Test graceful degradation when GitHub is unreachable

### Task 4.3: Run full test suite
- `make test` — all pass
- `make test-smoke` — smoke tests pass

## Phase 5: Documentation (1 task)

### Task 5.1: Update README with skills commands
- Add `crelay skills` section to README command reference
- Document initial setup: `crelay skills --repo owner/conductor-skills && crelay skills update`
