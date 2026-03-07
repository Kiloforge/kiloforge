# Implementation Plan: Fix 'crelay add' to Accept Remote URLs

**Track ID:** fix-add-remote-url_20260307130000Z

## Phase 1: URL Parsing and Clone Logic

### Task 1.1: Add remote URL parser utility [x]
- Create `repoNameFromURL(rawURL string) (string, error)` in `internal/cli/add.go` (or a small helper)
- Handle SSH format: `git@host:user/repo.git` → `repo`
- Handle HTTPS format: `https://host/user/repo.git` → `repo`
- Strip `.git` suffix if present
- Return error for unparseable URLs
- Tests: verify parsing for SSH, HTTPS, with/without `.git` suffix

### Task 1.2: Add clone helper [x]
- Create `cloneRepo(ctx context.Context, remoteURL, destDir string) error`
- Runs `git clone <remoteURL> <destDir>`
- Returns clear error if clone fails (auth, network, etc.)
- Tests: verify clone command is constructed correctly

## Phase 2: Rewrite Add Command

### Task 2.1: Change argument semantics [x]
- Update `Use:` to `add <remote-url>`
- Update `Long:` description to reflect remote URL input
- Make the argument required (`cobra.ExactArgs(1)`)
- Remove local path resolution (`filepath.Abs`, `.git` check)
- Parse remote URL to derive slug (or use `--name`)

### Task 2.2: Replace local repo logic with clone flow [x]
- Determine clone destination: `filepath.Join(cfg.DataDir, "repos", slug)`
- If directory already exists, skip clone (idempotent)
- Clone remote URL into managed directory
- Add `gitea` remote to cloned repo
- Push main branch to Gitea
- Store `OriginRemote` = the provided remote URL
- Store `ProjectDir` = the clone destination path
- Remove `--origin` flag (the arg itself IS the origin)
- Remove `detectOriginRemote()` function (no longer needed)

### Task 2.3: Update tests [x]
- Update existing add command tests for new argument semantics
- Test: SSH URL parsing and clone
- Test: HTTPS URL parsing and clone
- Test: `--name` override
- Test: duplicate project detection still works
- Test: error on invalid URL

### Verification 1
- [x] `crelay add git@github.com:user/repo.git` clones and registers
- [x] `crelay add https://github.com/user/repo.git` clones and registers
- [x] Slug derived from URL, not directory name
- [x] `--name` flag overrides derived slug
- [x] `--origin` flag removed
- [x] Tests pass
- [x] Build succeeds

## Phase 3: Update Docs and Help Text

### Task 3.1: Update command help and README [x]
- Update `crelay add` usage examples in README
- Update `docs/commands.md` if it exists
- Ensure `crelay add --help` shows correct usage

### Verification 2
- [x] Help text accurate
- [x] README examples correct
- [x] All tests pass
