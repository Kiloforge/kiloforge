package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckRequired(t *testing.T) {
	globalDir := t.TempDir()
	localDir := t.TempDir()

	// Install "kf-developer" globally.
	devDir := filepath.Join(globalDir, "kf-developer")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devDir, "SKILL.md"), []byte("# Dev"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Install "kf-reviewer" locally.
	revDir := filepath.Join(localDir, "kf-reviewer")
	if err := os.MkdirAll(revDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(revDir, "SKILL.md"), []byte("# Rev"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a directory without SKILL.md.
	os.MkdirAll(filepath.Join(globalDir, "empty-skill"), 0o755)

	tests := []struct {
		name      string
		required  []RequiredSkill
		globalDir string
		localDir  string
		wantCount int
		wantNames []string
	}{
		{
			name:      "found globally",
			required:  []RequiredSkill{{Name: "kf-developer", Reason: "dev"}},
			globalDir: globalDir,
			localDir:  localDir,
			wantCount: 0,
		},
		{
			name:      "found locally",
			required:  []RequiredSkill{{Name: "kf-reviewer", Reason: "rev"}},
			globalDir: globalDir,
			localDir:  localDir,
			wantCount: 0,
		},
		{
			name:      "missing skill",
			required:  []RequiredSkill{{Name: "kf-architect", Reason: "track gen"}},
			globalDir: globalDir,
			localDir:  localDir,
			wantCount: 1,
			wantNames: []string{"kf-architect"},
		},
		{
			name: "mixed found and missing",
			required: []RequiredSkill{
				{Name: "kf-developer", Reason: "dev"},
				{Name: "kf-architect", Reason: "track gen"},
			},
			globalDir: globalDir,
			localDir:  localDir,
			wantCount: 1,
			wantNames: []string{"kf-architect"},
		},
		{
			name: "both missing",
			required: []RequiredSkill{
				{Name: "foo", Reason: "foo"},
				{Name: "bar", Reason: "bar"},
			},
			globalDir: globalDir,
			localDir:  localDir,
			wantCount: 2,
			wantNames: []string{"foo", "bar"},
		},
		{
			name:      "empty dirs",
			required:  []RequiredSkill{{Name: "kf-developer", Reason: "dev"}},
			globalDir: "",
			localDir:  "",
			wantCount: 1,
			wantNames: []string{"kf-developer"},
		},
		{
			name:      "no required skills",
			required:  nil,
			globalDir: globalDir,
			localDir:  localDir,
			wantCount: 0,
		},
		{
			name:      "dir without SKILL.md",
			required:  []RequiredSkill{{Name: "empty-skill", Reason: "test"}},
			globalDir: globalDir,
			localDir:  localDir,
			wantCount: 1,
			wantNames: []string{"empty-skill"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			missing := CheckRequired(tt.required, tt.globalDir, tt.localDir)
			if len(missing) != tt.wantCount {
				t.Errorf("got %d missing, want %d", len(missing), tt.wantCount)
			}
			for i, name := range tt.wantNames {
				if i >= len(missing) {
					break
				}
				if missing[i].Name != name {
					t.Errorf("missing[%d].Name = %q, want %q", i, missing[i].Name, name)
				}
			}
		})
	}
}

func TestCheckStatus(t *testing.T) {
	globalDir := t.TempDir()

	// Install kf-developer with matching embedded content.
	devDir := filepath.Join(globalDir, "kf-developer")
	os.MkdirAll(devDir, 0o755)

	// Read embedded content and install the same.
	embeddedData, err := embeddedSkills.ReadFile("embedded/kf-developer/SKILL.md")
	if err != nil {
		t.Fatalf("read embedded: %v", err)
	}
	os.WriteFile(filepath.Join(devDir, "SKILL.md"), embeddedData, 0o644)

	// Install kf-reviewer with different content (outdated).
	revDir := filepath.Join(globalDir, "kf-reviewer")
	os.MkdirAll(revDir, 0o755)
	os.WriteFile(filepath.Join(revDir, "SKILL.md"), []byte("# Old content"), 0o644)

	required := []RequiredSkill{
		{Name: "kf-developer", Reason: "dev"},
		{Name: "kf-reviewer", Reason: "rev"},
		{Name: "kf-architect", Reason: "gen"},
	}

	statuses := CheckStatus(required, globalDir, "")

	// kf-developer: installed + current.
	if !statuses[0].Installed || !statuses[0].Current {
		t.Errorf("kf-developer: installed=%v current=%v, want true/true", statuses[0].Installed, statuses[0].Current)
	}
	// kf-reviewer: installed but outdated.
	if !statuses[1].Installed || statuses[1].Current {
		t.Errorf("kf-reviewer: installed=%v current=%v, want true/false", statuses[1].Installed, statuses[1].Current)
	}
	// kf-architect: not installed.
	if statuses[2].Installed {
		t.Errorf("kf-architect: installed=%v, want false", statuses[2].Installed)
	}
}

func TestInstallEmbedded(t *testing.T) {
	destDir := t.TempDir()

	path, err := InstallEmbedded("kf-developer", destDir)
	if err != nil {
		t.Fatalf("InstallEmbedded: %v", err)
	}

	// Verify SKILL.md exists.
	skillFile := filepath.Join(path, "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		t.Errorf("SKILL.md not found at %s", skillFile)
	}

	// Verify content matches embedded.
	installed, _ := os.ReadFile(skillFile)
	embedded, _ := embeddedSkills.ReadFile("embedded/kf-developer/SKILL.md")
	if string(installed) != string(embedded) {
		t.Error("installed content does not match embedded")
	}
}

func TestInstallEmbedded_WithSubdirs(t *testing.T) {
	destDir := t.TempDir()

	// kf-manage has subdirectories (resources/, scripts/).
	path, err := InstallEmbedded("kf-manage", destDir)
	if err != nil {
		t.Fatalf("InstallEmbedded: %v", err)
	}

	// Verify SKILL.md exists.
	if _, err := os.Stat(filepath.Join(path, "SKILL.md")); err != nil {
		t.Error("SKILL.md not found")
	}

	// Verify subdirectories exist.
	if _, err := os.Stat(filepath.Join(path, "resources")); err != nil {
		t.Error("resources/ not found")
	}
	if _, err := os.Stat(filepath.Join(path, "scripts")); err != nil {
		t.Error("scripts/ not found")
	}
}

func TestInstallEmbedded_NotFound(t *testing.T) {
	_, err := InstallEmbedded("nonexistent-skill", t.TempDir())
	if err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
}

func TestListEmbedded(t *testing.T) {
	names := ListEmbedded()
	if len(names) == 0 {
		t.Fatal("expected embedded skills")
	}

	// Verify known skills are present.
	nameSet := map[string]bool{}
	for _, n := range names {
		nameSet[n] = true
	}
	for _, expected := range []string{"kf-developer", "kf-reviewer", "kf-architect"} {
		if !nameSet[expected] {
			t.Errorf("expected %q in embedded skills", expected)
		}
	}
}

func TestRequiredSkillsForRole(t *testing.T) {
	tests := []struct {
		role     string
		wantName string
	}{
		{"developer", "kf-developer"},
		{"reviewer", "kf-reviewer"},
		{"interactive", "kf-architect"},
		{"architect", "kf-architect"},
		{"product-advisor", "kf-product-advisor"},
		{"setup", "kf-setup"},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			skills := RequiredSkillsForRole(tt.role)
			if len(skills) != 1 {
				t.Fatalf("got %d skills, want 1", len(skills))
			}
			if skills[0].Name != tt.wantName {
				t.Errorf("name = %q, want %q", skills[0].Name, tt.wantName)
			}
		})
	}

	// Unknown role returns nil.
	if skills := RequiredSkillsForRole("unknown"); skills != nil {
		t.Errorf("unknown role should return nil, got %v", skills)
	}
}

func TestSkillCommandForRole(t *testing.T) {
	tests := []struct {
		role string
		want string
	}{
		{"architect", "/kf-architect"},
		{"product-advisor", "/kf-product-advisor"},
		{"interactive", ""},
		{"developer", ""},
		{"reviewer", ""},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := SkillCommandForRole(tt.role)
			if got != tt.want {
				t.Errorf("SkillCommandForRole(%q) = %q, want %q", tt.role, got, tt.want)
			}
		})
	}
}
