package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildMockAgent builds the mock-agent binary to the given directory and returns its path.
func buildMockAgent(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "mock-agent")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	// Worktree-safe VCS env.
	cmd.Env = append(os.Environ(),
		"GIT_DIR="+gitCommonDir(t),
		"GIT_WORK_TREE="+gitWorkTree(t),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build mock-agent: %v\n%s", err, out)
	}
	return bin
}

func gitCommonDir(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--git-common-dir").CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse --git-common-dir: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func gitWorkTree(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func TestDefaultOutput(t *testing.T) {
	t.Parallel()
	bin := buildMockAgent(t)

	cmd := exec.Command(bin, "--output-format", "stream-json")
	cmd.Env = append(os.Environ(), "MOCK_AGENT_DELAY=0")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("mock-agent failed: %v", err)
	}

	lines := nonEmpty(strings.Split(string(out), "\n"))
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}

	// Verify init event.
	var init map[string]any
	mustUnmarshal(t, lines[0], &init)
	if init["type"] != "init" {
		t.Errorf("line 0: expected type=init, got %v", init["type"])
	}
	if init["session_id"] != "mock-session-001" {
		t.Errorf("line 0: expected session_id=mock-session-001, got %v", init["session_id"])
	}

	// Verify content_block_delta.
	var delta map[string]any
	mustUnmarshal(t, lines[1], &delta)
	if delta["type"] != "content_block_delta" {
		t.Errorf("line 1: expected type=content_block_delta, got %v", delta["type"])
	}

	// Verify result.
	var result map[string]any
	mustUnmarshal(t, lines[2], &result)
	if result["type"] != "result" {
		t.Errorf("line 2: expected type=result, got %v", result["type"])
	}
}

func TestCustomEvents(t *testing.T) {
	t.Parallel()
	bin := buildMockAgent(t)

	customEvents := `[{"type":"init","session_id":"custom-001"},{"type":"result","usage":{}}]`
	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(),
		"MOCK_AGENT_EVENTS="+customEvents,
		"MOCK_AGENT_DELAY=0",
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("mock-agent failed: %v", err)
	}

	lines := nonEmpty(strings.Split(string(out), "\n"))
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var init map[string]any
	mustUnmarshal(t, lines[0], &init)
	if init["session_id"] != "custom-001" {
		t.Errorf("expected session_id=custom-001, got %v", init["session_id"])
	}
}

func TestInteractiveMode(t *testing.T) {
	t.Parallel()
	bin := buildMockAgent(t)

	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(),
		"MOCK_AGENT_INTERACTIVE=true",
		"MOCK_AGENT_DELAY=0",
	)
	cmd.Stdin = strings.NewReader("hello world\n\n")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("mock-agent failed: %v", err)
	}

	lines := nonEmpty(strings.Split(string(out), "\n"))
	// Expect: init, content_block_delta (echo), result
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}

	// First line is init.
	var init map[string]any
	mustUnmarshal(t, lines[0], &init)
	if init["type"] != "init" {
		t.Errorf("expected type=init, got %v", init["type"])
	}

	// Second line should echo "hello world".
	var delta map[string]any
	mustUnmarshal(t, lines[1], &delta)
	if delta["type"] != "content_block_delta" {
		t.Errorf("expected type=content_block_delta, got %v", delta["type"])
	}
	d, _ := delta["delta"].(map[string]any)
	if d["text"] != "hello world" {
		t.Errorf("expected echoed text 'hello world', got %v", d["text"])
	}

	// Third line is result.
	var result map[string]any
	mustUnmarshal(t, lines[2], &result)
	if result["type"] != "result" {
		t.Errorf("expected type=result, got %v", result["type"])
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()
	bin := buildMockAgent(t)

	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(),
		"MOCK_AGENT_EXIT_CODE=42",
		"MOCK_AGENT_DELAY=0",
	)
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 42 {
		t.Errorf("expected exit code 42, got %d", exitErr.ExitCode())
	}
}

func TestFailAfter(t *testing.T) {
	t.Parallel()
	bin := buildMockAgent(t)

	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(),
		"MOCK_AGENT_FAIL_AFTER=1",
		"MOCK_AGENT_DELAY=0",
	)
	out, err := cmd.Output()
	if err == nil {
		t.Fatal("expected non-zero exit code from fail-after")
	}

	// Should have emitted exactly 1 event before crashing.
	lines := nonEmpty(strings.Split(string(out), "\n"))
	if len(lines) != 1 {
		t.Fatalf("expected 1 line before crash, got %d: %v", len(lines), lines)
	}
}

func TestDelay(t *testing.T) {
	t.Parallel()
	bin := buildMockAgent(t)

	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "MOCK_AGENT_DELAY=50")
	start := time.Now()
	out, err := cmd.Output()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("mock-agent failed: %v", err)
	}

	lines := nonEmpty(strings.Split(string(out), "\n"))
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	// With 3 events and 50ms delay between, expect at least ~100ms.
	if elapsed < 80*time.Millisecond {
		t.Errorf("expected at least 80ms for delays, got %v", elapsed)
	}
}

func nonEmpty(lines []string) []string {
	var result []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			result = append(result, l)
		}
	}
	return result
}

func mustUnmarshal(t *testing.T, data string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(data), v); err != nil {
		t.Fatalf("unmarshal %q: %v", data, err)
	}
}
