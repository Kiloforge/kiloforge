# Specification: Release Process with GoReleaser, GitHub Actions, and Homebrew

**Track ID:** release-process_20260309153000Z
**Type:** Chore
**Created:** 2026-03-09T15:30:00Z
**Status:** Draft

## Summary

Set up a complete release pipeline: `kf version` command with build-time injection, GoReleaser config for cross-platform binaries (full matrix: darwin/linux/windows × amd64/arm64), GitHub Actions workflow triggered by tag push (`v*`), Homebrew tap for macOS/Linux, and CHANGELOG.

## Context

Kiloforge has zero release infrastructure. All builds are local `make build`. There's no version command, no cross-compilation, no CI/CD, no distribution. Users need installable binaries and a repeatable release process.

## Codebase Analysis

### Current build
- `Makefile` has `build-backend` target: `cd backend && go build -o ../dist/kf ./cmd/kf`
- VCS stamping works via `GIT_DIR`/`GIT_WORK_TREE` env vars (from fix-buildvcs track)
- No `-ldflags` for version injection
- No `version` subcommand in CLI

### CLI entry point
- `backend/cmd/kf/main.go` — main package
- `backend/internal/adapter/cli/root.go` — Cobra root command, registers all subcommands

### Go module
- `backend/go.mod` — module path `github.com/...` (or local module name)

### Existing CI
- None — no `.github/workflows/` directory

## Acceptance Criteria

- [ ] `kf version` command outputs version, commit, build date, Go version, OS/arch
- [ ] Version injected at build time via `-ldflags -X`
- [ ] `.goreleaser.yaml` config with full platform matrix
- [ ] Binaries built for: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64, windows/arm64
- [ ] Archives: `.tar.gz` for darwin/linux, `.zip` for windows
- [ ] GitHub Actions workflow triggers on `v*` tag push
- [ ] Workflow runs tests, then GoReleaser to create GitHub Release with binaries
- [ ] Homebrew tap repository (`homebrew-tap`) with formula auto-updated by GoReleaser
- [ ] SHA256 checksums file generated
- [ ] CHANGELOG.md with initial v0.1.0 entry
- [ ] `make release-local` target for dry-run testing
- [ ] Release includes LICENSE and NOTICE files

## Dependencies

None.

## Blockers

None.

## Conflict Risk

- LOW across all pending tracks — this only adds new files (goreleaser config, workflow, version command) and minimal changes to existing code (ldflags in Makefile, version subcommand registration).

## Out of Scope

- Docker image publishing (future enhancement)
- Auto-release on merge to main (only tag-triggered)
- Signed binaries / notarization
- Linux package managers (apt, yum) — future enhancement
- Windows installer (MSI/NSIS) — future enhancement

## Technical Notes

### Version injection
```go
// backend/cmd/kf/main.go (or a version.go file)
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)
```

```bash
# GoReleaser handles this automatically via ldflags
```

### `kf version` output
```
kf version v0.1.0
  commit:  abc1234
  built:   2026-03-09T15:00:00Z
  go:      go1.24
  os/arch: darwin/arm64
```

### GoReleaser config (`.goreleaser.yaml`)
```yaml
version: 2
project_name: kf

before:
  hooks:
    - cd frontend && npm ci && npm run build
    - go mod tidy

builds:
  - id: kf
    dir: backend
    main: ./cmd/kf
    binary: kf
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - id: default
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    files:
      - LICENSE
      - NOTICE
      - README.md

checksum:
  name_template: "checksums.txt"

changelog:
  use: github-native

brews:
  - name: kf
    repository:
      owner: <github-org>
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    homepage: "https://kiloforge.dev"
    description: "Local AI agent orchestrator — Gitea + Claude Code"
    license: "Apache-2.0"
    install: |
      bin.install "kf"
    test: |
      system "#{bin}/kf", "version"

release:
  github:
    owner: <github-org>
    name: kiloforge
  prerelease: auto
  name_template: "v{{.Version}}"
```

### GitHub Actions workflow (`.github/workflows/release.yml`)
```yaml
name: Release
on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"

      - uses: actions/setup-node@v4
        with:
          node-version: "20"

      - name: Build frontend
        run: cd frontend && npm ci && npm run build

      - name: Run tests
        run: cd backend && go test ./...

      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

### Platform matrix (6 targets)
| OS | Arch | Archive | Notes |
|----|------|---------|-------|
| darwin | amd64 | .tar.gz | Intel Mac |
| darwin | arm64 | .tar.gz | Apple Silicon |
| linux | amd64 | .tar.gz | Standard servers |
| linux | arm64 | .tar.gz | ARM servers, Raspberry Pi |
| windows | amd64 | .zip | Standard Windows |
| windows | arm64 | .zip | ARM Windows |

### CGO_ENABLED=0
Required for cross-compilation. The pure-Go SQLite driver (`modernc.org/sqlite`) works without CGo, which is one reason it was chosen.

### Frontend embedding
The frontend must be built before GoReleaser runs the Go build, since the dashboard is embedded via `go:embed`. The `before.hooks` in goreleaser config handles this.

### Homebrew tap
Requires a separate GitHub repo `<org>/homebrew-tap`. GoReleaser auto-pushes the formula update on release. Users install via:
```bash
brew tap <org>/tap
brew install kf
```

---

_Generated by conductor-track-generator from prompt: "release process with goreleaser, tag-triggered CI, full platform matrix, homebrew"_
