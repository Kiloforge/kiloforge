package pool

import (
	"os/exec"

	"crelay/internal/core/port"
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
