package gitea

import (
	"context"
	"encoding/json"
	"fmt"
)

// Project represents a Gitea project board.
type Project struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// Column represents a column in a Gitea project board.
type Column struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

// CreateProject creates a new project board on the repository.
func (c *Client) CreateProject(ctx context.Context, repo, title, description string) (int, error) {
	payload := map[string]any{
		"title": title,
	}
	if description != "" {
		payload["description"] = description
	}

	data, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects", c.username, repo), payload)
	if err != nil {
		return 0, err
	}

	var result struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, err
	}
	return result.ID, nil
}

// GetProjects lists project boards for the repository.
func (c *Client) GetProjects(ctx context.Context, repo string) ([]Project, error) {
	data, err := c.do(ctx, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects", c.username, repo), nil)
	if err != nil {
		return nil, err
	}

	var result []Project
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateColumn adds a column to a project board.
func (c *Client) CreateColumn(ctx context.Context, projectID int, title string) (int, error) {
	payload := map[string]any{
		"title": title,
	}

	data, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/projects/%d/columns", projectID), payload)
	if err != nil {
		return 0, err
	}

	var result struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, err
	}
	return result.ID, nil
}

// GetColumns lists columns for a project board.
func (c *Client) GetColumns(ctx context.Context, projectID int) ([]Column, error) {
	data, err := c.do(ctx, "GET", fmt.Sprintf("/api/v1/projects/%d/columns", projectID), nil)
	if err != nil {
		return nil, err
	}

	var result []Column
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateCard adds an issue as a card to a project column.
func (c *Client) CreateCard(ctx context.Context, columnID, issueID int) (int, error) {
	payload := map[string]any{
		"content_id":   issueID,
		"content_type": "issue",
	}

	data, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/projects/columns/%d/cards", columnID), payload)
	if err != nil {
		return 0, err
	}

	var result struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, err
	}
	return result.ID, nil
}

// MoveCard moves a card to a different column.
func (c *Client) MoveCard(ctx context.Context, cardID, columnID int) error {
	payload := map[string]any{
		"column_id": columnID,
	}
	_, err := c.do(ctx, "PATCH", fmt.Sprintf("/api/v1/projects/columns/cards/%d", cardID), payload)
	return err
}
