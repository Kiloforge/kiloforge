package domain

import "time"

// BoardConfig stores the Gitea project board configuration for a project.
type BoardConfig struct {
	ProjectBoardID int            `json:"project_board_id"`
	Columns        map[string]int `json:"columns"`
	Labels         map[string]int `json:"labels"`
}

// TrackIssue maps a conductor track to a Gitea issue.
type TrackIssue struct {
	TrackID     string    `json:"track_id"`
	IssueNumber int       `json:"issue_number"`
	CardID      int       `json:"card_id"`
	Column      string    `json:"column"`
	LastSynced  time.Time `json:"last_synced"`
}
