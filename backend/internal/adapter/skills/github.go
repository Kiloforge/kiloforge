package skills

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Release represents a GitHub release.
type Release struct {
	TagName     string `json:"tag_name"`
	TarballURL  string `json:"tarball_url"`
	PublishedAt string `json:"published_at"`
}

// GitHubClient fetches release info from GitHub.
type GitHubClient struct {
	httpClient *http.Client
}

// NewGitHubClient creates a client with a 10s timeout.
func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewGitHubClientWith creates a client with a custom http.Client (for testing).
func NewGitHubClientWith(c *http.Client) *GitHubClient {
	return &GitHubClient{httpClient: c}
}

// LatestRelease fetches the latest release for the given repo (e.g., "owner/repo").
func (g *GitHubClient) LatestRelease(repo string) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	return fetchRelease(g.httpClient, url)
}

func fetchRelease(client *http.Client, url string) (*Release, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found")
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("rate limited by GitHub API")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from GitHub API", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &rel, nil
}
