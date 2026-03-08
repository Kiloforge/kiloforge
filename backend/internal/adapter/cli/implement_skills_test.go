package cli

import (
	"context"
	"testing"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/skills"
)

func TestPromptSkillInstall_NoRepo(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{SkillsDir: t.TempDir()}
	missing := []skills.RequiredSkill{
		{Name: "conductor-developer", Reason: "test"},
	}

	err := promptSkillInstall(context.Background(), cfg, missing, t.TempDir())
	if err == nil {
		t.Fatal("expected error when skills repo not configured")
	}
	if err.Error() != "skills repo not configured — run 'kf skills --repo owner/repo' first" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPromptSkillInstall_CancelledContext(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		SkillsDir:  t.TempDir(),
		SkillsRepo: "test/skills",
	}
	missing := []skills.RequiredSkill{
		{Name: "conductor-developer", Reason: "test"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := promptSkillInstall(ctx, cfg, missing, t.TempDir())
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}
