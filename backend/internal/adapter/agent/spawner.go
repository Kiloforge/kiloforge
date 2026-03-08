package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/skills"
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

// Spawner manages Claude agent lifecycle.
type Spawner struct {
	cfg                *config.Config
	store              port.AgentStore
	tracker            *QuotaTracker
	tracer             port.Tracer
	completionCallback CompletionCallback
}

// NewSpawner creates a spawner. If tracker is nil, stream parsing is disabled.
func NewSpawner(cfg *config.Config, store port.AgentStore, tracker *QuotaTracker) *Spawner {
	return &Spawner{cfg: cfg, store: store, tracker: tracker, tracer: port.NoopTracer{}}
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

// onCompletion invokes the completion callback if set.
func (s *Spawner) onCompletion(agentID, ref, status string) {
	if s.completionCallback != nil {
		s.completionCallback(agentID, ref, status)
	}
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

// SpawnReviewer launches a Claude agent to review a PR.
// The projectDir parameter specifies the working directory for the agent.
func (s *Spawner) SpawnReviewer(ctx context.Context, prNumber int, prURL string) (*domain.AgentInfo, error) {
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

	// Use current working directory as project dir (will be improved with 'kf add').
	projectDir, _ := os.Getwd()

	model := s.cfg.Model

	info := domain.AgentInfo{
		ID:          agentID,
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

	args := []string{"-p", prompt, "--session-id", sessionID, "--output-format", "stream-json"}
	if model != "" {
		args = append([]string{"--model", model}, args...)
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = projectDir

	lf, err := os.Create(logFile)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		lf.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = lf

	if err := cmd.Start(); err != nil {
		lf.Close()
		return nil, fmt.Errorf("start claude: %w", err)
	}

	info.PID = cmd.Process.Pid
	s.store.AddAgent(info)
	if err := s.store.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: save state: %v\n", err)
	}

	_, span := s.tracer.StartSpan(ctx, "agent/reviewer",
		port.StringAttr("agent.id", agentID),
		port.StringAttr("agent.role", "reviewer"),
		port.StringAttr("agent.ref", info.Ref),
		port.IntAttr("agent.pid", cmd.Process.Pid),
		port.StringAttr("session.id", sessionID),
	)
	span.AddEvent("agent.spawned", port.IntAttr("pid", cmd.Process.Pid))

	go s.monitorAgent(agentID, stdout, lf, cmd, span)

	return &info, nil
}

// SpawnDeveloperOpts configures a developer agent spawn.
type SpawnDeveloperOpts struct {
	TrackID     string // conductor track ID
	Flags       string // additional conductor-developer flags
	WorktreeDir string // working directory (worktree path); defaults to cwd
	LogDir      string // log directory; defaults to DataDir/logs
	Model       string // claude model alias (e.g., "opus", "sonnet")
}

// SpawnDeveloper launches a Claude agent to implement a track.
func (s *Spawner) SpawnDeveloper(ctx context.Context, opts SpawnDeveloperOpts) (*domain.AgentInfo, error) {
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

	args := []string{"-p", prompt, "--session-id", sessionID, "--output-format", "stream-json"}
	if model != "" {
		args = append([]string{"--model", model}, args...)
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = workDir

	lf, err := os.Create(logFile)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		lf.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = lf

	if err := cmd.Start(); err != nil {
		lf.Close()
		return nil, fmt.Errorf("start claude: %w", err)
	}

	info.PID = cmd.Process.Pid
	s.store.AddAgent(info)
	if err := s.store.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: save state: %v\n", err)
	}

	_, span := s.tracer.StartSpan(ctx, "agent/developer",
		port.StringAttr("agent.id", agentID),
		port.StringAttr("agent.role", "developer"),
		port.StringAttr("agent.ref", opts.TrackID),
		port.IntAttr("agent.pid", cmd.Process.Pid),
		port.StringAttr("agent.worktree", workDir),
		port.StringAttr("session.id", sessionID),
	)
	span.AddEvent("agent.spawned", port.IntAttr("pid", cmd.Process.Pid))

	go s.monitorAgent(agentID, stdout, lf, cmd, span)

	return &info, nil
}

// SpawnInteractiveOpts configures an interactive agent spawn.
type SpawnInteractiveOpts struct {
	WorkDir string // working directory; defaults to cwd
	Model   string // claude model alias
	Prompt  string // initial prompt; if set, passed via -p flag
	Ref     string // ref label (e.g., "track-gen"); defaults to "interactive"
}

// InteractiveAgent represents a running interactive Claude agent with IO handles.
type InteractiveAgent struct {
	Info   domain.AgentInfo
	Stdin  io.WriteCloser // write to agent's stdin
	Output <-chan []byte   // parsed text output as WS-ready messages
	Done   chan struct{}   // closed when agent exits
}

// SpawnInteractive launches a Claude agent in interactive mode with stdin connected.
func (s *Spawner) SpawnInteractive(ctx context.Context, opts SpawnInteractiveOpts) (*InteractiveAgent, error) {
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

	args := []string{"--session-id", sessionID, "--output-format", "stream-json"}
	if model != "" {
		args = append([]string{"--model", model}, args...)
	}
	if opts.Prompt != "" {
		args = append(args, "-p", opts.Prompt)
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = workDir

	lf, err := os.Create(logFile)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		lf.Close()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		lf.Close()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = lf

	if err := cmd.Start(); err != nil {
		lf.Close()
		return nil, fmt.Errorf("start claude: %w", err)
	}

	info.PID = cmd.Process.Pid
	s.store.AddAgent(info)
	if err := s.store.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: save state: %v\n", err)
	}

	_, span := s.tracer.StartSpan(ctx, "agent/interactive",
		port.StringAttr("agent.id", agentID),
		port.StringAttr("agent.role", "interactive"),
		port.IntAttr("agent.pid", cmd.Process.Pid),
		port.StringAttr("session.id", sessionID),
	)
	span.AddEvent("agent.spawned", port.IntAttr("pid", cmd.Process.Pid))

	output := make(chan []byte, 100)
	done := make(chan struct{})

	go s.monitorInteractive(agentID, stdout, lf, cmd, span, output, done)

	return &InteractiveAgent{
		Info:   info,
		Stdin:  stdin,
		Output: output,
		Done:   done,
	}, nil
}

// monitorInteractive reads stdout from an interactive agent, extracts text
// for WebSocket broadcast, and tracks quota usage.
func (s *Spawner) monitorInteractive(agentID string, stdout io.Reader, lf *os.File, cmd *exec.Cmd, span port.SpanEnder, output chan<- []byte, done chan struct{}) {
	defer lf.Close()
	defer span.End()
	defer close(done)
	defer close(output)

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(lf, line)

		if s.tracker != nil {
			if ev, err := ParseStreamLine(line); err == nil {
				s.tracker.RecordEvent(agentID, ev)
				if ev.Type == "result" && ev.Usage != nil {
					span.SetAttributes(
						port.IntAttr("tokens.input", ev.Usage.InputTokens),
						port.IntAttr("tokens.output", ev.Usage.OutputTokens),
						port.IntAttr("tokens.cache_read", ev.Usage.CacheReadTokens),
						port.IntAttr("tokens.cache_create", ev.Usage.CacheCreationTokens),
						port.Float64Attr("cost.usd", ev.CostUSD),
					)
				}
			}
		}

		// Extract displayable text and send to output channel.
		if text := ExtractText(line); text != "" {
			// Non-blocking send — drop if channel is full.
			select {
			case output <- []byte(text):
			default:
			}
		}
	}

	var finalStatus string
	if err := cmd.Wait(); err != nil {
		finalStatus = "failed"
		s.store.UpdateStatus(agentID, finalStatus)
		span.AddEvent("agent.failed")
		span.SetError(err)
	} else {
		finalStatus = "completed"
		s.store.UpdateStatus(agentID, finalStatus)
		span.AddEvent("agent.completed")
	}
	_ = s.store.Save()

	if s.tracker != nil {
		_ = s.tracker.Save()
	}

	s.onCompletion(agentID, "interactive", finalStatus)
}

// monitorAgent reads stdout from a CC process, logs each line, parses stream
// events for the quota tracker, and updates agent status on completion.
func (s *Spawner) monitorAgent(agentID string, stdout io.Reader, lf *os.File, cmd *exec.Cmd, span port.SpanEnder) {
	defer lf.Close()
	defer span.End()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(lf, line)

		if s.tracker != nil {
			if ev, err := ParseStreamLine(line); err == nil {
				s.tracker.RecordEvent(agentID, ev)
				if ev.Type == "result" && ev.Usage != nil {
					span.SetAttributes(
						port.IntAttr("tokens.input", ev.Usage.InputTokens),
						port.IntAttr("tokens.output", ev.Usage.OutputTokens),
						port.IntAttr("tokens.cache_read", ev.Usage.CacheReadTokens),
						port.IntAttr("tokens.cache_create", ev.Usage.CacheCreationTokens),
						port.Float64Attr("cost.usd", ev.CostUSD),
					)
				}
			}
		}
	}

	// Determine final status and get agent ref for callback.
	var finalStatus string
	if err := cmd.Wait(); err != nil {
		finalStatus = "failed"
		s.store.UpdateStatus(agentID, finalStatus)
		span.AddEvent("agent.failed")
		span.SetError(err)
	} else {
		finalStatus = "completed"
		s.store.UpdateStatus(agentID, finalStatus)
		span.AddEvent("agent.completed")
	}
	_ = s.store.Save()

	if s.tracker != nil {
		_ = s.tracker.Save()
	}

	// Look up the agent ref (track ID) for the callback.
	ref := ""
	if a, err := s.store.FindAgent(agentID); err == nil {
		ref = a.Ref
	}
	s.onCompletion(agentID, ref, finalStatus)
}
