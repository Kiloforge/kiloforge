package port

import "context"

// ProjectGiteaClient abstracts the Gitea operations needed by ProjectService.
type ProjectGiteaClient interface {
	CreateRepo(ctx context.Context, name string) error
	CreateWebhook(ctx context.Context, repoName string, orchPort int) error
	DeleteRepo(ctx context.Context, repoName string) error
	DeleteAllWebhooks(ctx context.Context, repoName string) error
	BaseURL() string
}
