package orchestration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const prTrackingFile = "pr-tracking.json"

// PRTracking links a PR to its developer and reviewer agents.
type PRTracking struct {
	PRNumber         int    `json:"pr_number"`
	TrackID          string `json:"track_id"`
	ProjectSlug      string `json:"project_slug"`
	DeveloperAgentID string `json:"developer_agent_id"`
	DeveloperSession string `json:"developer_session"`
	DeveloperWorkDir string `json:"developer_work_dir,omitempty"`
	ReviewerAgentID  string `json:"reviewer_agent_id,omitempty"`
	ReviewerSession  string `json:"reviewer_session,omitempty"`
	ReviewCycleCount int    `json:"review_cycle_count"`
	MaxReviewCycles  int    `json:"max_review_cycles"`
	Status           string `json:"status"` // "waiting-review", "in-review", "changes-requested", "approved", "escalated", "merged"
}

// PRTrackingPath returns the path for a project's PR tracking file.
func PRTrackingPath(dataDir, slug string) string {
	return filepath.Join(dataDir, "projects", slug, prTrackingFile)
}

// Save writes the tracking record to the given directory.
func (t *PRTracking) Save(dir string) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal pr tracking: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, prTrackingFile), append(data, '\n'), 0o644)
}

// LoadPRTracking reads a tracking record from the given directory.
func LoadPRTracking(dir string) (*PRTracking, error) {
	data, err := os.ReadFile(filepath.Join(dir, prTrackingFile))
	if err != nil {
		return nil, fmt.Errorf("read pr tracking: %w", err)
	}
	var t PRTracking
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse pr tracking: %w", err)
	}
	return &t, nil
}
