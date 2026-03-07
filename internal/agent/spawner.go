package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"crelay/internal/adapter/persistence/jsonfile"
	"crelay/internal/config"
	"crelay/internal/core/domain"

	"github.com/google/uuid"
)

// Spawner manages Claude agent lifecycle.
type Spawner struct {
	cfg     *config.Config
	store   *jsonfile.AgentStore
	tracker *QuotaTracker
}

// NewSpawner creates a spawner. If tracker is nil, stream parsing is disabled.
func NewSpawner(cfg *config.Config, store *jsonfile.AgentStore, tracker *QuotaTracker) *Spawner {
	return &Spawner{cfg: cfg, store: store, tracker: tracker}
}

// checkQuota returns an error if the tracker indicates rate limiting or budget exceeded.
func (s *Spawner) checkQuota() error {
	if s.tracker == nil {
		return nil
	}
	if s.tracker.IsRateLimited() {
		ra := s.tracker.RetryAfter()
		return fmt.Errorf("rate limited — retry after %s", ra.Round(time.Second))
	}
	if s.cfg.MaxSessionCostUSD > 0 {
		total := s.tracker.GetTotalUsage()
		if total.TotalCostUSD >= s.cfg.MaxSessionCostUSD {
			return fmt.Errorf("budget exceeded ($%.2f / $%.2f) — increase max_session_cost_usd or wait", total.TotalCostUSD, s.cfg.MaxSessionCostUSD)
		}
		if total.TotalCostUSD >= s.cfg.MaxSessionCostUSD*0.8 {
			fmt.Fprintf(os.Stderr, "warning: approaching budget limit ($%.2f / $%.2f)\n", total.TotalCostUSD, s.cfg.MaxSessionCostUSD)
		}
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

	prompt := fmt.Sprintf("/conductor-reviewer %s", prURL)

	// Use current working directory as project dir (will be improved with 'crelay add').
	projectDir, _ := os.Getwd()

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
	}

	cmd := exec.CommandContext(ctx, "claude",
		"-p", prompt,
		"--session-id", sessionID,
		"--output-format", "stream-json",
	)
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

	go s.monitorAgent(agentID, stdout, lf, cmd)

	return &info, nil
}

// SpawnDeveloperOpts configures a developer agent spawn.
type SpawnDeveloperOpts struct {
	TrackID     string // conductor track ID
	Flags       string // additional conductor-developer flags
	WorktreeDir string // working directory (worktree path); defaults to cwd
	LogDir      string // log directory; defaults to DataDir/logs
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

	prompt := fmt.Sprintf("/conductor-developer %s %s", opts.TrackID, opts.Flags)

	workDir := opts.WorktreeDir
	if workDir == "" {
		workDir, _ = os.Getwd()
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
	}

	cmd := exec.CommandContext(ctx, "claude",
		"-p", prompt,
		"--session-id", sessionID,
		"--output-format", "stream-json",
	)
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

	go s.monitorAgent(agentID, stdout, lf, cmd)

	return &info, nil
}

// monitorAgent reads stdout from a CC process, logs each line, parses stream
// events for the quota tracker, and updates agent status on completion.
func (s *Spawner) monitorAgent(agentID string, stdout io.Reader, lf *os.File, cmd *exec.Cmd) {
	defer lf.Close()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(lf, line)

		if s.tracker != nil {
			if ev, err := ParseStreamLine(line); err == nil {
				s.tracker.RecordEvent(agentID, ev)
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		s.store.UpdateStatus(agentID, "failed")
	} else {
		s.store.UpdateStatus(agentID, "completed")
	}
	_ = s.store.Save()

	if s.tracker != nil {
		_ = s.tracker.Save()
	}
}
