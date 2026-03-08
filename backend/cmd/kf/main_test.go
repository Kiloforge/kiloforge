package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestBinaryBuilds(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("go", "build", "-o", t.TempDir()+"/kf", ".")
	// Set GIT_DIR and GIT_WORK_TREE so VCS stamping works in git worktrees.
	cmd.Env = append(cmd.Environ(), "GIT_DIR="+gitCommonDir(t), "GIT_WORK_TREE="+gitWorkTree(t))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("binary build failed: %v\n%s", err, out)
	}
}

func gitCommonDir(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--git-common-dir").CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse --git-common-dir: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

func gitWorkTree(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}
