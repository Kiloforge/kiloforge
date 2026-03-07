package port

import "context"

// GiteaClient abstracts Gitea API operations.
type GiteaClient interface {
	MergePR(ctx context.Context, repo string, prNum int, method string) error
	CommentOnPR(ctx context.Context, repo string, prNum int, body string) error
	DeleteBranch(ctx context.Context, repo, branch string) error
	AddLabel(ctx context.Context, repo string, prNum int, label string) error
	GetPR(ctx context.Context, repo string, prNum int) (map[string]any, error)
	GetPRReviews(ctx context.Context, repo string, prNum int) ([]map[string]any, error)
}
