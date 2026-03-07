package domain

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestGitSSHEnv_WithKey(t *testing.T) {
	p := Project{SSHKeyPath: "/home/user/.ssh/id_ed25519"}
	env := p.GitSSHEnv()
	if len(env) != 1 {
		t.Fatalf("want 1 env var, got %d", len(env))
	}
	want := "GIT_SSH_COMMAND=ssh -i /home/user/.ssh/id_ed25519 -o IdentitiesOnly=yes"
	if env[0] != want {
		t.Errorf("got %q, want %q", env[0], want)
	}
}

func TestGitSSHEnv_Empty(t *testing.T) {
	p := Project{}
	env := p.GitSSHEnv()
	if env != nil {
		t.Errorf("want nil, got %v", env)
	}
}

func TestGitSSHEnv_TildeExpansion(t *testing.T) {
	p := Project{SSHKeyPath: "~/.ssh/id_ed25519"}
	env := p.GitSSHEnv()
	if len(env) != 1 {
		t.Fatalf("want 1 env var, got %d", len(env))
	}
	home, _ := os.UserHomeDir()
	if !strings.Contains(env[0], home+"/.ssh/id_ed25519") {
		t.Errorf("tilde not expanded: %q", env[0])
	}
	if strings.Contains(env[0], "~/") {
		t.Errorf("tilde still present: %q", env[0])
	}
}

func TestProject_UnmarshalWithoutSSHKeyPath(t *testing.T) {
	data := `{"slug":"test","repo_name":"test","project_dir":"/tmp/test","active":true}`
	var p Project
	if err := json.Unmarshal([]byte(data), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.SSHKeyPath != "" {
		t.Errorf("want empty SSHKeyPath, got %q", p.SSHKeyPath)
	}
	if p.Slug != "test" {
		t.Errorf("want slug 'test', got %q", p.Slug)
	}
}
