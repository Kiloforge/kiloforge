package agent

import (
	"context"
	"fmt"
	"os"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// ProcessStarter abstracts launching a claude --resume process.
type ProcessStarter interface {
	Start(ctx context.Context, sessionID, workDir, model string) (pid int, err error)
}

// ExecProcessStarter starts a real claude --resume process.
type ExecProcessStarter struct{}

// Start launches claude --resume <sessionID> in the given directory.
func (e *ExecProcessStarter) Start(ctx context.Context, sessionID, workDir, model string) (int, error) {
	return execClaudeResume(ctx, sessionID, workDir, model)
}

// RecoveryResult summarizes the outcome of recovering agents.
type RecoveryResult struct {
	Resumed []string
	Failed  []RecoveryFailure
	Skipped []string
}

// RecoveryFailure records why a specific agent could not be resumed.
type RecoveryFailure struct {
	AgentID string
	Reason  string
}

// RecoveryManager handles auto-recovery of suspended agents on startup.
type RecoveryManager struct {
	store       port.AgentStore
	starter     ProcessStarter
	reliability port.ReliabilityRecorder
}

// NewRecoveryManager creates a RecoveryManager.
func NewRecoveryManager(store port.AgentStore, starter ProcessStarter) *RecoveryManager {
	return &RecoveryManager{store: store, starter: starter}
}

// SetReliabilityRecorder sets the reliability event recorder.
func (rm *RecoveryManager) SetReliabilityRecorder(r port.ReliabilityRecorder) {
	rm.reliability = r
}

// DetectStale finds agents marked running/waiting/suspending whose process is
// dead and marks them suspended. It does NOT attempt to resume them. Returns
// the number of agents marked stale.
func (rm *RecoveryManager) DetectStale() int {
	running := rm.store.AgentsByStatus("running", "waiting", "suspending")
	count := 0
	for _, a := range running {
		if a.PID > 0 && !ProcessAlive(a.PID) {
			_ = rm.store.UpdateStatus(a.ID, string(domain.AgentStatusSuspended))
			count++
		}
	}
	if count > 0 {
		_ = rm.store.Save()
	}
	return count
}

// RecoverAll attempts to resume all suspended agents. Also detects stale
// "running" agents whose process is dead and marks them suspended first.
func (rm *RecoveryManager) RecoverAll(ctx context.Context) RecoveryResult {
	var result RecoveryResult

	// Detect stale agents: marked running but process is dead.
	rm.DetectStale()

	suspended := rm.store.AgentsByStatus("suspended")
	if len(suspended) == 0 {
		return result
	}

	// Resume developers first, then reviewers.
	var devs, others []domain.AgentInfo
	for _, a := range suspended {
		if a.Role == "developer" {
			devs = append(devs, a)
		} else {
			others = append(others, a)
		}
	}
	ordered := append(devs, others...)

	for _, a := range ordered {
		if a.SessionID == "" {
			rm.markResumeFailed(a, "no session ID")
			result.Failed = append(result.Failed, RecoveryFailure{AgentID: a.ID, Reason: "no session ID"})
			continue
		}

		workDir := a.WorktreeDir
		if workDir != "" {
			if _, err := os.Stat(workDir); err != nil {
				rm.markResumeFailed(a, "worktree missing")
				result.Failed = append(result.Failed, RecoveryFailure{AgentID: a.ID, Reason: "worktree missing"})
				continue
			}
		}

		pid, err := rm.starter.Start(ctx, a.SessionID, workDir, a.Model)
		if err != nil {
			errMsg := fmt.Sprintf("resume failed: %v", err)
			rm.markResumeFailed(a, errMsg)
			result.Failed = append(result.Failed, RecoveryFailure{AgentID: a.ID, Reason: errMsg})
			continue
		}

		a.Status = string(domain.AgentStatusRunning)
		a.PID = pid
		a.SuspendedAt = nil
		a.ShutdownReason = ""
		a.ResumeError = ""
		_ = rm.store.AddAgent(a) // upsert with all fields
		result.Resumed = append(result.Resumed, a.ID)
	}

	_ = rm.store.Save()
	return result
}

// markResumeFailed sets an agent to resume-failed with a reason and persists via upsert.
func (rm *RecoveryManager) markResumeFailed(a domain.AgentInfo, reason string) {
	a.Status = string(domain.AgentStatusResumeFailed)
	a.ResumeError = reason
	_ = rm.store.AddAgent(a) // upsert with ResumeError set
	rm.recordResumeFail(a.ID, a.Role, reason)
}

func (rm *RecoveryManager) recordResumeFail(agentID, role, reason string) {
	if rm.reliability != nil {
		_ = rm.reliability.RecordEvent(
			domain.RelEventAgentResumeFail, domain.SeverityError,
			agentID, role,
			map[string]any{"reason": reason},
		)
	}
}

// execClaudeResume runs `claude --resume <sessionID>` in the given directory.
func execClaudeResume(ctx context.Context, sessionID, workDir, model string) (int, error) {
	args := []string{"--resume", sessionID, "--output-format", "stream-json", "--verbose"}
	if model != "" {
		args = append([]string{"--model", model}, args...)
	}
	cmd := newCommand(ctx, "claude", args...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start claude: %w", err)
	}
	return cmd.Process.Pid, nil
}
