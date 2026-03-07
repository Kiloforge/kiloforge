package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Issue represents a Gitea issue.
type Issue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	State  string   `json:"state"`
	Labels []string `json:"-"`
}

// LabelDef defines a label to create.
type LabelDef struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// CreateIssue creates a new issue in the repository.
func (c *Client) CreateIssue(ctx context.Context, repo, title, body string, labels []string) (int, error) {
	payload := map[string]any{
		"title": title,
		"body":  body,
	}
	if len(labels) > 0 {
		payload["labels"] = labels
	}

	data, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/issues", c.username, repo), payload)
	if err != nil {
		return 0, err
	}

	var result struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, err
	}
	return result.Number, nil
}

// UpdateIssue updates an existing issue. Only non-empty fields are sent.
func (c *Client) UpdateIssue(ctx context.Context, repo string, issueNum int, title, body, state string) error {
	payload := map[string]any{}
	if title != "" {
		payload["title"] = title
	}
	if body != "" {
		payload["body"] = body
	}
	if state != "" {
		payload["state"] = state
	}
	_, err := c.do(ctx, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d", c.username, repo, issueNum), payload)
	return err
}

// GetIssues lists issues with optional state and label filters.
func (c *Client) GetIssues(ctx context.Context, repo, state string, labels []string) ([]Issue, error) {
	path := fmt.Sprintf("/api/v1/repos/%s/%s/issues", c.username, repo)
	var params []string
	if state != "" {
		params = append(params, "state="+state)
	}
	if len(labels) > 0 {
		params = append(params, "labels="+strings.Join(labels, ","))
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}

	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result []Issue
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// EnsureLabels creates any missing labels on the repository. Idempotent.
func (c *Client) EnsureLabels(ctx context.Context, repo string, labels []LabelDef) error {
	// Get existing labels.
	data, err := c.do(ctx, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/labels", c.username, repo), nil)
	if err != nil {
		return fmt.Errorf("get labels: %w", err)
	}

	var existing []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &existing); err != nil {
		return fmt.Errorf("parse labels: %w", err)
	}

	existingSet := make(map[string]bool, len(existing))
	for _, l := range existing {
		existingSet[l.Name] = true
	}

	for _, l := range labels {
		if existingSet[l.Name] {
			continue
		}
		_, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/labels", c.username, repo), l)
		if err != nil && !strings.Contains(err.Error(), "409") {
			return fmt.Errorf("create label %q: %w", l.Name, err)
		}
	}
	return nil
}
