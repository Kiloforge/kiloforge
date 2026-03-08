package agent

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// stubAgentStore is a minimal port.AgentStore for tests that don't need persistence.
type stubAgentStore struct {
	agents []domain.AgentInfo
}

var _ port.AgentStore = (*stubAgentStore)(nil)

func (s *stubAgentStore) Load() error                                  { return nil }
func (s *stubAgentStore) Save() error                                  { return nil }
func (s *stubAgentStore) AddAgent(info domain.AgentInfo)               { s.agents = append(s.agents, info) }
func (s *stubAgentStore) FindAgent(id string) (*domain.AgentInfo, error) {
	for i := range s.agents {
		if s.agents[i].ID == id {
			return &s.agents[i], nil
		}
	}
	return nil, fmt.Errorf("not found: %s", id)
}
func (s *stubAgentStore) FindByRef(ref string) *domain.AgentInfo { return nil }
func (s *stubAgentStore) UpdateStatus(id, status string) {
	for i := range s.agents {
		if s.agents[i].ID == id {
			s.agents[i].Status = status
		}
	}
}
func (s *stubAgentStore) HaltAgent(string) error                            { return nil }
func (s *stubAgentStore) Agents() []domain.AgentInfo                        { return s.agents }
func (s *stubAgentStore) AgentsByStatus(...string) []domain.AgentInfo       { return nil }

func TestCheckQuota_NilTracker(t *testing.T) {
	t.Parallel()

	s := &Spawner{cfg: &config.Config{}}
	if err := s.checkQuota(); err != nil {
		t.Errorf("nil tracker should not error, got: %v", err)
	}
}

func TestCheckQuota_NotRateLimited(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	s := &Spawner{cfg: &config.Config{}, tracker: tracker}
	if err := s.checkQuota(); err != nil {
		t.Errorf("should not error when not rate limited, got: %v", err)
	}
}

func TestCheckQuota_RateLimited(t *testing.T) {
	t.Parallel()

	tracker := NewQuotaTracker("")
	tracker.mu.Lock()
	tracker.rateLimitUntil = time.Now().Add(5 * time.Minute)
	tracker.mu.Unlock()

	s := &Spawner{cfg: &config.Config{}, tracker: tracker}
	err := s.checkQuota()
	if err == nil {
		t.Fatal("expected error when rate limited")
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestCheckQuota_BudgetIgnored(t *testing.T) {
	t.Parallel()

	// MaxSessionCostUSD is deprecated — budget should no longer block spawns.
	tracker := NewQuotaTracker("")
	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		CostUSD: 10.0,
		Usage:   &UsageData{InputTokens: 100000},
	})

	s := &Spawner{
		cfg:     &config.Config{MaxSessionCostUSD: 5.0},
		tracker: tracker,
	}

	if err := s.checkQuota(); err != nil {
		t.Errorf("budget should be ignored (deprecated), got: %v", err)
	}
}

func TestSetTracer(t *testing.T) {
	t.Parallel()

	s := NewSpawner(&config.Config{}, nil, nil)
	// Default tracer should be NoopTracer.
	if s.tracer == nil {
		t.Fatal("expected non-nil default tracer")
	}

	// SetTracer with nil should not replace the default.
	s.SetTracer(nil)
	if s.tracer == nil {
		t.Fatal("SetTracer(nil) should not set nil")
	}
}

func TestSetCompletionCallback(t *testing.T) {
	t.Parallel()

	s := NewSpawner(&config.Config{}, nil, nil)

	var called bool
	var gotID, gotRef, gotStatus string
	s.SetCompletionCallback(func(agentID, ref, status string) {
		called = true
		gotID = agentID
		gotRef = ref
		gotStatus = status
	})

	// Invoke callback directly to test it's wired.
	s.onCompletion("agent-123", "track-abc", "completed")

	if !called {
		t.Fatal("completion callback was not called")
	}
	if gotID != "agent-123" {
		t.Errorf("agentID = %q, want %q", gotID, "agent-123")
	}
	if gotRef != "track-abc" {
		t.Errorf("ref = %q, want %q", gotRef, "track-abc")
	}
	if gotStatus != "completed" {
		t.Errorf("status = %q, want %q", gotStatus, "completed")
	}
}

func TestOnCompletion_NilCallback(t *testing.T) {
	t.Parallel()

	s := NewSpawner(&config.Config{}, nil, nil)
	// Should not panic when no callback is set.
	s.onCompletion("agent-123", "track-abc", "completed")
}

func TestCheckQuota_HighCostAllowed(t *testing.T) {
	t.Parallel()

	// Budget enforcement is deprecated — high cost should not block spawns.
	tracker := NewQuotaTracker("")
	tracker.RecordEvent("agent-1", StreamEvent{
		Type:    "result",
		CostUSD: 100.0,
		Usage:   &UsageData{InputTokens: 1000000},
	})

	s := &Spawner{
		cfg:     &config.Config{},
		tracker: tracker,
	}

	if err := s.checkQuota(); err != nil {
		t.Errorf("should always allow spawn (budget deprecated), got: %v", err)
	}
}

func TestMonitorInteractive_ExtractsText(t *testing.T) {
	t.Parallel()

	// Use printf to emit stream-json lines that contain extractable text.
	streamJSON := `{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello from agent"}}
{"type":"result","subtype":"success","total_cost_usd":0.01,"usage":{"input_tokens":100,"output_tokens":50}}
`
	cmd := exec.Command("printf", "%s", streamJSON)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	lf, err := os.CreateTemp(t.TempDir(), "monitor-*.log")
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	output := make(chan []byte, 10)
	done := make(chan struct{})

	s := &Spawner{
		cfg:   &config.Config{},
		store: nil,
		tracer: port.NoopTracer{},
	}

	// monitorInteractive calls s.store.UpdateStatus — need a real store.
	s.cfg.DataDir = t.TempDir()
	s.store = &stubAgentStore{}

	_, span := port.NoopTracer{}.StartSpan(nil, "test")

	go s.monitorInteractive("test-agent", stdout, lf, cmd, span, output, done)

	// Should receive the extracted text.
	select {
	case msg := <-output:
		if string(msg) != "Hello from agent" {
			t.Errorf("output = %q, want %q", msg, "Hello from agent")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for output")
	}

	// Wait for done signal.
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for done")
	}
}

func TestInteractiveAgent_DoneClosedOnExit(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("true")
	stdout, _ := cmd.StdoutPipe()

	lf, _ := os.CreateTemp(t.TempDir(), "monitor-*.log")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	output := make(chan []byte, 10)
	done := make(chan struct{})

	s := &Spawner{
		cfg:    &config.Config{DataDir: t.TempDir()},
		store:  &stubAgentStore{},
		tracer: port.NoopTracer{},
	}

	_, span := port.NoopTracer{}.StartSpan(nil, "test")

	go s.monitorInteractive("test-agent", stdout, lf, cmd, span, output, done)

	select {
	case <-done:
		// Success — done channel closed when process exits.
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for done")
	}
}
