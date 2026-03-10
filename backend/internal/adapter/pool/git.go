package pool

import (
	"os/exec"
	"strings"

	"kiloforge/internal/core/port"
)

var _ port.GitRunner = (*execGitRunner)(nil)

// GitRunner abstracts git operations for testing.
type GitRunner interface {
	WorktreeAdd(path, branch string) error
	WorktreeRemove(path string) error
	ResetHardMain(worktreePath string) error
	CheckoutBranch(worktreePath, branch string) error
	CreateBranch(worktreePath, branch string) error
	DeleteBranch(branch string) error

	// Stash branch operations.
	AddAll(worktreePath string) error
	CommitWIP(worktreePath string) error
	HasCommitsAhead(worktreePath, base string) (bool, error)
	CreateStashBranch(worktreePath, stashBranch string) error
	ListStashBranches(trackID string) ([]string, error)
	MergeBranch(worktreePath, branch string) error
	DeleteBranches(branches []string) error
}

// execGitRunner runs real git commands.
type execGitRunner struct{}

func (r *execGitRunner) WorktreeAdd(path, branch string) error {
	return exec.Command("git", "worktree", "add", path, "-b", branch, "main").Run()
}

func (r *execGitRunner) WorktreeRemove(path string) error {
	return exec.Command("git", "worktree", "remove", path, "--force").Run()
}

func (r *execGitRunner) ResetHardMain(worktreePath string) error {
	return exec.Command("git", "-C", worktreePath, "reset", "--hard", "main").Run()
}

func (r *execGitRunner) CheckoutBranch(worktreePath, branch string) error {
	return exec.Command("git", "-C", worktreePath, "checkout", branch).Run()
}

func (r *execGitRunner) CreateBranch(worktreePath, branch string) error {
	return exec.Command("git", "-C", worktreePath, "checkout", "-b", branch).Run()
}

func (r *execGitRunner) DeleteBranch(branch string) error {
	return exec.Command("git", "branch", "-D", branch).Run()
}

func (r *execGitRunner) AddAll(worktreePath string) error {
	return exec.Command("git", "-C", worktreePath, "add", "-A").Run()
}

func (r *execGitRunner) CommitWIP(worktreePath string) error {
	cmd := exec.Command("git", "-C", worktreePath, "commit", "-m", "wip: auto-stash", "--allow-empty-message")
	// CommitWIP may fail if there's nothing to commit; callers should ignore that.
	return cmd.Run()
}

func (r *execGitRunner) HasCommitsAhead(worktreePath, base string) (bool, error) {
	out, err := exec.Command("git", "-C", worktreePath, "log", base+"..HEAD", "--oneline").Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func (r *execGitRunner) CreateStashBranch(worktreePath, stashBranch string) error {
	return exec.Command("git", "-C", worktreePath, "branch", stashBranch).Run()
}

func (r *execGitRunner) ListStashBranches(trackID string) ([]string, error) {
	pattern := "stash/" + trackID + "/*"
	out, err := exec.Command("git", "branch", "--list", pattern).Output()
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		b := strings.TrimSpace(line)
		if b != "" {
			branches = append(branches, b)
		}
	}
	return branches, nil
}

func (r *execGitRunner) MergeBranch(worktreePath, branch string) error {
	return exec.Command("git", "-C", worktreePath, "merge", branch, "--no-edit").Run()
}

func (r *execGitRunner) DeleteBranches(branches []string) error {
	if len(branches) == 0 {
		return nil
	}
	args := append([]string{"branch", "-D"}, branches...)
	return exec.Command("git", args...).Run()
}
