package service

import (
	"context"
	"fmt"
	"testing"
)

type stubGitRunner struct {
	outputs map[string]string // args key → output
	errors  map[string]error  // args key → error
}

func (r *stubGitRunner) RunGitCommand(_ context.Context, _ string, _ []string, args ...string) (string, error) {
	key := fmt.Sprintf("%v", args)
	if err, ok := r.errors[key]; ok {
		return "", err
	}
	if out, ok := r.outputs[key]; ok {
		return out, nil
	}
	return "", nil
}

func TestGitSyncService_CheckSyncStatus(t *testing.T) {
	t.Parallel()

	t.Run("ahead and behind", func(t *testing.T) {
		runner := &stubGitRunner{
			outputs: map[string]string{
				"[rev-list --left-right --count origin/main...main]": "3\t5\n",
			},
		}
		svc := NewGitSyncService(runner)
		status, err := svc.CheckSyncStatus(context.Background(), "/repo", nil, "main")
		if err != nil {
			t.Fatalf("CheckSyncStatus: %v", err)
		}
		if status.Ahead != 5 {
			t.Errorf("Ahead = %d, want 5", status.Ahead)
		}
		if status.Behind != 3 {
			t.Errorf("Behind = %d, want 3", status.Behind)
		}
	})

	t.Run("fetch failure", func(t *testing.T) {
		runner := &stubGitRunner{
			errors: map[string]error{
				"[fetch origin main]": fmt.Errorf("network error"),
			},
		}
		svc := NewGitSyncService(runner)
		_, err := svc.CheckSyncStatus(context.Background(), "/repo", nil, "main")
		if err == nil {
			t.Fatal("expected error on fetch failure")
		}
	})

	t.Run("zero ahead/behind", func(t *testing.T) {
		runner := &stubGitRunner{
			outputs: map[string]string{
				"[rev-list --left-right --count origin/main...main]": "0\t0\n",
			},
		}
		svc := NewGitSyncService(runner)
		status, err := svc.CheckSyncStatus(context.Background(), "/repo", nil, "main")
		if err != nil {
			t.Fatalf("CheckSyncStatus: %v", err)
		}
		if status.Ahead != 0 || status.Behind != 0 {
			t.Errorf("expected 0/0, got %d/%d", status.Ahead, status.Behind)
		}
	})
}

func TestGitSyncService_PushBranch(t *testing.T) {
	t.Parallel()

	t.Run("successful push", func(t *testing.T) {
		runner := &stubGitRunner{}
		svc := NewGitSyncService(runner)
		err := svc.PushBranch(context.Background(), "/repo", nil, "main")
		if err != nil {
			t.Fatalf("PushBranch: %v", err)
		}
	})

	t.Run("non-fast-forward error", func(t *testing.T) {
		runner := &stubGitRunner{
			errors: map[string]error{
				"[push origin main]": fmt.Errorf("non-fast-forward"),
			},
		}
		svc := NewGitSyncService(runner)
		err := svc.PushBranch(context.Background(), "/repo", nil, "main")
		if err == nil {
			t.Fatal("expected error")
		}
		if !containsStr(err.Error(), "diverged") {
			t.Errorf("error = %q, want diverged message", err.Error())
		}
	})
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
