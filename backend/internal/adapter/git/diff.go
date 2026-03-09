package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"kiloforge/internal/core/domain"
)

const diffTimeout = 30 * time.Second

// Diff returns a structured diff between the given branch and main.
func (gs *GitSync) Diff(ctx context.Context, projectDir, branch string) (*domain.DiffResult, error) {
	ctx, cancel := context.WithTimeout(ctx, diffTimeout)
	defer cancel()

	// Check branch exists.
	checkCmd := gs.gitCmd(ctx, projectDir, "", "rev-parse", "--verify", branch)
	if out, err := checkCmd.CombinedOutput(); err != nil {
		return nil, &BranchNotFoundError{Branch: branch, Detail: strings.TrimSpace(string(out))}
	}

	// Get unified diff.
	diffCmd := gs.gitCmd(ctx, projectDir, "", "diff", "main..."+branch, "-U3", "--no-color")
	diffOut, err := diffCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git diff failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git diff: %w", err)
	}

	files, err := ParseUnifiedDiff(string(diffOut))
	if err != nil {
		return nil, fmt.Errorf("parse diff: %w", err)
	}

	stats := domain.DiffStats{FilesChanged: len(files)}
	for _, f := range files {
		stats.Insertions += f.Insertions
		stats.Deletions += f.Deletions
	}

	return &domain.DiffResult{
		Branch: branch,
		Base:   "main",
		Stats:  stats,
		Files:  files,
	}, nil
}

// DiffWithMaxFiles returns a diff limited to maxFiles. If the actual file count
// exceeds maxFiles, the result is truncated and Truncated is set to true.
func (gs *GitSync) DiffWithMaxFiles(ctx context.Context, projectDir, branch string, maxFiles int) (*domain.DiffResult, error) {
	result, err := gs.Diff(ctx, projectDir, branch)
	if err != nil {
		return nil, err
	}

	if maxFiles > 0 && len(result.Files) > maxFiles {
		result.Files = result.Files[:maxFiles]
		result.Truncated = true
	}

	return result, nil
}

// BranchNotFoundError indicates that the requested branch does not exist.
type BranchNotFoundError struct {
	Branch string
	Detail string
}

func (e *BranchNotFoundError) Error() string {
	return fmt.Sprintf("branch %q not found: %s", e.Branch, e.Detail)
}
