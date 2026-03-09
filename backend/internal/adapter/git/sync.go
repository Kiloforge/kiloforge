package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Sync status constants.
const (
	StatusSynced  = "synced"
	StatusAhead   = "ahead"
	StatusBehind  = "behind"
	StatusDiverged = "diverged"
	StatusUnknown = "unknown"
)

// SyncStatusResult contains the result of a sync status check.
type SyncStatusResult struct {
	LocalBranch  string `json:"local_branch"`
	RemoteURL    string `json:"remote_url"`
	Ahead        int    `json:"ahead"`
	Behind       int    `json:"behind"`
	Status       string `json:"status"`
}

// PushResult contains the result of a push operation.
type PushResult struct {
	Success      bool   `json:"success"`
	LocalBranch  string `json:"local_branch"`
	RemoteBranch string `json:"remote_branch"`
}

// PullResult contains the result of a pull operation.
type PullResult struct {
	Success bool   `json:"success"`
	NewHead string `json:"new_head"`
}

// GitSync provides git sync operations (push, pull, fetch, status).
type GitSync struct{}

// New creates a new GitSync instance.
func New() *GitSync {
	return &GitSync{}
}

// FetchOrigin fetches from the origin remote.
func (gs *GitSync) FetchOrigin(ctx context.Context, projectDir, sshKeyPath string) error {
	cmd := gs.gitCmd(ctx, projectDir, sshKeyPath, "fetch", "origin")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fetch origin: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// PushToRemote pushes local main to a remote branch.
// Runs: git push origin localBranch:refs/heads/remoteBranch
func (gs *GitSync) PushToRemote(ctx context.Context, projectDir, localBranch, remoteBranch, sshKeyPath string) (*PushResult, error) {
	refspec := fmt.Sprintf("%s:refs/heads/%s", localBranch, remoteBranch)
	cmd := gs.gitCmd(ctx, projectDir, sshKeyPath, "push", "origin", refspec)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("push failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return &PushResult{
		Success:      true,
		LocalBranch:  localBranch,
		RemoteBranch: remoteBranch,
	}, nil
}

// PullFromRemote fetches from origin and fast-forward merges the specified branch.
// Returns an error if the branches have diverged.
func (gs *GitSync) PullFromRemote(ctx context.Context, projectDir, remoteBranch, sshKeyPath string) (*PullResult, error) {
	// Fetch first.
	if err := gs.FetchOrigin(ctx, projectDir, sshKeyPath); err != nil {
		return nil, err
	}

	// Check for divergence before attempting merge.
	ahead, behind, err := gs.revListCounts(ctx, projectDir, "main", "origin/"+remoteBranch)
	if err != nil {
		return nil, fmt.Errorf("check divergence: %w", err)
	}
	if ahead > 0 && behind > 0 {
		return nil, fmt.Errorf("branches have diverged (local %d ahead, %d behind origin/%s) — resolve manually", ahead, behind, remoteBranch)
	}

	// Fast-forward merge.
	cmd := gs.gitCmd(ctx, projectDir, "", "merge", "--ff-only", "origin/"+remoteBranch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("merge failed: %s: %w", strings.TrimSpace(string(out)), err)
	}

	// Get new HEAD.
	headCmd := gs.gitCmd(ctx, projectDir, "", "rev-parse", "--short", "HEAD")
	headOut, err := headCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("get HEAD: %w", err)
	}

	return &PullResult{
		Success: true,
		NewHead: strings.TrimSpace(string(headOut)),
	}, nil
}

// SyncStatus returns the ahead/behind counts and sync status.
// It fetches from origin first to ensure counts are current.
func (gs *GitSync) SyncStatus(ctx context.Context, projectDir, sshKeyPath string) (*SyncStatusResult, error) {
	// Fetch to get latest remote state.
	if err := gs.FetchOrigin(ctx, projectDir, sshKeyPath); err != nil {
		return nil, err
	}

	// Get current branch.
	branchCmd := gs.gitCmd(ctx, projectDir, "", "rev-parse", "--abbrev-ref", "HEAD")
	branchOut, err := branchCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("get branch: %w", err)
	}
	localBranch := strings.TrimSpace(string(branchOut))

	// Get remote URL.
	urlCmd := gs.gitCmd(ctx, projectDir, "", "remote", "get-url", "origin")
	urlOut, _ := urlCmd.Output()
	remoteURL := strings.TrimSpace(string(urlOut))

	// Get ahead/behind counts.
	ahead, behind, err := gs.revListCounts(ctx, projectDir, localBranch, "origin/"+localBranch)
	if err != nil {
		return &SyncStatusResult{
			LocalBranch: localBranch,
			RemoteURL:   remoteURL,
			Status:      StatusUnknown,
		}, nil
	}

	status := StatusSynced
	switch {
	case ahead > 0 && behind > 0:
		status = StatusDiverged
	case ahead > 0:
		status = StatusAhead
	case behind > 0:
		status = StatusBehind
	}

	return &SyncStatusResult{
		LocalBranch: localBranch,
		RemoteURL:   remoteURL,
		Ahead:       ahead,
		Behind:      behind,
		Status:      status,
	}, nil
}

// revListCounts returns the ahead/behind counts between local and remote refs.
func (gs *GitSync) revListCounts(ctx context.Context, projectDir, local, remote string) (ahead, behind int, err error) {
	cmd := gs.gitCmd(ctx, projectDir, "", "rev-list", "--left-right", "--count", local+"..."+remote)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("rev-list: %w", err)
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %q", string(out))
	}
	ahead, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse ahead count: %w", err)
	}
	behind, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse behind count: %w", err)
	}
	return ahead, behind, nil
}

// gitCmd creates a git command for the given directory with optional SSH key env.
// It clears GIT_DIR and GIT_WORK_TREE from the environment so that -C works
// correctly even when the parent process runs in a git worktree.
func (gs *GitSync) gitCmd(ctx context.Context, projectDir, sshKeyPath string, args ...string) *exec.Cmd {
	fullArgs := append([]string{"-C", projectDir}, args...)
	cmd := exec.CommandContext(ctx, "git", fullArgs...)

	// Build a clean env without GIT_DIR/GIT_WORK_TREE which would override -C.
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GIT_DIR=") || strings.HasPrefix(e, "GIT_WORK_TREE=") {
			continue
		}
		env = append(env, e)
	}
	if sshKeyPath != "" {
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o IdentitiesOnly=yes", sshKeyPath))
	}
	cmd.Env = env
	return cmd
}
