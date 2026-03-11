package port

import (
	"context"
)

// AgentSpawner abstracts agent spawning and resume.
type AgentSpawner interface {
	ResumeDeveloper(ctx context.Context, sessionID, workDir string) error
}
