package cli

import (
	"context"
	"testing"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/skills"
)

func TestPromptSkillInstall_CancelledContext(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{SkillsDir: t.TempDir()}
	missing := []skills.RequiredSkill{
		{Name: "kf-developer", Reason: "test"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := promptSkillInstall(ctx, cfg, missing, t.TempDir())
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}
