package rest

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/auth"
	"kiloforge/internal/adapter/config"
	gitadapter "kiloforge/internal/adapter/git"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/adapter/skills"
	"kiloforge/internal/adapter/tracing"
	wsAdapter "kiloforge/internal/adapter/ws"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
	"kiloforge/internal/core/service"
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
	AddProject(ctx context.Context, remoteURL, name string, opts ...service.AddProjectOpts) (*service.AddProjectResult, error)
	RemoveProject(ctx context.Context, slug string, cleanup bool) error
}

// InteractiveSpawner creates interactive agent sessions.
type InteractiveSpawner interface {
	SpawnInteractive(ctx context.Context, opts agent.SpawnInteractiveOpts) (*agent.InteractiveAgent, error)
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
	traceStore tracing.TraceReader
	boardSvc   *service.NativeBoardService
	eventBus   port.EventBus
	giteaURL      string
	sseClients    func() int
	cfg           *config.Config
	interSpawner  InteractiveSpawner
	wsSessions    *wsAdapter.SessionManager
}

// APIHandlerOpts configures the API handler.
type APIHandlerOpts struct {
	Agents        AgentLister
	Quota         QuotaReader
	LockMgr       *lock.Manager
	Projects      ProjectLister
	ProjectMgr    ProjectManager
	GitSync       *gitadapter.GitSync
	TraceStore    tracing.TraceReader
	BoardSvc      *service.NativeBoardService
	EventBus      port.EventBus
	GiteaURL      string
	SSEClients    func() int
	Cfg           *config.Config
	InterSpawner  InteractiveSpawner
	WSSessions    *wsAdapter.SessionManager
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
		traceStore:   opts.TraceStore,
		boardSvc:     opts.BoardSvc,
		eventBus:     opts.EventBus,
		giteaURL:     opts.GiteaURL,
		sseClients:   opts.SSEClients,
		cfg:          opts.Cfg,
		interSpawner: opts.InterSpawner,
		wsSessions:   opts.WSSessions,
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

// GetConfig implements gen.StrictServerInterface.
func (h *APIHandler) GetConfig(_ context.Context, _ gen.GetConfigRequestObject) (gen.GetConfigResponseObject, error) {
	if h.cfg == nil {
		return gen.GetConfig200JSONResponse{}, nil
	}
	return gen.GetConfig200JSONResponse{
		TracingEnabled:   h.cfg.IsTracingEnabled(),
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

	if req.Body.TracingEnabled != nil {
		v := *req.Body.TracingEnabled
		h.cfg.TracingEnabled = &v
	}
	if req.Body.DashboardEnabled != nil {
		v := *req.Body.DashboardEnabled
		h.cfg.DashboardEnabled = &v
	}

	if err := h.cfg.Save(); err != nil {
		return gen.UpdateConfig500JSONResponse{Error: fmt.Sprintf("save config: %v", err)}, nil
	}

	return gen.UpdateConfig200JSONResponse{
		TracingEnabled:   h.cfg.IsTracingEnabled(),
		DashboardEnabled: h.cfg.IsDashboardEnabled(),
	}, nil
}

// ListAgents implements gen.StrictServerInterface.
func (h *APIHandler) ListAgents(_ context.Context, _ gen.ListAgentsRequestObject) (gen.ListAgentsResponseObject, error) {
	if err := h.agents.Load(); err != nil {
		return gen.ListAgents500JSONResponse{Error: "failed to load agent state"}, nil
	}
	agents := h.agents.Agents()
	result := make(gen.ListAgents200JSONResponse, 0, len(agents))
	for _, a := range agents {
		result = append(result, domainAgentToGen(a, h.quota))
	}
	return result, nil
}

// SpawnInteractiveAgent implements gen.StrictServerInterface.
func (h *APIHandler) SpawnInteractiveAgent(ctx context.Context, req gen.SpawnInteractiveAgentRequestObject) (gen.SpawnInteractiveAgentResponseObject, error) {
	if h.interSpawner == nil || h.wsSessions == nil {
		return gen.SpawnInteractiveAgent500JSONResponse{Error: "interactive agents not configured"}, nil
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

	// Create bridge and register with WS session manager.
	bridge := wsAdapter.NewBridge(ia.Info.ID, ia.Stdin, ia.Done)
	h.wsSessions.RegisterBridge(ia.Info.ID, bridge)

	// Start output relay in background.
	go h.wsSessions.StartOutputRelay(ia.Info.ID, ia.Output)

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

	var opts []service.AddProjectOpts
	if req.Body.SshKey != nil && *req.Body.SshKey != "" {
		opts = append(opts, service.AddProjectOpts{SSHKeyPath: *req.Body.SshKey})
	}

	result, err := h.projectMgr.AddProject(ctx, req.Body.RemoteUrl, name, opts...)
	if err != nil {
		var existsErr *service.ProjectExistsError
		if errors.As(err, &existsErr) {
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
		var notFound *service.ProjectNotFoundError
		if errors.As(err, &notFound) {
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
		tracks, err := service.DiscoverTracks(p.ProjectDir)
		if err != nil {
			continue
		}
		for _, t := range tracks {
			result = append(result, gen.Track{
				Id:      t.ID,
				Title:   t.Title,
				Status:  gen.TrackStatus(t.Status),
				Project: &p.Slug,
			})
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
	for _, a := range agents {
		counts[a.Status]++
	}

	sseClients := 0
	if h.sseClients != nil {
		sseClients = h.sseClients()
	}

	resp := gen.StatusInfo{
		GiteaUrl:    h.giteaURL,
		AgentCounts: counts,
		TotalAgents: len(agents),
		SseClients:  sseClients,
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
	if h.cfg == nil || h.cfg.SkillsRepo == "" {
		return gen.GetSkillsStatus200JSONResponse{
			InstalledVersion: "",
			UpdateAvailable:  false,
			Skills:           []gen.SkillDetail{},
		}, nil
	}

	skillsDir := h.cfg.GetSkillsDir()
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
	if h.cfg == nil || h.cfg.SkillsRepo == "" {
		return gen.UpdateSkills400JSONResponse{Error: "no skills repo configured"}, nil
	}

	force := req.Body != nil && req.Body.Force != nil && *req.Body.Force
	skillsDir := h.cfg.GetSkillsDir()

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

	tracks, err := service.DiscoverTracks(projectDir)
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

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }

