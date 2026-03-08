package cli

import (
	"context"
	"fmt"
	"os"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/skills"

	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage conductor skills",
	Long:  `Install, update, and list conductor skills from a GitHub repository.`,
}

var skillsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Fetch and install the latest skills from the configured repo",
	RunE:  runSkillsUpdate,
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills with version and modification status",
	RunE:  runSkillsList,
}

func init() {
	skillsUpdateCmd.Flags().Bool("force", false, "Overwrite locally modified skills without confirmation")
	skillsCmd.Flags().String("repo", "", "Set the skills source repository (owner/repo)")
	skillsCmd.Flags().Bool("auto-update", false, "Enable auto-update of skills")
	skillsCmd.Flags().Bool("no-auto-update", false, "Disable auto-update of skills")
	skillsCmd.AddCommand(skillsUpdateCmd)
	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.PersistentPreRunE = runSkillsConfig
}

func runSkillsConfig(cmd *cobra.Command, args []string) error {
	repo, _ := cmd.Flags().GetString("repo")
	autoUpdate, _ := cmd.Flags().GetBool("auto-update")
	noAutoUpdate, _ := cmd.Flags().GetBool("no-auto-update")

	if repo == "" && !autoUpdate && !noAutoUpdate {
		return nil
	}

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}

	changed := false
	if repo != "" {
		cfg.SkillsRepo = repo
		changed = true
		fmt.Printf("Skills repo set to: %s\n", repo)
	}
	if autoUpdate {
		v := true
		cfg.AutoUpdateSkills = &v
		changed = true
		fmt.Println("Auto-update enabled")
	}
	if noAutoUpdate {
		v := false
		cfg.AutoUpdateSkills = &v
		changed = true
		fmt.Println("Auto-update disabled")
	}
	if changed {
		return cfg.Save()
	}
	return nil
}

func runSkillsUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}
	if cfg.SkillsRepo == "" {
		return fmt.Errorf("no skills repo configured — run 'kf skills --repo owner/repo' first")
	}

	force, _ := cmd.Flags().GetBool("force")
	skillsDir := cfg.GetSkillsDir()

	// Check latest release.
	gh := skills.NewGitHubClient()
	rel, err := gh.LatestRelease(cfg.SkillsRepo)
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}

	if cfg.SkillsVersion != "" && !skills.IsNewer(cfg.SkillsVersion, rel.TagName) {
		fmt.Printf("Skills are up to date (%s)\n", cfg.SkillsVersion)
		return nil
	}

	fmt.Printf("New version available: %s → %s\n", cfg.SkillsVersion, rel.TagName)

	// Check for modifications.
	if !force {
		manifest, _ := skills.LoadManifest()
		modified := skills.DetectModified(skillsDir, manifest)
		if len(modified) > 0 {
			fmt.Println("\nThe following skills have local modifications that will be overwritten:")
			for _, m := range modified {
				fmt.Printf("  • %s (%d files changed)\n", m.Name, len(m.Files))
			}
			fmt.Print("\nContinue? [y/N] ")
			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" {
				fmt.Println("Update cancelled. Use --force to skip this check.")
				return nil
			}
		}
	}

	// Install.
	fmt.Printf("Installing skills from %s...\n", cfg.SkillsRepo)
	inst := skills.NewInstaller()
	installed, err := inst.Install(rel.TarballURL, skillsDir)
	if err != nil {
		return fmt.Errorf("install skills: %w", err)
	}

	// Update manifest.
	checksums, _ := skills.ComputeChecksums(skillsDir)
	manifest := &skills.Manifest{
		Version:   rel.TagName,
		Checksums: checksums,
	}
	if err := manifest.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save manifest: %v\n", err)
	}

	// Update config version.
	cfg.SkillsVersion = rel.TagName
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save config: %v\n", err)
	}

	fmt.Printf("Installed %d skills (version %s):\n", len(installed), rel.TagName)
	for _, s := range installed {
		fmt.Printf("  • %s\n", s.Name)
	}
	return nil
}

// readLineCtx reads a line from stdin, respecting context cancellation.
// Returns the input string, or empty string if cancelled.
func readLineCtx(ctx context.Context) (string, bool) {
	ch := make(chan string, 1)
	go func() {
		var answer string
		fmt.Scanln(&answer)
		ch <- answer
	}()
	select {
	case <-ctx.Done():
		fmt.Println()
		return "", false
	case answer := <-ch:
		return answer, true
	}
}

// offerSkillsInstall prompts the user to install skills if repo is configured
// but no skills are installed yet. Falls back to embedded install when no repo.
func offerSkillsInstall(ctx context.Context, cfg *config.Config) {
	if cfg.SkillsRepo == "" {
		// No repo configured — use embedded skills silently.
		installEmbeddedSkills(cfg)
		return
	}
	skillsDir := cfg.GetSkillsDir()
	manifest, _ := skills.LoadManifest()
	installed := skills.ListInstalled(skillsDir, manifest)
	if len(installed) > 0 {
		return
	}

	fmt.Printf("\nSkills repo configured (%s) but no skills installed.\n", cfg.SkillsRepo)
	fmt.Print("Install skills now? [y/N] ")
	answer, ok := readLineCtx(ctx)
	if !ok {
		return
	}
	if answer != "y" && answer != "Y" {
		fmt.Println("Skipped. Run 'kf skills update' to install later.")
		return
	}

	gh := skills.NewGitHubClient()
	rel, err := gh.LatestRelease(cfg.SkillsRepo)
	if err != nil {
		fmt.Printf("Warning: could not check for skills: %v\n", err)
		return
	}

	fmt.Printf("Installing skills %s...\n", rel.TagName)
	inst := skills.NewInstaller()
	result, err := inst.Install(rel.TarballURL, skillsDir)
	if err != nil {
		fmt.Printf("Warning: skills installation failed: %v\n", err)
		return
	}

	checksums, _ := skills.ComputeChecksums(skillsDir)
	m := &skills.Manifest{Version: rel.TagName, Checksums: checksums}
	m.Save()
	cfg.SkillsVersion = rel.TagName
	cfg.Save()

	fmt.Printf("Installed %d skills:\n", len(result))
	for _, s := range result {
		fmt.Printf("  • %s\n", s.Name)
	}
}

// installEmbeddedSkills auto-installs all embedded skills without prompting.
// Skips skills that are already installed and up to date (hash match).
func installEmbeddedSkills(cfg *config.Config) {
	skillsDir := cfg.GetSkillsDir()
	installed, err := skills.InstallAllEmbedded(skillsDir)
	if err != nil {
		fmt.Printf("    Warning: skills installation failed: %v\n", err)
		return
	}
	if len(installed) == 0 {
		fmt.Println("    Skills already up to date")
		return
	}
	fmt.Printf("    Installed %d skill(s):\n", len(installed))
	for _, name := range installed {
		fmt.Printf("      • %s\n", name)
	}
}

func runSkillsList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}

	skillsDir := cfg.GetSkillsDir()
	manifest, _ := skills.LoadManifest()
	installed := skills.ListInstalled(skillsDir, manifest)

	if len(installed) == 0 {
		fmt.Println("No skills installed.")
		if cfg.SkillsRepo != "" {
			fmt.Printf("Run 'kf skills update' to install from %s\n", cfg.SkillsRepo)
		} else {
			fmt.Println("Run 'kf skills --repo owner/repo' to configure a source.")
		}
		return nil
	}

	fmt.Printf("Skills (version: %s, dir: %s):\n", cfg.SkillsVersion, skillsDir)
	for _, s := range installed {
		status := "✓"
		if s.Modified {
			status = "✎ modified"
		}
		fmt.Printf("  %s  %s\n", status, s.Name)
	}
	return nil
}
