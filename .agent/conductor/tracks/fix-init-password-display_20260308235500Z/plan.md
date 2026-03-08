# Implementation Plan: Fix Init Password Display

## Phase 1: Fix Password Handling (2 tasks)

### Task 1.1: Prevent password persistence in config.json
- **File:** `backend/internal/adapter/cli/init.go`
- Before `cfg.Save()`, clear `GiteaAdminPass` to prevent persistence
- Restore after save for display purposes
- Verify: `config.json` should never contain `gitea_admin_pass`

### Task 1.2: Fix re-init display and first-init display
- **File:** `backend/internal/adapter/cli/init.go`
- On re-init (Gitea already running): if password came from flag/env, display it. Otherwise, tell user password is not stored and suggest `--admin-pass`.
- On first init: display password in success output (already works, just verify)
- On all paths: if user provided `--admin-pass` or env var, always display it

## Phase 2: Verification (1 task)

### Task 2.1: Test and verify
- First init without flag → password generated, displayed, NOT in config.json
- Re-init without flag → "password not stored" message, suggests --admin-pass
- Init with --admin-pass → password displayed on both first and re-init
- Init with CRELAY_GITEA_ADMIN_PASS env → same as flag behavior
- Run test suite
