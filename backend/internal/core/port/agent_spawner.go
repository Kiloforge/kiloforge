package port

import (
	"context"

	"kiloforge/internal/core/domain"
)

// ReviewerOpts configures reviewer agent spawning.
type ReviewerOpts struct {
	PRNumber int
	PRURL    string
	WorkDir  string
	LogDir   string
}

// AgentSpawner abstracts agent spawning and resume.
type AgentSpawner interface {
	SpawnReviewer(ctx context.Context, opts ReviewerOpts) (*domain.AgentInfo, error)
	ResumeDeveloper(ctx context.Context, sessionID, workDir string) error
}
