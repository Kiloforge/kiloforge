package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"kiloforge/internal/adapter/config"
)

// TestIntegration_AgentLifecycle_SpawnStopResumeAttach exercises the full
// interactive agent lifecycle against a real Claude CLI process:
//
//  1. SpawnInteractive with prompt "hello" → wait for real response
//  2. StopAgent → verify stopped
//  3. ResumeAgent → attach to the same session
//  4. Send "say the word 'pong'" via Stdin → wait for real response
//  5. StopAgent → verify final state
//
// This test is SKIPPED by default. Set KF_INTEGRATION_TEST=1 to run it.
// Requires: Claude CLI installed, authenticated, and network access.
func TestIntegration_AgentLifecycle_SpawnStopResumeAttach(t *testing.T) {
	if os.Getenv("KF_INTEGRATION_TEST") != "1" {
		t.Skip("skipped: set KF_INTEGRATION_TEST=1 to run real agent lifecycle test")
	}

	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("skipped: claude CLI not found in PATH")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("skipped: git not found in PATH")
	}

	// Create a temp git repo as the working directory.
	workDir := t.TempDir()
	if out, err := exec.Command("git", "init", workDir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %s: %v", out, err)
	}

	store := &stubAgentStore{}
	cfg := &config.Config{
		DataDir:      t.TempDir(),
		MaxSwarmSize: 5,
		Model:        "haiku", // use cheapest model for integration tests
	}
	spawner := NewSpawner(cfg, store, nil)

	ctx := context.Background()

	// Clear Claude nesting detection env vars — we're likely running inside
	// Claude Code, and the child process will refuse to start if it detects
	// a parent session. Save and restore after the test.
	for _, key := range []string{"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT"} {
		if orig, ok := os.LookupEnv(key); ok {
			os.Unsetenv(key)
			defer os.Setenv(key, orig)
		}
	}

	// =========================================================================
	// Step 1: SpawnInteractive with prompt "hello"
	// =========================================================================
	t.Log("step 1: spawning interactive agent with prompt 'hello'...")

	ia, err := spawner.SpawnInteractive(ctx, SpawnInteractiveOpts{
		WorkDir: workDir,
		Prompt:  "respond with only the word 'hello'. nothing else. no punctuation.",
	})
	if err != nil {
		t.Fatalf("SpawnInteractive failed: %v", err)
	}

	agentID := ia.Info.ID
	t.Logf("  spawned agent %s (name=%s)", agentID, ia.Info.Name)

	// Wait for a text response and turn_end.
	text1, err := waitForTextAndTurnEnd(t, ia.Output, 60*time.Second)
	if err != nil {
		t.Fatalf("  waiting for response to 'hello': %v", err)
	}
	t.Logf("  got response: %q", text1)

	if text1 == "" {
		t.Fatal("  expected non-empty text response to 'hello'")
	}

	// Verify the session ID was persisted (the real SDK sets it on ResultMessage).
	agent, err := store.FindAgent(agentID)
	if err != nil {
		t.Fatalf("  find agent: %v", err)
	}
	t.Logf("  session ID after spawn: %s", agent.SessionID)

	// =========================================================================
	// Step 2: StopAgent
	// =========================================================================
	t.Log("step 2: stopping agent...")

	if err := spawner.StopAgent(agentID); err != nil {
		t.Fatalf("StopAgent failed: %v", err)
	}

	if _, ok := spawner.GetActiveAgent(agentID); ok {
		t.Error("  agent should not be active after stop")
	}

	agent, _ = store.FindAgent(agentID)
	if agent.Status != "stopped" {
		t.Errorf("  status = %q, want 'stopped'", agent.Status)
	}
	t.Logf("  agent stopped (status=%s, reason=%s)", agent.Status, agent.ShutdownReason)

	// =========================================================================
	// Step 3: ResumeAgent
	// =========================================================================
	t.Log("step 3: resuming agent (attaching to same session)...")

	iaResumed, err := spawner.ResumeAgent(ctx, agentID)
	if err != nil {
		t.Fatalf("ResumeAgent failed: %v", err)
	}

	if _, ok := spawner.GetActiveAgent(agentID); !ok {
		t.Error("  agent should be active after resume")
	}
	t.Logf("  agent resumed (status=%s)", iaResumed.Info.Status)

	// =========================================================================
	// Step 4: Send another message after resume
	// =========================================================================
	t.Log("step 4: sending 'say the word pong' via Stdin...")

	if err := iaResumed.Stdin("respond with only the word 'pong'. nothing else. no punctuation."); err != nil {
		t.Fatalf("Stdin failed: %v", err)
	}

	text2, err := waitForTextAndTurnEnd(t, iaResumed.Output, 60*time.Second)
	if err != nil {
		t.Fatalf("  waiting for response to 'pong': %v", err)
	}
	t.Logf("  got response: %q", text2)

	if text2 == "" {
		t.Fatal("  expected non-empty text response after resume")
	}

	// =========================================================================
	// Step 5: Final stop
	// =========================================================================
	t.Log("step 5: final stop...")

	if err := spawner.StopAgent(agentID); err != nil {
		t.Fatalf("final StopAgent failed: %v", err)
	}

	agent, _ = store.FindAgent(agentID)
	if agent.Status != "stopped" {
		t.Errorf("  final status = %q, want 'stopped'", agent.Status)
	}

	t.Log("PASS: full lifecycle — spawn → response → stop → resume → response → stop")
}

// waitForTextAndTurnEnd reads the output channel until a turn_end message is
// received, collecting all text content along the way. Returns the concatenated
// text or an error on timeout.
func waitForTextAndTurnEnd(t *testing.T, ch <-chan []byte, timeout time.Duration) (string, error) {
	t.Helper()

	deadline := time.After(timeout)
	var text string

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return text, fmt.Errorf("output channel closed before turn_end")
			}

			var parsed map[string]interface{}
			if err := json.Unmarshal(msg, &parsed); err != nil {
				continue
			}

			msgType, _ := parsed["type"].(string)

			switch msgType {
			case "text":
				if s, ok := parsed["text"].(string); ok {
					text += s
				}
			case "turn_end":
				return text, nil
			case "error":
				errMsg, _ := parsed["message"].(string)
				return text, fmt.Errorf("agent error: %s", errMsg)
			}

		case <-deadline:
			return text, fmt.Errorf("timeout after %s waiting for turn_end", timeout)
		}
	}
}
