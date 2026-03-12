package agent

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"kiloforge/internal/adapter/config"

	"github.com/schlunsen/claude-agent-sdk-go/types"
)

// mockSessionFactory creates SDKSessions backed by mockSDKClients.
// Each call to NewSession/NewResumeSession pops from a FIFO queue of
// pre-configured mocks that the test controls via response channels.
type mockSessionFactory struct {
	sessions []*mockSDKClient

	// lastResumeSessionID records the sessionID passed to NewResumeSession
	// so the test can verify the correct session was resumed.
	lastResumeSessionID string
}

func (f *mockSessionFactory) pop() *mockSDKClient {
	if len(f.sessions) == 0 {
		panic("mockSessionFactory: no sessions queued")
	}
	m := f.sessions[0]
	f.sessions = f.sessions[1:]
	return m
}

func (f *mockSessionFactory) NewSession(_ context.Context, _, _, _ string, _ map[string]string) (*SDKSession, error) {
	mock := f.pop()
	return newTestSDKSessionWithMock(mock), nil
}

func (f *mockSessionFactory) NewResumeSession(_ context.Context, _, _, _ string, sessionID string, _ map[string]string) (*SDKSession, error) {
	f.lastResumeSessionID = sessionID
	mock := f.pop()
	return newTestSDKSessionWithMock(mock), nil
}

// noopAuth is an auth checker that always succeeds (bypasses Claude CLI auth).
func noopAuth(_ context.Context) error { return nil }

// TestAgentLifecycle_SpawnStopResumeAttach exercises the full interactive agent
// lifecycle through the real Spawner methods:
//
//  1. SpawnInteractive with prompt "hello" → receive response
//  2. StopAgent → verify stopped
//  3. ResumeAgent → verify resumed with correct session ID
//  4. Send another message via the resumed agent → receive response
//  5. StopAgent again → verify final state
//
// All SDK calls are mocked via mockSessionFactory — no real server or Claude CLI needed.
func TestAgentLifecycle_SpawnStopResumeAttach(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// --- Setup ---

	// Create a temp git repo for workDir (SpawnInteractive checks for .git).
	workDir := t.TempDir()
	exec.Command("git", "init", workDir).Run()

	store := &stubAgentStore{}
	cfg := &config.Config{
		DataDir:      t.TempDir(),
		MaxSwarmSize: 5,
	}

	// Mock A: for the initial SpawnInteractive. Will respond to "hello".
	mockA := &mockSDKClient{
		connected:  true,
		responseCh: make(chan types.Message, 10),
	}
	// Mock B: for ResumeAgent. Will respond to "what is 2+2?".
	mockB := &mockSDKClient{
		connected:  true,
		responseCh: make(chan types.Message, 10),
	}

	factory := &mockSessionFactory{
		sessions: []*mockSDKClient{mockA, mockB},
	}

	spawner := NewSpawner(cfg, store, nil)
	spawner.SetSessionFactory(factory)
	spawner.SetAuthChecker(noopAuth)

	ctx := context.Background()

	// =========================================================================
	// Step 1: SpawnInteractive with prompt "hello"
	// =========================================================================
	ia, err := spawner.SpawnInteractive(ctx, SpawnInteractiveOpts{
		WorkDir: workDir,
		Model:   "sonnet",
		Prompt:  "hello",
	})
	if err != nil {
		t.Fatalf("SpawnInteractive: %v", err)
	}

	agentID := ia.Info.ID
	t.Logf("spawned agent %s (name=%s)", agentID, ia.Info.Name)

	// Verify agent is in the active map.
	if _, ok := spawner.GetActiveAgent(agentID); !ok {
		t.Fatal("agent should be active after spawn")
	}

	// Simulate Claude responding to "hello".
	costUSD := 0.001
	mockA.responseCh <- &types.AssistantMessage{
		Content: []types.ContentBlock{
			&types.TextBlock{Text: "Hello! How can I help you?"},
		},
	}
	mockA.responseCh <- &types.ResultMessage{
		SessionID:    "real-session-abc",
		TotalCostUSD: &costUSD,
	}
	close(mockA.responseCh)

	// Wait for relay to process messages.
	waitForOutput(t, ia.Output, 5*time.Second)

	// Verify the session ID callback persisted the real session ID.
	agent, err := store.FindAgent(agentID)
	if err != nil {
		t.Fatalf("find agent: %v", err)
	}
	if agent.SessionID != "real-session-abc" {
		t.Errorf("session ID after spawn = %q, want %q", agent.SessionID, "real-session-abc")
	}

	// =========================================================================
	// Step 2: StopAgent
	// =========================================================================
	if err := spawner.StopAgent(agentID); err != nil {
		t.Fatalf("StopAgent: %v", err)
	}

	// Verify removed from active map.
	if _, ok := spawner.GetActiveAgent(agentID); ok {
		t.Error("agent should not be active after stop")
	}

	// Verify store status is "stopped".
	agent, err = store.FindAgent(agentID)
	if err != nil {
		t.Fatalf("find agent after stop: %v", err)
	}
	if agent.Status != "stopped" {
		t.Errorf("status after stop = %q, want %q", agent.Status, "stopped")
	}
	if agent.ShutdownReason != "user_stopped" {
		t.Errorf("shutdown reason = %q, want %q", agent.ShutdownReason, "user_stopped")
	}

	// =========================================================================
	// Step 3: ResumeAgent (attach to the same session)
	// =========================================================================
	iaResumed, err := spawner.ResumeAgent(ctx, agentID)
	if err != nil {
		t.Fatalf("ResumeAgent: %v", err)
	}

	// Verify the factory received the correct session ID for resume.
	if factory.lastResumeSessionID != "real-session-abc" {
		t.Errorf("resume used session ID %q, want %q", factory.lastResumeSessionID, "real-session-abc")
	}

	// Verify agent is active again.
	if _, ok := spawner.GetActiveAgent(agentID); !ok {
		t.Error("agent should be active after resume")
	}
	if iaResumed.Info.Status != "running" {
		t.Errorf("status after resume = %q, want %q", iaResumed.Info.Status, "running")
	}

	// =========================================================================
	// Step 4: Send another message via the resumed agent
	// =========================================================================
	if err := iaResumed.Stdin("what is 2+2?"); err != nil {
		t.Fatalf("second query via Stdin: %v", err)
	}

	// Simulate response to the second message.
	costUSD2 := 0.002
	mockB.responseCh <- &types.AssistantMessage{
		Content: []types.ContentBlock{
			&types.TextBlock{Text: "2+2 equals 4."},
		},
	}
	mockB.responseCh <- &types.ResultMessage{
		SessionID:    "real-session-abc",
		TotalCostUSD: &costUSD2,
	}
	close(mockB.responseCh)

	// Wait for relay to process.
	waitForOutput(t, iaResumed.Output, 5*time.Second)

	// Verify agent is still active after the second exchange.
	if _, ok := spawner.GetActiveAgent(agentID); !ok {
		t.Error("agent should still be active after second query")
	}

	// =========================================================================
	// Step 5: Final stop
	// =========================================================================
	if err := spawner.StopAgent(agentID); err != nil {
		t.Fatalf("final StopAgent: %v", err)
	}

	agent, _ = store.FindAgent(agentID)
	if agent.Status != "stopped" {
		t.Errorf("final status = %q, want %q", agent.Status, "stopped")
	}

	t.Log("lifecycle complete: spawn → response → stop → resume → response → stop")
}

// waitForOutput drains the output channel until it blocks or the deadline fires.
// This ensures relayResponse has finished processing before we make assertions.
func waitForOutput(t *testing.T, ch <-chan []byte, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return // channel closed
			}
		case <-deadline:
			return
		case <-time.After(200 * time.Millisecond):
			return // no more messages for 200ms — relay is likely done
		}
	}
}

// TestMockSessionFactory_NewSession verifies the factory wiring.
func TestMockSessionFactory_NewSession(t *testing.T) {
	t.Parallel()

	mock := &mockSDKClient{connected: true, responseCh: make(chan types.Message)}
	factory := &mockSessionFactory{sessions: []*mockSDKClient{mock}}

	session, err := factory.NewSession(context.Background(), "", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session == nil {
		t.Fatal("session should not be nil")
	}
}

// TestMockSessionFactory_NewResumeSession verifies resume wiring and session ID capture.
func TestMockSessionFactory_NewResumeSession(t *testing.T) {
	t.Parallel()

	mock := &mockSDKClient{connected: true, responseCh: make(chan types.Message)}
	factory := &mockSessionFactory{sessions: []*mockSDKClient{mock}}

	session, err := factory.NewResumeSession(context.Background(), "", "", "", "sess-123", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session == nil {
		t.Fatal("session should not be nil")
	}
	if factory.lastResumeSessionID != "sess-123" {
		t.Errorf("lastResumeSessionID = %q, want %q", factory.lastResumeSessionID, "sess-123")
	}
}

// TestSetSessionFactory verifies the setter.
func TestSetSessionFactory(t *testing.T) {
	t.Parallel()

	s := NewSpawner(&config.Config{}, nil, nil)

	// Default should be RealSessionFactory.
	if _, ok := s.sessionFactory.(RealSessionFactory); !ok {
		t.Error("default factory should be RealSessionFactory")
	}

	// Setting nil should be a no-op.
	s.SetSessionFactory(nil)
	if _, ok := s.sessionFactory.(RealSessionFactory); !ok {
		t.Error("nil SetSessionFactory should not change the factory")
	}

	// Setting a mock should work.
	mock := &mockSessionFactory{}
	s.SetSessionFactory(mock)
	if s.sessionFactory != mock {
		t.Error("SetSessionFactory should set the factory")
	}
}

// TestRealSessionFactory_ImplementsInterface is a compile-time check.
func TestRealSessionFactory_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ SessionFactory = RealSessionFactory{}
	var _ SessionFactory = (*mockSessionFactory)(nil)
}
