package service

import (
	"fmt"
)

// SkillsConfigStore abstracts skills-related config read/write.
type SkillsConfigStore interface {
	GetSkillsRepo() string
	SetSkillsRepo(repo string)
	GetSkillsVersion() string
	SetSkillsVersion(version string)
	GetAutoUpdateSkills() *bool
	SetAutoUpdateSkills(enabled *bool)
	GetSkillsDir() string
	Save() error
}

// SkillsReleaseChecker checks for new skill releases.
type SkillsReleaseChecker interface {
	LatestRelease(repo string) (*SkillRelease, error)
}

// SkillRelease describes a skills release.
type SkillRelease struct {
	TagName    string
	TarballURL string
}

// SkillsInstaller installs skills from a tarball URL.
type SkillsInstaller interface {
	Install(tarballURL, destDir string) ([]InstalledSkillInfo, error)
}

// InstalledSkillInfo describes a skill that was installed.
type InstalledSkillInfo struct {
	Name string
	Path string
}

// SkillsManifestManager manages skill manifest (checksums and versioning).
type SkillsManifestManager interface {
	LoadManifest() (*SkillsManifest, error)
	SaveManifest(m *SkillsManifest) error
	ComputeChecksums(skillsDir string) (map[string]string, error)
	DetectModified(skillsDir string, manifest *SkillsManifest) []ModifiedSkillInfo
	ListInstalled(skillsDir string, manifest *SkillsManifest) []InstalledSkillDetail
}

// SkillsManifest tracks installed versions and file checksums.
type SkillsManifest struct {
	Version   string
	Checksums map[string]string
}

// ModifiedSkillInfo describes a skill with local modifications.
type ModifiedSkillInfo struct {
	Name  string
	Files []string
}

// InstalledSkillDetail describes an installed skill with modification status.
type InstalledSkillDetail struct {
	Name     string
	Modified bool
}

// SkillsVersionChecker compares versions.
type SkillsVersionChecker interface {
	IsNewer(current, latest string) bool
}

// SkillsService orchestrates skills configuration, update checking,
// and installation workflows.
type SkillsService struct {
	config   SkillsConfigStore
	checker  SkillsReleaseChecker
	installer SkillsInstaller
	manifest SkillsManifestManager
	version  SkillsVersionChecker
}

// NewSkillsService creates a new SkillsService.
func NewSkillsService(
	config SkillsConfigStore,
	checker SkillsReleaseChecker,
	installer SkillsInstaller,
	manifest SkillsManifestManager,
	version SkillsVersionChecker,
) *SkillsService {
	return &SkillsService{
		config:    config,
		checker:   checker,
		installer: installer,
		manifest:  manifest,
		version:   version,
	}
}

// SkillsConfigUpdate describes a set of config changes.
type SkillsConfigUpdate struct {
	Repo         *string
	AutoUpdate   *bool
	NoAutoUpdate *bool
}

// SkillsConfigResult describes what changed.
type SkillsConfigResult struct {
	RepoChanged       bool
	AutoUpdateChanged bool
	NewRepo           string
	AutoUpdateEnabled bool
}

// UpdateConfig applies configuration changes and saves.
func (s *SkillsService) UpdateConfig(update SkillsConfigUpdate) (*SkillsConfigResult, error) {
	result := &SkillsConfigResult{}

	if update.Repo != nil && *update.Repo != "" {
		s.config.SetSkillsRepo(*update.Repo)
		result.RepoChanged = true
		result.NewRepo = *update.Repo
	}
	if update.AutoUpdate != nil && *update.AutoUpdate {
		v := true
		s.config.SetAutoUpdateSkills(&v)
		result.AutoUpdateChanged = true
		result.AutoUpdateEnabled = true
	}
	if update.NoAutoUpdate != nil && *update.NoAutoUpdate {
		v := false
		s.config.SetAutoUpdateSkills(&v)
		result.AutoUpdateChanged = true
		result.AutoUpdateEnabled = false
	}

	if result.RepoChanged || result.AutoUpdateChanged {
		if err := s.config.Save(); err != nil {
			return nil, fmt.Errorf("save config: %w", err)
		}
	}

	return result, nil
}

// UpdateCheckResult describes the result of checking for updates.
type UpdateCheckResult struct {
	UpToDate       bool
	CurrentVersion string
	NewVersion     string
	Release        *SkillRelease
	Modified       []ModifiedSkillInfo
}

// CheckForUpdates checks if a newer skills version is available.
func (s *SkillsService) CheckForUpdates() (*UpdateCheckResult, error) {
	repo := s.config.GetSkillsRepo()
	if repo == "" {
		return nil, fmt.Errorf("no skills repo configured")
	}

	rel, err := s.checker.LatestRelease(repo)
	if err != nil {
		return nil, fmt.Errorf("check for updates: %w", err)
	}

	currentVersion := s.config.GetSkillsVersion()
	if currentVersion != "" && !s.version.IsNewer(currentVersion, rel.TagName) {
		return &UpdateCheckResult{
			UpToDate:       true,
			CurrentVersion: currentVersion,
		}, nil
	}

	// Check for local modifications.
	skillsDir := s.config.GetSkillsDir()
	manifest, _ := s.manifest.LoadManifest()
	modified := s.manifest.DetectModified(skillsDir, manifest)

	return &UpdateCheckResult{
		CurrentVersion: currentVersion,
		NewVersion:     rel.TagName,
		Release:        rel,
		Modified:       modified,
	}, nil
}

// InstallUpdateResult describes the result of installing an update.
type InstallUpdateResult struct {
	Installed []InstalledSkillInfo
	Version   string
}

// InstallUpdate installs skills from a release and updates manifest/config.
func (s *SkillsService) InstallUpdate(release *SkillRelease) (*InstallUpdateResult, error) {
	skillsDir := s.config.GetSkillsDir()

	installed, err := s.installer.Install(release.TarballURL, skillsDir)
	if err != nil {
		return nil, fmt.Errorf("install skills: %w", err)
	}

	// Update manifest.
	checksums, _ := s.manifest.ComputeChecksums(skillsDir)
	manifest := &SkillsManifest{
		Version:   release.TagName,
		Checksums: checksums,
	}
	if err := s.manifest.SaveManifest(manifest); err != nil {
		// Non-fatal, log as warning.
		return &InstallUpdateResult{Installed: installed, Version: release.TagName},
			fmt.Errorf("installed but manifest save failed: %w", err)
	}

	// Update config version.
	s.config.SetSkillsVersion(release.TagName)
	if err := s.config.Save(); err != nil {
		// Non-fatal.
		return &InstallUpdateResult{Installed: installed, Version: release.TagName},
			fmt.Errorf("installed but config save failed: %w", err)
	}

	return &InstallUpdateResult{Installed: installed, Version: release.TagName}, nil
}

// ListInstalledSkills returns all installed skills with modification status.
func (s *SkillsService) ListInstalledSkills() ([]InstalledSkillDetail, string, string) {
	skillsDir := s.config.GetSkillsDir()
	manifest, _ := s.manifest.LoadManifest()
	installed := s.manifest.ListInstalled(skillsDir, manifest)
	return installed, s.config.GetSkillsVersion(), skillsDir
}
