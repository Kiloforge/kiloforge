package rest

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"crelay/internal/core/domain"
	"crelay/internal/core/port"
	"crelay/internal/core/service"
	"crelay/internal/core/testutil"
)

// mockBoardGitea implements port.BoardGiteaClient for board sync tests.
type mockBoardGitea struct {
	movedCards []movedCardCall
	closedIssues []closedIssueCall
}

type movedCardCall struct {
	cardID, columnID int
}

type closedIssueCall struct {
	repo     string
	issueNum int
}

func (m *mockBoardGitea) EnsureLabel(_ context.Context, _, _, _ string) (int, error) { return 1, nil }
func (m *mockBoardGitea) CreateIssue(_ context.Context, _, _, _ string, _ []string) (int, error) {
	return 1, nil
}
func (m *mockBoardGitea) UpdateIssue(_ context.Context, repo string, issueNum int, _, _, state string) error {
	if state == "closed" {
		m.closedIssues = append(m.closedIssues, closedIssueCall{repo, issueNum})
	}
	return nil
}
func (m *mockBoardGitea) CreateProject(_ context.Context, _, _, _ string) (int, error) { return 1, nil }
func (m *mockBoardGitea) ListProjects(_ context.Context, _ string) ([]port.ProjectInfo, error) {
	return nil, nil
}
func (m *mockBoardGitea) CreateColumn(_ context.Context, _ int, _ string) (int, error) {
	return 1, nil
}
func (m *mockBoardGitea) ListColumns(_ context.Context, _ int) ([]port.ColumnInfo, error) {
	return nil, nil
}
func (m *mockBoardGitea) CreateCard(_ context.Context, _, _ int) (int, error) { return 1, nil }
func (m *mockBoardGitea) MoveCard(_ context.Context, cardID, columnID int) error {
	m.movedCards = append(m.movedCards, movedCardCall{cardID, columnID})
	return nil
}

// mockBoardStore implements service.BoardStore for testing.
type mockBoardStore struct {
	config      *domain.BoardConfig
	trackIssues map[string]domain.TrackIssue
}

func newMockBoardStore() *mockBoardStore {
	return &mockBoardStore{
		trackIssues: make(map[string]domain.TrackIssue),
	}
}

func (m *mockBoardStore) GetBoardConfig(_ string) (*domain.BoardConfig, error) {
	return m.config, nil
}
func (m *mockBoardStore) SaveBoardConfig(_ string, cfg *domain.BoardConfig) error {
	m.config = cfg
	return nil
}
func (m *mockBoardStore) GetTrackIssue(_ string, trackID string) (*domain.TrackIssue, error) {
	ti, ok := m.trackIssues[trackID]
	if !ok {
		return nil, nil
	}
	return &ti, nil
}
func (m *mockBoardStore) SaveTrackIssue(_ string, ti domain.TrackIssue) error {
	m.trackIssues[ti.TrackID] = ti
	return nil
}
func (m *mockBoardStore) ListTrackIssues(_ string) ([]domain.TrackIssue, error) {
	var result []domain.TrackIssue
	for _, ti := range m.trackIssues {
		result = append(result, ti)
	}
	return result, nil
}

func newTestBoardSyncer(gitea *mockBoardGitea, store *mockBoardStore) *boardSyncer {
	return &boardSyncer{
		svc:       service.NewBoardService(gitea, store),
		store:     store,
		adminUser: "conductor",
		logger:    log.New(os.Stderr, "[test] ", 0),
	}
}

func TestBoardSync_IsSelfTriggered(t *testing.T) {
	t.Parallel()
	bs := newTestBoardSyncer(&mockBoardGitea{}, newMockBoardStore())

	tests := []struct {
		name    string
		payload map[string]any
		want    bool
	}{
		{"admin user", map[string]any{"sender": map[string]any{"login": "conductor"}}, true},
		{"other user", map[string]any{"sender": map[string]any{"login": "alice"}}, false},
		{"no sender", map[string]any{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bs.isSelfTriggered(tt.payload); got != tt.want {
				t.Errorf("isSelfTriggered = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBoardSync_HandleLabelUpdated(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	store := newMockBoardStore()
	store.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{
			"suggested": 10,
			"approved":  11,
			"in_progress": 12,
		},
	}
	store.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "suggested",
		LastSynced: time.Now(),
	}

	bs := newTestBoardSyncer(gitea, store)

	issue := map[string]any{
		"number": float64(42),
		"labels": []any{
			map[string]any{"name": "status:approved"},
			map[string]any{"name": "type:feature"},
		},
	}

	bs.handleLabelUpdated(context.Background(), "myapp", issue)

	if len(gitea.movedCards) != 1 {
		t.Fatalf("expected 1 card move, got %d", len(gitea.movedCards))
	}
	if gitea.movedCards[0].cardID != 50 {
		t.Errorf("cardID: want 50, got %d", gitea.movedCards[0].cardID)
	}
	if gitea.movedCards[0].columnID != 11 {
		t.Errorf("columnID: want 11 (approved), got %d", gitea.movedCards[0].columnID)
	}

	// Verify mapping updated.
	ti := store.trackIssues["track-1"]
	if ti.Column != "approved" {
		t.Errorf("column: want approved, got %s", ti.Column)
	}
}

func TestBoardSync_HandleIssueClosed(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	store := newMockBoardStore()
	store.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{
			"suggested": 10,
			"completed": 14,
		},
	}
	store.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "suggested",
	}

	bs := newTestBoardSyncer(gitea, store)
	bs.handleIssueClosed(context.Background(), "myapp", map[string]any{"number": float64(42)})

	if len(gitea.movedCards) != 1 {
		t.Fatalf("expected 1 card move, got %d", len(gitea.movedCards))
	}
	if gitea.movedCards[0].columnID != 14 {
		t.Errorf("columnID: want 14 (completed), got %d", gitea.movedCards[0].columnID)
	}
}

func TestBoardSync_HandleIssueAssigned(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	store := newMockBoardStore()
	store.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{
			"suggested":   10,
			"in_progress": 12,
		},
	}
	store.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "suggested",
	}

	bs := newTestBoardSyncer(gitea, store)
	bs.handleIssueAssigned(context.Background(), "myapp", map[string]any{"number": float64(42)})

	if len(gitea.movedCards) != 1 {
		t.Fatalf("expected 1 card move, got %d", len(gitea.movedCards))
	}
	if gitea.movedCards[0].columnID != 12 {
		t.Errorf("columnID: want 12 (in_progress), got %d", gitea.movedCards[0].columnID)
	}
}

func TestBoardSync_HandleIssueAssigned_SkipsIfAlreadyInProgress(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	store := newMockBoardStore()
	store.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{
			"in_progress": 12,
		},
	}
	store.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "in_progress",
	}

	bs := newTestBoardSyncer(gitea, store)
	bs.handleIssueAssigned(context.Background(), "myapp", map[string]any{"number": float64(42)})

	if len(gitea.movedCards) != 0 {
		t.Errorf("expected no card moves (already in progress), got %d", len(gitea.movedCards))
	}
}

func TestBoardSync_HandlePROpened(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	store := newMockBoardStore()
	store.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{
			"in_progress": 12,
			"in_review":   13,
		},
	}
	store.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "in_progress",
	}

	bs := newTestBoardSyncer(gitea, store)
	bs.handlePROpened(context.Background(), "myapp", "track-1", 5)

	if len(gitea.movedCards) != 1 {
		t.Fatalf("expected 1 card move, got %d", len(gitea.movedCards))
	}
	if gitea.movedCards[0].columnID != 13 {
		t.Errorf("columnID: want 13 (in_review), got %d", gitea.movedCards[0].columnID)
	}
}

func TestBoardSync_HandlePRMerged(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	store := newMockBoardStore()
	store.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{
			"in_review": 13,
			"completed": 14,
		},
	}
	store.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "in_review",
	}

	bs := newTestBoardSyncer(gitea, store)
	bs.handlePRMerged(context.Background(), "myapp", "track-1", 5)

	if len(gitea.movedCards) != 1 {
		t.Fatalf("expected 1 card move, got %d", len(gitea.movedCards))
	}
	if gitea.movedCards[0].columnID != 14 {
		t.Errorf("columnID: want 14 (completed), got %d", gitea.movedCards[0].columnID)
	}

	// Issue should be closed.
	if len(gitea.closedIssues) != 1 {
		t.Fatalf("expected 1 issue closed, got %d", len(gitea.closedIssues))
	}
	if gitea.closedIssues[0].issueNum != 42 {
		t.Errorf("closed issue: want 42, got %d", gitea.closedIssues[0].issueNum)
	}
}

func TestBoardSync_AdminEventsSkipped(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	store := newMockBoardStore()
	bs := newTestBoardSyncer(gitea, store)

	// Admin-triggered event should be skipped
	payload := map[string]any{
		"sender": map[string]any{"login": "conductor"},
	}
	if !bs.isSelfTriggered(payload) {
		t.Error("expected admin event to be self-triggered")
	}

	// Non-admin event should not be skipped
	payload = map[string]any{
		"sender": map[string]any{"login": "alice"},
	}
	if bs.isSelfTriggered(payload) {
		t.Error("expected non-admin event to not be self-triggered")
	}
}

func TestBoardSync_NoTrackIssue_Noop(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	store := newMockBoardStore()
	store.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{"completed": 14},
	}
	// No track issues registered

	bs := newTestBoardSyncer(gitea, store)
	bs.handleIssueClosed(context.Background(), "myapp", map[string]any{"number": float64(999)})

	if len(gitea.movedCards) != 0 {
		t.Errorf("expected no card moves for unknown issue, got %d", len(gitea.movedCards))
	}
}

// --- Lifecycle integration tests ---

func newTestBoardSyncerWithLifecycle(
	gitea *mockBoardGitea,
	bstore *mockBoardStore,
	agentStore *testutil.MockAgentStore,
	spawner *testutil.MockAgentSpawner,
	pool *testutil.MockPoolReturner,
) (*boardSyncer, *[]string) {
	bs := newTestBoardSyncer(gitea, bstore)
	var poolReturner port.PoolReturner
	if pool != nil {
		poolReturner = pool
	}
	lifecycle := service.NewLifecycleService(agentStore, spawner, poolReturner, &testutil.MockLogger{})
	bs.lifecycle = lifecycle

	var comments []string
	bs.commentFn = func(_ context.Context, _ string, _ int, body string) {
		comments = append(comments, body)
	}
	return bs, &comments
}

func TestBoardSync_BackwardMove_HaltsDeveloper(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	bstore := newMockBoardStore()
	bstore.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{
			"suggested":   10,
			"approved":    11,
			"in_progress": 12,
		},
	}
	bstore.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "in_progress",
	}

	agentStore := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "running", StartedAt: time.Now()},
		},
	}

	bs, comments := newTestBoardSyncerWithLifecycle(gitea, bstore, agentStore, &testutil.MockAgentSpawner{}, nil)

	issue := map[string]any{
		"number": float64(42),
		"labels": []any{
			map[string]any{"name": "status:approved"},
		},
	}
	bs.handleLabelUpdated(context.Background(), "myapp", issue)

	agent, _ := agentStore.FindAgent("dev-1")
	if agent.Status != "halted" {
		t.Errorf("agent status = %q, want halted", agent.Status)
	}
	if len(*comments) == 0 {
		t.Error("expected comment to be posted")
	}
}

func TestBoardSync_BackwardMove_InReview_HaltsBoth(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	bstore := newMockBoardStore()
	bstore.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{
			"approved":    11,
			"in_progress": 12,
			"in_review":   13,
		},
	}
	bstore.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "in_review",
	}

	agentStore := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "running", StartedAt: time.Now()},
			{ID: "rev-1", Ref: "PR #5", Status: "running", StartedAt: time.Now()},
		},
	}

	prTracking := &domain.PRTracking{TrackID: "track-1", ReviewerAgentID: "rev-1"}
	bs, _ := newTestBoardSyncerWithLifecycle(gitea, bstore, agentStore, &testutil.MockAgentSpawner{}, nil)
	bs.prLoader = func(_ string) (*domain.PRTracking, error) {
		return prTracking, nil
	}

	issue := map[string]any{
		"number": float64(42),
		"labels": []any{
			map[string]any{"name": "status:approved"},
		},
	}
	bs.handleLabelUpdated(context.Background(), "myapp", issue)

	dev, _ := agentStore.FindAgent("dev-1")
	if dev.Status != "halted" {
		t.Errorf("developer status = %q, want halted", dev.Status)
	}
	rev, _ := agentStore.FindAgent("rev-1")
	if rev.Status != "halted" {
		t.Errorf("reviewer status = %q, want halted", rev.Status)
	}
}

func TestBoardSync_BackwardMove_AlreadyHalted(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	bstore := newMockBoardStore()
	bstore.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{"approved": 11, "in_progress": 12},
	}
	bstore.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "in_progress",
	}

	agentStore := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "halted", StartedAt: time.Now()},
		},
	}

	bs, _ := newTestBoardSyncerWithLifecycle(gitea, bstore, agentStore, &testutil.MockAgentSpawner{}, nil)

	issue := map[string]any{
		"number": float64(42),
		"labels": []any{map[string]any{"name": "status:approved"}},
	}
	bs.handleLabelUpdated(context.Background(), "myapp", issue)

	agent, _ := agentStore.FindAgent("dev-1")
	if agent.Status != "halted" {
		t.Errorf("status = %q, want halted (unchanged)", agent.Status)
	}
}

func TestBoardSync_BackwardMove_NoAgent(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	bstore := newMockBoardStore()
	bstore.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{"approved": 11, "in_progress": 12},
	}
	bstore.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "in_progress",
	}

	agentStore := &testutil.MockAgentStore{}
	bs, _ := newTestBoardSyncerWithLifecycle(gitea, bstore, agentStore, &testutil.MockAgentSpawner{}, nil)

	issue := map[string]any{
		"number": float64(42),
		"labels": []any{map[string]any{"name": "status:approved"}},
	}
	bs.handleLabelUpdated(context.Background(), "myapp", issue)
}

func TestBoardSync_ForwardMove_ResumesDeveloper(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	bstore := newMockBoardStore()
	bstore.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{"approved": 11, "in_progress": 12},
	}
	bstore.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "approved",
	}

	workDir := t.TempDir()
	agentStore := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "halted", SessionID: "sess-1", WorktreeDir: workDir, StartedAt: time.Now()},
		},
	}
	spawner := &testutil.MockAgentSpawner{}
	bs, comments := newTestBoardSyncerWithLifecycle(gitea, bstore, agentStore, spawner, nil)

	issue := map[string]any{
		"number": float64(42),
		"labels": []any{map[string]any{"name": "status:in-progress"}},
	}
	bs.handleLabelUpdated(context.Background(), "myapp", issue)

	agent, _ := agentStore.FindAgent("dev-1")
	if agent.Status != "running" {
		t.Errorf("agent status = %q, want running", agent.Status)
	}
	if len(spawner.ResumeCalls) != 1 {
		t.Errorf("expected 1 resume call, got %d", len(spawner.ResumeCalls))
	}
	if len(*comments) == 0 {
		t.Error("expected resume comment")
	}
}

func TestBoardSync_ForwardMove_ResumeFailed(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	bstore := newMockBoardStore()
	bstore.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{"approved": 11, "in_progress": 12},
	}
	bstore.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "approved",
	}

	agentStore := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "halted", SessionID: "", StartedAt: time.Now()},
		},
	}
	bs, comments := newTestBoardSyncerWithLifecycle(gitea, bstore, agentStore, &testutil.MockAgentSpawner{}, nil)

	issue := map[string]any{
		"number": float64(42),
		"labels": []any{map[string]any{"name": "status:in-progress"}},
	}
	bs.handleLabelUpdated(context.Background(), "myapp", issue)

	agent, _ := agentStore.FindAgent("dev-1")
	if agent.Status != "resume-failed" {
		t.Errorf("agent status = %q, want resume-failed", agent.Status)
	}
	if len(*comments) == 0 {
		t.Error("expected failure comment")
	}
}

func TestBoardSync_IssueClosed_Rejection(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	bstore := newMockBoardStore()
	bstore.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{"in_progress": 12, "completed": 14},
	}
	bstore.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "in_progress",
	}

	agentStore := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "running", StartedAt: time.Now()},
		},
	}
	pool := &testutil.MockPoolReturner{}
	bs, comments := newTestBoardSyncerWithLifecycle(gitea, bstore, agentStore, &testutil.MockAgentSpawner{}, pool)

	bs.handleIssueClosed(context.Background(), "myapp", map[string]any{"number": float64(42)})

	agent, _ := agentStore.FindAgent("dev-1")
	if agent.Status != "stopped" {
		t.Errorf("agent status = %q, want stopped", agent.Status)
	}
	if len(pool.Calls) != 1 || pool.Calls[0] != "track-1" {
		t.Errorf("pool calls = %v, want [track-1]", pool.Calls)
	}
	if len(*comments) == 0 {
		t.Error("expected rejection comment")
	}
}

func TestBoardSync_RejectedLabel(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	bstore := newMockBoardStore()
	bstore.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{"in_progress": 12},
	}
	bstore.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "in_progress",
	}

	agentStore := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "dev-1", Ref: "track-1", Status: "running", StartedAt: time.Now()},
		},
	}
	pool := &testutil.MockPoolReturner{}
	bs, comments := newTestBoardSyncerWithLifecycle(gitea, bstore, agentStore, &testutil.MockAgentSpawner{}, pool)

	var closedIssues []int
	bs.updateIssueFn = func(_ context.Context, _ string, issueNum int, _ string) {
		closedIssues = append(closedIssues, issueNum)
	}

	issue := map[string]any{
		"number": float64(42),
		"labels": []any{
			map[string]any{"name": "rejected"},
			map[string]any{"name": "status:in-progress"},
		},
	}
	bs.handleLabelUpdated(context.Background(), "myapp", issue)

	agent, _ := agentStore.FindAgent("dev-1")
	if agent.Status != "stopped" {
		t.Errorf("agent status = %q, want stopped", agent.Status)
	}
	if len(closedIssues) != 1 || closedIssues[0] != 42 {
		t.Errorf("closed issues = %v, want [42]", closedIssues)
	}
	if len(*comments) == 0 {
		t.Error("expected rejection comment")
	}
}

func TestBoardSync_RejectedLabel_NoAgent(t *testing.T) {
	t.Parallel()

	gitea := &mockBoardGitea{}
	bstore := newMockBoardStore()
	bstore.config = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{"in_progress": 12},
	}
	bstore.trackIssues["track-1"] = domain.TrackIssue{
		TrackID: "track-1", IssueNumber: 42, CardID: 50, Column: "in_progress",
	}

	agentStore := &testutil.MockAgentStore{}
	bs, _ := newTestBoardSyncerWithLifecycle(gitea, bstore, agentStore, &testutil.MockAgentSpawner{}, nil)

	issue := map[string]any{
		"number": float64(42),
		"labels": []any{map[string]any{"name": "rejected"}},
	}
	bs.handleLabelUpdated(context.Background(), "myapp", issue)
}
