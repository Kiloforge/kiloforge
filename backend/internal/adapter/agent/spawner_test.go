package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/skills"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"

	"github.com/schlunsen/claude-agent-sdk-go/types"
)

// stubAgentStore is a minimal port.AgentStore for tests that don't need persistence.
type stubAgentStore struct {
	agents []domain.AgentInfo
}

var _ port.AgentStore = (*stubAgentStore)(nil)

func (s *stubAgentStore) Load() error { return nil }
func (s *stubAgentStore) Save() error { return nil }
func (s *stubAgentStore) AddAgent(info domain.AgentInfo) error {
	s.agents = append(s.agents, info)
	return nil
}
func (s *stubAgentStore) FindAgent(id string) (*domain.AgentInfo, error) {
	for i := range s.agents {
		if s.agents[i].ID == id {
			return &s.agents[i], nil
		}
	}
	return nil, fmt.Errorf("not found: %s", id)
}
func (s *stubAgentStore) FindByRef(ref string) *domain.AgentInfo { return nil }
func (s *stubAgentStore) UpdateStatus(id, status string) error {
	for i := range s.agents {
		if s.agents[i].ID == id {
			s.agents[i].Status = status
		}
	}
	return nil
}
func (s *stubAgentStore) HaltAgent(string) error { return nil }
func (s *stubAgentStore) RemoveAgent(id string) error {
	for i := range s.agents {
		if s.agents[i].ID == id {
			s.agents = append(s.agents[:i], s.agents[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("not found: %s", id)
}
func (s *stubAgentStore) Agents() []domain.AgentInfo                  { return s.agents }
func (s *stubAgentStore) AgentsByStatus(...string) []domain.AgentInfo { return nil }
func (s *stubAgentStore) ListAgents(_ domain.PageOpts, _ ...string) (domain.Page[domain.AgentInfo], error) {
	return domain.Page[domain.AgentInfo]{Items: s.agents, TotalCount: len(s.agents)}, nil
}

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
	if s.tracer == nil {
		t.Fatal("expected non-nil default tracer")
	}

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
	s.onCompletion("agent-123", "track-abc", "completed")
}

func TestCheckQuota_HighCostAllowed(t *testing.T) {
	t.Parallel()

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

func TestValidateSkills_Developer(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	cfg := &config.Config{SkillsDir: globalDir}
	s := NewSpawner(cfg, nil, nil)

	err := s.ValidateSkills("developer", "", "")
	if err == nil {
		t.Fatal("expected error when skills are missing")
	}
	var errMissing *ErrSkillsMissing
	if !errors.As(err, &errMissing) {
		t.Fatalf("expected ErrSkillsMissing, got %T", err)
	}
	if len(errMissing.Missing) != 1 || errMissing.Missing[0].Name != "kf-developer" {
		t.Errorf("unexpected missing skills: %v", errMissing.Missing)
	}

	devDir := filepath.Join(globalDir, "kf-developer")
	os.MkdirAll(devDir, 0o755)
	os.WriteFile(filepath.Join(devDir, "SKILL.md"), []byte("# Dev"), 0o644)

	if err := s.ValidateSkills("developer", "", ""); err != nil {
		t.Errorf("expected no error after install, got: %v", err)
	}
}

func TestValidateSkills_Reviewer(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	cfg := &config.Config{SkillsDir: globalDir}
	s := NewSpawner(cfg, nil, nil)

	err := s.ValidateSkills("reviewer", "", "")
	if err == nil {
		t.Fatal("expected error when reviewer skill is missing")
	}

	workDir := t.TempDir()
	localDir := filepath.Join(workDir, ".claude", "skills", "kf-reviewer")
	os.MkdirAll(localDir, 0o755)
	os.WriteFile(filepath.Join(localDir, "SKILL.md"), []byte("# Rev"), 0o644)

	if err := s.ValidateSkills("reviewer", workDir, ""); err != nil {
		t.Errorf("expected no error after local install, got: %v", err)
	}
}

func TestValidateSkills_WorktreeProjectDir(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	cfg := &config.Config{SkillsDir: globalDir}
	s := NewSpawner(cfg, nil, nil)

	// workDir (worktree) has no skills, projectDir has the skill installed.
	workDir := t.TempDir()
	projectDir := t.TempDir()

	// Skill not in workDir, not in globalDir, not in projectDir — should fail.
	err := s.ValidateSkills("developer", workDir, projectDir)
	if err == nil {
		t.Fatal("expected error when skills are missing from all directories")
	}

	// Install skill in projectDir — should pass.
	projSkillDir := filepath.Join(projectDir, ".claude", "skills", "kf-developer")
	os.MkdirAll(projSkillDir, 0o755)
	os.WriteFile(filepath.Join(projSkillDir, "SKILL.md"), []byte("# Dev"), 0o644)

	if err := s.ValidateSkills("developer", workDir, projectDir); err != nil {
		t.Errorf("expected no error when skill exists in projectDir, got: %v", err)
	}
}

func TestValidateSkills_ProjectDirSameAsWorkDir(t *testing.T) {
	t.Parallel()

	globalDir := t.TempDir()
	cfg := &config.Config{SkillsDir: globalDir}
	s := NewSpawner(cfg, nil, nil)

	// When projectDir == workDir, should not duplicate the check.
	workDir := t.TempDir()
	skillDir := filepath.Join(workDir, ".claude", "skills", "kf-developer")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Dev"), 0o644)

	if err := s.ValidateSkills("developer", workDir, workDir); err != nil {
		t.Errorf("expected no error when projectDir == workDir, got: %v", err)
	}
}

func TestValidateSkills_UnknownRole(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{SkillsDir: t.TempDir()}
	s := NewSpawner(cfg, nil, nil)

	if err := s.ValidateSkills("unknown", "", ""); err != nil {
		t.Errorf("unexpected error for unknown role: %v", err)
	}
}

func TestErrSkillsMissing_Error(t *testing.T) {
	t.Parallel()

	err := &ErrSkillsMissing{
		Missing: []skills.RequiredSkill{
			{Name: "kf-developer", Reason: "dev"},
			{Name: "kf-reviewer", Reason: "rev"},
		},
	}
	msg := err.Error()
	if msg != "required skills not installed: kf-developer, kf-reviewer" {
		t.Errorf("unexpected error message: %q", msg)
	}
}

func TestResultToStreamEvent(t *testing.T) {
	t.Parallel()

	cost := 0.0342
	result := &types.ResultMessage{
		Type:         "result",
		Subtype:      "success",
		SessionID:    "sess-123",
		TotalCostUSD: &cost,
		Usage: map[string]interface{}{
			"input_tokens":                float64(12500),
			"output_tokens":               float64(3200),
			"cache_read_input_tokens":     float64(8000),
			"cache_creation_input_tokens": float64(1500),
		},
	}

	ev := resultToStreamEvent(result)
	if ev.Type != "result" {
		t.Errorf("Type: want %q, got %q", "result", ev.Type)
	}
	if ev.CostUSD != 0.0342 {
		t.Errorf("CostUSD: want %f, got %f", 0.0342, ev.CostUSD)
	}
	if ev.Usage == nil {
		t.Fatal("Usage is nil")
	}
	if ev.Usage.InputTokens != 12500 {
		t.Errorf("InputTokens: want 12500, got %d", ev.Usage.InputTokens)
	}
	if ev.Usage.OutputTokens != 3200 {
		t.Errorf("OutputTokens: want 3200, got %d", ev.Usage.OutputTokens)
	}
	if ev.Usage.CacheReadTokens != 8000 {
		t.Errorf("CacheReadTokens: want 8000, got %d", ev.Usage.CacheReadTokens)
	}
	if ev.Usage.CacheCreationTokens != 1500 {
		t.Errorf("CacheCreationTokens: want 1500, got %d", ev.Usage.CacheCreationTokens)
	}
}

func TestResultToStreamEvent_NilCost(t *testing.T) {
	t.Parallel()

	result := &types.ResultMessage{
		Type:    "result",
		Subtype: "error_during_execution",
	}

	ev := resultToStreamEvent(result)
	if ev.CostUSD != 0 {
		t.Errorf("CostUSD: want 0, got %f", ev.CostUSD)
	}
	if ev.Usage != nil {
		t.Error("Usage should be nil")
	}
}

func TestExtractUsageInfo(t *testing.T) {
	t.Parallel()

	if got := extractUsageInfo(nil); got != nil {
		t.Errorf("expected nil, got %v", got)
	}

	usage := map[string]interface{}{
		"input_tokens":                float64(100),
		"output_tokens":               float64(50),
		"cache_read_input_tokens":     float64(200),
		"cache_creation_input_tokens": float64(10),
	}
	info := extractUsageInfo(usage)
	if info == nil {
		t.Fatal("expected non-nil UsageInfo")
	}
	if info.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", info.InputTokens)
	}
	if info.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", info.OutputTokens)
	}
	if info.CacheReadTokens != 200 {
		t.Errorf("CacheReadTokens = %d, want 200", info.CacheReadTokens)
	}
	if info.CacheCreationTokens != 10 {
		t.Errorf("CacheCreationTokens = %d, want 10", info.CacheCreationTokens)
	}
}

func TestStopAgent_NotRunning(t *testing.T) {
	t.Parallel()

	s := NewSpawner(&config.Config{}, &stubAgentStore{}, nil)
	err := s.StopAgent("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-running agent")
	}
	if err.Error() != "agent not running: nonexistent" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStopAgent_Running(t *testing.T) {
	t.Parallel()

	store := &stubAgentStore{
		agents: []domain.AgentInfo{
			{ID: "agent-1", Status: "running", SessionID: "sess-1"},
		},
	}
	s := NewSpawner(&config.Config{}, store, nil)

	// Create a mock SDK session.
	ctx, cancel := context.WithCancel(context.Background())
	session := &SDKSession{
		ctx:    ctx,
		cancel: cancel,
		output: make(chan []byte, 10),
		done:   make(chan struct{}),
	}

	ia := &InteractiveAgent{
		Info:       store.agents[0],
		Done:       session.done,
		sdkSession: session,
	}

	s.activeMu.Lock()
	s.activeAgents["agent-1"] = ia
	s.activeMu.Unlock()

	err := s.StopAgent("agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify removed from active agents.
	if _, ok := s.GetActiveAgent("agent-1"); ok {
		t.Error("agent should have been removed from active registry")
	}

	// Verify store status updated.
	agent, _ := store.FindAgent("agent-1")
	if agent.Status != "stopped" {
		t.Errorf("status = %q, want %q", agent.Status, "stopped")
	}
}

func TestResumeAgent_NotFound(t *testing.T) {
	t.Parallel()

	store := &stubAgentStore{}
	s := NewSpawner(&config.Config{}, store, nil)
	_, err := s.ResumeAgent(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}
}

func TestResumeAgent_AlreadyRunning(t *testing.T) {
	t.Parallel()

	store := &stubAgentStore{
		agents: []domain.AgentInfo{
			{ID: "agent-1", Status: "running"},
		},
	}
	s := NewSpawner(&config.Config{}, store, nil)

	// Register as active.
	s.activeMu.Lock()
	s.activeAgents["agent-1"] = &InteractiveAgent{}
	s.activeMu.Unlock()

	_, err := s.ResumeAgent(context.Background(), "agent-1")
	if err == nil {
		t.Fatal("expected error for already-running agent")
	}
	if err.Error() != "agent already running: agent-1" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResumeDeveloper_Success(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not available")
	}

	store := &stubAgentStore{
		agents: []domain.AgentInfo{
			{
				ID:          "dev-1",
				Role:        "developer",
				Status:      "suspended",
				SessionID:   "sess-1",
				WorktreeDir: os.TempDir(),
				Model:       "sonnet",
			},
		},
	}
	s := NewSpawner(&config.Config{}, store, nil)

	info, err := s.ResumeDeveloper(context.Background(), "dev-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ID != "dev-1" {
		t.Errorf("expected agent ID dev-1, got %s", info.ID)
	}
	if info.Status != "running" {
		t.Errorf("expected status running, got %s", info.Status)
	}
}

func TestResumeDeveloper_NotFound(t *testing.T) {
	t.Parallel()

	store := &stubAgentStore{}
	s := NewSpawner(&config.Config{}, store, nil)

	_, err := s.ResumeDeveloper(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}
}

func TestResumeDeveloper_AlreadyActive(t *testing.T) {
	t.Parallel()

	store := &stubAgentStore{
		agents: []domain.AgentInfo{
			{ID: "dev-1", Role: "developer", Status: "running", SessionID: "s1"},
		},
	}
	s := NewSpawner(&config.Config{}, store, nil)

	_, err := s.ResumeDeveloper(context.Background(), "dev-1")
	if err == nil {
		t.Fatal("expected error for active agent")
	}
}

func TestResumeDeveloper_NoSessionID(t *testing.T) {
	t.Parallel()

	store := &stubAgentStore{
		agents: []domain.AgentInfo{
			{ID: "dev-1", Role: "developer", Status: "suspended", SessionID: ""},
		},
	}
	s := NewSpawner(&config.Config{}, store, nil)

	_, err := s.ResumeDeveloper(context.Background(), "dev-1")
	if err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

func TestAgentEnv_ContainsIdentityVars(t *testing.T) {
	t.Parallel()

	s := NewSpawner(&config.Config{OrchestratorPort: 4001}, nil, nil)
	env := s.agentEnv("agent-123", "sess-456", "developer")

	expected := map[string]string{
		"KF_ORCH_URL":   "http://localhost:4001",
		"KF_AGENT_ID":   "agent-123",
		"KF_SESSION_ID": "sess-456",
		"KF_AGENT_ROLE": "developer",
	}
	for k, v := range expected {
		if env[k] != v {
			t.Errorf("%s = %q, want %q", k, env[k], v)
		}
	}
	if len(env) != len(expected) {
		t.Errorf("env has %d keys, want %d", len(env), len(expected))
	}
}

func TestAgentEnv_EmptyValues(t *testing.T) {
	t.Parallel()

	s := NewSpawner(&config.Config{OrchestratorPort: 4001}, nil, nil)
	env := s.agentEnv("", "", "")

	// Even with empty values, all keys should be present
	for _, k := range []string{"KF_ORCH_URL", "KF_AGENT_ID", "KF_SESSION_ID", "KF_AGENT_ROLE"} {
		if _, ok := env[k]; !ok {
			t.Errorf("missing key %s", k)
		}
	}
}

func TestEnsureGitRepo_InitializesWhenMissing(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	// dir has no .git — ensureGitRepo should create one
	if err := ensureGitRepo(context.Background(), dir); err != nil {
		t.Fatalf("ensureGitRepo failed: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, ".git"))
	if err != nil {
		t.Fatalf(".git not created: %v", err)
	}
	if !info.IsDir() {
		t.Error(".git should be a directory")
	}
}

func TestEnsureGitRepo_SkipsExistingRepo(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	// Pre-create a .git directory
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, ".git", "marker")
	os.WriteFile(marker, []byte("test"), 0o644)

	if err := ensureGitRepo(context.Background(), dir); err != nil {
		t.Fatalf("ensureGitRepo failed: %v", err)
	}

	// Marker file should still exist (git init was NOT re-run)
	if _, err := os.Stat(marker); err != nil {
		t.Error("marker file missing — git init may have re-run on existing repo")
	}
}

// spyAnalytics records analytics events for testing.
type spyAnalytics struct {
	events []spyEvent
}

type spyEvent struct {
	Name  string
	Props map[string]any
}

func (s *spyAnalytics) Track(_ context.Context, event string, props map[string]any) {
	s.events = append(s.events, spyEvent{Name: event, Props: props})
}

func (s *spyAnalytics) Shutdown(_ context.Context) error { return nil }

func TestTrackEvent_AgentSessionStarted(t *testing.T) {
	t.Parallel()

	spy := &spyAnalytics{}
	s := NewSpawner(&config.Config{}, nil, nil)
	s.SetAnalyticsTracker(spy)

	// trackEvent is the shared helper — verify it records events.
	s.trackEvent("agent_session_started", map[string]any{"role": "developer", "model": "sonnet"})

	if len(spy.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(spy.events))
	}
	if spy.events[0].Name != "agent_session_started" {
		t.Errorf("event name = %q, want %q", spy.events[0].Name, "agent_session_started")
	}
	if spy.events[0].Props["role"] != "developer" {
		t.Errorf("role = %v, want %q", spy.events[0].Props["role"], "developer")
	}
}

func TestTrackEvent_AgentSessionResumed(t *testing.T) {
	t.Parallel()

	spy := &spyAnalytics{}
	s := NewSpawner(&config.Config{}, nil, nil)
	s.SetAnalyticsTracker(spy)

	s.trackEvent("agent_session_resumed", map[string]any{"role": "developer", "agent_id": "dev-1"})

	if len(spy.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(spy.events))
	}
	if spy.events[0].Name != "agent_session_resumed" {
		t.Errorf("event name = %q, want %q", spy.events[0].Name, "agent_session_resumed")
	}
}

func TestTrackEvent_NilAnalytics(t *testing.T) {
	t.Parallel()

	s := NewSpawner(&config.Config{}, nil, nil)
	// Should not panic with nil analytics.
	s.trackEvent("agent_session_started", map[string]any{"role": "developer"})
}

func TestIntFromMap(t *testing.T) {
	t.Parallel()

	m := map[string]interface{}{
		"float":   float64(42),
		"int":     int(7),
		"string":  "not a number",
		"missing": nil,
	}

	if got := intFromMap(m, "float"); got != 42 {
		t.Errorf("float: got %d, want 42", got)
	}
	if got := intFromMap(m, "int"); got != 7 {
		t.Errorf("int: got %d, want 7", got)
	}
	if got := intFromMap(m, "string"); got != 0 {
		t.Errorf("string: got %d, want 0", got)
	}
	if got := intFromMap(m, "nonexistent"); got != 0 {
		t.Errorf("nonexistent: got %d, want 0", got)
	}
}
