// mock-agent is a test binary that simulates Claude CLI stream-JSON output.
//
// It accepts the same flags as the claude CLI (--output-format, --model, --verbose,
// --print, -p) and outputs configurable stream-JSON events to stdout.
//
// Behavior is controlled via environment variables:
//
//   - MOCK_AGENT_EVENTS:      JSON array of stream-JSON events to emit (overrides defaults)
//   - MOCK_AGENT_DELAY:       Milliseconds between events (default: 100)
//   - MOCK_AGENT_EXIT_CODE:   Process exit code (default: 0)
//   - MOCK_AGENT_INTERACTIVE: Enable stdin echo mode ("true"/"false", default: "false")
//   - MOCK_AGENT_FAIL_AFTER:  Emit N events then exit(1) (disabled if unset or 0)
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Default stream-JSON events emitted when MOCK_AGENT_EVENTS is not set.
var defaultEvents = []json.RawMessage{
	json.RawMessage(`{"type":"init","session_id":"mock-session-001"}`),
	json.RawMessage(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello from mock agent"}}`),
	json.RawMessage(`{"type":"result","usage":{"input_tokens":100,"output_tokens":50},"cost":{"input_cost":0.001,"output_cost":0.0005}}`),
}

func main() {
	os.Exit(run())
}

func run() int {
	// Accept claude CLI flags (ignored, but parsed for compatibility).
	_ = flag.String("output-format", "stream-json", "output format")
	_ = flag.String("model", "", "model name")
	_ = flag.Bool("verbose", false, "verbose output")
	_ = flag.String("print", "", "print mode (non-interactive)")
	flag.StringVar(new(string), "p", "", "shorthand for --print")
	flag.Parse()

	delay := envDuration("MOCK_AGENT_DELAY", 100*time.Millisecond)
	exitCode := envInt("MOCK_AGENT_EXIT_CODE", 0)
	interactive := envBool("MOCK_AGENT_INTERACTIVE", false)
	failAfter := envInt("MOCK_AGENT_FAIL_AFTER", 0)

	if interactive {
		return runInteractive(delay, exitCode, failAfter)
	}
	return runNonInteractive(delay, exitCode, failAfter)
}

func runNonInteractive(delay time.Duration, exitCode, failAfter int) int {
	events := loadEvents()
	for i, ev := range events {
		if failAfter > 0 && i >= failAfter {
			fmt.Fprintf(os.Stderr, "mock-agent: failing after %d events\n", failAfter)
			return 1
		}
		fmt.Fprintln(os.Stdout, string(ev))
		if i < len(events)-1 {
			time.Sleep(delay)
		}
	}
	return exitCode
}

func runInteractive(delay time.Duration, exitCode, failAfter int) int {
	// Emit init event.
	emitted := 0
	initEvent := `{"type":"init","session_id":"mock-session-001"}`
	fmt.Fprintln(os.Stdout, initEvent)
	emitted++

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		if failAfter > 0 && emitted >= failAfter {
			fmt.Fprintf(os.Stderr, "mock-agent: failing after %d events\n", failAfter)
			return 1
		}
		// Echo input as content_block_delta.
		delta := map[string]any{
			"type": "content_block_delta",
			"delta": map[string]any{
				"type": "text_delta",
				"text": line,
			},
		}
		data, _ := json.Marshal(delta)
		fmt.Fprintln(os.Stdout, string(data))
		emitted++
		time.Sleep(delay)
	}

	// Emit result event.
	result := `{"type":"result","usage":{"input_tokens":100,"output_tokens":50},"cost":{"input_cost":0.001,"output_cost":0.0005}}`
	fmt.Fprintln(os.Stdout, result)
	return exitCode
}

func loadEvents() []json.RawMessage {
	raw := os.Getenv("MOCK_AGENT_EVENTS")
	if raw == "" {
		return defaultEvents
	}
	var events []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &events); err != nil {
		fmt.Fprintf(os.Stderr, "mock-agent: invalid MOCK_AGENT_EVENTS JSON: %v\n", err)
		return defaultEvents
	}
	return events
}

func envDuration(key string, defaultVal time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	ms, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return time.Duration(ms) * time.Millisecond
}

func envInt(key string, defaultVal int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return v
}

func envBool(key string, defaultVal bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	return raw == "true" || raw == "1"
}
