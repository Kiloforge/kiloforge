package skills

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func createTestTarball(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add top-level directory (GitHub-style).
	tw.WriteHeader(&tar.Header{
		Name:     "owner-repo-abc123/",
		Typeflag: tar.TypeDir,
		Mode:     0o755,
	})

	for name, content := range files {
		tw.WriteHeader(&tar.Header{
			Name:     "owner-repo-abc123/" + name,
			Size:     int64(len(content)),
			Mode:     0o644,
			Typeflag: tar.TypeReg,
		})
		tw.Write([]byte(content))
	}
	tw.Close()
	gw.Close()
	return buf.Bytes()
}

func TestInstall_ExtractsSkills(t *testing.T) {
	t.Parallel()
	tarball := createTestTarball(t, map[string]string{
		"kf-developer/SKILL.md": "# Developer Skill",
		"kf-reviewer/SKILL.md":  "# Reviewer Skill",
		"README.md":                    "# Not a skill",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(tarball)
	}))
	defer srv.Close()

	destDir := t.TempDir()
	inst := NewInstallerWith(srv.Client())
	installed, err := inst.Install(srv.URL, destDir)
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}
	if len(installed) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(installed))
	}

	// Verify files exist.
	for _, s := range installed {
		skillFile := filepath.Join(s.Path, "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			t.Errorf("skill %s SKILL.md not found: %v", s.Name, err)
		}
	}
}

func TestInstall_DownloadError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	inst := NewInstallerWith(srv.Client())
	_, err := inst.Install(srv.URL, t.TempDir())
	if err == nil {
		t.Fatal("expected error for failed download")
	}
}

func TestListInstalled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create two skill directories.
	os.MkdirAll(filepath.Join(dir, "skill-a"), 0o755)
	os.WriteFile(filepath.Join(dir, "skill-a", "SKILL.md"), []byte("# A"), 0o644)
	os.MkdirAll(filepath.Join(dir, "skill-b"), 0o755)
	os.WriteFile(filepath.Join(dir, "skill-b", "SKILL.md"), []byte("# B"), 0o644)
	// Create a non-skill directory.
	os.MkdirAll(filepath.Join(dir, "not-a-skill"), 0o755)
	os.WriteFile(filepath.Join(dir, "not-a-skill", "README.md"), []byte("nope"), 0o644)

	manifest := &Manifest{Checksums: map[string]string{}}
	skills := ListInstalled(dir, manifest)
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}
}

func TestDetectModified(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create a skill.
	os.MkdirAll(filepath.Join(dir, "my-skill"), 0o755)
	os.WriteFile(filepath.Join(dir, "my-skill", "SKILL.md"), []byte("original"), 0o644)

	// Compute checksums.
	checksums, err := ComputeChecksums(dir)
	if err != nil {
		t.Fatalf("ComputeChecksums: %v", err)
	}
	manifest := &Manifest{Checksums: checksums}

	// No modifications yet.
	modified := DetectModified(dir, manifest)
	if len(modified) != 0 {
		t.Errorf("expected 0 modified, got %d", len(modified))
	}

	// Modify the file.
	os.WriteFile(filepath.Join(dir, "my-skill", "SKILL.md"), []byte("modified!"), 0o644)
	modified = DetectModified(dir, manifest)
	if len(modified) != 1 {
		t.Fatalf("expected 1 modified, got %d", len(modified))
	}
	if modified[0].Name != "my-skill" {
		t.Errorf("modified skill name = %q, want my-skill", modified[0].Name)
	}
}

func TestIsNewer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		current, latest string
		want            bool
	}{
		{"v1.0.0", "v1.0.1", true},
		{"v1.0.0", "v1.1.0", true},
		{"v1.0.0", "v2.0.0", true},
		{"v1.2.3", "v1.2.3", false},
		{"v2.0.0", "v1.9.9", false},
		{"1.0.0", "v1.0.1", true},
		{"v1.0.0", "1.0.1", true},
		{"", "v1.0.0", false},
		{"v1.0.0", "", false},
		{"invalid", "v1.0.0", false},
	}
	for _, tt := range tests {
		got := IsNewer(tt.current, tt.latest)
		if got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
		}
	}
}
