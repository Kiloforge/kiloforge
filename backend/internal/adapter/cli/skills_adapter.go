package cli

import (
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/skills"
	"kiloforge/internal/core/service"
)

// configSkillsAdapter wraps config.Config to implement service.SkillsConfigStore.
type configSkillsAdapter struct {
	cfg *config.Config
}

func (a *configSkillsAdapter) GetSkillsRepo() string       { return a.cfg.SkillsRepo }
func (a *configSkillsAdapter) SetSkillsRepo(repo string)   { a.cfg.SkillsRepo = repo }
func (a *configSkillsAdapter) GetSkillsVersion() string    { return a.cfg.SkillsVersion }
func (a *configSkillsAdapter) SetSkillsVersion(v string)   { a.cfg.SkillsVersion = v }
func (a *configSkillsAdapter) GetAutoUpdateSkills() *bool  { return a.cfg.AutoUpdateSkills }
func (a *configSkillsAdapter) SetAutoUpdateSkills(v *bool) { a.cfg.AutoUpdateSkills = v }
func (a *configSkillsAdapter) GetSkillsDir() string        { return a.cfg.GetSkillsDir() }
func (a *configSkillsAdapter) Save() error                 { return a.cfg.Save() }

// releaseCheckerAdapter wraps skills.GitHubClient to implement service.SkillsReleaseChecker.
type releaseCheckerAdapter struct {
	client *skills.GitHubClient
}

func (a *releaseCheckerAdapter) LatestRelease(repo string) (*service.SkillRelease, error) {
	rel, err := a.client.LatestRelease(repo)
	if err != nil {
		return nil, err
	}
	return &service.SkillRelease{
		TagName:    rel.TagName,
		TarballURL: rel.TarballURL,
	}, nil
}

// installerAdapter wraps skills.Installer to implement service.SkillsInstaller.
type installerAdapter struct {
	inst *skills.Installer
}

func (a *installerAdapter) Install(tarballURL, destDir string) ([]service.InstalledSkillInfo, error) {
	installed, err := a.inst.Install(tarballURL, destDir)
	if err != nil {
		return nil, err
	}
	result := make([]service.InstalledSkillInfo, len(installed))
	for i, s := range installed {
		result[i] = service.InstalledSkillInfo{Name: s.Name, Path: s.Path}
	}
	return result, nil
}

// manifestAdapter wraps skills manifest functions to implement service.SkillsManifestManager.
type manifestAdapter struct{}

func (a *manifestAdapter) LoadManifest() (*service.SkillsManifest, error) {
	m, err := skills.LoadManifest()
	if err != nil {
		return nil, err
	}
	return &service.SkillsManifest{Version: m.Version, Checksums: m.Checksums}, nil
}

func (a *manifestAdapter) SaveManifest(m *service.SkillsManifest) error {
	sm := &skills.Manifest{Version: m.Version, Checksums: m.Checksums}
	return sm.Save()
}

func (a *manifestAdapter) ComputeChecksums(skillsDir string) (map[string]string, error) {
	return skills.ComputeChecksums(skillsDir)
}

func (a *manifestAdapter) DetectModified(skillsDir string, m *service.SkillsManifest) []service.ModifiedSkillInfo {
	sm := &skills.Manifest{Version: m.Version, Checksums: m.Checksums}
	modified := skills.DetectModified(skillsDir, sm)
	result := make([]service.ModifiedSkillInfo, len(modified))
	for i, mod := range modified {
		result[i] = service.ModifiedSkillInfo{Name: mod.Name, Files: mod.Files}
	}
	return result
}

func (a *manifestAdapter) ListInstalled(skillsDir string, m *service.SkillsManifest) []service.InstalledSkillDetail {
	sm := &skills.Manifest{Version: m.Version, Checksums: m.Checksums}
	installed := skills.ListInstalled(skillsDir, sm)
	result := make([]service.InstalledSkillDetail, len(installed))
	for i, s := range installed {
		result[i] = service.InstalledSkillDetail{Name: s.Name, Modified: s.Modified}
	}
	return result
}

// versionCheckerAdapter wraps skills.IsNewer to implement service.SkillsVersionChecker.
type versionCheckerAdapter struct{}

func (a *versionCheckerAdapter) IsNewer(current, latest string) bool {
	return skills.IsNewer(current, latest)
}

// newSkillsService constructs a SkillsService with all adapters wired.
func newSkillsService(cfg *config.Config) *service.SkillsService {
	return service.NewSkillsService(
		&configSkillsAdapter{cfg: cfg},
		&releaseCheckerAdapter{client: skills.NewGitHubClient()},
		&installerAdapter{inst: skills.NewInstaller()},
		&manifestAdapter{},
		&versionCheckerAdapter{},
	)
}
