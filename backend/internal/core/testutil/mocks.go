package testutil

import (
	"context"
	"fmt"
	"sync"

	"crelay/internal/core/domain"
	"crelay/internal/core/port"
)

// Compile-time interface checks.
var (
	_ port.AgentStore    = (*MockAgentStore)(nil)
	_ port.AgentSpawner  = (*MockAgentSpawner)(nil)
	_ port.GiteaClient   = (*MockGiteaClient)(nil)
	_ port.Merger        = (*MockMerger)(nil)
	_ port.PoolReturner  = (*MockPoolReturner)(nil)
	_ port.Logger        = (*MockLogger)(nil)
	_ port.GitRunner     = (*MockGitRunner)(nil)
)

// MockAgentStore is an in-memory AgentStore.
type MockAgentStore struct {
	mu     sync.Mutex
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

func (m *MockAgentStore) AddAgent(info domain.AgentInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AgentData = append(m.AgentData, info)
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

func (m *MockAgentStore) UpdateStatus(idPrefix, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.AgentData {
		if m.AgentData[i].ID == idPrefix {
			m.AgentData[i].Status = status
			return
		}
	}
}

func (m *MockAgentStore) HaltAgent(idPrefix string) error {
	if m.HaltErr != nil {
		return m.HaltErr
	}
	_, err := m.FindAgent(idPrefix)
	return err
}

func (m *MockAgentStore) Agents() []domain.AgentInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.AgentInfo, len(m.AgentData))
	copy(out, m.AgentData)
	return out
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

// MockGiteaClient records Gitea API calls.
type MockGiteaClient struct {
	mu    sync.Mutex
	Calls []GiteaCall

	MergeErr    error
	CommentErr  error
	DeleteErr   error
	LabelErr    error
	GetPRErr    error
	GetReviewsErr error

	PRData      map[string]any
	ReviewsData []map[string]any
}

type GiteaCall struct {
	Method string
	Repo   string
	Args   []any
}

func (m *MockGiteaClient) MergePR(_ context.Context, repo string, prNum int, method string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GiteaCall{Method: "MergePR", Repo: repo, Args: []any{prNum, method}})
	return m.MergeErr
}

func (m *MockGiteaClient) CommentOnPR(_ context.Context, repo string, prNum int, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GiteaCall{Method: "CommentOnPR", Repo: repo, Args: []any{prNum, body}})
	return m.CommentErr
}

func (m *MockGiteaClient) DeleteBranch(_ context.Context, repo, branch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GiteaCall{Method: "DeleteBranch", Repo: repo, Args: []any{branch}})
	return m.DeleteErr
}

func (m *MockGiteaClient) AddLabel(_ context.Context, repo string, prNum int, label string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GiteaCall{Method: "AddLabel", Repo: repo, Args: []any{prNum, label}})
	return m.LabelErr
}

func (m *MockGiteaClient) GetPR(_ context.Context, repo string, prNum int) (map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GiteaCall{Method: "GetPR", Repo: repo, Args: []any{prNum}})
	if m.GetPRErr != nil {
		return nil, m.GetPRErr
	}
	if m.PRData != nil {
		return m.PRData, nil
	}
	return map[string]any{"number": prNum}, nil
}

func (m *MockGiteaClient) GetPRReviews(_ context.Context, repo string, prNum int) ([]map[string]any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GiteaCall{Method: "GetPRReviews", Repo: repo, Args: []any{prNum}})
	if m.GetReviewsErr != nil {
		return nil, m.GetReviewsErr
	}
	return m.ReviewsData, nil
}

// MockMerger records merge operations.
type MockMerger struct {
	mu    sync.Mutex
	Calls []GiteaCall

	MergeErr   error
	CommentErr error
	DeleteErr  error
}

func (m *MockMerger) MergePR(_ context.Context, repo string, prNum int, method string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GiteaCall{Method: "MergePR", Repo: repo, Args: []any{prNum, method}})
	return m.MergeErr
}

func (m *MockMerger) CommentOnPR(_ context.Context, repo string, prNum int, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GiteaCall{Method: "CommentOnPR", Repo: repo, Args: []any{prNum, body}})
	return m.CommentErr
}

func (m *MockMerger) DeleteBranch(_ context.Context, repo, branch string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, GiteaCall{Method: "DeleteBranch", Repo: repo, Args: []any{branch}})
	return m.DeleteErr
}

// MockPoolReturner records pool return calls.
type MockPoolReturner struct {
	mu    sync.Mutex
	Calls []string

	ReturnErr error
}

func (m *MockPoolReturner) ReturnByTrackID(trackID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, trackID)
	return m.ReturnErr
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
