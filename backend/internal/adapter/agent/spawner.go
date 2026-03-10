package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/prereq"
	"kiloforge/internal/adapter/skills"
	"kiloforge/internal/adapter/ws"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"

	"github.com/google/uuid"
)

// ErrSkillsMissing is returned when required skills are not installed.
type ErrSkillsMissing struct {
	Missing []skills.RequiredSkill
}

func (e *ErrSkillsMissing) Error() string {
	names := make([]string, len(e.Missing))
	for i, s := range e.Missing {
		names[i] = s.Name
	}
	return fmt.Sprintf("required skills not installed: %s", strings.Join(names, ", "))
}

// ValidateSkills checks that the required skills for a given role are installed.
// It checks both the global skills directory (from config) and the local
// .claude/skills/ directory relative to workDir.
// Returns ErrSkillsMissing if any required skills are not found.
func (s *Spawner) ValidateSkills(role, workDir string) error {
	required := skills.RequiredSkillsForRole(role)
	if len(required) == 0 {
		return nil
	}

	globalDir := s.cfg.GetSkillsDir()

	localDir := ""
	if workDir != "" {
		localDir = filepath.Join(workDir, ".claude", "skills")
	}

	missing := skills.CheckRequired(required, globalDir, localDir)
	if len(missing) > 0 {
		return &ErrSkillsMissing{Missing: missing}
	}
	return nil
}

// CompletionCallback is called when an agent process exits.
// It receives the agent ID, ref (track ID), and final status.
type CompletionCallback func(agentID, ref, status string)

// SessionEndCallback is called when an interactive agent session ends.
// Used to clean up resources like WS bridges.
type SessionEndCallback func(agentID string)

// Spawner manages Claude agent lifecycle.
type Spawner struct {
	cfg                *config.Config
	store              port.AgentStore
	tracker            *QuotaTracker
	tracer             port.Tracer
	completionCallback CompletionCallback
	sessionEndCallback SessionEndCallback

	activeMu     sync.RWMutex
	activeAgents map[string]*InteractiveAgent
}

// CleanClaudeEnv returns os.Environ() with Claude-internal env vars removed
// to prevent "nested session" detection in child claude processes.
// Deprecated: The SDK handles environment cleaning automatically for SDK-based agents.
// This function is retained for non-SDK callers (e.g., direct exec.Command usage in server.go).
func CleanClaudeEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "CLAUDECODE=") ||
			strings.HasPrefix(e, "CLAUDE_CODE_ENTRYPOINT=") {
			continue
		}
		env = append(env, e)
	}
	return env
}

// NewSpawner creates a spawner. If tracker is nil, stream parsing is disabled.
func NewSpawner(cfg *config.Config, store port.AgentStore, tracker *QuotaTracker) *Spawner {
	return &Spawner{
		cfg:          cfg,
		store:        store,
		tracker:      tracker,
		tracer:       port.NoopTracer{},
		activeAgents: make(map[string]*InteractiveAgent),
	}
}

// GetActiveAgent returns a running interactive agent by ID.
func (s *Spawner) GetActiveAgent(id string) (*InteractiveAgent, bool) {
	s.activeMu.RLock()
	defer s.activeMu.RUnlock()
	a, ok := s.activeAgents[id]
	return a, ok
}

// SetTracer sets the distributed tracer for agent lifecycle spans.
func (s *Spawner) SetTracer(t port.Tracer) {
	if t != nil {
		s.tracer = t
	}
}

// SetCompletionCallback sets the function called when an agent process exits.
func (s *Spawner) SetCompletionCallback(fn CompletionCallback) {
	s.completionCallback = fn
}

// SetSessionEndCallback sets the function called when an interactive session ends.
// Typically used to call SessionManager.UnregisterBridge.
func (s *Spawner) SetSessionEndCallback(fn SessionEndCallback) {
	s.sessionEndCallback = fn
}

// onCompletion invokes the completion callback if set.
func (s *Spawner) onCompletion(agentID, ref, status string) {
	if s.completionCallback != nil {
		s.completionCallback(agentID, ref, status)
	}
}

// checkAuth verifies Claude CLI authentication before spawning.
func (s *Spawner) checkAuth(ctx context.Context) error {
	if err := prereq.CheckClaudeAuthCached(ctx); err != nil {
		return fmt.Errorf("claude auth: %w", err)
	}
	return nil
}

// checkQuota returns an error if the tracker indicates rate limiting.
// Budget enforcement via MaxSessionCostUSD is deprecated — subscription
// rate limits are the primary constraint.
func (s *Spawner) checkQuota() error {
	if s.tracker == nil {
		return nil
	}
	if s.tracker.IsRateLimited() {
		ra := s.tracker.RetryAfter()
		return fmt.Errorf("rate limited — retry after %s", ra.Round(time.Second))
	}
	return nil
}

// SpawnReviewer launches a Claude agent to review a PR using the SDK Query function.
func (s *Spawner) SpawnReviewer(ctx context.Context, prNumber int, prURL string) (*domain.AgentInfo, error) {
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	if err := s.checkQuota(); err != nil {
		return nil, fmt.Errorf("spawn blocked: %w", err)
	}

	agentID := uuid.New().String()
	sessionID := uuid.New().String()
	logDir := filepath.Join(s.cfg.DataDir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	logFile := filepath.Join(logDir, agentID+".log")

	prompt := fmt.Sprintf("/kf-reviewer %s", prURL)

	projectDir, _ := os.Getwd()
	model := s.cfg.Model

	info := domain.AgentInfo{
		ID:          agentID,
		Name:        GenerateName(),
		Role:        "reviewer",
		Ref:         fmt.Sprintf("PR #%d", prNumber),
		Status:      "running",
		SessionID:   sessionID,
		WorktreeDir: projectDir,
		LogFile:     logFile,
		StartedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Model:       model,
	}

	if err := s.store.AddAgent(info); err != nil {
		fmt.Fprintf(os.Stderr, "warning: add agent: %v\n", err)
	}
	if err := s.store.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: save state: %v\n", err)
	}

	_, span := s.tracer.StartSpan(ctx, "agent/reviewer",
		port.StringAttr("agent.id", agentID),
		port.StringAttr("agent.name", info.Name),
		port.StringAttr("agent.role", "reviewer"),
		port.StringAttr("agent.ref", info.Ref),
		port.StringAttr("session.id", sessionID),
	)
	span.AddEvent("agent.spawned")

	go s.runSDKAgent(context.Background(), agentID, info.Ref, prompt, projectDir, model, logFile, span)

	return &info, nil
}

// SpawnDeveloperOpts configures a developer agent spawn.
type SpawnDeveloperOpts struct {
	TrackID     string // conductor track ID
	Flags       string // additional kf-developer flags
	WorktreeDir string // working directory (worktree path); defaults to cwd
	LogDir      string // log directory; defaults to DataDir/logs
	Model       string // claude model alias (e.g., "opus", "sonnet")
}

// SpawnDeveloper launches a Claude agent to implement a track using the SDK Query function.
func (s *Spawner) SpawnDeveloper(ctx context.Context, opts SpawnDeveloperOpts) (*domain.AgentInfo, error) {
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	if err := s.checkQuota(); err != nil {
		return nil, fmt.Errorf("spawn blocked: %w", err)
	}

	agentID := uuid.New().String()
	sessionID := uuid.New().String()

	logDir := opts.LogDir
	if logDir == "" {
		logDir = filepath.Join(s.cfg.DataDir, "logs")
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	logFile := filepath.Join(logDir, agentID+".log")

	prompt := fmt.Sprintf("/kf-developer %s %s", opts.TrackID, opts.Flags)

	workDir := opts.WorktreeDir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	model := opts.Model
	if model == "" {
		model = s.cfg.Model
	}

	info := domain.AgentInfo{
		ID:          agentID,
		Name:        GenerateName(),
		Role:        "developer",
		Ref:         opts.TrackID,
		Status:      "running",
		SessionID:   sessionID,
		WorktreeDir: workDir,
		LogFile:     logFile,
		StartedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Model:       model,
	}

	if err := s.store.AddAgent(info); err != nil {
		fmt.Fprintf(os.Stderr, "warning: add agent: %v\n", err)
	}
	if err := s.store.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: save state: %v\n", err)
	}

	_, span := s.tracer.StartSpan(ctx, "agent/developer",
		port.StringAttr("agent.id", agentID),
		port.StringAttr("agent.name", info.Name),
		port.StringAttr("agent.role", "developer"),
		port.StringAttr("agent.ref", opts.TrackID),
		port.StringAttr("agent.worktree", workDir),
		port.StringAttr("session.id", sessionID),
	)
	span.AddEvent("agent.spawned")

	go s.runSDKAgent(context.Background(), agentID, opts.TrackID, prompt, workDir, model, logFile, span)

	return &info, nil
}

// runSDKAgent executes a one-shot SDK Query and updates agent state on completion.
func (s *Spawner) runSDKAgent(ctx context.Context, agentID, ref, prompt, workDir, model, logFile string, span port.SpanEnder) {
	defer span.End()

	finalStatus, err := QueryOneShot(ctx, prompt, workDir, model, logFile, s.tracker, agentID, span)
	if err != nil {
		finalStatus = "failed"
		if uerr := s.store.UpdateStatus(agentID, finalStatus); uerr != nil {
			fmt.Fprintf(os.Stderr, "warning: update status: %v\n", uerr)
		}
		span.AddEvent("agent.failed")
		span.SetError(err)
	} else {
		if uerr := s.store.UpdateStatus(agentID, finalStatus); uerr != nil {
			fmt.Fprintf(os.Stderr, "warning: update status: %v\n", uerr)
		}
		span.AddEvent("agent." + finalStatus)
	}
	_ = s.store.Save()

	if s.tracker != nil {
		_ = s.tracker.Save()
	}

	s.onCompletion(agentID, ref, finalStatus)
}

// SpawnInteractiveOpts configures an interactive agent spawn.
type SpawnInteractiveOpts struct {
	WorkDir string // working directory; defaults to cwd
	Model   string // claude model alias
	Prompt  string // initial prompt; if set, sent as the first query
	Ref     string // ref label (e.g., "track-gen"); defaults to "interactive"
}

// InteractiveAgent represents a running interactive Claude agent with IO handles.
type InteractiveAgent struct {
	Info        domain.AgentInfo
	Stdin       ws.InputHandler    // SDK-based input handler
	Output      <-chan []byte      // structured messages for WS relay
	Done        chan struct{}      // closed when agent exits
	sdkSession  *SDKSession        // SDK session for turn-based input
	cancelRelay context.CancelFunc // cancels the current relay goroutine
}

// SetCancelRelay stores the cancel function for the current relay goroutine.
func (ia *InteractiveAgent) SetCancelRelay(cancel context.CancelFunc) {
	ia.cancelRelay = cancel
}

// CancelRelay cancels the current relay goroutine if one is active.
func (ia *InteractiveAgent) CancelRelay() {
	if ia.cancelRelay != nil {
		ia.cancelRelay()
	}
}

// SpawnInteractive launches a Claude agent in interactive mode using the SDK Client.
func (s *Spawner) SpawnInteractive(ctx context.Context, opts SpawnInteractiveOpts) (*InteractiveAgent, error) {
	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	if err := s.checkQuota(); err != nil {
		return nil, fmt.Errorf("spawn blocked: %w", err)
	}

	agentID := uuid.New().String()
	sessionID := uuid.New().String()

	logDir := filepath.Join(s.cfg.DataDir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	logFile := filepath.Join(logDir, agentID+".log")

	workDir := opts.WorkDir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	model := opts.Model
	if model == "" {
		model = s.cfg.Model
	}

	ref := opts.Ref
	if ref == "" {
		ref = "interactive"
	}

	info := domain.AgentInfo{
		ID:          agentID,
		Name:        GenerateName(),
		Role:        "interactive",
		Ref:         ref,
		Status:      "running",
		SessionID:   sessionID,
		WorktreeDir: workDir,
		LogFile:     logFile,
		StartedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Model:       model,
	}

	// Create SDK session with a detached context so the agent process
	// outlives the HTTP request that spawned it. The session manages its
	// own lifecycle via session.Close() / session.cancel.
	session, err := NewSDKSession(context.Background(), workDir, model, logFile)
	if err != nil {
		return nil, fmt.Errorf("create SDK session: %w", err)
	}

	// Open log file for structured output.
	lf, err := os.Create(logFile)
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("create log file: %w", err)
	}
	session.SetLogFile(lf)

	// Connect to Claude CLI.
	if err := session.Connect(context.Background()); err != nil {
		lf.Close()
		session.Close()
		return nil, fmt.Errorf("SDK connect: %w", err)
	}

	if err := s.store.AddAgent(info); err != nil {
		fmt.Fprintf(os.Stderr, "warning: add agent: %v\n", err)
	}
	if err := s.store.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: save state: %v\n", err)
	}

	_, span := s.tracer.StartSpan(ctx, "agent/interactive",
		port.StringAttr("agent.id", agentID),
		port.StringAttr("agent.name", info.Name),
		port.StringAttr("agent.role", "interactive"),
		port.StringAttr("session.id", sessionID),
	)
	span.AddEvent("agent.spawned")

	// If initial prompt is set, send the first query.
	// Use the session's own context (not the HTTP request context).
	if opts.Prompt != "" {
		if err := session.Query(session.ctx, opts.Prompt, s.tracker, agentID, span); err != nil {
			span.End()
			session.Close()
			return nil, fmt.Errorf("initial query: %w", err)
		}
	}

	// Create input handler that sends subsequent queries via SDK.
	// Uses the session's own context so queries outlive the original HTTP request.
	inputHandler := func(text string) error {
		return session.Query(session.ctx, text, s.tracker, agentID, span)
	}

	ia := &InteractiveAgent{
		Info:       info,
		Stdin:      inputHandler,
		Output:     session.Output(),
		Done:       session.done,
		sdkSession: session,
	}

	// Register in active agents map.
	s.activeMu.Lock()
	s.activeAgents[agentID] = ia
	s.activeMu.Unlock()

	// Monitor session lifecycle in background.
	go s.monitorSDKSession(agentID, ref, session, span)

	return ia, nil
}

// StopAgent stops a running interactive agent.
func (s *Spawner) StopAgent(id string) error {
	s.activeMu.Lock()
	ia, ok := s.activeAgents[id]
	if ok {
		delete(s.activeAgents, id)
	}
	s.activeMu.Unlock()

	if !ok {
		return fmt.Errorf("agent not running: %s", id)
	}

	// Cancel relay goroutine and close SDK session.
	ia.CancelRelay()
	ia.sdkSession.Close()

	// Update store.
	now := time.Now()
	if uerr := s.store.UpdateStatus(id, "stopped"); uerr != nil {
		fmt.Fprintf(os.Stderr, "warning: update status: %v\n", uerr)
	}
	agent, err := s.store.FindAgent(id)
	if err == nil {
		agent.ShutdownReason = "user_stopped"
		agent.FinishedAt = &now
		if uerr := s.store.AddAgent(*agent); uerr != nil { // upsert
			fmt.Fprintf(os.Stderr, "warning: add agent: %v\n", uerr)
		}
	}
	_ = s.store.Save()

	return nil
}

// ResumeAgent resumes a stopped/completed/failed interactive agent session.
func (s *Spawner) ResumeAgent(ctx context.Context, id string) (*InteractiveAgent, error) {
	// Check not already running.
	s.activeMu.RLock()
	_, running := s.activeAgents[id]
	s.activeMu.RUnlock()
	if running {
		return nil, fmt.Errorf("agent already running: %s", id)
	}

	agent, err := s.store.FindAgent(id)
	if err != nil {
		return nil, err
	}

	if agent.IsActive() {
		return nil, fmt.Errorf("agent already running: %s", id)
	}

	if err := s.checkAuth(ctx); err != nil {
		return nil, err
	}
	if err := s.checkQuota(); err != nil {
		return nil, fmt.Errorf("spawn blocked: %w", err)
	}

	workDir := agent.WorktreeDir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	model := agent.Model
	if model == "" {
		model = s.cfg.Model
	}

	logFile := agent.LogFile

	// Create SDK session with resume — detached from HTTP request context.
	session, err := NewSDKSessionWithResume(context.Background(), workDir, model, logFile, agent.SessionID)
	if err != nil {
		return nil, fmt.Errorf("create resumed SDK session: %w", err)
	}

	// Open log file for append.
	lf, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("open log file: %w", err)
	}
	session.SetLogFile(lf)

	// Connect to Claude CLI — detached from HTTP request context.
	if err := session.Connect(context.Background()); err != nil {
		lf.Close()
		session.Close()
		return nil, fmt.Errorf("SDK connect: %w", err)
	}

	// Update agent status.
	if uerr := s.store.UpdateStatus(id, "running"); uerr != nil {
		fmt.Fprintf(os.Stderr, "warning: update status: %v\n", uerr)
	}
	_ = s.store.Save()

	_, span := s.tracer.StartSpan(ctx, "agent/interactive-resume",
		port.StringAttr("agent.id", id),
		port.StringAttr("agent.role", agent.Role),
		port.StringAttr("session.id", agent.SessionID),
	)
	span.AddEvent("agent.resumed")

	inputHandler := func(text string) error {
		return session.Query(session.ctx, text, s.tracker, id, span)
	}

	ia := &InteractiveAgent{
		Info:       *agent,
		Stdin:      inputHandler,
		Output:     session.Output(),
		Done:       session.done,
		sdkSession: session,
	}
	ia.Info.Status = "running"
	ia.Info.FinishedAt = nil

	s.activeMu.Lock()
	s.activeAgents[id] = ia
	s.activeMu.Unlock()

	go s.monitorSDKSession(id, agent.Ref, session, span)

	return ia, nil
}

// monitorSDKSession waits for the SDK session to end and updates agent state.
func (s *Spawner) monitorSDKSession(agentID, ref string, session *SDKSession, span port.SpanEnder) {
	<-session.ctx.Done()

	// Remove from active agents registry.
	s.activeMu.Lock()
	delete(s.activeAgents, agentID)
	s.activeMu.Unlock()

	// Clean up WS bridge.
	if s.sessionEndCallback != nil {
		s.sessionEndCallback(agentID)
	}

	span.AddEvent("agent.completed")
	span.End()

	if uerr := s.store.UpdateStatus(agentID, "completed"); uerr != nil {
		fmt.Fprintf(os.Stderr, "warning: update status: %v\n", uerr)
	}
	_ = s.store.Save()

	if s.tracker != nil {
		_ = s.tracker.Save()
	}

	s.onCompletion(agentID, ref, "completed")
}
