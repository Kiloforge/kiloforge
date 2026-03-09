package service

import (
	"fmt"
	"testing"
)

// --- Test doubles ---

type stubSkillsConfig struct {
	repo         string
	version      string
	autoUpdate   *bool
	skillsDir    string
	saved        bool
	saveErr      error
}

func (s *stubSkillsConfig) GetSkillsRepo() string           { return s.repo }
func (s *stubSkillsConfig) SetSkillsRepo(repo string)       { s.repo = repo }
func (s *stubSkillsConfig) GetSkillsVersion() string        { return s.version }
func (s *stubSkillsConfig) SetSkillsVersion(v string)       { s.version = v }
func (s *stubSkillsConfig) GetAutoUpdateSkills() *bool       { return s.autoUpdate }
func (s *stubSkillsConfig) SetAutoUpdateSkills(v *bool)      { s.autoUpdate = v }
func (s *stubSkillsConfig) GetSkillsDir() string             { return s.skillsDir }
func (s *stubSkillsConfig) Save() error                      { s.saved = true; return s.saveErr }

type stubReleaseChecker struct {
	release *SkillRelease
	err     error
}

func (s *stubReleaseChecker) LatestRelease(repo string) (*SkillRelease, error) {
	return s.release, s.err
}

type stubSkillsInstaller struct {
	installed []InstalledSkillInfo
	err       error
}

func (s *stubSkillsInstaller) Install(tarballURL, destDir string) ([]InstalledSkillInfo, error) {
	return s.installed, s.err
}

type stubManifestManager struct {
	manifest  *SkillsManifest
	modified  []ModifiedSkillInfo
	installed []InstalledSkillDetail
	checksums map[string]string
}

func (s *stubManifestManager) LoadManifest() (*SkillsManifest, error) {
	if s.manifest == nil {
		return &SkillsManifest{Checksums: map[string]string{}}, nil
	}
	return s.manifest, nil
}
func (s *stubManifestManager) SaveManifest(m *SkillsManifest) error { return nil }
func (s *stubManifestManager) ComputeChecksums(dir string) (map[string]string, error) {
	return s.checksums, nil
}
func (s *stubManifestManager) DetectModified(dir string, m *SkillsManifest) []ModifiedSkillInfo {
	return s.modified
}
func (s *stubManifestManager) ListInstalled(dir string, m *SkillsManifest) []InstalledSkillDetail {
	return s.installed
}

type stubVersionChecker struct {
	newer bool
}

func (s *stubVersionChecker) IsNewer(current, latest string) bool { return s.newer }

// --- Tests ---

func TestSkillsService_UpdateConfig(t *testing.T) {
	t.Parallel()

	t.Run("set repo", func(t *testing.T) {
		cfg := &stubSkillsConfig{}
		svc := NewSkillsService(cfg, nil, nil, nil, nil)

		repo := "owner/repo"
		result, err := svc.UpdateConfig(SkillsConfigUpdate{Repo: &repo})
		if err != nil {
			t.Fatalf("UpdateConfig: %v", err)
		}
		if !result.RepoChanged {
			t.Error("RepoChanged should be true")
		}
		if result.NewRepo != "owner/repo" {
			t.Errorf("NewRepo = %q, want %q", result.NewRepo, "owner/repo")
		}
		if !cfg.saved {
			t.Error("config should have been saved")
		}
	})

	t.Run("enable auto-update", func(t *testing.T) {
		cfg := &stubSkillsConfig{}
		svc := NewSkillsService(cfg, nil, nil, nil, nil)

		v := true
		result, err := svc.UpdateConfig(SkillsConfigUpdate{AutoUpdate: &v})
		if err != nil {
			t.Fatalf("UpdateConfig: %v", err)
		}
		if !result.AutoUpdateChanged || !result.AutoUpdateEnabled {
			t.Error("AutoUpdate should be changed and enabled")
		}
	})

	t.Run("no changes skips save", func(t *testing.T) {
		cfg := &stubSkillsConfig{}
		svc := NewSkillsService(cfg, nil, nil, nil, nil)

		_, err := svc.UpdateConfig(SkillsConfigUpdate{})
		if err != nil {
			t.Fatalf("UpdateConfig: %v", err)
		}
		if cfg.saved {
			t.Error("config should not have been saved when nothing changed")
		}
	})
}

func TestSkillsService_CheckForUpdates(t *testing.T) {
	t.Parallel()

	t.Run("no repo configured", func(t *testing.T) {
		cfg := &stubSkillsConfig{repo: ""}
		svc := NewSkillsService(cfg, nil, nil, nil, nil)

		_, err := svc.CheckForUpdates()
		if err == nil {
			t.Fatal("expected error for missing repo")
		}
	})

	t.Run("up to date", func(t *testing.T) {
		cfg := &stubSkillsConfig{repo: "owner/repo", version: "v1.0.0", skillsDir: "/tmp/skills"}
		checker := &stubReleaseChecker{release: &SkillRelease{TagName: "v1.0.0"}}
		version := &stubVersionChecker{newer: false}
		manifest := &stubManifestManager{}
		svc := NewSkillsService(cfg, checker, nil, manifest, version)

		result, err := svc.CheckForUpdates()
		if err != nil {
			t.Fatalf("CheckForUpdates: %v", err)
		}
		if !result.UpToDate {
			t.Error("should be up to date")
		}
	})

	t.Run("new version available", func(t *testing.T) {
		cfg := &stubSkillsConfig{repo: "owner/repo", version: "v1.0.0", skillsDir: "/tmp/skills"}
		checker := &stubReleaseChecker{release: &SkillRelease{TagName: "v2.0.0", TarballURL: "https://example.com/tar"}}
		version := &stubVersionChecker{newer: true}
		manifest := &stubManifestManager{}
		svc := NewSkillsService(cfg, checker, nil, manifest, version)

		result, err := svc.CheckForUpdates()
		if err != nil {
			t.Fatalf("CheckForUpdates: %v", err)
		}
		if result.UpToDate {
			t.Error("should not be up to date")
		}
		if result.NewVersion != "v2.0.0" {
			t.Errorf("NewVersion = %q, want %q", result.NewVersion, "v2.0.0")
		}
	})

	t.Run("release check fails", func(t *testing.T) {
		cfg := &stubSkillsConfig{repo: "owner/repo"}
		checker := &stubReleaseChecker{err: fmt.Errorf("network error")}
		svc := NewSkillsService(cfg, checker, nil, nil, nil)

		_, err := svc.CheckForUpdates()
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSkillsService_InstallUpdate(t *testing.T) {
	t.Parallel()

	t.Run("successful install", func(t *testing.T) {
		cfg := &stubSkillsConfig{skillsDir: "/tmp/skills"}
		installer := &stubSkillsInstaller{
			installed: []InstalledSkillInfo{
				{Name: "kf-developer", Path: "/tmp/skills/kf-developer"},
			},
		}
		manifest := &stubManifestManager{checksums: map[string]string{"kf-developer/SKILL.md": "abc123"}}
		svc := NewSkillsService(cfg, nil, installer, manifest, nil)

		release := &SkillRelease{TagName: "v2.0.0", TarballURL: "https://example.com/tar"}
		result, err := svc.InstallUpdate(release)
		if err != nil {
			t.Fatalf("InstallUpdate: %v", err)
		}
		if len(result.Installed) != 1 {
			t.Fatalf("got %d installed, want 1", len(result.Installed))
		}
		if result.Version != "v2.0.0" {
			t.Errorf("Version = %q, want %q", result.Version, "v2.0.0")
		}
		if cfg.version != "v2.0.0" {
			t.Errorf("config version = %q, want %q", cfg.version, "v2.0.0")
		}
	})

	t.Run("install failure", func(t *testing.T) {
		cfg := &stubSkillsConfig{skillsDir: "/tmp/skills"}
		installer := &stubSkillsInstaller{err: fmt.Errorf("download failed")}
		svc := NewSkillsService(cfg, nil, installer, nil, nil)

		release := &SkillRelease{TagName: "v2.0.0", TarballURL: "https://example.com/tar"}
		_, err := svc.InstallUpdate(release)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSkillsService_ListInstalledSkills(t *testing.T) {
	t.Parallel()

	cfg := &stubSkillsConfig{version: "v1.0.0", skillsDir: "/tmp/skills"}
	manifest := &stubManifestManager{
		installed: []InstalledSkillDetail{
			{Name: "kf-developer", Modified: false},
			{Name: "kf-reviewer", Modified: true},
		},
	}
	svc := NewSkillsService(cfg, nil, nil, manifest, nil)

	installed, version, dir := svc.ListInstalledSkills()
	if len(installed) != 2 {
		t.Fatalf("got %d installed, want 2", len(installed))
	}
	if version != "v1.0.0" {
		t.Errorf("version = %q, want %q", version, "v1.0.0")
	}
	if dir != "/tmp/skills" {
		t.Errorf("dir = %q, want %q", dir, "/tmp/skills")
	}
}
