package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"crelay/internal/core/port"
)

var _ port.BoardGiteaClient = (*Client)(nil)

// EnsureLabel creates a label if it doesn't exist and returns its ID.
func (c *Client) EnsureLabel(ctx context.Context, repo, name, color string) (int, error) {
	payload := map[string]any{
		"name":  name,
		"color": color,
	}
	data, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/labels", c.username, repo), payload)
	if err == nil {
		var result struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			return 0, fmt.Errorf("parse created label: %w", err)
		}
		return result.ID, nil
	}

	// If 409 conflict, label already exists — find it.
	if !strings.Contains(err.Error(), "409") {
		return 0, fmt.Errorf("create label %q: %w", name, err)
	}

	data, err = c.do(ctx, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/labels", c.username, repo), nil)
	if err != nil {
		return 0, fmt.Errorf("get labels: %w", err)
	}
	var labels []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &labels); err != nil {
		return 0, fmt.Errorf("parse labels: %w", err)
	}
	for _, l := range labels {
		if l.Name == name {
			return l.ID, nil
		}
	}
	return 0, fmt.Errorf("label %q not found after 409", name)
}

// ListProjects lists project boards as port.ProjectInfo.
func (c *Client) ListProjects(ctx context.Context, repo string) ([]port.ProjectInfo, error) {
	projects, err := c.GetProjects(ctx, repo)
	if err != nil {
		return nil, err
	}
	result := make([]port.ProjectInfo, len(projects))
	for i, p := range projects {
		result[i] = port.ProjectInfo{ID: p.ID, Title: p.Title}
	}
	return result, nil
}

// ListColumns lists columns as port.ColumnInfo.
func (c *Client) ListColumns(ctx context.Context, projectID int) ([]port.ColumnInfo, error) {
	columns, err := c.GetColumns(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]port.ColumnInfo, len(columns))
	for i, col := range columns {
		result[i] = port.ColumnInfo{ID: col.ID, Title: col.Title}
	}
	return result, nil
}
