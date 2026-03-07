package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"conductor-relay/internal/config"
	"conductor-relay/internal/state"

	"github.com/google/uuid"
)

// Spawner manages Claude agent lifecycle.
type Spawner struct {
	cfg   *config.Config
	store *state.Store
}

func NewSpawner(cfg *config.Config, store *state.Store) *Spawner {
	return &Spawner{cfg: cfg, store: store}
}

// SpawnReviewer launches a Claude agent to review a PR.
func (s *Spawner) SpawnReviewer(ctx context.Context, prNumber int, prURL string) (*state.AgentInfo, error) {
	agentID := uuid.New().String()
	sessionID := uuid.New().String()
	logDir := filepath.Join(s.cfg.DataDir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	logFile := filepath.Join(logDir, agentID+".log")

	prompt := fmt.Sprintf("/conductor-reviewer %s", prURL)

	info := state.AgentInfo{
		ID:          agentID,
		Role:        "reviewer",
		Ref:         fmt.Sprintf("PR #%d", prNumber),
		Status:      "running",
		SessionID:   sessionID,
		WorktreeDir: s.cfg.ProjectDir,
		LogFile:     logFile,
		StartedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	cmd := exec.CommandContext(ctx, "claude",
		"-p", prompt,
		"--session-id", sessionID,
		"--output-format", "stream-json",
	)
	cmd.Dir = s.cfg.ProjectDir

	// Capture output to log file.
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
	if err := s.store.Save(s.cfg.DataDir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: save state: %v\n", err)
	}

	// Stream output to log file in background.
	go func() {
		defer lf.Close()
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large JSON lines.
		for scanner.Scan() {
			fmt.Fprintln(lf, scanner.Text())
		}

		// Wait for process to finish.
		if err := cmd.Wait(); err != nil {
			s.store.UpdateStatus(agentID, "failed")
		} else {
			s.store.UpdateStatus(agentID, "completed")
		}
		_ = s.store.Save(s.cfg.DataDir)
	}()

	return &info, nil
}

// SpawnDeveloper launches a Claude agent to implement a track.
func (s *Spawner) SpawnDeveloper(ctx context.Context, trackID string, flags string) (*state.AgentInfo, error) {
	agentID := uuid.New().String()
	sessionID := uuid.New().String()
	logDir := filepath.Join(s.cfg.DataDir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	logFile := filepath.Join(logDir, agentID+".log")

	prompt := fmt.Sprintf("/conductor-developer %s %s", trackID, flags)

	info := state.AgentInfo{
		ID:          agentID,
		Role:        "developer",
		Ref:         trackID,
		Status:      "running",
		SessionID:   sessionID,
		WorktreeDir: s.cfg.ProjectDir,
		LogFile:     logFile,
		StartedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	cmd := exec.CommandContext(ctx, "claude",
		"-p", prompt,
		"--session-id", sessionID,
		"--output-format", "stream-json",
	)
	cmd.Dir = s.cfg.ProjectDir

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
	if err := s.store.Save(s.cfg.DataDir); err != nil {
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
		_ = s.store.Save(s.cfg.DataDir)
	}()

	return &info, nil
}
