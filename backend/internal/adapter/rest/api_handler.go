package rest

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/auth"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/prereq"
	gitadapter "kiloforge/internal/adapter/git"
	"kiloforge/internal/adapter/lock"
	"kiloforge/pkg/kf"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/adapter/skills"
	"kiloforge/internal/adapter/tracing"
	wsAdapter "kiloforge/internal/adapter/ws"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// AgentLister provides read access to agent state.
type AgentLister interface {
	Agents() []domain.AgentInfo
	FindAgent(idPrefix string) (*domain.AgentInfo, error)
	Load() error
}

// QuotaReader provides read access to quota data.
type QuotaReader interface {
	GetAgentUsage(agentID string) *agent.AgentUsage
	GetTotalUsage() agent.TotalUsage
	IsRateLimited() bool
	RetryAfter() time.Duration
}

// ProjectLister provides read access to registered projects for API handlers.
type ProjectLister interface {
	List() []domain.Project
}

// ProjectManager handles project add/remove operations.
type ProjectManager interface {
	AddProject(ctx context.Context, remoteURL, name string, opts ...domain.AddProjectOpts) (*domain.AddProjectResult, error)
	RemoveProject(ctx context.Context, slug string, cleanup bool) error
}

// InteractiveSpawner creates and manages interactive agent sessions.
type InteractiveSpawner interface {
	SpawnInteractive(ctx context.Context, opts agent.SpawnInteractiveOpts) (*agent.InteractiveAgent, error)
	StopAgent(id string) error
	ResumeAgent(ctx context.Context, id string) (*agent.InteractiveAgent, error)
	GetActiveAgent(id string) (*agent.InteractiveAgent, bool)
}

// AgentRemover can remove agent records from persistence.
type AgentRemover interface {
	RemoveAgent(id string) error
}

// ConsentChecker provides consent state access.
type ConsentChecker interface {
	HasAgentPermissionsConsent() bool
	RecordAgentPermissionsConsent() error
}

// APIHandler implements gen.StrictServerInterface by delegating to existing
// adapters for agents, locks, quota, and tracks.
type APIHandler struct {
	agents     AgentLister
	quota      QuotaReader
	lockMgr    *lock.Manager
	projects   ProjectLister
	projectMgr ProjectManager
	gitSync    *gitadapter.GitSync
	diffProv   port.DiffProvider
	traceStore tracing.TraceReader
	boardSvc     port.BoardService
	trackReader  port.TrackReader
	eventBus     port.EventBus
	giteaURL      string
	sseClients    func() int
	cfg           *config.Config
	interSpawner  InteractiveSpawner
	wsSessions    *wsAdapter.SessionManager
	consent       ConsentChecker
	agentRemover  AgentRemover

	adminMu          sync.Mutex
	runningAdminAgent string // agent ID of currently running admin op, empty if none
}

// APIHandlerOpts configures the API handler.
type APIHandlerOpts struct {
	Agents        AgentLister
	Quota         QuotaReader
	LockMgr       *lock.Manager
	Projects      ProjectLister
	ProjectMgr    ProjectManager
	GitSync       *gitadapter.GitSync
	DiffProvider  port.DiffProvider
	TraceStore    tracing.TraceReader
	BoardSvc      port.BoardService
	TrackReader   port.TrackReader
	EventBus      port.EventBus
	GiteaURL      string
	SSEClients    func() int
	Cfg           *config.Config
	InterSpawner  InteractiveSpawner
	WSSessions    *wsAdapter.SessionManager
	Consent       ConsentChecker
	AgentRemover  AgentRemover
}

// NewAPIHandler creates a new handler implementing StrictServerInterface.
func NewAPIHandler(opts APIHandlerOpts) *APIHandler {
	return &APIHandler{
		agents:       opts.Agents,
		quota:        opts.Quota,
		lockMgr:      opts.LockMgr,
		projects:     opts.Projects,
		projectMgr:   opts.ProjectMgr,
		gitSync:      opts.GitSync,
		diffProv:     opts.DiffProvider,
		traceStore:   opts.TraceStore,
		boardSvc:     opts.BoardSvc,
		trackReader:  opts.TrackReader,
		eventBus:     opts.EventBus,
		giteaURL:     opts.GiteaURL,
		sseClients:   opts.SSEClients,
		cfg:          opts.Cfg,
		interSpawner: opts.InterSpawner,
		wsSessions:   opts.WSSessions,
		consent:      opts.Consent,
		agentRemover: opts.AgentRemover,
	}
}

// Compile-time check.
var _ gen.StrictServerInterface = (*APIHandler)(nil)

// GetHealth implements gen.StrictServerInterface.
func (h *APIHandler) GetHealth(_ context.Context, _ gen.GetHealthRequestObject) (gen.GetHealthResponseObject, error) {
	projectCount := 0
	if h.projects != nil {
		projectCount = len(h.projects.List())
	}
	return gen.GetHealth200JSONResponse{
		Status:   "ok",
		Projects: projectCount,
	}, nil
}

// GetPreflight implements gen.StrictServerInterface.
func (h *APIHandler) GetPreflight(ctx context.Context, _ gen.GetPreflightRequestObject) (gen.GetPreflightResponseObject, error) {
	resp := gen.GetPreflight200JSONResponse{
		ClaudeAuthenticated: true,
		SkillsOk:           true,
		ConsentGiven:        true,
		SetupRequired:       h.isSetupRequired(),
	}

	// Auth check.
	if err := prereq.CheckClaudeAuthCached(ctx); err != nil {
		resp.ClaudeAuthenticated = false
		msg := err.Error()
		resp.ClaudeAuthError = &msg
	}

	// Skills check (interactive role as baseline).
	if h.cfg != nil {
		required := skills.RequiredSkillsForRole("interactive")
		if len(required) > 0 {
			globalDir := h.cfg.GetSkillsDir()
			missing := skills.CheckRequired(required, globalDir, "")
			if len(missing) > 0 {
				resp.SkillsOk = false
				names := make([]string, len(missing))
				for i, m := range missing {
					names[i] = m.Name
				}
				resp.SkillsMissing = &names
			}
		}
	}

	// Consent check.
	if h.consent != nil && !h.consent.HasAgentPermissionsConsent() {
		resp.ConsentGiven = false
	}

	return resp, nil
}

// GetConfig implements gen.StrictServerInterface.
func (h *APIHandler) GetConfig(_ context.Context, _ gen.GetConfigRequestObject) (gen.GetConfigResponseObject, error) {
	if h.cfg == nil {
		return gen.GetConfig200JSONResponse{}, nil
	}
	return gen.GetConfig200JSONResponse{
		DashboardEnabled: h.cfg.IsDashboardEnabled(),
	}, nil
}

// UpdateConfig implements gen.StrictServerInterface.
func (h *APIHandler) UpdateConfig(_ context.Context, req gen.UpdateConfigRequestObject) (gen.UpdateConfigResponseObject, error) {
	if h.cfg == nil {
		return gen.UpdateConfig500JSONResponse{Error: "config not available"}, nil
	}
	if req.Body == nil {
		return gen.UpdateConfig400JSONResponse{Error: "request body required"}, nil
	}

	if req.Body.DashboardEnabled != nil {
		v := *req.Body.DashboardEnabled
		h.cfg.DashboardEnabled = &v
	}

	if err := h.cfg.Save(); err != nil {
		return gen.UpdateConfig500JSONResponse{Error: fmt.Sprintf("save config: %v", err)}, nil
	}

	return gen.UpdateConfig200JSONResponse{
		DashboardEnabled: h.cfg.IsDashboardEnabled(),
	}, nil
}

// GetAgentPermissionsConsent implements gen.StrictServerInterface.
func (h *APIHandler) GetAgentPermissionsConsent(_ context.Context, _ gen.GetAgentPermissionsConsentRequestObject) (gen.GetAgentPermissionsConsentResponseObject, error) {
	if h.consent == nil {
		return gen.GetAgentPermissionsConsent200JSONResponse{Consented: false}, nil
	}
	consented := h.consent.HasAgentPermissionsConsent()
	return gen.GetAgentPermissionsConsent200JSONResponse{Consented: consented}, nil
}

// RecordAgentPermissionsConsent implements gen.StrictServerInterface.
func (h *APIHandler) RecordAgentPermissionsConsent(_ context.Context, _ gen.RecordAgentPermissionsConsentRequestObject) (gen.RecordAgentPermissionsConsentResponseObject, error) {
	if h.consent == nil {
		return gen.RecordAgentPermissionsConsent500JSONResponse{Error: "consent store not configured"}, nil
	}
	if err := h.consent.RecordAgentPermissionsConsent(); err != nil {
		return gen.RecordAgentPermissionsConsent500JSONResponse{Error: fmt.Sprintf("record consent: %v", err)}, nil
	}
	return gen.RecordAgentPermissionsConsent200JSONResponse{Consented: true}, nil
}

// checkConsent returns a 403 error string if consent is required but not given.
// Returns empty string if consent is granted or consent store is not configured.
func (h *APIHandler) checkConsent() string {
	if h.consent == nil {
		return ""
	}
	if h.consent.HasAgentPermissionsConsent() {
		return ""
	}
	return "agent_permissions_not_consented: user must consent to agent permissions before spawning agents. POST /api/consent/agent-permissions to consent."
}

// checkSetup returns the project slug if kiloforge setup is required for the given project.
// Returns empty string if setup is complete or project not found.
func (h *APIHandler) checkSetup(projectSlug string) string {
	if projectSlug == "" || h.projects == nil || h.trackReader == nil {
		return ""
	}
	proj, ok := h.findProject(projectSlug)
	if !ok {
		return ""
	}
	if !h.trackReader.IsInitialized(proj.ProjectDir) {
		return projectSlug
	}
	return ""
}

// isSetupRequired checks if any registered project is missing kiloforge setup.
func (h *APIHandler) isSetupRequired() bool {
	if h.projects == nil || h.trackReader == nil {
		return false
	}
	for _, p := range h.projects.List() {
		if !h.trackReader.IsInitialized(p.ProjectDir) {
			return true
		}
	}
	return false
}

// checkClaudeAuth returns a non-empty error string if the Claude CLI is not authenticated.
func (h *APIHandler) checkClaudeAuth(ctx context.Context) string {
	if err := prereq.CheckClaudeAuthCached(ctx); err != nil {
		return err.Error()
	}
	return ""
}

// ListAgents implements gen.StrictServerInterface.
func (h *APIHandler) ListAgents(_ context.Context, req gen.ListAgentsRequestObject) (gen.ListAgentsResponseObject, error) {
	if err := h.agents.Load(); err != nil {
		return gen.ListAgents500JSONResponse{Error: "failed to load agent state"}, nil
	}
	agents := h.agents.Agents()
	// Default: active=true — show active + recently finished (30 min TTL).
	showAll := req.Params.Active != nil && !*req.Params.Active
	if !showAll {
		agents = filterActiveAgents(agents, time.Now().Add(-30*time.Minute))
	}
	result := make(gen.ListAgents200JSONResponse, 0, len(agents))
	for _, a := range agents {
		result = append(result, domainAgentToGen(a, h.quota))
	}
	return result, nil
}

// filterActiveAgents returns agents that are active (running/waiting) or
// finished after the given cutoff time (display TTL).
func filterActiveAgents(agents []domain.AgentInfo, cutoff time.Time) []domain.AgentInfo {
	var filtered []domain.AgentInfo
	for _, a := range agents {
		if a.IsActive() {
			filtered = append(filtered, a)
			continue
		}
		if a.FinishedAt != nil && a.FinishedAt.After(cutoff) {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

// SpawnInteractiveAgent implements gen.StrictServerInterface.
func (h *APIHandler) SpawnInteractiveAgent(ctx context.Context, req gen.SpawnInteractiveAgentRequestObject) (gen.SpawnInteractiveAgentResponseObject, error) {
	if h.interSpawner == nil || h.wsSessions == nil {
		return gen.SpawnInteractiveAgent500JSONResponse{Error: "interactive agents not configured"}, nil
	}

	// Check Claude CLI authentication.
	if msg := h.checkClaudeAuth(ctx); msg != "" {
		return gen.SpawnInteractiveAgent401JSONResponse{Error: msg}, nil
	}

	// Check agent permissions consent.
	if msg := h.checkConsent(); msg != "" {
		return gen.SpawnInteractiveAgent403JSONResponse{Error: msg}, nil
	}

	// Validate required skills for interactive agents.
	if resp := h.checkSkillsForRole("interactive", ""); resp != nil {
		return gen.SpawnInteractiveAgent412JSONResponse(*resp), nil
	}

	// Check kiloforge setup if a project is specified.
	if req.Body != nil && req.Body.Project != nil && *req.Body.Project != "" {
		if slug := h.checkSetup(*req.Body.Project); slug != "" {
			return gen.SpawnInteractiveAgent428JSONResponse{
				Error:   "kiloforge setup required",
				Project: slug,
			}, nil
		}
	}

	opts := agent.SpawnInteractiveOpts{}
	if req.Body != nil {
		if req.Body.WorkDir != nil {
			opts.WorkDir = *req.Body.WorkDir
		}
		if req.Body.Model != nil {
			opts.Model = *req.Body.Model
		}
	}

	ia, err := h.interSpawner.SpawnInteractive(ctx, opts)
	if err != nil {
		if strings.Contains(err.Error(), "rate limited") {
			return gen.SpawnInteractiveAgent429JSONResponse{Error: err.Error()}, nil
		}
		return gen.SpawnInteractiveAgent500JSONResponse{Error: err.Error()}, nil
	}

	// Create SDK bridge and register with WS session manager.
	bridge := wsAdapter.NewSDKBridge(ia.Info.ID, ia.Stdin, ia.Done)
	h.wsSessions.RegisterBridge(ia.Info.ID, bridge)

	// Start structured message relay in background with cancellable context.
	relayCtx, cancelRelay := context.WithCancel(context.Background())
	ia.SetCancelRelay(cancelRelay)
	go h.wsSessions.StartStructuredRelay(relayCtx, ia.Info.ID, ia.Output)

	return gen.SpawnInteractiveAgent201JSONResponse(domainAgentToGen(ia.Info, h.quota)), nil
}

// GetAgent implements gen.StrictServerInterface.
func (h *APIHandler) GetAgent(_ context.Context, req gen.GetAgentRequestObject) (gen.GetAgentResponseObject, error) {
	if err := h.agents.Load(); err != nil {
		return gen.GetAgent500JSONResponse{Error: "failed to load agent state"}, nil
	}
	a, err := h.agents.FindAgent(req.Id)
	if err != nil {
		return gen.GetAgent404JSONResponse{Error: "agent not found"}, nil
	}
	return gen.GetAgent200JSONResponse(domainAgentToGen(*a, h.quota)), nil
}

// GetAgentLog implements gen.StrictServerInterface.
func (h *APIHandler) GetAgentLog(_ context.Context, req gen.GetAgentLogRequestObject) (gen.GetAgentLogResponseObject, error) {
	if err := h.agents.Load(); err != nil {
		return gen.GetAgentLog500JSONResponse{Error: "failed to load agent state"}, nil
	}
	a, err := h.agents.FindAgent(req.Id)
	if err != nil {
		return gen.GetAgentLog404JSONResponse{Error: "agent not found"}, nil
	}
	if a.LogFile == "" {
		return gen.GetAgentLog404JSONResponse{Error: "no log file for agent"}, nil
	}

	f, err := os.Open(a.LogFile)
	if err != nil {
		return gen.GetAgentLog404JSONResponse{Error: "log file not accessible"}, nil
	}
	defer f.Close()

	lines := 100
	if req.Params.Lines != nil && *req.Params.Lines > 0 {
		lines = *req.Params.Lines
	}

	var allLines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	start := 0
	if len(allLines) > lines {
		start = len(allLines) - lines
	}
	tail := allLines[start:]

	return gen.GetAgentLog200JSONResponse{
		AgentId: a.ID,
		Lines:   tail,
		Total:   len(allLines),
	}, nil
}

// StopAgent implements gen.StrictServerInterface.
func (h *APIHandler) StopAgent(_ context.Context, req gen.StopAgentRequestObject) (gen.StopAgentResponseObject, error) {
	if h.interSpawner == nil {
		return gen.StopAgent409JSONResponse{Error: "interactive agents not configured"}, nil
	}

	if err := h.interSpawner.StopAgent(req.Id); err != nil {
		if strings.Contains(err.Error(), "not running") {
			return gen.StopAgent409JSONResponse{Error: err.Error()}, nil
		}
		return gen.StopAgent404JSONResponse{Error: err.Error()}, nil
	}

	// Unregister WS bridge.
	if h.wsSessions != nil {
		h.wsSessions.UnregisterBridge(req.Id)
	}

	// Return updated agent.
	a, err := h.agents.FindAgent(req.Id)
	if err != nil {
		return gen.StopAgent404JSONResponse{Error: "agent not found after stop"}, nil
	}
	return gen.StopAgent200JSONResponse(domainAgentToGen(*a, h.quota)), nil
}

// ResumeAgent implements gen.StrictServerInterface.
func (h *APIHandler) ResumeAgent(ctx context.Context, req gen.ResumeAgentRequestObject) (gen.ResumeAgentResponseObject, error) {
	if h.interSpawner == nil || h.wsSessions == nil {
		return gen.ResumeAgent409JSONResponse{Error: "interactive agents not configured"}, nil
	}

	ia, err := h.interSpawner.ResumeAgent(ctx, req.Id)
	if err != nil {
		if strings.Contains(err.Error(), "already running") {
			return gen.ResumeAgent409JSONResponse{Error: err.Error()}, nil
		}
		if strings.Contains(err.Error(), "not found") {
			return gen.ResumeAgent404JSONResponse{Error: err.Error()}, nil
		}
		return gen.ResumeAgent409JSONResponse{Error: err.Error()}, nil
	}

	// Create SDK bridge and register with WS session manager.
	bridge := wsAdapter.NewSDKBridge(ia.Info.ID, ia.Stdin, ia.Done)
	h.wsSessions.RegisterBridge(ia.Info.ID, bridge)

	// Start structured message relay in background with cancellable context.
	relayCtx, cancelRelay := context.WithCancel(context.Background())
	ia.SetCancelRelay(cancelRelay)
	go h.wsSessions.StartStructuredRelay(relayCtx, ia.Info.ID, ia.Output)

	return gen.ResumeAgent200JSONResponse(domainAgentToGen(ia.Info, h.quota)), nil
}

// DeleteAgent implements gen.StrictServerInterface.
func (h *APIHandler) DeleteAgent(_ context.Context, req gen.DeleteAgentRequestObject) (gen.DeleteAgentResponseObject, error) {
	if h.agentRemover == nil {
		return gen.DeleteAgent409JSONResponse{Error: "agent removal not configured"}, nil
	}

	// Check agent is not running.
	if h.interSpawner != nil {
		if _, ok := h.interSpawner.GetActiveAgent(req.Id); ok {
			return gen.DeleteAgent409JSONResponse{Error: "agent still running — stop it first"}, nil
		}
	}

	// Find agent to get log file path.
	a, err := h.agents.FindAgent(req.Id)
	if err != nil {
		return gen.DeleteAgent404JSONResponse{Error: "agent not found"}, nil
	}

	// Delete log file if exists.
	if a.LogFile != "" {
		_ = os.Remove(a.LogFile)
	}

	// Remove from store.
	if err := h.agentRemover.RemoveAgent(a.ID); err != nil {
		return gen.DeleteAgent404JSONResponse{Error: err.Error()}, nil
	}

	return gen.DeleteAgent204Response{}, nil
}

// GetQuota implements gen.StrictServerInterface.
func (h *APIHandler) GetQuota(_ context.Context, _ gen.GetQuotaRequestObject) (gen.GetQuotaResponseObject, error) {
	if h.quota == nil {
		return gen.GetQuota200JSONResponse{
			EstimatedCostUsd:    0,
			InputTokens:         0,
			OutputTokens:        0,
			CacheReadTokens:     0,
			CacheCreationTokens: 0,
			AgentCount:          0,
			RateLimited:         false,
		}, nil
	}
	total := h.quota.GetTotalUsage()
	resp := gen.QuotaInfo{
		EstimatedCostUsd:    total.TotalCostUSD,
		InputTokens:         total.InputTokens,
		OutputTokens:        total.OutputTokens,
		CacheReadTokens:     total.CacheReadTokens,
		CacheCreationTokens: total.CacheCreationTokens,
		AgentCount:          total.AgentCount,
		RateLimited:         h.quota.IsRateLimited(),
	}
	if h.quota.IsRateLimited() {
		resp.RetryAfterSeconds = intPtr(int(h.quota.RetryAfter().Seconds()))
	}

	// Per-agent breakdown.
	if err := h.agents.Load(); err == nil {
		agents := h.agents.Agents()
		var perAgent []gen.QuotaAgentUsage
		for _, a := range agents {
			if usage := h.quota.GetAgentUsage(a.ID); usage != nil {
				perAgent = append(perAgent, gen.QuotaAgentUsage{
					AgentId:             a.ID,
					EstimatedCostUsd:    usage.TotalCostUSD,
					InputTokens:         usage.InputTokens,
					OutputTokens:        usage.OutputTokens,
					CacheReadTokens:     usage.CacheReadTokens,
					CacheCreationTokens: usage.CacheCreationTokens,
				})
			}
		}
		if len(perAgent) > 0 {
			resp.Agents = &perAgent
		}
	}

	return gen.GetQuota200JSONResponse(resp), nil
}

// ListProjects implements gen.StrictServerInterface.
func (h *APIHandler) ListProjects(_ context.Context, _ gen.ListProjectsRequestObject) (gen.ListProjectsResponseObject, error) {
	if h.projects == nil {
		return gen.ListProjects200JSONResponse{}, nil
	}
	projects := h.projects.List()
	result := make(gen.ListProjects200JSONResponse, 0, len(projects))
	for _, p := range projects {
		proj := gen.Project{
			Slug:     p.Slug,
			RepoName: p.RepoName,
			Active:   p.Active,
		}
		if p.OriginRemote != "" {
			proj.OriginRemote = &p.OriginRemote
		}
		result = append(result, proj)
	}
	return result, nil
}

// AddProject implements gen.StrictServerInterface.
func (h *APIHandler) AddProject(ctx context.Context, req gen.AddProjectRequestObject) (gen.AddProjectResponseObject, error) {
	if h.projectMgr == nil {
		return gen.AddProject500JSONResponse{Error: "project management not configured"}, nil
	}
	if req.Body == nil || req.Body.RemoteUrl == "" {
		return gen.AddProject400JSONResponse{Error: "remote_url is required"}, nil
	}

	name := ""
	if req.Body.Name != nil {
		name = *req.Body.Name
	}

	var opts []domain.AddProjectOpts
	if req.Body.SshKey != nil && *req.Body.SshKey != "" {
		opts = append(opts, domain.AddProjectOpts{SSHKeyPath: *req.Body.SshKey})
	}

	result, err := h.projectMgr.AddProject(ctx, req.Body.RemoteUrl, name, opts...)
	if err != nil {
		if errors.Is(err, domain.ErrProjectExists) {
			return gen.AddProject409JSONResponse{Error: err.Error()}, nil
		}
		return gen.AddProject400JSONResponse{Error: err.Error()}, nil
	}

	p := result.Project
	if h.eventBus != nil {
		h.eventBus.Publish(domain.NewProjectUpdateEvent(map[string]any{
			"slug":      p.Slug,
			"repo_name": p.RepoName,
			"active":    p.Active,
		}))
	}
	resp := gen.AddProject201JSONResponse{
		Slug:     p.Slug,
		RepoName: p.RepoName,
		Active:   p.Active,
	}
	if p.OriginRemote != "" {
		resp.OriginRemote = &p.OriginRemote
	}
	return resp, nil
}

// RemoveProject implements gen.StrictServerInterface.
func (h *APIHandler) RemoveProject(ctx context.Context, req gen.RemoveProjectRequestObject) (gen.RemoveProjectResponseObject, error) {
	if h.projectMgr == nil {
		return gen.RemoveProject500JSONResponse{Error: "project management not configured"}, nil
	}

	cleanup := false
	if req.Params.Cleanup != nil {
		cleanup = *req.Params.Cleanup
	}

	err := h.projectMgr.RemoveProject(ctx, req.Slug, cleanup)
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return gen.RemoveProject404JSONResponse{Error: err.Error()}, nil
		}
		return gen.RemoveProject500JSONResponse{Error: err.Error()}, nil
	}
	if h.eventBus != nil {
		h.eventBus.Publish(domain.NewProjectRemovedEvent(req.Slug))
	}
	return gen.RemoveProject204Response{}, nil
}

// ListTracks implements gen.StrictServerInterface.
func (h *APIHandler) ListTracks(_ context.Context, req gen.ListTracksRequestObject) (gen.ListTracksResponseObject, error) {
	if h.projects == nil {
		return gen.ListTracks200JSONResponse{}, nil
	}
	projects := h.projects.List()
	var result gen.ListTracks200JSONResponse
	for _, p := range projects {
		if req.Params.Project != nil && *req.Params.Project != p.Slug {
			continue
		}
		tracks, err := h.trackReader.DiscoverTracks(p.ProjectDir)
		if err != nil {
			continue
		}
		for _, t := range tracks {
			track := gen.Track{
				Id:      t.ID,
				Title:   t.Title,
				Status:  gen.TrackStatus(t.Status),
				Project: &p.Slug,
			}
			if t.DepsCount > 0 {
				track.DepsCount = &t.DepsCount
				track.DepsMet = &t.DepsMet
			}
			if t.ConflictCount > 0 {
				track.ConflictCount = &t.ConflictCount
			}
			result = append(result, track)
		}
	}
	if result == nil {
		result = gen.ListTracks200JSONResponse{}
	}
	return result, nil
}

// GetStatus implements gen.StrictServerInterface.
func (h *APIHandler) GetStatus(_ context.Context, _ gen.GetStatusRequestObject) (gen.GetStatusResponseObject, error) {
	if err := h.agents.Load(); err != nil {
		return gen.GetStatus500JSONResponse{Error: "failed to load state"}, nil
	}
	agents := h.agents.Agents()
	counts := make(map[string]int)
	activeCount := 0
	for _, a := range agents {
		if a.IsActive() {
			counts[a.Status]++
			activeCount++
		}
	}

	sseClients := 0
	if h.sseClients != nil {
		sseClients = h.sseClients()
	}

	resp := gen.StatusInfo{
		GiteaUrl:     h.giteaURL,
		AgentCounts:  counts,
		ActiveAgents: activeCount,
		TotalAgents:  len(agents),
		SseClients:   sseClients,
	}
	if h.quota != nil {
		rl := h.quota.IsRateLimited()
		resp.RateLimited = &rl
		total := h.quota.GetTotalUsage()
		resp.EstimatedCostUsd = &total.TotalCostUSD
	}

	return gen.GetStatus200JSONResponse(resp), nil
}

// ListSSHKeys implements gen.StrictServerInterface.
func (h *APIHandler) ListSSHKeys(_ context.Context, _ gen.ListSSHKeysRequestObject) (gen.ListSSHKeysResponseObject, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return gen.ListSSHKeys200JSONResponse{Keys: []gen.SSHKeyInfo{}}, nil
	}
	sshDir := filepath.Join(home, ".ssh")
	keys := auth.DiscoverSSHKeys(sshDir)
	result := make([]gen.SSHKeyInfo, 0, len(keys))
	for _, k := range keys {
		info := gen.SSHKeyInfo{
			Name: k.Name,
			Path: k.Path,
			Type: k.Type,
		}
		if k.PubContent != "" {
			// Extract comment from pub key content (last field).
			parts := strings.SplitN(k.PubContent, " ", 3)
			if len(parts) >= 3 {
				info.Comment = &parts[2]
			}
		}
		result = append(result, info)
	}
	return gen.ListSSHKeys200JSONResponse{Keys: result}, nil
}

// ListLocks implements gen.StrictServerInterface.
func (h *APIHandler) ListLocks(_ context.Context, _ gen.ListLocksRequestObject) (gen.ListLocksResponseObject, error) {
	locks := h.lockMgr.List()
	result := make(gen.ListLocks200JSONResponse, 0, len(locks))
	for _, l := range locks {
		result = append(result, lockToGen(&l))
	}
	return result, nil
}

// AcquireLock implements gen.StrictServerInterface.
func (h *APIHandler) AcquireLock(ctx context.Context, req gen.AcquireLockRequestObject) (gen.AcquireLockResponseObject, error) {
	if req.Body == nil || req.Body.Holder == "" {
		return gen.AcquireLock400JSONResponse{Error: "holder required"}, nil
	}

	ttlSec := 60
	if req.Body.TtlSeconds != nil && *req.Body.TtlSeconds > 0 {
		ttlSec = *req.Body.TtlSeconds
	}
	ttl := time.Duration(ttlSec) * time.Second

	timeoutSec := 0
	if req.Body.TimeoutSeconds != nil && *req.Body.TimeoutSeconds > 0 {
		timeoutSec = *req.Body.TimeoutSeconds
	}

	// Non-blocking (timeout=0): use an already-cancelled context so Acquire
	// returns immediately if the lock is held. Positive timeout: use a deadline.
	if timeoutSec > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		defer cancel()
	} else {
		// Create an already-cancelled context for truly non-blocking acquire.
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		cancel() // cancel immediately — Acquire will return ErrTimeout at once if lock is held
	}

	l, err := h.lockMgr.Acquire(ctx, req.Scope, req.Body.Holder, ttl)
	if err != nil {
		var currentHolder string
		locks := h.lockMgr.List()
		for _, existing := range locks {
			if existing.Scope == req.Scope {
				currentHolder = existing.Holder
				break
			}
		}
		return gen.AcquireLock409JSONResponse{
			Error:         "timeout waiting for lock",
			CurrentHolder: strPtr(currentHolder),
		}, nil
	}
	if h.eventBus != nil {
		h.eventBus.Publish(domain.NewLockUpdateEvent(map[string]string{
			"scope":      l.Scope,
			"holder":     l.Holder,
			"expires_at": l.ExpiresAt.Format(time.RFC3339),
		}))
	}
	return gen.AcquireLock200JSONResponse(lockToGen(l)), nil
}

// HeartbeatLock implements gen.StrictServerInterface.
func (h *APIHandler) HeartbeatLock(_ context.Context, req gen.HeartbeatLockRequestObject) (gen.HeartbeatLockResponseObject, error) {
	if req.Body == nil || req.Body.Holder == "" {
		return gen.HeartbeatLock400JSONResponse{Error: "holder required"}, nil
	}

	ttlSec := 60
	if req.Body.TtlSeconds != nil && *req.Body.TtlSeconds > 0 {
		ttlSec = *req.Body.TtlSeconds
	}

	l, err := h.lockMgr.Heartbeat(req.Scope, req.Body.Holder, time.Duration(ttlSec)*time.Second)
	if err != nil {
		return gen.HeartbeatLock404JSONResponse{Error: err.Error()}, nil
	}
	if h.eventBus != nil {
		h.eventBus.Publish(domain.NewLockUpdateEvent(map[string]string{
			"scope":      l.Scope,
			"holder":     l.Holder,
			"expires_at": l.ExpiresAt.Format(time.RFC3339),
		}))
	}
	return gen.HeartbeatLock200JSONResponse(lockToGen(l)), nil
}

// ReleaseLock implements gen.StrictServerInterface.
func (h *APIHandler) ReleaseLock(_ context.Context, req gen.ReleaseLockRequestObject) (gen.ReleaseLockResponseObject, error) {
	if req.Body == nil || req.Body.Holder == "" {
		return gen.ReleaseLock400JSONResponse{Error: "holder required"}, nil
	}

	if err := h.lockMgr.Release(req.Scope, req.Body.Holder); err != nil {
		return gen.ReleaseLock404JSONResponse{Error: err.Error()}, nil
	}
	if h.eventBus != nil {
		h.eventBus.Publish(domain.NewLockReleasedEvent(req.Scope))
	}
	return gen.ReleaseLock200JSONResponse{Released: true}, nil
}

// domainAgentToGen converts a domain.AgentInfo to the generated Agent model.
func domainAgentToGen(a domain.AgentInfo, quota QuotaReader) gen.Agent {
	g := gen.Agent{
		Id:        a.ID,
		Role:      gen.AgentRole(a.Role),
		Ref:       a.Ref,
		Status:    gen.AgentStatus(a.Status),
		Pid:       a.PID,
		StartedAt: a.StartedAt,
		UpdatedAt: a.UpdatedAt,
	}
	if a.SessionID != "" {
		g.SessionId = &a.SessionID
	}
	if a.WorktreeDir != "" {
		g.WorktreeDir = &a.WorktreeDir
	}
	if a.LogFile != "" {
		g.LogFile = &a.LogFile
	}
	if a.SuspendedAt != nil {
		g.SuspendedAt = a.SuspendedAt
	}
	if a.FinishedAt != nil {
		g.FinishedAt = a.FinishedAt
	}
	if a.ShutdownReason != "" {
		g.ShutdownReason = &a.ShutdownReason
	}
	if !a.StartedAt.IsZero() {
		uptime := int(time.Since(a.StartedAt).Seconds())
		g.UptimeSeconds = &uptime
	}
	if a.Model != "" {
		g.Model = &a.Model
	}
	if quota != nil {
		if usage := quota.GetAgentUsage(a.ID); usage != nil {
			g.EstimatedCostUsd = &usage.TotalCostUSD
			g.InputTokens = &usage.InputTokens
			g.OutputTokens = &usage.OutputTokens
			g.CacheReadTokens = &usage.CacheReadTokens
			g.CacheCreationTokens = &usage.CacheCreationTokens
		}
	}
	return g
}

// lockToGen converts a lock.Lock to the generated LockInfo model.
func lockToGen(l *lock.Lock) gen.LockInfo {
	remaining := time.Until(l.ExpiresAt).Seconds()
	if remaining < 0 {
		remaining = 0
	}
	return gen.LockInfo{
		Scope:               l.Scope,
		Holder:              l.Holder,
		AcquiredAt:          l.AcquiredAt,
		ExpiresAt:           l.ExpiresAt,
		TtlRemainingSeconds: remaining,
	}
}

// GetSkillsStatus implements gen.StrictServerInterface.
func (h *APIHandler) GetSkillsStatus(_ context.Context, _ gen.GetSkillsStatusRequestObject) (gen.GetSkillsStatusResponseObject, error) {
	if h.cfg == nil {
		return gen.GetSkillsStatus200JSONResponse{
			InstalledVersion: "",
			UpdateAvailable:  false,
			Skills:           []gen.SkillDetail{},
		}, nil
	}

	skillsDir := h.cfg.GetSkillsDir()

	// When no repo is configured, report status of embedded skills.
	if h.cfg.SkillsRepo == "" {
		required := skills.AllRequiredSkills()
		statuses := skills.CheckStatus(required, skillsDir, "")
		embeddedLabel := "embedded"

		resp := gen.SkillsStatus{
			InstalledVersion: embeddedLabel,
			UpdateAvailable:  false,
			Repo:             &embeddedLabel,
			Skills:           make([]gen.SkillDetail, 0, len(statuses)),
		}
		for _, s := range statuses {
			resp.Skills = append(resp.Skills, gen.SkillDetail{
				Name:     s.Name,
				Modified: !s.Current && s.Installed,
			})
		}
		return gen.GetSkillsStatus200JSONResponse(resp), nil
	}

	manifest, _ := skills.LoadManifest()
	installed := skills.ListInstalled(skillsDir, manifest)

	resp := gen.SkillsStatus{
		InstalledVersion: h.cfg.SkillsVersion,
		UpdateAvailable:  false,
		Repo:             &h.cfg.SkillsRepo,
		Skills:           make([]gen.SkillDetail, 0, len(installed)),
	}

	for _, s := range installed {
		detail := gen.SkillDetail{
			Name:     s.Name,
			Modified: s.Modified,
		}
		resp.Skills = append(resp.Skills, detail)
	}

	// Check for available update (non-blocking, best-effort).
	gh := skills.NewGitHubClient()
	rel, err := gh.LatestRelease(h.cfg.SkillsRepo)
	if err == nil {
		resp.AvailableVersion = &rel.TagName
		if h.cfg.SkillsVersion == "" || skills.IsNewer(h.cfg.SkillsVersion, rel.TagName) {
			resp.UpdateAvailable = true
		}
	}

	return gen.GetSkillsStatus200JSONResponse(resp), nil
}

// UpdateSkills implements gen.StrictServerInterface.
func (h *APIHandler) UpdateSkills(_ context.Context, req gen.UpdateSkillsRequestObject) (gen.UpdateSkillsResponseObject, error) {
	if h.cfg == nil {
		return gen.UpdateSkills400JSONResponse{Error: "not initialized"}, nil
	}

	skillsDir := h.cfg.GetSkillsDir()

	// When no repo is configured, re-extract from embedded assets.
	if h.cfg.SkillsRepo == "" {
		installed, err := skills.InstallAllEmbedded(skillsDir)
		if err != nil {
			return gen.UpdateSkills500JSONResponse{Error: fmt.Sprintf("install embedded: %v", err)}, nil
		}
		return gen.UpdateSkills200JSONResponse{
			Version:        "embedded",
			InstalledCount: len(installed),
			Skills:         &installed,
		}, nil
	}

	force := req.Body != nil && req.Body.Force != nil && *req.Body.Force

	// Check latest release.
	gh := skills.NewGitHubClient()
	rel, err := gh.LatestRelease(h.cfg.SkillsRepo)
	if err != nil {
		return gen.UpdateSkills500JSONResponse{Error: fmt.Sprintf("check for updates: %v", err)}, nil
	}

	if h.cfg.SkillsVersion != "" && !skills.IsNewer(h.cfg.SkillsVersion, rel.TagName) {
		return gen.UpdateSkills200JSONResponse{
			Version:        h.cfg.SkillsVersion,
			InstalledCount: 0,
		}, nil
	}

	// Check for modifications unless force.
	if !force {
		manifest, _ := skills.LoadManifest()
		modified := skills.DetectModified(skillsDir, manifest)
		if len(modified) > 0 {
			details := make([]gen.SkillDetail, 0, len(modified))
			for _, m := range modified {
				details = append(details, gen.SkillDetail{
					Name:         m.Name,
					Modified:     true,
					ChangedFiles: &m.Files,
				})
			}
			return gen.UpdateSkills409JSONResponse{
				Error:    fmt.Sprintf("%d skill(s) have local modifications", len(modified)),
				Modified: details,
			}, nil
		}
	}

	// Install.
	inst := skills.NewInstaller()
	result, err := inst.Install(rel.TarballURL, skillsDir)
	if err != nil {
		return gen.UpdateSkills500JSONResponse{Error: fmt.Sprintf("install: %v", err)}, nil
	}

	// Update manifest.
	checksums, _ := skills.ComputeChecksums(skillsDir)
	newManifest := &skills.Manifest{Version: rel.TagName, Checksums: checksums}
	_ = newManifest.Save()

	// Update config.
	h.cfg.SkillsVersion = rel.TagName
	_ = h.cfg.Save()

	names := make([]string, 0, len(result))
	for _, s := range result {
		names = append(names, s.Name)
	}

	return gen.UpdateSkills200JSONResponse{
		Version:        rel.TagName,
		InstalledCount: len(result),
		Skills:         &names,
	}, nil
}

// ListTraces implements gen.StrictServerInterface.
func (h *APIHandler) ListTraces(_ context.Context, req gen.ListTracesRequestObject) (gen.ListTracesResponseObject, error) {
	if h.traceStore == nil {
		return gen.ListTraces200JSONResponse{}, nil
	}

	var traces []tracing.TraceSummary
	switch {
	case req.Params.TrackId != nil && *req.Params.TrackId != "":
		traces = h.traceStore.FindByTrackID(*req.Params.TrackId)
	case req.Params.SessionId != nil && *req.Params.SessionId != "":
		traces = h.traceStore.FindBySessionID(*req.Params.SessionId)
	default:
		traces = h.traceStore.ListTraces()
	}

	result := make(gen.ListTraces200JSONResponse, 0, len(traces))
	for _, t := range traces {
		result = append(result, gen.TraceSummary{
			TraceId:   t.TraceID,
			RootName:  t.RootName,
			SpanCount: t.SpanCount,
			StartTime: t.StartTime,
			EndTime:   t.EndTime,
		})
	}
	return result, nil
}

// GetTrace implements gen.StrictServerInterface.
func (h *APIHandler) GetTrace(_ context.Context, req gen.GetTraceRequestObject) (gen.GetTraceResponseObject, error) {
	if h.traceStore == nil {
		return gen.GetTrace404JSONResponse{Error: "tracing not enabled"}, nil
	}
	spans := h.traceStore.GetTrace(req.TraceId)
	if len(spans) == 0 {
		return gen.GetTrace404JSONResponse{Error: "trace not found"}, nil
	}

	genSpans := make([]gen.SpanInfo, 0, len(spans))
	for _, sp := range spans {
		s := gen.SpanInfo{
			TraceId:    sp.TraceID,
			SpanId:     sp.SpanID,
			Name:       sp.Name,
			StartTime:  sp.StartTime,
			EndTime:    sp.EndTime,
			DurationMs: sp.DurationMs,
			Status:     sp.Status,
		}
		if sp.ParentID != "" {
			s.ParentId = &sp.ParentID
		}
		if len(sp.Attributes) > 0 {
			attrs := map[string]string{}
			for k, v := range sp.Attributes {
				attrs[k] = v
			}
			s.Attributes = &attrs
		}
		if len(sp.Events) > 0 {
			events := make([]gen.SpanEvent, 0, len(sp.Events))
			for _, ev := range sp.Events {
				e := gen.SpanEvent{
					Name:      ev.Name,
					Timestamp: ev.Timestamp,
				}
				if len(ev.Attributes) > 0 {
					ea := map[string]string{}
					for k, v := range ev.Attributes {
						ea[k] = v
					}
					e.Attributes = &ea
				}
				events = append(events, e)
			}
			s.Events = &events
		}
		genSpans = append(genSpans, s)
	}

	return gen.GetTrace200JSONResponse{
		TraceId: req.TraceId,
		Spans:   genSpans,
	}, nil
}

// GetBoard implements gen.StrictServerInterface.
func (h *APIHandler) GetBoard(_ context.Context, req gen.GetBoardRequestObject) (gen.GetBoardResponseObject, error) {
	if h.boardSvc == nil {
		return gen.GetBoard500JSONResponse{Error: "board service not configured"}, nil
	}
	board, err := h.boardSvc.GetBoard(req.Project)
	if err != nil {
		return gen.GetBoard500JSONResponse{Error: err.Error()}, nil
	}

	// Auto-sync if board is empty (first load or after reset).
	if len(board.Cards) == 0 {
		if proj, ok := h.findProject(req.Project); ok {
			tracks, discoverErr := h.trackReader.DiscoverTracks(proj.ProjectDir)
			if discoverErr == nil && len(tracks) > 0 {
				result, syncErr := h.boardSvc.SyncFromTracks(req.Project, tracks, nil)
				if syncErr == nil && (result.Created > 0 || result.Updated > 0) {
					board, _ = h.boardSvc.GetBoard(req.Project)
					if h.eventBus != nil {
						h.eventBus.Publish(domain.NewBoardUpdateEvent(board))
					}
				}
			}
		}
	}

	return gen.GetBoard200JSONResponse(domainBoardToGen(board)), nil
}

// MoveCard implements gen.StrictServerInterface.
func (h *APIHandler) MoveCard(_ context.Context, req gen.MoveCardRequestObject) (gen.MoveCardResponseObject, error) {
	if h.boardSvc == nil {
		return gen.MoveCard500JSONResponse{Error: "board service not configured"}, nil
	}
	if req.Body == nil {
		return gen.MoveCard400JSONResponse{Error: "request body required"}, nil
	}
	result, err := h.boardSvc.MoveCard(req.Project, req.Body.TrackId, string(req.Body.ToColumn))
	if err != nil {
		return gen.MoveCard400JSONResponse{Error: err.Error()}, nil
	}
	if h.eventBus != nil {
		h.eventBus.Publish(domain.NewBoardUpdateEvent(map[string]string{
			"track_id":    result.TrackID,
			"from_column": result.FromColumn,
			"to_column":   result.ToColumn,
		}))
	}
	return gen.MoveCard200JSONResponse{
		TrackId:    result.TrackID,
		FromColumn: result.FromColumn,
		ToColumn:   result.ToColumn,
	}, nil
}

// SyncBoard implements gen.StrictServerInterface.
func (h *APIHandler) SyncBoard(_ context.Context, req gen.SyncBoardRequestObject) (gen.SyncBoardResponseObject, error) {
	if h.boardSvc == nil {
		return gen.SyncBoard500JSONResponse{Error: "board service not configured"}, nil
	}
	if h.projects == nil {
		return gen.SyncBoard400JSONResponse{Error: "no projects configured"}, nil
	}

	// Find project.
	var projectDir string
	for _, p := range h.projects.List() {
		if p.Slug == req.Project {
			projectDir = p.ProjectDir
			break
		}
	}
	if projectDir == "" {
		return gen.SyncBoard400JSONResponse{Error: fmt.Sprintf("project %q not found", req.Project)}, nil
	}

	tracks, err := h.trackReader.DiscoverTracks(projectDir)
	if err != nil {
		return gen.SyncBoard500JSONResponse{Error: fmt.Sprintf("discover tracks: %v", err)}, nil
	}

	result, err := h.boardSvc.SyncFromTracks(req.Project, tracks, nil)
	if err != nil {
		return gen.SyncBoard500JSONResponse{Error: err.Error()}, nil
	}
	if h.eventBus != nil && (result.Created > 0 || result.Updated > 0) {
		board, boardErr := h.boardSvc.GetBoard(req.Project)
		if boardErr == nil {
			h.eventBus.Publish(domain.NewBoardUpdateEvent(board))
		}
	}
	return gen.SyncBoard200JSONResponse{
		Created:   result.Created,
		Updated:   result.Updated,
		Unchanged: result.Unchanged,
	}, nil
}

// domainBoardToGen converts a domain.BoardState to the generated BoardState.
func domainBoardToGen(b *domain.BoardState) gen.BoardState {
	cards := make(map[string]gen.BoardCard, len(b.Cards))
	for id, c := range b.Cards {
		card := gen.BoardCard{
			TrackId:   c.TrackID,
			Title:     c.Title,
			Column:    gen.BoardCardColumn(c.Column),
			Position:  c.Position,
			MovedAt:   c.MovedAt,
			CreatedAt: c.CreatedAt,
		}
		if c.Type != "" {
			card.Type = &c.Type
		}
		if c.AgentID != "" {
			card.AgentId = &c.AgentID
		}
		if c.AgentStatus != "" {
			card.AgentStatus = &c.AgentStatus
		}
		if c.AssignedWorker != "" {
			card.AssignedWorker = &c.AssignedWorker
		}
		if c.PRNumber != 0 {
			card.PrNumber = &c.PRNumber
		}
		cards[id] = card
	}
	return gen.BoardState{
		Columns: b.Columns,
		Cards:   cards,
	}
}

// PushProject implements gen.StrictServerInterface.
func (h *APIHandler) PushProject(ctx context.Context, req gen.PushProjectRequestObject) (gen.PushProjectResponseObject, error) {
	if h.gitSync == nil {
		return gen.PushProject500JSONResponse{Error: "git sync not configured"}, nil
	}
	p, ok := h.findProject(req.Slug)
	if !ok {
		return gen.PushProject404JSONResponse{Error: fmt.Sprintf("project %q not found", req.Slug)}, nil
	}
	if req.Body == nil || req.Body.RemoteBranch == "" {
		return gen.PushProject400JSONResponse{Error: "remote_branch is required"}, nil
	}

	result, err := h.gitSync.PushToRemote(ctx, p.ProjectDir, "main", req.Body.RemoteBranch, p.SSHKeyPath)
	if err != nil {
		return gen.PushProject500JSONResponse{Error: err.Error()}, nil
	}

	if h.eventBus != nil {
		h.eventBus.Publish(domain.NewProjectUpdateEvent(map[string]any{
			"slug":   p.Slug,
			"action": "push",
		}))
	}

	return gen.PushProject200JSONResponse{
		Success:      result.Success,
		LocalBranch:  result.LocalBranch,
		RemoteBranch: result.RemoteBranch,
	}, nil
}

// PullProject implements gen.StrictServerInterface.
func (h *APIHandler) PullProject(ctx context.Context, req gen.PullProjectRequestObject) (gen.PullProjectResponseObject, error) {
	if h.gitSync == nil {
		return gen.PullProject500JSONResponse{Error: "git sync not configured"}, nil
	}
	p, ok := h.findProject(req.Slug)
	if !ok {
		return gen.PullProject404JSONResponse{Error: fmt.Sprintf("project %q not found", req.Slug)}, nil
	}

	remoteBranch := "main"
	if req.Body != nil && req.Body.RemoteBranch != nil && *req.Body.RemoteBranch != "" {
		remoteBranch = *req.Body.RemoteBranch
	}

	result, err := h.gitSync.PullFromRemote(ctx, p.ProjectDir, remoteBranch, p.SSHKeyPath)
	if err != nil {
		if strings.Contains(err.Error(), "diverged") {
			return gen.PullProject409JSONResponse{Error: err.Error()}, nil
		}
		return gen.PullProject500JSONResponse{Error: err.Error()}, nil
	}

	if h.eventBus != nil {
		h.eventBus.Publish(domain.NewProjectUpdateEvent(map[string]any{
			"slug":   p.Slug,
			"action": "pull",
		}))
	}

	return gen.PullProject200JSONResponse{
		Success: result.Success,
		NewHead: result.NewHead,
	}, nil
}

// GetSyncStatus implements gen.StrictServerInterface.
func (h *APIHandler) GetSyncStatus(ctx context.Context, req gen.GetSyncStatusRequestObject) (gen.GetSyncStatusResponseObject, error) {
	if h.gitSync == nil {
		return gen.GetSyncStatus500JSONResponse{Error: "git sync not configured"}, nil
	}
	p, ok := h.findProject(req.Slug)
	if !ok {
		return gen.GetSyncStatus404JSONResponse{Error: fmt.Sprintf("project %q not found", req.Slug)}, nil
	}

	status, err := h.gitSync.SyncStatus(ctx, p.ProjectDir, p.SSHKeyPath)
	if err != nil {
		return gen.GetSyncStatus500JSONResponse{Error: err.Error()}, nil
	}

	resp := gen.GetSyncStatus200JSONResponse{
		LocalBranch: status.LocalBranch,
		Ahead:       status.Ahead,
		Behind:      status.Behind,
		Status:      gen.SyncStatusResponseStatus(status.Status),
	}
	if status.RemoteURL != "" {
		resp.RemoteUrl = &status.RemoteURL
	}
	return resp, nil
}

// GetProjectDiff implements gen.StrictServerInterface.
func (h *APIHandler) GetProjectDiff(ctx context.Context, req gen.GetProjectDiffRequestObject) (gen.GetProjectDiffResponseObject, error) {
	if h.diffProv == nil {
		return gen.GetProjectDiff500JSONResponse{Error: "diff provider not configured"}, nil
	}
	p, ok := h.findProject(req.Slug)
	if !ok {
		return gen.GetProjectDiff404JSONResponse{Error: fmt.Sprintf("project %q not found", req.Slug)}, nil
	}

	maxFiles := 100
	if req.Params.MaxFiles != nil {
		maxFiles = *req.Params.MaxFiles
	}

	result, err := h.diffProv.DiffWithMaxFiles(ctx, p.ProjectDir, req.Params.Branch, maxFiles)
	if err != nil {
		if _, ok := err.(*gitadapter.BranchNotFoundError); ok {
			return gen.GetProjectDiff404JSONResponse{Error: err.Error()}, nil
		}
		return gen.GetProjectDiff500JSONResponse{Error: err.Error()}, nil
	}

	resp := gen.GetProjectDiff200JSONResponse{
		Branch: result.Branch,
		Base:   result.Base,
		Stats: gen.DiffStats{
			FilesChanged: result.Stats.FilesChanged,
			Insertions:   result.Stats.Insertions,
			Deletions:    result.Stats.Deletions,
		},
		Files: make([]gen.FileDiff, len(result.Files)),
	}
	if result.Truncated {
		t := true
		resp.Truncated = &t
	}

	for i, f := range result.Files {
		gf := gen.FileDiff{
			Path:       f.Path,
			Status:     gen.FileDiffStatus(f.Status),
			Insertions: f.Insertions,
			Deletions:  f.Deletions,
			IsBinary:   f.IsBinary,
			Hunks:      make([]gen.DiffHunk, len(f.Hunks)),
		}
		if f.OldPath != "" {
			gf.OldPath = &f.OldPath
		}
		for j, hunk := range f.Hunks {
			gh := gen.DiffHunk{
				OldStart: hunk.OldStart,
				OldLines: hunk.OldLines,
				NewStart: hunk.NewStart,
				NewLines: hunk.NewLines,
				Header:   hunk.Header,
				Lines:    make([]gen.DiffLine, len(hunk.Lines)),
			}
			for k, line := range hunk.Lines {
				gh.Lines[k] = gen.DiffLine{
					Type:    gen.DiffLineType(line.Type),
					Content: line.Content,
					OldNo:   line.OldNo,
					NewNo:   line.NewNo,
				}
			}
			gf.Hunks[j] = gh
		}
		resp.Files[i] = gf
	}

	return resp, nil
}

// GetProjectBranches implements gen.StrictServerInterface.
func (h *APIHandler) GetProjectBranches(_ context.Context, req gen.GetProjectBranchesRequestObject) (gen.GetProjectBranchesResponseObject, error) {
	_, ok := h.findProject(req.Slug)
	if !ok {
		return gen.GetProjectBranches404JSONResponse{Error: fmt.Sprintf("project %q not found", req.Slug)}, nil
	}

	// Build branch info from active agents that have worktree directories.
	var branches []gen.BranchInfo
	if h.agents != nil {
		for _, a := range h.agents.Agents() {
			if a.WorktreeDir == "" {
				continue
			}
			branches = append(branches, gen.BranchInfo{
				Branch:  a.Ref,
				AgentId: &a.ID,
				TrackId: &a.Ref,
				Status:  a.Status,
			})
		}
	}
	if branches == nil {
		branches = []gen.BranchInfo{}
	}
	return gen.GetProjectBranches200JSONResponse(branches), nil
}

// findProject looks up a project by slug from the projects list.
func (h *APIHandler) findProject(slug string) (domain.Project, bool) {
	if h.projects == nil {
		return domain.Project{}, false
	}
	for _, p := range h.projects.List() {
		if p.Slug == slug {
			return p, true
		}
	}
	return domain.Project{}, false
}

// GenerateTracks implements gen.StrictServerInterface.
func (h *APIHandler) GenerateTracks(ctx context.Context, req gen.GenerateTracksRequestObject) (gen.GenerateTracksResponseObject, error) {
	if h.interSpawner == nil || h.wsSessions == nil {
		return gen.GenerateTracks500JSONResponse{Error: "interactive agents not configured"}, nil
	}
	if req.Body == nil || req.Body.Prompt == "" {
		return gen.GenerateTracks500JSONResponse{Error: "prompt is required"}, nil
	}

	// Check Claude CLI authentication.
	if msg := h.checkClaudeAuth(ctx); msg != "" {
		return gen.GenerateTracks401JSONResponse{Error: msg}, nil
	}

	// Check agent permissions consent.
	if msg := h.checkConsent(); msg != "" {
		return gen.GenerateTracks403JSONResponse{Error: msg}, nil
	}

	// Validate required skills for track generation.
	if resp := h.checkSkillsForRole("interactive", ""); resp != nil {
		return gen.GenerateTracks412JSONResponse(*resp), nil
	}

	// Resolve project working directory.
	var workDir string
	var projectSlug string
	if req.Body.Project != nil && *req.Body.Project != "" {
		projectSlug = *req.Body.Project
		if h.projects != nil {
			for _, p := range h.projects.List() {
				if p.Slug == projectSlug {
					workDir = p.ProjectDir
					break
				}
			}
		}
		if workDir == "" {
			return gen.GenerateTracks500JSONResponse{Error: fmt.Sprintf("project %q not found", projectSlug)}, nil
		}
	}

	// Check kiloforge setup is complete before allowing track generation.
	if slug := h.checkSetup(projectSlug); slug != "" {
		return gen.GenerateTracks428JSONResponse{
			Error:   "kiloforge setup required",
			Project: slug,
		}, nil
	}

	fullPrompt := fmt.Sprintf("/kf-architect I would like to generate one or more tracks and the specifications are the following: %s", req.Body.Prompt)

	opts := agent.SpawnInteractiveOpts{
		WorkDir: workDir,
		Prompt:  fullPrompt,
		Ref:     "track-gen",
	}

	ia, err := h.interSpawner.SpawnInteractive(ctx, opts)
	if err != nil {
		if strings.Contains(err.Error(), "rate limited") {
			return gen.GenerateTracks429JSONResponse{Error: err.Error()}, nil
		}
		return gen.GenerateTracks500JSONResponse{Error: err.Error()}, nil
	}

	// Create bridge and register with WS session manager.
	bridge := wsAdapter.NewSDKBridge(ia.Info.ID, ia.Stdin, ia.Done)
	h.wsSessions.RegisterBridge(ia.Info.ID, bridge)

	// Start structured message relay in background with cancellable context.
	relayCtx, cancelRelay := context.WithCancel(context.Background())
	ia.SetCancelRelay(cancelRelay)
	go h.wsSessions.StartStructuredRelay(relayCtx, ia.Info.ID, ia.Output)

	// Auto-sync board when track-gen agent completes.
	if projectSlug != "" && h.boardSvc != nil {
		go func() {
			<-ia.Done
			projectDir := workDir
			tracks, err := h.trackReader.DiscoverTracks(projectDir)
			if err != nil {
				return
			}
			result, err := h.boardSvc.SyncFromTracks(projectSlug, tracks, nil)
			if err != nil {
				return
			}
			if h.eventBus != nil && (result.Created > 0 || result.Updated > 0) {
				board, boardErr := h.boardSvc.GetBoard(projectSlug)
				if boardErr == nil {
					h.eventBus.Publish(domain.NewBoardUpdateEvent(board))
				}
			}
		}()
	}

	wsURL := fmt.Sprintf("/ws/agent/%s", ia.Info.ID)

	return gen.GenerateTracks201JSONResponse{
		AgentId: ia.Info.ID,
		WsUrl:   wsURL,
	}, nil
}

// adminSkillMap maps operation names to skill prompts.
var adminSkillMap = map[gen.AdminOperationRequestOperation]string{
	gen.BulkArchive:    "/kf-bulk-archive",
	gen.CompactArchive: "/kf-compact-archive",
	gen.Report:         "/kf-report",
}

// adminArchiveOps are operations that modify tracks and should trigger board sync.
var adminArchiveOps = map[gen.AdminOperationRequestOperation]bool{
	gen.BulkArchive:    true,
	gen.CompactArchive: true,
}

// RunAdminOperation implements gen.StrictServerInterface.
func (h *APIHandler) RunAdminOperation(ctx context.Context, req gen.RunAdminOperationRequestObject) (gen.RunAdminOperationResponseObject, error) {
	if h.interSpawner == nil || h.wsSessions == nil {
		return gen.RunAdminOperation500JSONResponse{Error: "interactive agents not configured"}, nil
	}
	if req.Body == nil {
		return gen.RunAdminOperation500JSONResponse{Error: "request body required"}, nil
	}

	// Validate operation.
	skillPrompt, ok := adminSkillMap[req.Body.Operation]
	if !ok {
		return gen.RunAdminOperation500JSONResponse{Error: fmt.Sprintf("unknown operation: %s", req.Body.Operation)}, nil
	}

	// Check Claude CLI authentication.
	if msg := h.checkClaudeAuth(ctx); msg != "" {
		return gen.RunAdminOperation401JSONResponse{Error: msg}, nil
	}

	// Check agent permissions consent.
	if msg := h.checkConsent(); msg != "" {
		return gen.RunAdminOperation403JSONResponse{Error: msg}, nil
	}

	// Validate required skills.
	if resp := h.checkSkillsForRole("interactive", ""); resp != nil {
		return gen.RunAdminOperation412JSONResponse{Error: resp.Error}, nil
	}

	// Check kiloforge setup if a project is specified.
	if req.Body.Project != nil && *req.Body.Project != "" {
		if slug := h.checkSetup(*req.Body.Project); slug != "" {
			return gen.RunAdminOperation428JSONResponse{
				Error:   "kiloforge setup required",
				Project: slug,
			}, nil
		}
	}

	// Concurrency guard — only one admin operation at a time.
	h.adminMu.Lock()
	if h.runningAdminAgent != "" {
		h.adminMu.Unlock()
		return gen.RunAdminOperation412JSONResponse{Error: fmt.Sprintf("admin operation already running (agent %s)", h.runningAdminAgent)}, nil
	}

	// Resolve project working directory.
	var workDir string
	var projectSlug string
	if req.Body.Project != nil && *req.Body.Project != "" {
		projectSlug = *req.Body.Project
		if h.projects != nil {
			for _, p := range h.projects.List() {
				if p.Slug == projectSlug {
					workDir = p.ProjectDir
					break
				}
			}
		}
		if workDir == "" {
			h.adminMu.Unlock()
			return gen.RunAdminOperation500JSONResponse{Error: fmt.Sprintf("project %q not found", projectSlug)}, nil
		}
	}

	opts := agent.SpawnInteractiveOpts{
		WorkDir: workDir,
		Prompt:  skillPrompt,
		Ref:     "admin",
	}

	ia, err := h.interSpawner.SpawnInteractive(ctx, opts)
	if err != nil {
		h.adminMu.Unlock()
		if strings.Contains(err.Error(), "rate limited") {
			return gen.RunAdminOperation429JSONResponse{Error: err.Error()}, nil
		}
		return gen.RunAdminOperation500JSONResponse{Error: err.Error()}, nil
	}

	h.runningAdminAgent = ia.Info.ID
	h.adminMu.Unlock()

	// Create bridge and register with WS session manager.
	bridge := wsAdapter.NewSDKBridge(ia.Info.ID, ia.Stdin, ia.Done)
	h.wsSessions.RegisterBridge(ia.Info.ID, bridge)

	// Start structured message relay in background with cancellable context.
	relayCtx, cancelRelay := context.WithCancel(context.Background())
	ia.SetCancelRelay(cancelRelay)
	go h.wsSessions.StartStructuredRelay(relayCtx, ia.Info.ID, ia.Output)

	// Clear concurrency guard and auto-sync board on completion.
	go func() {
		<-ia.Done

		h.adminMu.Lock()
		h.runningAdminAgent = ""
		h.adminMu.Unlock()

		// Auto-sync board after archive operations.
		if adminArchiveOps[req.Body.Operation] && projectSlug != "" && h.boardSvc != nil {
			tracks, err := h.trackReader.DiscoverTracks(workDir)
			if err != nil {
				return
			}
			result, err := h.boardSvc.SyncFromTracks(projectSlug, tracks, nil)
			if err != nil {
				return
			}
			if h.eventBus != nil && (result.Created > 0 || result.Updated > 0) {
				board, boardErr := h.boardSvc.GetBoard(projectSlug)
				if boardErr == nil {
					h.eventBus.Publish(domain.NewBoardUpdateEvent(board))
				}
			}
		}
	}()

	wsURL := fmt.Sprintf("/ws/agent/%s", ia.Info.ID)

	return gen.RunAdminOperation201JSONResponse{
		AgentId: ia.Info.ID,
		WsUrl:   wsURL,
	}, nil
}

// GetTrackDetail implements gen.StrictServerInterface.
func (h *APIHandler) GetTrackDetail(_ context.Context, req gen.GetTrackDetailRequestObject) (gen.GetTrackDetailResponseObject, error) {
	if h.projects == nil {
		return gen.GetTrackDetail404JSONResponse{Error: "no projects configured"}, nil
	}

	projectSlug := req.Params.Project
	var projectDir string
	for _, p := range h.projects.List() {
		if p.Slug == projectSlug {
			projectDir = p.ProjectDir
			break
		}
	}
	if projectDir == "" {
		return gen.GetTrackDetail404JSONResponse{Error: fmt.Sprintf("project %q not found", projectSlug)}, nil
	}

	detail, err := h.trackReader.GetTrackDetail(projectDir, req.TrackId)
	if err != nil {
		return gen.GetTrackDetail404JSONResponse{Error: err.Error()}, nil
	}

	resp := gen.TrackDetail{
		Id:     detail.ID,
		Title:  detail.Title,
		Status: detail.Status,
	}
	if detail.Type != "" {
		resp.Type = &detail.Type
	}
	if detail.Spec != "" {
		resp.Spec = &detail.Spec
	}
	if detail.Plan != "" {
		resp.Plan = &detail.Plan
	}
	if detail.Phases.Total > 0 {
		resp.PhasesTotal = &detail.Phases.Total
		resp.PhasesCompleted = &detail.Phases.Completed
	}
	if detail.Tasks.Total > 0 {
		resp.TasksTotal = &detail.Tasks.Total
		resp.TasksCompleted = &detail.Tasks.Completed
	}
	if detail.CreatedAt != "" {
		resp.CreatedAt = &detail.CreatedAt
	}
	if detail.UpdatedAt != "" {
		resp.UpdatedAt = &detail.UpdatedAt
	}
	if len(detail.Dependencies) > 0 {
		deps := make([]gen.TrackDependency, len(detail.Dependencies))
		for i, d := range detail.Dependencies {
			deps[i] = gen.TrackDependency{
				Id:     d.ID,
				Title:  &d.Title,
				Status: gen.TrackDependencyStatus(d.Status),
			}
		}
		resp.Dependencies = &deps
	}
	if len(detail.Conflicts) > 0 {
		conflicts := make([]gen.TrackConflict, len(detail.Conflicts))
		for i, c := range detail.Conflicts {
			conflicts[i] = gen.TrackConflict{
				TrackId:    c.TrackID,
				TrackTitle: &c.TrackTitle,
				Risk:       gen.TrackConflictRisk(c.Risk),
				Note:       &c.Note,
			}
		}
		resp.Conflicts = &conflicts
	}

	return gen.GetTrackDetail200JSONResponse(resp), nil
}

// DeleteTrack implements gen.StrictServerInterface.
func (h *APIHandler) DeleteTrack(_ context.Context, req gen.DeleteTrackRequestObject) (gen.DeleteTrackResponseObject, error) {
	trackID := req.TrackId

	// Find the project to locate track artifacts.
	var projectSlug string
	var projectDir string
	if req.Params.Project != nil && *req.Params.Project != "" {
		projectSlug = *req.Params.Project
	}

	if projectSlug != "" && h.projects != nil {
		for _, p := range h.projects.List() {
			if p.Slug == projectSlug {
				projectDir = p.ProjectDir
				break
			}
		}
	} else if h.projects != nil && h.trackReader != nil {
		// Try to find the track across all projects using SDK.
		for _, p := range h.projects.List() {
			tracks, err := h.trackReader.DiscoverTracks(p.ProjectDir)
			if err != nil {
				continue
			}
			for _, t := range tracks {
				if t.ID == trackID {
					projectSlug = p.Slug
					projectDir = p.ProjectDir
					break
				}
			}
			if projectDir != "" {
				break
			}
		}
	}

	if projectDir == "" {
		return gen.DeleteTrack404JSONResponse{Error: fmt.Sprintf("track %q not found in any project", trackID)}, nil
	}

	// Remove track via SDK (deletes track dir, removes from registry, prunes deps/conflicts).
	if h.trackReader != nil {
		if err := h.trackReader.RemoveTrack(projectDir, trackID); err != nil {
			return gen.DeleteTrack500JSONResponse{Error: fmt.Sprintf("remove track: %v", err)}, nil
		}
	}

	// Remove board card if board service is available.
	if h.boardSvc != nil && projectSlug != "" {
		if removed, err := h.boardSvc.RemoveCard(projectSlug, trackID); err == nil && removed {
			if h.eventBus != nil {
				board, boardErr := h.boardSvc.GetBoard(projectSlug)
				if boardErr == nil {
					h.eventBus.Publish(domain.NewBoardUpdateEvent(board))
				}
			}
		}
	}

	return gen.DeleteTrack204Response{}, nil
}

// checkSkillsForRole validates required skills for a role and returns a
// SkillsMissingResponse if any are missing, or nil if all are present.
func (h *APIHandler) checkSkillsForRole(role, workDir string) *gen.SkillsMissingResponse {
	if h.cfg == nil {
		return nil
	}
	required := skills.RequiredSkillsForRole(role)
	if len(required) == 0 {
		return nil
	}

	globalDir := h.cfg.GetSkillsDir()
	localDir := ""
	if workDir != "" {
		localDir = filepath.Join(workDir, ".claude", "skills")
	}

	missing := skills.CheckRequired(required, globalDir, localDir)
	if len(missing) == 0 {
		return nil
	}

	missingItems := make([]struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
	}, len(missing))
	for i, m := range missing {
		missingItems[i].Name = m.Name
		missingItems[i].Reason = m.Reason
	}

	return &gen.SkillsMissingResponse{
		Error:         fmt.Sprintf("required skills not installed: %d missing", len(missing)),
		MissingSkills: missingItems,
	}
}

// GetProjectMetadata implements gen.StrictServerInterface.
func (h *APIHandler) GetProjectMetadata(_ context.Context, req gen.GetProjectMetadataRequestObject) (gen.GetProjectMetadataResponseObject, error) {
	proj, ok := h.findProject(req.Slug)
	if !ok {
		return gen.GetProjectMetadata404JSONResponse{Error: "project not found"}, nil
	}

	kfClient := kf.NewClientFromProject(proj.ProjectDir)
	if !kfClient.IsInitialized() {
		return gen.GetProjectMetadata404JSONResponse{Error: "kiloforge not initialized for this project"}, nil
	}

	info, err := kfClient.GetProjectInfo()
	if err != nil {
		return gen.GetProjectMetadata404JSONResponse{Error: fmt.Sprintf("failed to read project metadata: %v", err)}, nil
	}

	trackSummary, err := kfClient.GetTrackSummary()
	if err != nil {
		trackSummary = &kf.TrackSummary{}
	}

	resp := gen.GetProjectMetadata200JSONResponse{
		Product:   info.Product,
		TechStack: info.TechStack,
		TrackSummary: gen.TrackSummary{
			Total:      trackSummary.Total,
			Pending:    trackSummary.Pending,
			InProgress: trackSummary.InProgress,
			Completed:  trackSummary.Completed,
			Archived:   trackSummary.Archived,
		},
	}

	if info.ProductGuidelines != "" {
		resp.ProductGuidelines = &info.ProductGuidelines
	}
	if info.Workflow != "" {
		resp.Workflow = &info.Workflow
	}

	for _, link := range info.QuickLinks {
		resp.QuickLinks = append(resp.QuickLinks, gen.QuickLink{
			Label: link.Label,
			Path:  link.Path,
		})
	}
	if resp.QuickLinks == nil {
		resp.QuickLinks = []gen.QuickLink{}
	}

	if len(info.StyleGuides) > 0 {
		guides := make([]gen.StyleGuideEntry, len(info.StyleGuides))
		for i, sg := range info.StyleGuides {
			guides[i] = gen.StyleGuideEntry{Name: sg.Name, Content: sg.Content}
		}
		resp.StyleGuides = &guides
	}

	return resp, nil
}

// GetProjectSetupStatus implements gen.StrictServerInterface.
func (h *APIHandler) GetProjectSetupStatus(_ context.Context, req gen.GetProjectSetupStatusRequestObject) (gen.GetProjectSetupStatusResponseObject, error) {
	proj, ok := h.findProject(req.Slug)
	if !ok {
		return gen.GetProjectSetupStatus404JSONResponse{Error: "project not found"}, nil
	}
	initialized := h.trackReader != nil && h.trackReader.IsInitialized(proj.ProjectDir)
	return gen.GetProjectSetupStatus200JSONResponse{
		SetupComplete: initialized,
		ProjectSlug:   req.Slug,
	}, nil
}

// StartProjectSetup implements gen.StrictServerInterface.
func (h *APIHandler) StartProjectSetup(ctx context.Context, req gen.StartProjectSetupRequestObject) (gen.StartProjectSetupResponseObject, error) {
	if h.interSpawner == nil || h.wsSessions == nil {
		return gen.StartProjectSetup500JSONResponse{Error: "interactive agents not configured"}, nil
	}

	// Check Claude CLI authentication.
	if msg := h.checkClaudeAuth(ctx); msg != "" {
		return gen.StartProjectSetup401JSONResponse{Error: msg}, nil
	}

	// Check agent permissions consent.
	if msg := h.checkConsent(); msg != "" {
		return gen.StartProjectSetup403JSONResponse{Error: msg}, nil
	}

	proj, ok := h.findProject(req.Slug)
	if !ok {
		return gen.StartProjectSetup404JSONResponse{Error: "project not found"}, nil
	}

	// Validate required skills (setup needs kf-setup specifically).
	if resp := h.checkSkillsForRole("setup", proj.ProjectDir); resp != nil {
		return gen.StartProjectSetup412JSONResponse(*resp), nil
	}

	opts := agent.SpawnInteractiveOpts{
		WorkDir: proj.ProjectDir,
		Prompt:  "/kf-setup",
		Ref:     "setup",
	}

	ia, err := h.interSpawner.SpawnInteractive(ctx, opts)
	if err != nil {
		return gen.StartProjectSetup500JSONResponse{Error: err.Error()}, nil
	}

	// Create bridge and register with WS session manager.
	bridge := wsAdapter.NewSDKBridge(ia.Info.ID, ia.Stdin, ia.Done)
	h.wsSessions.RegisterBridge(ia.Info.ID, bridge)

	// Start structured message relay in background with cancellable context.
	relayCtx, cancelRelay := context.WithCancel(context.Background())
	ia.SetCancelRelay(cancelRelay)
	go h.wsSessions.StartStructuredRelay(relayCtx, ia.Info.ID, ia.Output)

	wsURL := fmt.Sprintf("/ws/agent/%s", ia.Info.ID)

	return gen.StartProjectSetup201JSONResponse{
		AgentId: ia.Info.ID,
		WsUrl:   wsURL,
	}, nil
}

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }

