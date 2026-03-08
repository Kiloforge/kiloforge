package main

import (
	"os/exec"
	"testing"
)

func TestBinaryBuilds(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", t.TempDir()+"/kf", ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("binary build failed: %v\n%s", err, out)
	}
}
