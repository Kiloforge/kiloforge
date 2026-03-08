package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckRequired(t *testing.T) {
	// Setup: create temp dirs for global and local skills.
	globalDir := t.TempDir()
	localDir := t.TempDir()

	// Install "conductor-developer" globally.
	devDir := filepath.Join(globalDir, "conductor-developer")
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devDir, "SKILL.md"), []byte("# Dev"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Install "conductor-reviewer" locally.
	revDir := filepath.Join(localDir, "conductor-reviewer")
	if err := os.MkdirAll(revDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(revDir, "SKILL.md"), []byte("# Rev"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		required   []RequiredSkill
		globalDir  string
		localDir   string
		wantCount  int
		wantNames  []string
	}{
		{
			name: "all found globally",
			required: []RequiredSkill{
				{Name: "conductor-developer", Reason: "dev"},
			},
			globalDir: globalDir,
			localDir:  localDir,
			wantCount: 0,
		},
		{
			name: "found locally",
			required: []RequiredSkill{
				{Name: "conductor-reviewer", Reason: "rev"},
			},
			globalDir: globalDir,
			localDir:  localDir,
			wantCount: 0,
		},
		{
			name: "missing skill",
			required: []RequiredSkill{
				{Name: "conductor-track-generator", Reason: "track gen"},
			},
			globalDir: globalDir,
			localDir:  localDir,
			wantCount: 1,
			wantNames: []string{"conductor-track-generator"},
		},
		{
			name: "mixed found and missing",
			required: []RequiredSkill{
				{Name: "conductor-developer", Reason: "dev"},
				{Name: "conductor-track-generator", Reason: "track gen"},
			},
			globalDir: globalDir,
			localDir:  localDir,
			wantCount: 1,
			wantNames: []string{"conductor-track-generator"},
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
			name: "empty dirs",
			required: []RequiredSkill{
				{Name: "conductor-developer", Reason: "dev"},
			},
			globalDir: "",
			localDir:  "",
			wantCount: 1,
			wantNames: []string{"conductor-developer"},
		},
		{
			name:      "no required skills",
			required:  nil,
			globalDir: globalDir,
			localDir:  localDir,
			wantCount: 0,
		},
		{
			name: "dir without SKILL.md",
			required: []RequiredSkill{
				{Name: "empty-skill", Reason: "test"},
			},
			globalDir: globalDir,
			localDir:  localDir,
			wantCount: 1,
			wantNames: []string{"empty-skill"},
		},
	}

	// Create a directory without SKILL.md for the last test case.
	os.MkdirAll(filepath.Join(globalDir, "empty-skill"), 0o755)

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
