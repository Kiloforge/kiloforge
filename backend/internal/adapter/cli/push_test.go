package cli

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"crelay/internal/core/domain"
)

func TestPushCmd_FlagsRegistered(t *testing.T) {
	if f := pushCmd.Flags().Lookup("branch"); f == nil {
		t.Fatal("--branch flag not registered")
	}
	if f := pushCmd.Flags().Lookup("all"); f == nil {
		t.Fatal("--all flag not registered")
	}
}

func TestPushCmd_RequiresSlugOrAll(t *testing.T) {
	// Reset flags for test.
	flagPushAll = false
	err := runPush(pushCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no slug and no --all")
	}
	if !strings.Contains(err.Error(), "slug required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPushCmd_AllWithSlugErrors(t *testing.T) {
	flagPushAll = true
	defer func() { flagPushAll = false }()
	err := runPush(pushCmd, []string{"myapp"})
	if err == nil {
		t.Fatal("expected error when --all with slug")
	}
	if !strings.Contains(err.Error(), "cannot use --all") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGitCmd_SSHEnv(t *testing.T) {
	p := domain.Project{
		ProjectDir: "/tmp/test-repo",
		SSHKeyPath: "/home/user/.ssh/id_ed25519",
	}
	cmd := gitCmd(context.Background(), p, "status")

	// Verify -C flag.
	args := cmd.Args
	if len(args) < 4 {
		t.Fatalf("expected at least 4 args, got %d: %v", len(args), args)
	}
	if args[1] != "-C" || args[2] != "/tmp/test-repo" || args[3] != "status" {
		t.Errorf("unexpected args: %v", args)
	}

	// Verify SSH env is set. Check the last GIT_SSH_COMMAND value since
	// env vars are resolved last-wins (the system may also set one).
	var lastSSH string
	for _, e := range cmd.Env {
		if strings.HasPrefix(e, "GIT_SSH_COMMAND=") {
			lastSSH = e
		}
	}
	if lastSSH == "" {
		t.Error("GIT_SSH_COMMAND not set in env")
	} else if !strings.Contains(lastSSH, "/home/user/.ssh/id_ed25519") {
		t.Errorf("SSH command missing key path: %s", lastSSH)
	}
}

func TestGitCmd_NoSSHEnv(t *testing.T) {
	p := domain.Project{
		ProjectDir: "/tmp/test-repo",
	}
	cmd := gitCmd(context.Background(), p, "push", "origin", "main")

	for _, e := range cmd.Env {
		if strings.HasPrefix(e, "GIT_SSH_COMMAND=") {
			t.Error("GIT_SSH_COMMAND should not be set when no SSHKeyPath")
		}
	}
}

func TestGitCmd_CustomBranch(t *testing.T) {
	p := domain.Project{ProjectDir: "/tmp/test-repo"}
	cmd := gitCmd(context.Background(), p, "push", "origin", "feature-x")
	args := cmd.Args
	// git -C /tmp/test-repo push origin feature-x
	if args[len(args)-1] != "feature-x" {
		t.Errorf("expected branch 'feature-x', got args: %v", args)
	}
}

func TestPushProject_NoOriginRemote(t *testing.T) {
	p := domain.Project{
		Slug:       "test",
		ProjectDir: "/tmp/nonexistent",
	}
	err := pushProject(context.Background(), p, "main")
	if err == nil {
		t.Fatal("expected error when no origin remote")
	}
	if !strings.Contains(err.Error(), "no origin remote") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPushAll_EmptyProjects(t *testing.T) {
	// Verify --all with no projects doesn't error, just prints message.
	// We can't fully test the actual push without a real repo.
	_ = exec.Command("true") // satisfy import
	err := pushAll(context.Background(), nil)
	if err != nil {
		t.Errorf("expected no error for empty projects, got: %v", err)
	}
}
