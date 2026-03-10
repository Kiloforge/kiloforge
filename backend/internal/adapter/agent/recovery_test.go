package agent

import (
	"context"
	"fmt"
	"os"
	"testing"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/testutil"
)

// mockProcessStarter records start calls and returns configurable results.
type mockProcessStarter struct {
	calls   []mockStartCall
	err     error
	nextPID int
}

type mockStartCall struct {
	sessionID string
	workDir   string
}

func (m *mockProcessStarter) Start(_ context.Context, sessionID, workDir, _ string) (int, error) {
	m.calls = append(m.calls, mockStartCall{sessionID: sessionID, workDir: workDir})
	if m.err != nil {
		return 0, m.err
	}
	m.nextPID++
	return 1000 + m.nextPID, nil
}

func TestRecoverAll_NoSuspended(t *testing.T) {
	store := &testutil.MockAgentStore{}
	starter := &mockProcessStarter{}
	rm := NewRecoveryManager(store, starter)
	result := rm.RecoverAll(context.Background())

	if len(result.Resumed) != 0 || len(result.Failed) != 0 {
		t.Errorf("expected empty result, got resumed=%d failed=%d",
			len(result.Resumed), len(result.Failed))
	}
	if len(starter.calls) != 0 {
		t.Errorf("expected no start calls, got %d", len(starter.calls))
	}
}

func TestRecoverAll_ResumesSuccessfully(t *testing.T) {
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "agent-1", Status: "suspended", SessionID: "sess-1", WorktreeDir: os.TempDir(), Role: "developer"},
		},
	}
	starter := &mockProcessStarter{}
	rm := NewRecoveryManager(store, starter)
	result := rm.RecoverAll(context.Background())

	if len(result.Resumed) != 1 || result.Resumed[0] != "agent-1" {
		t.Errorf("expected agent-1 resumed, got %v", result.Resumed)
	}
	if len(starter.calls) != 1 || starter.calls[0].sessionID != "sess-1" {
		t.Errorf("expected start call with sess-1, got %v", starter.calls)
	}
	agent, _ := store.FindAgent("agent-1")
	if agent.Status != "running" {
		t.Errorf("expected running, got %s", agent.Status)
	}
}

func TestRecoverAll_MissingSessionID(t *testing.T) {
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "agent-1", Status: "suspended", SessionID: "", WorktreeDir: os.TempDir()},
		},
	}
	starter := &mockProcessStarter{}
	rm := NewRecoveryManager(store, starter)
	result := rm.RecoverAll(context.Background())

	if len(result.Failed) != 1 || result.Failed[0].Reason != "no session ID" {
		t.Errorf("expected failure with 'no session ID', got %v", result.Failed)
	}
	agent, _ := store.FindAgent("agent-1")
	if agent.Status != "resume-failed" {
		t.Errorf("expected resume-failed, got %s", agent.Status)
	}
}

func TestRecoverAll_MissingWorktree(t *testing.T) {
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "agent-1", Status: "suspended", SessionID: "sess-1", WorktreeDir: "/nonexistent/path/abc123"},
		},
	}
	starter := &mockProcessStarter{}
	rm := NewRecoveryManager(store, starter)
	result := rm.RecoverAll(context.Background())

	if len(result.Failed) != 1 || result.Failed[0].Reason != "worktree missing" {
		t.Errorf("expected failure with 'worktree missing', got %v", result.Failed)
	}
}

func TestRecoverAll_StartFails(t *testing.T) {
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "agent-1", Status: "suspended", SessionID: "sess-1", WorktreeDir: os.TempDir()},
		},
	}
	starter := &mockProcessStarter{err: fmt.Errorf("session expired")}
	rm := NewRecoveryManager(store, starter)
	result := rm.RecoverAll(context.Background())

	if len(result.Failed) != 1 {
		t.Fatalf("expected 1 failure, got %d", len(result.Failed))
	}
	if result.Failed[0].Reason != "resume failed: session expired" {
		t.Errorf("expected 'resume failed: session expired', got %q", result.Failed[0].Reason)
	}
}

func TestRecoverAll_StaleAgents(t *testing.T) {
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			// PID 999999 should not exist — stale running agent
			{ID: "stale-1", Status: "running", PID: 999999, SessionID: "sess-1", WorktreeDir: os.TempDir()},
		},
	}
	starter := &mockProcessStarter{}
	rm := NewRecoveryManager(store, starter)
	result := rm.RecoverAll(context.Background())

	// Stale agent should be detected, marked suspended, then resumed.
	if len(result.Resumed) != 1 {
		t.Errorf("expected 1 resumed (was stale), got %d", len(result.Resumed))
	}
}

func TestRecoverAll_DevelopersFirst(t *testing.T) {
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "rev-1", Status: "suspended", SessionID: "s1", WorktreeDir: os.TempDir(), Role: "reviewer"},
			{ID: "dev-1", Status: "suspended", SessionID: "s2", WorktreeDir: os.TempDir(), Role: "developer"},
		},
	}
	starter := &mockProcessStarter{}
	rm := NewRecoveryManager(store, starter)
	rm.RecoverAll(context.Background())

	if len(starter.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(starter.calls))
	}
	if starter.calls[0].sessionID != "s2" {
		t.Errorf("expected developer (s2) first, got %s", starter.calls[0].sessionID)
	}
	if starter.calls[1].sessionID != "s1" {
		t.Errorf("expected reviewer (s1) second, got %s", starter.calls[1].sessionID)
	}
}
