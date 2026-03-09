# Implementation Plan: Interactive SSH Key Selection for Add Command

**Track ID:** ssh-key-selection_20260309100000Z

## Phase 1: SSH Key Discovery

- [x] Task 1.1: Add `SSHKeyInfo` struct to `auth/sshkey.go` — fields: `Name`, `Path`, `Type` (ed25519/rsa/ecdsa), `PubContent`
- [x] Task 1.2: Add `DiscoverSSHKeys(sshDir string) []SSHKeyInfo` — scan `~/.ssh/` for private key files, detect type from name
- [x] Task 1.3: Unit tests for `DiscoverSSHKeys` — mock directory with multiple key types, empty directory, no keys

## Phase 2: Interactive Selection Prompt

- [x] Task 2.1: Add `PromptSSHKeySelection(keys []SSHKeyInfo, r io.Reader, w io.Writer) (string, error)` in a new `cli/prompt.go` — display numbered list, read selection, return chosen key path
- [x] Task 2.2: Handle edge cases — single key auto-select, no keys found (return empty), invalid input (re-prompt), "skip" option
- [x] Task 2.3: Detect non-interactive stdin (nil reader) — fall back to auto-select first key
- [x] Task 2.4: Unit tests for prompt — simulate stdin input, test single key, no keys, invalid input, skip selection

## Phase 3: Integration into Add Command

- [x] Task 3.1: In `runAdd()`, after detecting SSH remote and no `--ssh-key` flag: call `DiscoverSSHKeys`, then `PromptSSHKeySelection`
- [x] Task 3.2: Wire selected key path into existing `sshKeyPath`/`sshEnv` variables (rest of add flow unchanged)
- [x] Task 3.3: Skip prompt entirely for HTTPS remotes

## Phase 4: Verification

- [x] Task 4.1: Verify `go test ./...` passes
- [x] Task 4.2: Verify `make build` succeeds
- [x] Task 4.3: Manual verification — `kf add git@github.com:user/repo.git` shows key selection prompt
