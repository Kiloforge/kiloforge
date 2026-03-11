package testutil

import (
	"context"
	"fmt"
	"sync"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// Compile-time interface checks.
var (
	_ port.AgentStore       = (*MockAgentStore)(nil)
	_ port.AgentSpawner     = (*MockAgentSpawner)(nil)
	_ port.Logger           = (*MockLogger)(nil)
	_ port.GitRunner        = (*MockGitRunner)(nil)
	_ port.AnalyticsTracker = (*SpyAnalytics)(nil)
)

// MockAgentStore is an in-memory AgentStore.
type MockAgentStore struct {
	mu        sync.Mutex
	AgentData []domain.AgentInfo

	// Injectable errors.
	SaveErr error
	LoadErr error
	HaltErr error
}

func (m *MockAgentStore) Load() error {
	if m.LoadErr != nil {
		return m.LoadErr
	}
	return nil
}

func (m *MockAgentStore) Save() error {
	if m.SaveErr != nil {
		return m.SaveErr
	}
	return nil
}

func (m *MockAgentStore) AddAgent(info domain.AgentInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Upsert: replace if exists, append if new.
	for i := range m.AgentData {
		if m.AgentData[i].ID == info.ID {
			m.AgentData[i] = info
			return nil
		}
	}
	m.AgentData = append(m.AgentData, info)
	return nil
}

func (m *MockAgentStore) FindAgent(idPrefix string) (*domain.AgentInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.AgentData {
		if m.AgentData[i].ID == idPrefix {
			return &m.AgentData[i], nil
		}
	}
	return nil, domain.ErrAgentNotFound
}

func (m *MockAgentStore) UpdateStatus(idPrefix, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.AgentData {
		if m.AgentData[i].ID == idPrefix {
			m.AgentData[i].Status = status
			return nil
		}
	}
	return nil
}

func (m *MockAgentStore) HaltAgent(idPrefix string) error {
	if m.HaltErr != nil {
		return m.HaltErr
	}
	_, err := m.FindAgent(idPrefix)
	return err
}

func (m *MockAgentStore) RemoveAgent(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.AgentData {
		if m.AgentData[i].ID == id {
			m.AgentData = append(m.AgentData[:i], m.AgentData[i+1:]...)
			return nil
		}
	}
	return domain.ErrAgentNotFound
}

func (m *MockAgentStore) Agents() []domain.AgentInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.AgentInfo, len(m.AgentData))
	copy(out, m.AgentData)
	return out
}

func (m *MockAgentStore) FindByRef(ref string) *domain.AgentInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	var best *domain.AgentInfo
	for i := range m.AgentData {
		if m.AgentData[i].Ref == ref {
			if best == nil || m.AgentData[i].StartedAt.After(best.StartedAt) {
				best = &m.AgentData[i]
			}
		}
	}
	return best
}

func (m *MockAgentStore) ListAgents(opts domain.PageOpts, statuses ...string) (domain.Page[domain.AgentInfo], error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	opts.Normalize()
	var filtered []domain.AgentInfo
	if len(statuses) > 0 {
		set := make(map[string]bool, len(statuses))
		for _, s := range statuses {
			set[s] = true
		}
		for _, a := range m.AgentData {
			if set[a.Status] {
				filtered = append(filtered, a)
			}
		}
	} else {
		filtered = append(filtered, m.AgentData...)
	}
	total := len(filtered)
	if len(filtered) > opts.Limit {
		filtered = filtered[:opts.Limit]
	}
	return domain.Page[domain.AgentInfo]{Items: filtered, TotalCount: total}, nil
}

func (m *MockAgentStore) AgentsByStatus(statuses ...string) []domain.AgentInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	set := make(map[string]bool, len(statuses))
	for _, st := range statuses {
		set[st] = true
	}
	var result []domain.AgentInfo
	for _, a := range m.AgentData {
		if set[a.Status] {
			result = append(result, a)
		}
	}
	return result
}

// MockAgentSpawner records spawn/resume calls.
type MockAgentSpawner struct {
	mu            sync.Mutex
	ReviewerCalls []port.ReviewerOpts
	ResumeCalls   []ResumeCall

	SpawnErr  error
	ResumeErr error
}

type ResumeCall struct {
	SessionID string
	WorkDir   string
}

func (m *MockAgentSpawner) SpawnReviewer(_ context.Context, opts port.ReviewerOpts) (*domain.AgentInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SpawnErr != nil {
		return nil, m.SpawnErr
	}
	m.ReviewerCalls = append(m.ReviewerCalls, opts)
	return &domain.AgentInfo{
		ID:        fmt.Sprintf("reviewer-%d", opts.PRNumber),
		Role:      "reviewer",
		Ref:       fmt.Sprintf("PR #%d", opts.PRNumber),
		Status:    "running",
		SessionID: "mock-session",
	}, nil
}

func (m *MockAgentSpawner) ResumeDeveloper(_ context.Context, sessionID, workDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ResumeErr != nil {
		return m.ResumeErr
	}
	m.ResumeCalls = append(m.ResumeCalls, ResumeCall{SessionID: sessionID, WorkDir: workDir})
	return nil
}

// MockLogger discards log output (silent logger for tests).
type MockLogger struct {
	mu       sync.Mutex
	Messages []string
}

func (m *MockLogger) Printf(format string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, fmt.Sprintf(format, args...))
}

// MockGitRunner records git operations.
type MockGitRunner struct {
	mu    sync.Mutex
	Calls []GitRunnerCall

	WorktreeAddErr    error
	WorktreeRemoveErr error
	ResetHardMainErr  error
	CheckoutErr       error
	CreateBranchErr   error
	DeleteBranchErr   error

	// Stash operation controls.
	CommitWIPErr   error
	HasAhead       bool
	HasAheadErr    error
	StashBranches  []string
	ListStashErr   error
	MergeBranchErr error
}

type GitRunnerCall struct {
	Method string
	Args   []string
}

func (m *MockGitRunner) WorktreeAdd(path, branch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GitRunnerCall{Method: "WorktreeAdd", Args: []string{path, branch}})
	return m.WorktreeAddErr
}

func (m *MockGitRunner) WorktreeRemove(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GitRunnerCall{Method: "WorktreeRemove", Args: []string{path}})
	return m.WorktreeRemoveErr
}

func (m *MockGitRunner) ResetHardMain(worktreePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GitRunnerCall{Method: "ResetHardMain", Args: []string{worktreePath}})
	return m.ResetHardMainErr
}

func (m *MockGitRunner) CheckoutBranch(worktreePath, branch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GitRunnerCall{Method: "CheckoutBranch", Args: []string{worktreePath, branch}})
	return m.CheckoutErr
}

func (m *MockGitRunner) CreateBranch(worktreePath, branch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GitRunnerCall{Method: "CreateBranch", Args: []string{worktreePath, branch}})
	return m.CreateBranchErr
}

func (m *MockGitRunner) DeleteBranch(branch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GitRunnerCall{Method: "DeleteBranch", Args: []string{branch}})
	return m.DeleteBranchErr
}

func (m *MockGitRunner) AddAll(worktreePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GitRunnerCall{Method: "AddAll", Args: []string{worktreePath}})
	return nil
}

func (m *MockGitRunner) CommitWIP(worktreePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GitRunnerCall{Method: "CommitWIP", Args: []string{worktreePath}})
	return m.CommitWIPErr
}

func (m *MockGitRunner) HasCommitsAhead(worktreePath, base string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GitRunnerCall{Method: "HasCommitsAhead", Args: []string{worktreePath, base}})
	return m.HasAhead, m.HasAheadErr
}

func (m *MockGitRunner) CreateStashBranch(worktreePath, stashBranch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GitRunnerCall{Method: "CreateStashBranch", Args: []string{worktreePath, stashBranch}})
	return nil
}

func (m *MockGitRunner) ListStashBranches(trackID string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GitRunnerCall{Method: "ListStashBranches", Args: []string{trackID}})
	return m.StashBranches, m.ListStashErr
}

func (m *MockGitRunner) MergeBranch(worktreePath, branch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GitRunnerCall{Method: "MergeBranch", Args: []string{worktreePath, branch}})
	return m.MergeBranchErr
}

func (m *MockGitRunner) DeleteBranches(branches []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GitRunnerCall{Method: "DeleteBranches", Args: branches})
	return nil
}

// SpyAnalytics records analytics events for testing.
type SpyAnalytics struct {
	mu     sync.Mutex
	events []SpyAnalyticsEvent
}

// SpyAnalyticsEvent records a single analytics event.
type SpyAnalyticsEvent struct {
	Name  string
	Props map[string]any
}

func (s *SpyAnalytics) Track(_ context.Context, event string, props map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, SpyAnalyticsEvent{Name: event, Props: props})
}

func (s *SpyAnalytics) Shutdown(_ context.Context) error { return nil }

// Events returns a copy of the recorded events.
func (s *SpyAnalytics) Events() []SpyAnalyticsEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]SpyAnalyticsEvent, len(s.events))
	copy(cp, s.events)
	return cp
}
