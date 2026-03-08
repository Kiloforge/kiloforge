# Implementation Plan: Fix Init Password Display (Root Cause)

**Track ID:** fix-password-display-v3_20260309083826Z

## Phase 1: Remove Root Cause

- [ ] Task 1.1: Remove lines 83-85 from `backend/internal/adapter/gitea/manager.go` — the `m.cfg.GiteaAdminPass = ""` mutation and its comment
- [ ] Task 1.2: Add test in `manager_test.go` — verify `Configure()` does NOT clear `cfg.GiteaAdminPass` after returning

## Phase 2: Verify Display Works

- [ ] Task 2.1: Verify `init.go` line 146 will now display the password (code review — no change needed, just confirm the fix flows through)
- [ ] Task 2.2: Verify `json_adapter.Save()` still strips password from config.json (code review — already correct)
- [ ] Task 2.3: Run `make test` — all tests pass
- [ ] Task 2.4: Run `make build` — compiles cleanly

## Phase 3: Regression Prevention

- [ ] Task 3.1: Add integration-style test or comment in `init.go` documenting the password lifecycle: generate → use → display → discard (never persist)
- [ ] Task 3.2: Grep for any other places that clear `GiteaAdminPass` on a shared config pointer — ensure none exist
