package port

// PoolReturner abstracts returning a worktree to the pool.
type PoolReturner interface {
	ReturnByTrackID(trackID string) error
}
