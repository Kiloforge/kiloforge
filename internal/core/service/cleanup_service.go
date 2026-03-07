package service

import (
	"context"
	"fmt"

	"crelay/internal/core/domain"
	"crelay/internal/core/port"
)

// CleanupOpts configures the merge and cleanup sequence.
type CleanupOpts struct {
	Tracking    *domain.PRTracking
	AgentStore  port.AgentStore
	Merger      port.Merger
	PoolReturn  port.PoolReturner
	MergeMethod string
}

// MergeAndCleanup executes the full post-approval sequence:
// merge PR, post comment, delete remote branch, return worktree, update state.
func MergeAndCleanup(ctx context.Context, opts CleanupOpts) error {
	t := opts.Tracking
	method := opts.MergeMethod
	if method == "" {
		method = "merge"
	}

	if err := opts.Merger.MergePR(ctx, t.ProjectSlug, t.PRNumber, method); err != nil {
		return fmt.Errorf("merge PR #%d: %w", t.PRNumber, err)
	}

	comment := fmt.Sprintf(
		"Merge complete. Track `%s` implementation merged.\n\n"+
			"Developer session: `%s`\nReviewer session: `%s`",
		t.TrackID, t.DeveloperSession, t.ReviewerSession,
	)
	_ = opts.Merger.CommentOnPR(ctx, t.ProjectSlug, t.PRNumber, comment)
	_ = opts.Merger.DeleteBranch(ctx, t.ProjectSlug, t.TrackID)

	if opts.PoolReturn != nil {
		if err := opts.PoolReturn.ReturnByTrackID(t.TrackID); err != nil {
			fmt.Printf("warning: return worktree: %v\n", err)
		}
	}

	if opts.AgentStore != nil {
		if t.DeveloperAgentID != "" {
			_ = opts.AgentStore.HaltAgent(t.DeveloperAgentID)
			opts.AgentStore.UpdateStatus(t.DeveloperAgentID, "completed")
		}
		if t.ReviewerAgentID != "" {
			_ = opts.AgentStore.HaltAgent(t.ReviewerAgentID)
			opts.AgentStore.UpdateStatus(t.ReviewerAgentID, "completed")
		}
		_ = opts.AgentStore.Save()
	}

	t.Status = "merged"
	return nil
}
