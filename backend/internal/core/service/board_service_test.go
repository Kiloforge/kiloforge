package service_test

import (
	"context"
	"testing"

	"crelay/internal/core/domain"
	"crelay/internal/core/port"
	"crelay/internal/core/service"
)

// mockBoardGitea implements port.BoardGiteaClient for testing.
type mockBoardGitea struct {
	labels       map[string]int
	projects     []port.ProjectInfo
	columns      map[int][]port.ColumnInfo
	nextLabelID  int
	nextProjID   int
	nextColID    int
	nextIssueNum int
	nextCardID   int
	issues       []createdIssue
	cards        []createdCard
	movedCards   []movedCard
}

type createdIssue struct {
	repo, title, body string
	labels            []string
}

type createdCard struct {
	columnID, issueID int
}

type movedCard struct {
	cardID, columnID int
}

func newMockBoardGitea() *mockBoardGitea {
	return &mockBoardGitea{
		labels:       make(map[string]int),
		columns:      make(map[int][]port.ColumnInfo),
		nextLabelID:  100,
		nextProjID:   1,
		nextColID:    10,
		nextIssueNum: 1,
		nextCardID:   50,
	}
}

func (m *mockBoardGitea) EnsureLabel(_ context.Context, _, name, _ string) (int, error) {
	if id, ok := m.labels[name]; ok {
		return id, nil
	}
	m.nextLabelID++
	m.labels[name] = m.nextLabelID
	return m.nextLabelID, nil
}

func (m *mockBoardGitea) CreateIssue(_ context.Context, repo, title, body string, labels []string) (int, error) {
	m.issues = append(m.issues, createdIssue{repo, title, body, labels})
	m.nextIssueNum++
	return m.nextIssueNum - 1, nil
}

func (m *mockBoardGitea) UpdateIssue(_ context.Context, _ string, _ int, _, _, _ string) error {
	return nil
}

func (m *mockBoardGitea) CreateProject(_ context.Context, _, _, _ string) (int, error) {
	m.nextProjID++
	return m.nextProjID - 1, nil
}

func (m *mockBoardGitea) ListProjects(_ context.Context, _ string) ([]port.ProjectInfo, error) {
	return m.projects, nil
}

func (m *mockBoardGitea) CreateColumn(_ context.Context, projID int, title string) (int, error) {
	m.nextColID++
	m.columns[projID] = append(m.columns[projID], port.ColumnInfo{ID: m.nextColID, Title: title})
	return m.nextColID, nil
}

func (m *mockBoardGitea) ListColumns(_ context.Context, projID int) ([]port.ColumnInfo, error) {
	return m.columns[projID], nil
}

func (m *mockBoardGitea) CreateCard(_ context.Context, columnID, issueID int) (int, error) {
	m.cards = append(m.cards, createdCard{columnID, issueID})
	m.nextCardID++
	return m.nextCardID - 1, nil
}

func (m *mockBoardGitea) MoveCard(_ context.Context, cardID, columnID int) error {
	m.movedCards = append(m.movedCards, movedCard{cardID, columnID})
	return nil
}

// mockBoardStore implements service.BoardStore for testing.
type mockBoardStore struct {
	configs     map[string]*domain.BoardConfig
	trackIssues map[string]map[string]domain.TrackIssue
}

func newMockBoardStore() *mockBoardStore {
	return &mockBoardStore{
		configs:     make(map[string]*domain.BoardConfig),
		trackIssues: make(map[string]map[string]domain.TrackIssue),
	}
}

func (m *mockBoardStore) GetBoardConfig(slug string) (*domain.BoardConfig, error) {
	return m.configs[slug], nil
}

func (m *mockBoardStore) SaveBoardConfig(slug string, cfg *domain.BoardConfig) error {
	m.configs[slug] = cfg
	return nil
}

func (m *mockBoardStore) GetTrackIssue(slug, trackID string) (*domain.TrackIssue, error) {
	if m.trackIssues[slug] == nil {
		return nil, nil
	}
	ti, ok := m.trackIssues[slug][trackID]
	if !ok {
		return nil, nil
	}
	return &ti, nil
}

func (m *mockBoardStore) SaveTrackIssue(slug string, ti domain.TrackIssue) error {
	if m.trackIssues[slug] == nil {
		m.trackIssues[slug] = make(map[string]domain.TrackIssue)
	}
	m.trackIssues[slug][ti.TrackID] = ti
	return nil
}

func (m *mockBoardStore) ListTrackIssues(slug string) ([]domain.TrackIssue, error) {
	var result []domain.TrackIssue
	for _, ti := range m.trackIssues[slug] {
		result = append(result, ti)
	}
	return result, nil
}

func TestSetupBoard_CreatesLabelsAndColumns(t *testing.T) {
	t.Parallel()

	gitea := newMockBoardGitea()
	store := newMockBoardStore()
	svc := service.NewBoardService(gitea, store)

	project := domain.Project{Slug: "myapp", RepoName: "myapp"}
	cfg, err := svc.SetupBoard(context.Background(), project)
	if err != nil {
		t.Fatalf("SetupBoard: %v", err)
	}

	if cfg.ProjectBoardID == 0 {
		t.Error("expected non-zero board ID")
	}
	if len(cfg.Labels) != 8 {
		t.Errorf("expected 8 labels, got %d", len(cfg.Labels))
	}
	if len(cfg.Columns) != 5 {
		t.Errorf("expected 5 columns, got %d", len(cfg.Columns))
	}

	// Verify persisted.
	saved := store.configs["myapp"]
	if saved == nil {
		t.Fatal("expected config to be saved")
	}
	if saved.ProjectBoardID != cfg.ProjectBoardID {
		t.Error("saved config doesn't match")
	}
}

func TestSetupBoard_Idempotent(t *testing.T) {
	t.Parallel()

	gitea := newMockBoardGitea()
	store := newMockBoardStore()
	store.configs["myapp"] = &domain.BoardConfig{
		ProjectBoardID: 99,
		Columns:        map[string]int{"suggested": 1},
		Labels:         map[string]int{"type:feature": 10},
	}

	svc := service.NewBoardService(gitea, store)
	project := domain.Project{Slug: "myapp", RepoName: "myapp"}
	cfg, err := svc.SetupBoard(context.Background(), project)
	if err != nil {
		t.Fatalf("SetupBoard: %v", err)
	}
	if cfg.ProjectBoardID != 99 {
		t.Errorf("expected existing board ID 99, got %d", cfg.ProjectBoardID)
	}
}

func TestPublishTrack_CreatesIssueAndCard(t *testing.T) {
	t.Parallel()

	gitea := newMockBoardGitea()
	store := newMockBoardStore()
	store.configs["myapp"] = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{
			"suggested":   10,
			"approved":    11,
			"in_progress": 12,
			"in_review":   13,
			"completed":   14,
		},
		Labels: map[string]int{
			"type:feature":      100,
			"status:suggested":  101,
		},
	}

	svc := service.NewBoardService(gitea, store)
	project := domain.Project{Slug: "myapp", RepoName: "myapp"}
	track := service.TrackEntry{ID: "track-1", Title: "Test Track", Status: service.StatusPending}

	err := svc.PublishTrack(context.Background(), project, track, "feature", "# Spec content")
	if err != nil {
		t.Fatalf("PublishTrack: %v", err)
	}

	if len(gitea.issues) != 1 {
		t.Fatalf("expected 1 issue created, got %d", len(gitea.issues))
	}
	if gitea.issues[0].title != "Test Track" {
		t.Errorf("issue title: want %q, got %q", "Test Track", gitea.issues[0].title)
	}
	if len(gitea.cards) != 1 {
		t.Fatalf("expected 1 card created, got %d", len(gitea.cards))
	}
	if gitea.cards[0].columnID != 10 {
		t.Errorf("card column: want 10 (suggested), got %d", gitea.cards[0].columnID)
	}

	// Check mapping saved.
	ti, _ := store.GetTrackIssue("myapp", "track-1")
	if ti == nil {
		t.Fatal("expected track issue to be saved")
	}
}

func TestPublishTrack_Idempotent(t *testing.T) {
	t.Parallel()

	gitea := newMockBoardGitea()
	store := newMockBoardStore()
	store.configs["myapp"] = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns:        map[string]int{"suggested": 10},
	}
	store.trackIssues["myapp"] = map[string]domain.TrackIssue{
		"track-1": {TrackID: "track-1", IssueNumber: 42},
	}

	svc := service.NewBoardService(gitea, store)
	project := domain.Project{Slug: "myapp", RepoName: "myapp"}
	track := service.TrackEntry{ID: "track-1", Title: "Test", Status: service.StatusPending}

	err := svc.PublishTrack(context.Background(), project, track, "feature", "")
	if err != nil {
		t.Fatalf("PublishTrack: %v", err)
	}
	if len(gitea.issues) != 0 {
		t.Error("expected no issues created for already-published track")
	}
}

func TestSyncTracks_NewAndChanged(t *testing.T) {
	t.Parallel()

	gitea := newMockBoardGitea()
	store := newMockBoardStore()
	store.configs["myapp"] = &domain.BoardConfig{
		ProjectBoardID: 1,
		Columns: map[string]int{
			"suggested":   10,
			"in_progress": 12,
			"completed":   14,
		},
		Labels: map[string]int{},
	}
	// track-existing was published in "suggested" column
	store.trackIssues["myapp"] = map[string]domain.TrackIssue{
		"track-existing": {TrackID: "track-existing", IssueNumber: 5, CardID: 50, Column: "suggested"},
	}

	svc := service.NewBoardService(gitea, store)
	project := domain.Project{Slug: "myapp", RepoName: "myapp"}

	tracks := []service.TrackEntry{
		{ID: "track-new", Title: "New Track", Status: service.StatusPending},
		{ID: "track-existing", Title: "Existing", Status: service.StatusInProgress},
	}

	result, err := svc.SyncTracks(context.Background(), project, tracks, nil, nil)
	if err != nil {
		t.Fatalf("SyncTracks: %v", err)
	}
	if result.Created != 1 {
		t.Errorf("Created: want 1, got %d", result.Created)
	}
	if result.Updated != 1 {
		t.Errorf("Updated: want 1, got %d", result.Updated)
	}
	if result.Unchanged != 0 {
		t.Errorf("Unchanged: want 0, got %d", result.Unchanged)
	}

	// Verify card was moved.
	if len(gitea.movedCards) != 1 {
		t.Fatalf("expected 1 moved card, got %d", len(gitea.movedCards))
	}
	if gitea.movedCards[0].columnID != 12 {
		t.Errorf("moved to column: want 12 (in_progress), got %d", gitea.movedCards[0].columnID)
	}
}

func TestStatusToColumn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status string
		want   string
	}{
		{service.StatusPending, "Suggested"},
		{service.StatusApproved, "Approved"},
		{service.StatusInProgress, "In Progress"},
		{service.StatusInReview, "In Review"},
		{service.StatusComplete, "Completed"},
		{"unknown", "Suggested"},
	}
	for _, tt := range tests {
		if got := service.StatusToColumn(tt.status); got != tt.want {
			t.Errorf("StatusToColumn(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestColumnToStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		column string
		want   string
	}{
		{"Suggested", service.StatusPending},
		{"Approved", service.StatusApproved},
		{"In Progress", service.StatusInProgress},
		{"In Review", service.StatusInReview},
		{"Completed", service.StatusComplete},
		{"Unknown", service.StatusPending},
	}
	for _, tt := range tests {
		if got := service.ColumnToStatus(tt.column); got != tt.want {
			t.Errorf("ColumnToStatus(%q) = %q, want %q", tt.column, got, tt.want)
		}
	}
}
