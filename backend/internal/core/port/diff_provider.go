package port

import (
	"context"

	"kiloforge/internal/core/domain"
)

// DiffProvider abstracts git diff operations for testability.
type DiffProvider interface {
	Diff(ctx context.Context, projectDir, branch string) (*domain.DiffResult, error)
	DiffWithMaxFiles(ctx context.Context, projectDir, branch string, maxFiles int) (*domain.DiffResult, error)
}
