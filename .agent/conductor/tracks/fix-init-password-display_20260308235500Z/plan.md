# Implementation Plan: Fix Init Password Display

## Phase 1: Fix Password Handling (2 tasks)

### Task 1.1: Prevent password persistence in config.json [x]
- **File:** `backend/internal/adapter/config/json_adapter.go`
- `Save()` now strips GiteaAdminPass before marshalling (copies config to avoid mutation)
- Verify: `config.json` should never contain `gitea_admin_pass`

### Task 1.2: Fix re-init display and first-init display [x]
- **File:** `backend/internal/adapter/cli/init.go`
- On re-init (Gitea already running): if password came from flag/env, display it. Otherwise, tell user password is not stored and suggest `--admin-pass`.
- On first init: display password in success output (already works, verified)
- On all paths: if user provided `--admin-pass` or env var, always display it

## Phase 2: Verification (1 task)

### Task 2.1: Test and verify [x]
- First init without flag → password generated, displayed, NOT in config.json ✓
- Re-init without flag → "password not stored" message, suggests --admin-pass ✓
- Init with --admin-pass → password displayed on both first and re-init ✓
- Init with KF_GITEA_ADMIN_PASS env → same as flag behavior ✓
- Run test suite ✓ (make test + make build pass)
