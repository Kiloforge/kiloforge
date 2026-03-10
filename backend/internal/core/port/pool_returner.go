package port

// PoolReturner abstracts returning a worktree to the pool.
type PoolReturner interface {
	ReturnByTrackID(trackID string) error
	StashByTrackID(trackID string) error
	CleanupStash(trackID string) error
}
