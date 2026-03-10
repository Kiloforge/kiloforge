package domain

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ProjectStatus represents the state of a registered project.
type ProjectStatus string

const (
	ProjectActive   ProjectStatus = "active"
	ProjectInactive ProjectStatus = "inactive"
)

// Project represents a registered project in the kiloforge system.
type Project struct {
	Slug         string    `json:"slug"`
	RepoName     string    `json:"repo_name"`
	ProjectDir   string    `json:"project_dir"`
	MirrorDir    string    `json:"mirror_dir,omitempty"`
	OriginRemote string    `json:"origin_remote,omitempty"`
	SSHKeyPath   string    `json:"ssh_key_path,omitempty"`
	RegisteredAt time.Time `json:"registered_at"`
	Active       bool      `json:"active"`
}

// AddProjectOpts contains optional parameters for adding a project.
type AddProjectOpts struct {
	SSHKeyPath string // Path to SSH private key for cloning.
}

// AddProjectResult contains details about a newly added project.
type AddProjectResult struct {
	Project   Project
	EmptyRepo bool // true if the repo had no commits (push was skipped)
}

// GitSSHEnv returns environment variables for git commands to use the
// project's configured SSH key. Returns nil if no SSH key is configured.
func (p Project) GitSSHEnv() []string {
	if p.SSHKeyPath == "" {
		return nil
	}
	path := p.SSHKeyPath
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = home + path[1:]
		}
	}
	return []string{
		fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o IdentitiesOnly=yes", path),
	}
}
