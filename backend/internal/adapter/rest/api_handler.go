package rest

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"crelay/internal/adapter/agent"
	"crelay/internal/adapter/config"
	"crelay/internal/adapter/lock"
	"crelay/internal/adapter/rest/gen"
	"crelay/internal/adapter/skills"
	"crelay/internal/core/domain"
	"crelay/internal/core/service"
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

// APIHandler implements gen.StrictServerInterface by delegating to existing
// adapters for agents, locks, quota, and tracks.
type APIHandler struct {
	agents     AgentLister
	quota      QuotaReader
	lockMgr    *lock.Manager
	projects   ProjectLister
	giteaURL   string
	sseClients func() int
	cfg        *config.Config
}

// APIHandlerOpts configures the API handler.
type APIHandlerOpts struct {
	Agents     AgentLister
	Quota      QuotaReader
	LockMgr    *lock.Manager
	Projects   ProjectLister
	GiteaURL   string
	SSEClients func() int
	Cfg        *config.Config
}

// NewAPIHandler creates a new handler implementing StrictServerInterface.
func NewAPIHandler(opts APIHandlerOpts) *APIHandler {
	return &APIHandler{
		agents:     opts.Agents,
		quota:      opts.Quota,
		lockMgr:    opts.LockMgr,
		projects:   opts.Projects,
		giteaURL:   opts.GiteaURL,
		sseClients: opts.SSEClients,
		cfg:        opts.Cfg,
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
			TotalCostUsd: 0,
			RateLimited:  false,
		}, nil
	}
	total := h.quota.GetTotalUsage()
	resp := gen.QuotaInfo{
		TotalCostUsd: total.TotalCostUSD,
		InputTokens:  intPtr(total.InputTokens),
		OutputTokens: intPtr(total.OutputTokens),
		AgentCount:   intPtr(total.AgentCount),
		RateLimited:  h.quota.IsRateLimited(),
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
					AgentId:      a.ID,
					CostUsd:      usage.TotalCostUSD,
					InputTokens:  usage.InputTokens,
					OutputTokens: usage.OutputTokens,
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
		resp.TotalCostUsd = &total.TotalCostUSD
	}

	return gen.GetStatus200JSONResponse(resp), nil
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
	if quota != nil {
		if usage := quota.GetAgentUsage(a.ID); usage != nil {
			g.CostUsd = &usage.TotalCostUSD
			g.InputTokens = &usage.InputTokens
			g.OutputTokens = &usage.OutputTokens
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

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }

