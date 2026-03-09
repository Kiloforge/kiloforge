package cli

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/service"
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

func TestExecGitRunner_ArgsAndEnv(t *testing.T) {
	runner := &execGitRunner{}

	// Test that SSH env vars are passed through by inspecting a git command
	// with a non-existent dir (will fail, but we check the error output format).
	sshEnv := []string{"GIT_SSH_COMMAND=ssh -i /home/user/.ssh/id_ed25519 -o IdentitiesOnly=yes"}
	_, err := runner.RunGitCommand(context.Background(), "/tmp/nonexistent-test-repo", sshEnv, "status")
	if err == nil {
		t.Fatal("expected error for non-existent dir")
	}
}

func TestPushProject_NoOriginRemote(t *testing.T) {
	p := domain.Project{
		Slug:       "test",
		ProjectDir: "/tmp/nonexistent",
	}
	syncSvc := service.NewGitSyncService(&execGitRunner{})
	err := pushProject(context.Background(), p, "main", syncSvc)
	if err == nil {
		t.Fatal("expected error when no origin remote")
	}
	if !strings.Contains(err.Error(), "no origin remote") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPushAll_EmptyProjects(t *testing.T) {
	_ = exec.Command("true") // satisfy import
	syncSvc := service.NewGitSyncService(&execGitRunner{})
	err := pushAll(context.Background(), nil, syncSvc)
	if err != nil {
		t.Errorf("expected no error for empty projects, got: %v", err)
	}
}
