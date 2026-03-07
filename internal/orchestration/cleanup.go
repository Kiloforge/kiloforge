package orchestration

import (
	"context"
	"fmt"

	"crelay/internal/state"
)

// Merger abstracts Gitea API operations needed for merge cleanup.
type Merger interface {
	MergePR(ctx context.Context, repo string, prNum int, method string) error
	CommentOnPR(ctx context.Context, repo string, prNum int, body string) error
	DeleteBranch(ctx context.Context, repo, branch string) error
}

// PoolReturner abstracts returning a worktree to the pool.
type PoolReturner interface {
	ReturnByTrackID(trackID string) error
}

// CleanupOpts configures the merge and cleanup sequence.
type CleanupOpts struct {
	Tracking    *PRTracking
	Store       *state.Store
	Merger      Merger
	PoolReturn  PoolReturner
	DataDir     string
	MergeMethod string // "merge", "rebase", "squash"
}

// MergeAndCleanup executes the full post-approval sequence:
// merge PR, post comment, delete remote branch, return worktree, update state.
func MergeAndCleanup(ctx context.Context, opts CleanupOpts) error {
	t := opts.Tracking
	method := opts.MergeMethod
	if method == "" {
		method = "merge"
	}

	// 1. Merge PR via API.
	if err := opts.Merger.MergePR(ctx, t.ProjectSlug, t.PRNumber, method); err != nil {
		return fmt.Errorf("merge PR #%d: %w", t.PRNumber, err)
	}

	// 2. Post final comment.
	comment := fmt.Sprintf(
		"Merge complete. Track `%s` implementation merged.\n\n"+
			"Developer session: `%s`\nReviewer session: `%s`",
		t.TrackID, t.DeveloperSession, t.ReviewerSession,
	)
	// Best effort — don't fail on comment error.
	_ = opts.Merger.CommentOnPR(ctx, t.ProjectSlug, t.PRNumber, comment)

	// 3. Delete remote branch (best effort).
	_ = opts.Merger.DeleteBranch(ctx, t.ProjectSlug, t.TrackID)

	// 4. Return worktree to pool.
	if opts.PoolReturn != nil {
		if err := opts.PoolReturn.ReturnByTrackID(t.TrackID); err != nil {
			// Log but don't fail — pool state can be fixed manually.
			fmt.Printf("warning: return worktree: %v\n", err)
		}
	}

	// 5. Terminate agent processes (best effort) and update state.
	if t.DeveloperAgentID != "" {
		_ = opts.Store.HaltAgent(t.DeveloperAgentID) // SIGINT
		opts.Store.UpdateStatus(t.DeveloperAgentID, "completed")
	}
	if t.ReviewerAgentID != "" {
		_ = opts.Store.HaltAgent(t.ReviewerAgentID) // SIGINT
		opts.Store.UpdateStatus(t.ReviewerAgentID, "completed")
	}
	if opts.DataDir != "" {
		_ = opts.Store.Save(opts.DataDir)
	}

	// 6. Update tracking status.
	t.Status = "merged"

	return nil
}
