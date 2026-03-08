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

// GiteaIssueClient extends GiteaClient with issue management operations.
type GiteaIssueClient interface {
	GiteaClient

	// Issue management
	CreateIssue(ctx context.Context, repo, title, body string, labels []string) (int, error)
	UpdateIssue(ctx context.Context, repo string, issueNum int, title, body, state string) error
}
