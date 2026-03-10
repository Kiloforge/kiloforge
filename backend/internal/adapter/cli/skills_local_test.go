package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallLocalSkills(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()

	installed, err := installLocalSkills(projectDir)
	if err != nil {
		t.Fatalf("installLocalSkills: %v", err)
	}

	// Should have installed at least one skill.
	if len(installed) == 0 {
		t.Fatal("expected at least one skill installed")
	}

	// Skills should be in {projectDir}/.claude/skills/.
	skillsDir := filepath.Join(projectDir, ".claude", "skills")
	if _, err := os.Stat(skillsDir); err != nil {
		t.Fatalf("skills dir not created: %v", err)
	}

	// Verify at least kf-developer was installed.
	devSkill := filepath.Join(skillsDir, "kf-developer", "SKILL.md")
	if _, err := os.Stat(devSkill); err != nil {
		t.Errorf("kf-developer SKILL.md not found: %v", err)
	}
}

func TestInstallLocalSkills_Idempotent(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()

	// First install.
	first, err := installLocalSkills(projectDir)
	if err != nil {
		t.Fatalf("first install: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("expected skills on first install")
	}

	// Second install should skip all (hashes match).
	second, err := installLocalSkills(projectDir)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if len(second) != 0 {
		t.Errorf("expected 0 skills on second install, got %d", len(second))
	}
}
