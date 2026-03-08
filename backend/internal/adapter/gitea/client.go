package gitea

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"kiloforge/internal/core/port"
)

var _ port.GiteaClient = (*Client)(nil)
var _ port.Merger = (*Client)(nil)

// Client wraps the Gitea REST API.
type Client struct {
	baseURL  string
	username string
	password string
	token    string
	http     *http.Client
}

func NewClient(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		http:     &http.Client{},
	}
}

// NewClientWithToken creates a Client that authenticates with an API token.
// Use this for all post-init operations where token auth is available.
func NewClientWithToken(baseURL, username, token string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		token:    token,
		http:     &http.Client{},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	} else {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return respBody, fmt.Errorf("API %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// CreateToken creates an API access token.
func (c *Client) CreateToken(ctx context.Context, name string) (string, error) {
	payload := map[string]any{
		"name":   name,
		"scopes": []string{"all"},
	}
	data, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/users/%s/tokens", c.username), payload)
	if err != nil {
		return "", err
	}

	var result struct {
		SHA1 string `json:"sha1"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	return result.SHA1, nil
}

// CreateRepo creates a new repository.
func (c *Client) CreateRepo(ctx context.Context, name string) error {
	payload := map[string]any{
		"name":          name,
		"auto_init":     false,
		"private":       false,
		"default_branch": "main",
	}
	_, err := c.do(ctx, "POST", "/api/v1/user/repos", payload)
	return err
}

// CreateWebhook registers a webhook on the repository.
func (c *Client) CreateWebhook(ctx context.Context, repoName string, orchPort int) error {
	// Use host.docker.internal so the container can reach the host orchestrator.
	hookURL := fmt.Sprintf("http://host.docker.internal:%d/webhook", orchPort)

	payload := map[string]any{
		"type":   "gitea",
		"active": true,
		"config": map[string]any{
			"url":          hookURL,
			"content_type": "json",
			"secret":       "",
		},
		"events": []string{
			"issues",
			"issue_comment",
			"pull_request",
			"pull_request_review",
			"pull_request_comment",
			"push",
		},
	}
	_, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/hooks", c.username, repoName), payload)
	return err
}

// CheckVersion calls the Gitea version API to verify the server is running.
func (c *Client) CheckVersion(ctx context.Context) (string, error) {
	data, err := c.do(ctx, "GET", "/api/v1/version", nil)
	if err != nil {
		return "", err
	}
	var result struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	return result.Version, nil
}

// AddSSHKey registers an SSH public key for the authenticated user.
// Returns nil on success or if the key already exists (HTTP 422).
func (c *Client) AddSSHKey(ctx context.Context, title, pubKey string) error {
	payload := map[string]any{
		"title": title,
		"key":   pubKey,
	}
	_, err := c.do(ctx, "POST", "/api/v1/user/keys", payload)
	if err != nil && isAlreadyExists(err) {
		return nil
	}
	return err
}

func isAlreadyExists(err error) bool {
	return err != nil && strings.Contains(err.Error(), "422")
}

// GetPR fetches a pull request by number.
func (c *Client) GetPR(ctx context.Context, repoName string, prNumber int) (map[string]any, error) {
	data, err := c.do(ctx, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", c.username, repoName, prNumber), nil)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CommentOnPR posts a comment on a PR (uses the issues endpoint since PRs are issues in Gitea).
func (c *Client) CommentOnPR(ctx context.Context, repoName string, prNumber int, body string) error {
	payload := map[string]any{"body": body}
	_, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments", c.username, repoName, prNumber), payload)
	return err
}

// AddLabel creates a label on the repo (if needed) and adds it to the given PR/issue.
func (c *Client) AddLabel(ctx context.Context, repoName string, prNumber int, labelName string) error {
	// Create the label on the repo.
	createPayload := map[string]any{
		"name":  labelName,
		"color": "#e11d48",
	}
	data, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/labels", c.username, repoName), createPayload)
	if err != nil && !strings.Contains(err.Error(), "409") {
		return fmt.Errorf("create label: %w", err)
	}

	// Extract label ID.
	var label struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &label); err != nil {
		return fmt.Errorf("parse label: %w", err)
	}

	// Add label to issue/PR.
	addPayload := map[string]any{
		"labels": []int64{label.ID},
	}
	_, err = c.do(ctx, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/labels", c.username, repoName, prNumber), addPayload)
	return err
}

// MergePR merges a pull request via the API.
// Method can be "merge", "rebase", or "squash".
func (c *Client) MergePR(ctx context.Context, repoName string, prNumber int, method string) error {
	payload := map[string]any{
		"Do": method,
	}
	_, err := c.do(ctx, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d/merge", c.username, repoName, prNumber), payload)
	return err
}

// DeleteBranch deletes a branch on the remote repository.
func (c *Client) DeleteBranch(ctx context.Context, repoName, branch string) error {
	_, err := c.do(ctx, "DELETE", fmt.Sprintf("/api/v1/repos/%s/%s/branches/%s", c.username, repoName, branch), nil)
	return err
}

// GetPRReviews fetches all reviews for a PR.
func (c *Client) GetPRReviews(ctx context.Context, repoName string, prNumber int) ([]map[string]any, error) {
	data, err := c.do(ctx, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d/reviews", c.username, repoName, prNumber), nil)
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
