package agent

import (
	"context"
	"fmt"
	"os"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
	"kiloforge/internal/core/service"
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
	store          port.AgentStore
	starter        ProcessStarter
	reliabilitySvc *service.ReliabilityService
}

// NewRecoveryManager creates a RecoveryManager.
func NewRecoveryManager(store port.AgentStore, starter ProcessStarter) *RecoveryManager {
	return &RecoveryManager{store: store, starter: starter}
}

// SetReliabilityService sets the reliability service for recording resume failure events.
func (rm *RecoveryManager) SetReliabilityService(svc *service.ReliabilityService) {
	rm.reliabilitySvc = svc
}

// RecoverAll attempts to resume all suspended agents. Also detects stale
// "running" agents whose process is dead and marks them suspended first.
func (rm *RecoveryManager) RecoverAll(ctx context.Context) RecoveryResult {
	var result RecoveryResult

	// Detect stale agents: marked running but process is dead.
	running := rm.store.AgentsByStatus("running", "waiting", "suspending")
	for _, a := range running {
		if a.PID > 0 && !ProcessAlive(a.PID) {
			_ = rm.store.UpdateStatus(a.ID, string(domain.AgentStatusSuspended))
		}
	}

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
			_ = rm.store.UpdateStatus(a.ID, string(domain.AgentStatusResumeFailed))
			if ag, err := rm.store.FindAgent(a.ID); err == nil {
				ag.ResumeError = "no session ID"
			}
			result.Failed = append(result.Failed, RecoveryFailure{AgentID: a.ID, Reason: "no session ID"})
			rm.recordResumeFailure(a.ID, a.Ref, "no session ID")
			continue
		}

		workDir := a.WorktreeDir
		if workDir != "" {
			if _, err := os.Stat(workDir); err != nil {
				_ = rm.store.UpdateStatus(a.ID, string(domain.AgentStatusResumeFailed))
				if ag, err := rm.store.FindAgent(a.ID); err == nil {
					ag.ResumeError = "worktree missing"
				}
				result.Failed = append(result.Failed, RecoveryFailure{AgentID: a.ID, Reason: "worktree missing"})
				rm.recordResumeFailure(a.ID, a.Ref, "worktree missing")
				continue
			}
		}

		pid, err := rm.starter.Start(ctx, a.SessionID, workDir, a.Model)
		if err != nil {
			_ = rm.store.UpdateStatus(a.ID, string(domain.AgentStatusResumeFailed))
			errMsg := fmt.Sprintf("resume failed: %v", err)
			if ag, err := rm.store.FindAgent(a.ID); err == nil {
				ag.ResumeError = errMsg
			}
			result.Failed = append(result.Failed, RecoveryFailure{AgentID: a.ID, Reason: errMsg})
			rm.recordResumeFailure(a.ID, a.Ref, errMsg)
			continue
		}

		_ = rm.store.UpdateStatus(a.ID, string(domain.AgentStatusRunning))
		if ag, err := rm.store.FindAgent(a.ID); err == nil {
			ag.PID = pid
			ag.SuspendedAt = nil
			ag.ShutdownReason = ""
			ag.ResumeError = ""
		}
		result.Resumed = append(result.Resumed, a.ID)
	}

	_ = rm.store.Save()
	return result
}

func (rm *RecoveryManager) recordResumeFailure(agentID, ref, reason string) {
	if rm.reliabilitySvc != nil {
		_ = rm.reliabilitySvc.RecordEvent(domain.RelEvtAgentResumeFail, domain.SeverityError, agentID, ref, map[string]any{
			"reason": reason,
		})
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
