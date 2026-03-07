package port

// GitRunner abstracts git operations for worktree management.
type GitRunner interface {
	WorktreeAdd(path, branch string) error
	WorktreeRemove(path string) error
	ResetHardMain(worktreePath string) error
	CheckoutBranch(worktreePath, branch string) error
	CreateBranch(worktreePath, branch string) error
	DeleteBranch(branch string) error
}
