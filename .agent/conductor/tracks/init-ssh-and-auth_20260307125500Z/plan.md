# Implementation Plan: SSH Key Auto-Registration and Randomized Admin Password

**Track ID:** init-ssh-and-auth_20260307125500Z

## Phase 1: Randomized Admin Password

### Task 1.1: Add password generation utility
- Create `internal/auth/password.go`
- `GeneratePassword(length int) string` — crypto/rand based, alphanumeric
- Tests: verify length, character set, uniqueness across calls

### Task 1.2: Add GiteaAdminPass to config as a resolved field
- Add `GiteaAdminPass` field to `Config` struct (if not already there from config refactor)
- Remove the hardcoded `GiteaAdminPass` constant from `config.go`
- Update all call sites that referenced `config.GiteaAdminPass` constant to use `cfg.GiteaAdminPass` field
- Tests: config serialization with password field

### Task 1.3: Implement password resolution in init
- Add `--admin-pass` flag to `crelay init` (and `crelay up`)
- Resolution: flag > saved config > generate random
- Save resolved password to config
- Print password in init success output
- Tests: verify resolution order

### Verification 1
- [ ] Password generated when no config exists
- [ ] Saved password reused on subsequent init
- [ ] Flag overrides saved password
- [ ] Hardcoded constant removed
- [ ] All call sites updated

## Phase 2: SSH Key Auto-Registration

### Task 2.1: Implement SSH key detection
- Create `internal/auth/sshkey.go`
- `DetectSSHKey() (string, string, error)` — returns (path, content, error)
- Search order: id_ed25519.pub, id_rsa.pub, id_ecdsa.pub
- Tests: mock filesystem, verify detection order

### Task 2.2: Add SSH key API to Gitea client
- Add `AddSSHKey(ctx, title, pubKey string) error` to `internal/gitea/client.go`
- `POST /api/v1/user/keys` with title and key content
- Handle 422 (key already exists) gracefully
- Tests: verify API call structure

### Task 2.3: Integrate SSH key registration into init
- After admin user configuration, detect and register SSH key
- Add `--ssh-key` flag for custom path
- If no key found: warn and continue
- If key already registered (422): log and continue
- Print registered key path in init output

### Task 2.4: Update README and docs
- Document `--admin-pass` and `--ssh-key` flags
- Note that password is generated and stored on first init
- Update security notes

### Verification 2
- [ ] SSH key auto-detected and registered
- [ ] Custom key path works via flag
- [ ] Missing SSH key is non-fatal warning
- [ ] Already-registered key handled gracefully
- [ ] Docs updated
- [ ] Build and tests pass
