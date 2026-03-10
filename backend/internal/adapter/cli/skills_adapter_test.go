package cli

import (
	"testing"

	"kiloforge/internal/adapter/config"
)

func TestConfigSkillsAdapter_GetSet(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		SkillsRepo:    "org/skills",
		SkillsVersion: "v1.0.0",
	}
	adapter := &configSkillsAdapter{cfg: cfg}

	if got := adapter.GetSkillsRepo(); got != "org/skills" {
		t.Errorf("GetSkillsRepo() = %q, want %q", got, "org/skills")
	}
	if got := adapter.GetSkillsVersion(); got != "v1.0.0" {
		t.Errorf("GetSkillsVersion() = %q, want %q", got, "v1.0.0")
	}

	adapter.SetSkillsRepo("new/repo")
	if got := adapter.GetSkillsRepo(); got != "new/repo" {
		t.Errorf("after SetSkillsRepo, GetSkillsRepo() = %q, want %q", got, "new/repo")
	}

	adapter.SetSkillsVersion("v2.0.0")
	if got := adapter.GetSkillsVersion(); got != "v2.0.0" {
		t.Errorf("after SetSkillsVersion, GetSkillsVersion() = %q, want %q", got, "v2.0.0")
	}

	b := true
	adapter.SetAutoUpdateSkills(&b)
	got := adapter.GetAutoUpdateSkills()
	if got == nil || !*got {
		t.Error("after SetAutoUpdateSkills(true), expected true")
	}
}

func TestVersionCheckerAdapter_IsNewer(t *testing.T) {
	t.Parallel()

	checker := &versionCheckerAdapter{}

	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"v1.0.0", "v2.0.0", true},
		{"v2.0.0", "v1.0.0", false},
		{"v1.0.0", "v1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.current+"→"+tt.latest, func(t *testing.T) {
			got := checker.IsNewer(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}
