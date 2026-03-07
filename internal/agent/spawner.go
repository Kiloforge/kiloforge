package agent

import (
	"bufio"
	"context"
	"fmt"
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
	cfg   *config.Config
	store *jsonfile.AgentStore
}

func NewSpawner(cfg *config.Config, store *jsonfile.AgentStore) *Spawner {
	return &Spawner{cfg: cfg, store: store}
}

// SpawnReviewer launches a Claude agent to review a PR.
// The projectDir parameter specifies the working directory for the agent.
func (s *Spawner) SpawnReviewer(ctx context.Context, prNumber int, prURL string) (*domain.AgentInfo, error) {
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

	go func() {
		defer lf.Close()
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			fmt.Fprintln(lf, scanner.Text())
		}

		if err := cmd.Wait(); err != nil {
			s.store.UpdateStatus(agentID, "failed")
		} else {
			s.store.UpdateStatus(agentID, "completed")
		}
		_ = s.store.Save()
	}()

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

	go func() {
		defer lf.Close()
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			fmt.Fprintln(lf, scanner.Text())
		}

		if err := cmd.Wait(); err != nil {
			s.store.UpdateStatus(agentID, "failed")
		} else {
			s.store.UpdateStatus(agentID, "completed")
		}
		_ = s.store.Save()
	}()

	return &info, nil
}
