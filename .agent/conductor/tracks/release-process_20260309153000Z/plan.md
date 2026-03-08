# Implementation Plan: Release Process with GoReleaser, GitHub Actions, and Homebrew

**Track ID:** release-process_20260309153000Z

## Phase 1: Version Command

- [ ] Task 1.1: Create `backend/cmd/kf/version.go` — define `version`, `commit`, `date` variables with default "dev" values
- [ ] Task 1.2: Add `version` subcommand to `backend/internal/adapter/cli/root.go` — prints version, commit, date, Go version, OS/arch
- [ ] Task 1.3: Update Makefile `build-backend` target to inject version via `-ldflags -X` using git describe
- [ ] Task 1.4: Verify `kf version` outputs correct info after `make build`

## Phase 2: GoReleaser Configuration

- [ ] Task 2.1: Install goreleaser locally for testing (`go install github.com/goreleaser/goreleaser/v2@latest`)
- [ ] Task 2.2: Create `.goreleaser.yaml` — builds config with full platform matrix (darwin/linux/windows × amd64/arm64), CGO_ENABLED=0, ldflags, archives (.tar.gz + .zip for windows), checksums
- [ ] Task 2.3: Add `before.hooks` for frontend build (`cd frontend && npm ci && npm run build`)
- [ ] Task 2.4: Configure Homebrew tap section (repository, formula, install/test blocks)
- [ ] Task 2.5: Add `make release-local` target — runs `goreleaser release --snapshot --clean` for dry-run testing
- [ ] Task 2.6: Test `make release-local` — verify 6 binaries are produced, archives are correct format

## Phase 3: GitHub Actions Workflow

- [ ] Task 3.1: Create `.github/workflows/release.yml` — triggered on `v*` tag push, sets up Go 1.24 + Node 20, builds frontend, runs tests, runs goreleaser
- [ ] Task 3.2: Create `.github/workflows/ci.yml` — triggered on push/PR to main, runs `go test ./...` and `npm run build` (basic CI)
- [ ] Task 3.3: Document required GitHub secrets: `HOMEBREW_TAP_TOKEN` for tap repo push

## Phase 4: CHANGELOG and Documentation

- [ ] Task 4.1: Create `CHANGELOG.md` with initial v0.1.0 entry summarizing current feature set
- [ ] Task 4.2: Update `README.md` — add installation section (Homebrew, binary download, build from source)
- [ ] Task 4.3: Add release process documentation to README or CONTRIBUTING.md (how to tag and release)

## Phase 5: Verification

- [ ] Task 5.1: Verify `make build` still works with ldflags
- [ ] Task 5.2: Verify `make release-local` produces all 6 platform binaries
- [ ] Task 5.3: Verify `go test ./...` passes
- [ ] Task 5.4: Verify `.github/workflows/release.yml` syntax is valid (act or yamllint)
