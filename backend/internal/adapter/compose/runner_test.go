package compose

import (
	"context"
	"testing"
)

func TestDetect_FindsAvailableVariant(t *testing.T) {
	t.Parallel()

	runner, err := Detect()
	if err != nil {
		t.Skipf("no docker compose available: %v", err)
	}

	if len(runner.args) == 0 {
		t.Fatal("expected non-empty args")
	}

	// Should be either ["docker", "compose"] or ["docker-compose"]
	if runner.args[0] != "docker" && runner.args[0] != "docker-compose" {
		t.Errorf("unexpected first arg: %s", runner.args[0])
	}
}

func TestRunner_Version(t *testing.T) {
	t.Parallel()

	runner, err := Detect()
	if err != nil {
		t.Skipf("no docker compose available: %v", err)
	}

	v := runner.Version()
	if v == "" {
		t.Error("expected non-empty version")
	}
}

func TestRunner_CommandBuilding(t *testing.T) {
	t.Parallel()

	// Use a known runner for deterministic testing.
	runner := &Runner{args: []string{"docker", "compose"}}

	cmd := runner.command(context.Background(), "/tmp/test", "up", "-d")
	args := cmd.Args

	// Should contain: docker compose -f /tmp/test/docker-compose.yml -p crelay up -d
	expected := []string{
		"docker", "compose",
		"-f", "/tmp/test/docker-compose.yml",
		"-p", "crelay",
		"up", "-d",
	}

	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}

	for i, want := range expected {
		if args[i] != want {
			t.Errorf("arg[%d]: want %q, got %q", i, want, args[i])
		}
	}
}

func TestRunner_StopCommandBuilding(t *testing.T) {
	t.Parallel()

	runner := &Runner{args: []string{"docker", "compose"}}

	cmd := runner.command(context.Background(), "/tmp/test", "stop")
	args := cmd.Args

	// Should contain: docker compose -f /tmp/test/docker-compose.yml -p crelay stop
	expected := []string{
		"docker", "compose",
		"-f", "/tmp/test/docker-compose.yml",
		"-p", "crelay",
		"stop",
	}

	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}

	for i, want := range expected {
		if args[i] != want {
			t.Errorf("arg[%d]: want %q, got %q", i, want, args[i])
		}
	}
}
