package port

import (
	"context"
)

// BoardGiteaClient abstracts the Gitea operations needed by the board service.
type BoardGiteaClient interface {
	// Labels
	EnsureLabel(ctx context.Context, repo, name, color string) (int, error)

	// Issues
	CreateIssue(ctx context.Context, repo, title, body string, labels []string) (int, error)
	UpdateIssue(ctx context.Context, repo string, issueNum int, title, body, state string) error

	// Project boards
	CreateProject(ctx context.Context, repo, title, description string) (int, error)
	ListProjects(ctx context.Context, repo string) ([]ProjectInfo, error)
	CreateColumn(ctx context.Context, projectID int, title string) (int, error)
	ListColumns(ctx context.Context, projectID int) ([]ColumnInfo, error)
	CreateCard(ctx context.Context, columnID, issueID int) (int, error)
	MoveCard(ctx context.Context, cardID, columnID int) error
}

// ProjectInfo is a minimal project board representation for the port layer.
type ProjectInfo struct {
	ID    int
	Title string
}

// ColumnInfo is a minimal column representation for the port layer.
type ColumnInfo struct {
	ID    int
	Title string
}
