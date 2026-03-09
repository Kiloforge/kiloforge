# Implementation Plan: SSH Key Auto-Registration and Randomized Admin Password

**Track ID:** init-ssh-and-auth_20260307125500Z

## Phase 1: Randomized Admin Password

### Task 1.1: Add password generation utility
- [x] Create `internal/auth/password.go`
- [x] `GeneratePassword(length int) string` — crypto/rand based, alphanumeric
- [x] Tests: verify length, character set, uniqueness across calls

### Task 1.2: Add GiteaAdminPass to config as a resolved field
- [x] Remove the hardcoded `GiteaAdminPass` default from `defaults.go`
- [x] Update defaults test
- [x] Field already existed on Config struct from config refactor

### Task 1.3: Implement password resolution in init
- [x] Add `--admin-pass` flag to `kf init`
- [x] Resolution: flag > saved config > generate random
- [x] Save resolved password to config
- [x] Print password in init success output

### Verification 1
- [x] Password generated when no config exists
- [x] Saved password reused on subsequent init
- [x] Flag overrides saved password
- [x] Hardcoded constant removed
- [x] All call sites updated

## Phase 2: SSH Key Auto-Registration

### Task 2.1: Implement SSH key detection
- [x] Create `internal/auth/sshkey.go`
- [x] `DetectSSHKey(sshDir) (string, string, error)` — returns (path, content, error)
- [x] Search order: id_ed25519.pub, id_rsa.pub, id_ecdsa.pub
- [x] Tests: verify detection order, fallback, missing key, trimming

### Task 2.2: Add SSH key API to Gitea client
- [x] Add `AddSSHKey(ctx, title, pubKey string) error` to `internal/gitea/client.go`
- [x] `POST /api/v1/user/keys` with title and key content
- [x] Handle 422 (key already exists) gracefully
- [x] Tests: verify API call structure, 422 handling, error propagation

### Task 2.3: Integrate SSH key registration into init
- [x] After admin user configuration, detect and register SSH key
- [x] Add `--ssh-key` flag for custom path
- [x] If no key found: warn and continue
- [x] If key already registered (422): log and continue
- [x] Print registered key path in init output

### Task 2.4: Update README and docs
- [x] Document `--admin-pass` and `--ssh-key` flags
- [x] Note that password is generated and stored on first init
- [x] Update security notes (removed hardcoded password reference)

### Verification 2
- [x] SSH key auto-detected and registered
- [x] Custom key path works via flag
- [x] Missing SSH key is non-fatal warning
- [x] Already-registered key handled gracefully
- [x] Docs updated
- [x] Build and tests pass
