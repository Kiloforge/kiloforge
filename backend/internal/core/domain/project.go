package domain

import "time"

// ProjectStatus represents the state of a registered project.
type ProjectStatus string

const (
	ProjectActive   ProjectStatus = "active"
	ProjectInactive ProjectStatus = "inactive"
)

// Project represents a registered project in the crelay system.
type Project struct {
	Slug         string    `json:"slug"`
	RepoName     string    `json:"repo_name"`
	ProjectDir   string    `json:"project_dir"`
	OriginRemote string    `json:"origin_remote,omitempty"`
	RegisteredAt time.Time `json:"registered_at"`
	Active       bool      `json:"active"`
}
