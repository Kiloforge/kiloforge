# Implementation Plan: Remove Password from Config Persistence

## Phase 1: Token-Based Client Constructor (3 tasks)

### Task 1.1: Add NewClientWithToken constructor
- [x] **File:** `backend/internal/adapter/gitea/client.go`
- Add `NewClientWithToken(baseURL, username, token string) *Client` that sets the token directly
- Keep `NewClient(baseURL, username, password)` for init-time BasicAuth usage

### Task 1.2: Add test for NewClientWithToken
- [x] **File:** `backend/internal/adapter/gitea/client_test.go`
- Test that `NewClientWithToken` sets token auth (Authorization header is `token <value>`, not BasicAuth)
- Test that requests do NOT include BasicAuth when token is set

### Task 1.3: Verify existing client tests still pass
- [x] Run `go test ./internal/adapter/gitea/...`

## Phase 2: Clear Password After Init (3 tasks)

### Task 2.1: Clear GiteaAdminPass in Configure()
- [x] **File:** `backend/internal/adapter/gitea/manager.go`
- After token creation succeeds in `Configure()`, set `m.cfg.GiteaAdminPass = ""`
- This ensures the subsequent `cfg.Save()` in `init.go` omits the field (via `omitempty`)

### Task 2.2: Refactor CLI call sites to use token-only client
- [x] **Files:** `backend/internal/adapter/cli/up.go`, `status.go`, `add.go`, `sync.go`, `board.go`, `implement.go`, `down.go`, `serve.go`
- **File:** `backend/internal/adapter/rest/server.go`
- Replace pattern: `gitea.NewClient(url, user, pass) + if token != "" { client.SetToken(token) }`
- With: `gitea.NewClientWithToken(url, user, cfg.APIToken)`

### Task 2.3: Update test helpers
- [x] **File:** `backend/internal/adapter/rest/server_test.go`
- Update test helpers that create clients to use `NewClientWithToken` where appropriate

## Phase 3: Verification (2 tasks)

### Task 3.1: Add integration test for password clearing
- [x] **File:** `backend/internal/adapter/config/json_adapter_test.go`
- Test that after saving config with empty password, JSON omits `gitea_admin_pass`
- Test that loading an old config with `gitea_admin_pass` present still works (backward compat)

### Task 3.2: Run full test suite
- [x] `go test ./...` — all tests pass
