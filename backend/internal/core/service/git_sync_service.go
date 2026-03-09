package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// GitCommandRunner abstracts git command execution for sync operations.
type GitCommandRunner interface {
	RunGitCommand(ctx context.Context, dir string, sshEnv []string, args ...string) (string, error)
}

// GitSyncService provides git sync status checking.
type GitSyncService struct {
	git GitCommandRunner
}

// NewGitSyncService creates a new GitSyncService.
func NewGitSyncService(git GitCommandRunner) *GitSyncService {
	return &GitSyncService{git: git}
}

// SyncStatus describes the ahead/behind relationship with a remote branch.
type SyncStatus struct {
	Ahead  int
	Behind int
}

// CheckSyncStatus fetches the remote and returns ahead/behind counts.
func (s *GitSyncService) CheckSyncStatus(ctx context.Context, dir string, sshEnv []string, branch string) (*SyncStatus, error) {
	// Fetch origin.
	if _, err := s.git.RunGitCommand(ctx, dir, sshEnv, "fetch", "origin", branch); err != nil {
		return nil, fmt.Errorf("fetch origin: %w", err)
	}

	// Check ahead/behind.
	out, err := s.git.RunGitCommand(ctx, dir, sshEnv,
		"rev-list", "--left-right", "--count",
		fmt.Sprintf("origin/%s...%s", branch, branch))
	if err != nil {
		return nil, fmt.Errorf("rev-list: %w", err)
	}

	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) != 2 {
		return &SyncStatus{}, nil
	}

	behind, _ := strconv.Atoi(parts[0])
	ahead, _ := strconv.Atoi(parts[1])
	return &SyncStatus{Ahead: ahead, Behind: behind}, nil
}

// PushBranch pushes a branch to origin.
func (s *GitSyncService) PushBranch(ctx context.Context, dir string, sshEnv []string, branch string) error {
	_, err := s.git.RunGitCommand(ctx, dir, sshEnv, "push", "origin", branch)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "non-fast-forward") {
			return fmt.Errorf("origin has diverged — pull and resolve conflicts first")
		}
		return fmt.Errorf("push failed: %w", err)
	}
	return nil
}
