package port

import "context"

// Merger abstracts Gitea API operations needed for merge cleanup.
type Merger interface {
	MergePR(ctx context.Context, repo string, prNum int, method string) error
	CommentOnPR(ctx context.Context, repo string, prNum int, body string) error
	DeleteBranch(ctx context.Context, repo, branch string) error
}
