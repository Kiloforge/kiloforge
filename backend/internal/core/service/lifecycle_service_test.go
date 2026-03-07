package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"crelay/internal/core/domain"
	"crelay/internal/core/testutil"
)

func newTestLifecycleService(agents *testutil.MockAgentStore, spawner *testutil.MockAgentSpawner, pool *testutil.MockPoolReturner) *LifecycleService {
	logger := &testutil.MockLogger{}
	return NewLifecycleService(agents, spawner, pool, logger)
}

func TestIsBackwardMove(t *testing.T) {
	t.Parallel()
	tests := []struct {
		from, to string
		want     bool
	}{
		{"in_progress", "approved", true},
		{"in_progress", "suggested", true},
		{"in_review", "in_progress", true},
		{"in_review", "approved", true},
		{"approved", "in_progress", false},
		{"suggested", "approved", false},
		{"in_progress", "in_progress", false},
		{"unknown", "approved", false},
	}
	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			t.Parallel()
			if got := IsBackwardMove(tt.from, tt.to); got != tt.want {
				t.Errorf("IsBackwardMove(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestIsForwardMove(t *testing.T) {
	t.Parallel()
	tests := []struct {
		from, to string
		want     bool
	}{
		{"approved", "in_progress", true},
		{"suggested", "in_progress", true},
		{"in_progress", "approved", false},
		{"in_progress", "in_progress", false},
	}
	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			t.Parallel()
			if got := IsForwardMove(tt.from, tt.to); got != tt.want {
				t.Errorf("IsForwardMove(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestHandleBackwardMove_HaltsDeveloper(t *testing.T) {
	t.Parallel()
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "running", StartedAt: time.Now()},
		},
	}
	svc := newTestLifecycleService(store, &testutil.MockAgentSpawner{}, nil)
	svc.HandleBackwardMove(context.Background(), "track-1", "in_progress", "approved", nil)
	agent, _ := store.FindAgent("dev-1")
	if agent.Status != "halted" {
		t.Errorf("status = %q, want halted", agent.Status)
	}
}

func TestHandleBackwardMove_HaltsBothAgents(t *testing.T) {
	t.Parallel()
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "running", StartedAt: time.Now()},
			{ID: "rev-1", Ref: "PR #5", Status: "running", StartedAt: time.Now()},
		},
	}
	prTracking := &domain.PRTracking{TrackID: "track-1", ReviewerAgentID: "rev-1"}
	svc := newTestLifecycleService(store, &testutil.MockAgentSpawner{}, nil)
	svc.HandleBackwardMove(context.Background(), "track-1", "in_review", "approved", prTracking)
	dev, _ := store.FindAgent("dev-1")
	if dev.Status != "halted" {
		t.Errorf("developer status = %q, want halted", dev.Status)
	}
	rev, _ := store.FindAgent("rev-1")
	if rev.Status != "halted" {
		t.Errorf("reviewer status = %q, want halted", rev.Status)
	}
}

func TestHandleBackwardMove_AlreadyHalted(t *testing.T) {
	t.Parallel()
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "halted", StartedAt: time.Now()},
		},
	}
	svc := newTestLifecycleService(store, &testutil.MockAgentSpawner{}, nil)
	svc.HandleBackwardMove(context.Background(), "track-1", "in_progress", "approved", nil)
	agent, _ := store.FindAgent("dev-1")
	if agent.Status != "halted" {
		t.Errorf("status = %q, want halted (unchanged)", agent.Status)
	}
}

func TestHandleBackwardMove_NoAgent(t *testing.T) {
	t.Parallel()
	store := &testutil.MockAgentStore{}
	svc := newTestLifecycleService(store, &testutil.MockAgentSpawner{}, nil)
	svc.HandleBackwardMove(context.Background(), "track-nonexistent", "in_progress", "approved", nil)
}

func TestHandleRepromotion_ResumesDeveloper(t *testing.T) {
	t.Parallel()
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "halted", SessionID: "sess-1", WorktreeDir: t.TempDir(), StartedAt: time.Now()},
		},
	}
	spawner := &testutil.MockAgentSpawner{}
	svc := newTestLifecycleService(store, spawner, nil)
	resumed, reason := svc.HandleRepromotion(context.Background(), "track-1", "in_progress", nil)
	if !resumed {
		t.Errorf("expected resumed=true, got false (reason: %s)", reason)
	}
	agent, _ := store.FindAgent("dev-1")
	if agent.Status != "running" {
		t.Errorf("status = %q, want running", agent.Status)
	}
	if len(spawner.ResumeCalls) != 1 {
		t.Fatalf("expected 1 resume call, got %d", len(spawner.ResumeCalls))
	}
}

func TestHandleRepromotion_MissingSession(t *testing.T) {
	t.Parallel()
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "halted", SessionID: "", StartedAt: time.Now()},
		},
	}
	svc := newTestLifecycleService(store, &testutil.MockAgentSpawner{}, nil)
	resumed, reason := svc.HandleRepromotion(context.Background(), "track-1", "in_progress", nil)
	if resumed {
		t.Error("expected resumed=false")
	}
	if reason != "no session to resume" {
		t.Errorf("reason = %q, want 'no session to resume'", reason)
	}
	agent, _ := store.FindAgent("dev-1")
	if agent.Status != "resume-failed" {
		t.Errorf("status = %q, want resume-failed", agent.Status)
	}
}

func TestHandleRepromotion_MissingWorktree(t *testing.T) {
	t.Parallel()
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "halted", SessionID: "sess-1", WorktreeDir: "/nonexistent/path", StartedAt: time.Now()},
		},
	}
	svc := newTestLifecycleService(store, &testutil.MockAgentSpawner{}, nil)
	resumed, reason := svc.HandleRepromotion(context.Background(), "track-1", "in_progress", nil)
	if resumed {
		t.Error("expected resumed=false")
	}
	if reason != "worktree not found" {
		t.Errorf("reason = %q, want 'worktree not found'", reason)
	}
}

func TestHandleRepromotion_ResumeError(t *testing.T) {
	t.Parallel()
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "halted", SessionID: "sess-1", WorktreeDir: t.TempDir(), StartedAt: time.Now()},
		},
	}
	spawner := &testutil.MockAgentSpawner{ResumeErr: errors.New("spawn error")}
	svc := newTestLifecycleService(store, spawner, nil)
	resumed, _ := svc.HandleRepromotion(context.Background(), "track-1", "in_progress", nil)
	if resumed {
		t.Error("expected resumed=false")
	}
	agent, _ := store.FindAgent("dev-1")
	if agent.Status != "resume-failed" {
		t.Errorf("status = %q, want resume-failed", agent.Status)
	}
}

func TestHandleRepromotion_NotHalted(t *testing.T) {
	t.Parallel()
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "running", SessionID: "sess-1", StartedAt: time.Now()},
		},
	}
	svc := newTestLifecycleService(store, &testutil.MockAgentSpawner{}, nil)
	resumed, _ := svc.HandleRepromotion(context.Background(), "track-1", "in_progress", nil)
	if resumed {
		t.Error("expected resumed=false for non-halted agent")
	}
}

func TestHandleRejection_StopsAgentAndReturnsWorktree(t *testing.T) {
	t.Parallel()
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "running", StartedAt: time.Now()},
		},
	}
	pool := &testutil.MockPoolReturner{}
	svc := newTestLifecycleService(store, &testutil.MockAgentSpawner{}, pool)
	svc.HandleRejection(context.Background(), "track-1", nil)
	agent, _ := store.FindAgent("dev-1")
	if agent.Status != "stopped" {
		t.Errorf("status = %q, want stopped", agent.Status)
	}
	if len(pool.Calls) != 1 || pool.Calls[0] != "track-1" {
		t.Errorf("pool calls = %v, want [track-1]", pool.Calls)
	}
}

func TestHandleRejection_NoAgent(t *testing.T) {
	t.Parallel()
	store := &testutil.MockAgentStore{}
	pool := &testutil.MockPoolReturner{}
	svc := newTestLifecycleService(store, &testutil.MockAgentSpawner{}, pool)
	svc.HandleRejection(context.Background(), "track-nonexistent", nil)
}

func TestHandleRejection_AlreadyCompleted(t *testing.T) {
	t.Parallel()
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "completed", StartedAt: time.Now()},
		},
	}
	pool := &testutil.MockPoolReturner{}
	svc := newTestLifecycleService(store, &testutil.MockAgentSpawner{}, pool)
	svc.HandleRejection(context.Background(), "track-1", nil)
	agent, _ := store.FindAgent("dev-1")
	if agent.Status != "completed" {
		t.Errorf("status should remain completed, got %q", agent.Status)
	}
}
