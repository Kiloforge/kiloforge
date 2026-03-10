package port

// GitRunner abstracts git operations for worktree management.
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
