package skills

import (
	"context"
	"log"
	"time"
)

// AutoUpdater periodically checks for skill updates.
type AutoUpdater struct {
	repo      string
	skillsDir string
	interval  time.Duration
}

// NewAutoUpdater creates an auto-updater that checks for new skill versions.
func NewAutoUpdater(repo, skillsDir string) *AutoUpdater {
	return &AutoUpdater{
		repo:      repo,
		skillsDir: skillsDir,
		interval:  24 * time.Hour,
	}
}

// Start begins periodic update checks. Runs the first check immediately.
func (u *AutoUpdater) Start(ctx context.Context) {
	go u.run(ctx)
}

func (u *AutoUpdater) run(ctx context.Context) {
	u.check()

	ticker := time.NewTicker(u.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.check()
		}
	}
}

func (u *AutoUpdater) check() {
	gh := NewGitHubClient()
	rel, err := gh.LatestRelease(u.repo)
	if err != nil {
		log.Printf("[skills] Update check failed: %v", err)
		return
	}

	manifest, err := LoadManifest()
	if err != nil {
		log.Printf("[skills] Could not load manifest: %v", err)
		return
	}

	if manifest.Version != "" && !IsNewer(manifest.Version, rel.TagName) {
		return
	}

	// Check for local modifications.
	modified := DetectModified(u.skillsDir, manifest)
	if len(modified) > 0 {
		log.Printf("[skills] Update available (%s → %s) but %d skill(s) have local modifications — skipping auto-install", manifest.Version, rel.TagName, len(modified))
		return
	}

	log.Printf("[skills] Installing update %s → %s...", manifest.Version, rel.TagName)
	inst := NewInstaller()
	installed, err := inst.Install(rel.TarballURL, u.skillsDir)
	if err != nil {
		log.Printf("[skills] Auto-update failed: %v", err)
		return
	}

	checksums, _ := ComputeChecksums(u.skillsDir)
	newManifest := &Manifest{Version: rel.TagName, Checksums: checksums}
	if err := newManifest.Save(); err != nil {
		log.Printf("[skills] Warning: could not save manifest: %v", err)
	}

	log.Printf("[skills] Updated %d skills to %s", len(installed), rel.TagName)
}
