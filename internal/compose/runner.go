package compose

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	ComposeFileName = "docker-compose.yml"
	ProjectName     = "crelay"
)

// Runner abstracts docker compose CLI invocation, supporting both v2 (docker compose) and v1 (docker-compose).
type Runner struct {
	args []string // e.g., ["docker", "compose"] or ["docker-compose"]
}

// Detect returns a Runner configured for the available CLI variant.
// It tries v2 (docker compose) first, then falls back to v1 (docker-compose).
func Detect() (*Runner, error) {
	// Try v2: docker compose version
	if err := exec.Command("docker", "compose", "version").Run(); err == nil {
		return &Runner{args: []string{"docker", "compose"}}, nil
	}

	// Try v1: docker-compose version
	if err := exec.Command("docker-compose", "version").Run(); err == nil {
		return &Runner{args: []string{"docker-compose"}}, nil
	}

	return nil, fmt.Errorf(
		"docker compose not found: install Docker Desktop (includes compose v2) " +
			"or install docker-compose standalone")
}

// Version returns the compose CLI version string.
func (r *Runner) Version() string {
	cmd := exec.Command(r.args[0], append(r.args[1:], "version", "--short")...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// command builds an exec.Cmd for a compose operation.
func (r *Runner) command(ctx context.Context, composeDir string, composeArgs ...string) *exec.Cmd {
	cmdArgs := make([]string, len(r.args)-1)
	copy(cmdArgs, r.args[1:])
	cmdArgs = append(cmdArgs, "-f", filepath.Join(composeDir, ComposeFileName), "-p", ProjectName)
	cmdArgs = append(cmdArgs, composeArgs...)
	return exec.CommandContext(ctx, r.args[0], cmdArgs...)
}

// Up runs `docker compose up -d`.
func (r *Runner) Up(ctx context.Context, composeDir string) error {
	cmd := r.command(ctx, composeDir, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Down runs `docker compose down`, optionally removing volumes.
func (r *Runner) Down(ctx context.Context, composeDir string, removeVolumes bool) error {
	args := []string{"down"}
	if removeVolumes {
		args = append(args, "--volumes")
	}
	cmd := r.command(ctx, composeDir, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Ps runs `docker compose ps` and returns the output.
func (r *Runner) Ps(ctx context.Context, composeDir string) (string, error) {
	cmd := r.command(ctx, composeDir, "ps", "--format", "table")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return out.String(), err
}

// Exec runs a command inside a service container.
func (r *Runner) Exec(ctx context.Context, composeDir string, service string, cmdArgs ...string) ([]byte, error) {
	args := append([]string{"exec", "-T", service}, cmdArgs...)
	cmd := r.command(ctx, composeDir, args...)
	return cmd.CombinedOutput()
}
