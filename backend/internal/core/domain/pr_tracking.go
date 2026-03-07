package domain

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
	Status           string `json:"status"`
}
