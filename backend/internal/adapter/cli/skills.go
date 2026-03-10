package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/skills"
	"kiloforge/internal/core/service"

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
		return fmt.Errorf("%s", notInitializedError())
	}

	svc := newSkillsService(cfg)
	result, err := svc.UpdateConfig(service.SkillsConfigUpdate{
		Repo:         &repo,
		AutoUpdate:   &autoUpdate,
		NoAutoUpdate: &noAutoUpdate,
	})
	if err != nil {
		return err
	}

	if result.RepoChanged {
		fmt.Printf("Skills repo set to: %s\n", result.NewRepo)
	}
	if result.AutoUpdateChanged {
		if result.AutoUpdateEnabled {
			fmt.Println("Auto-update enabled")
		} else {
			fmt.Println("Auto-update disabled")
		}
	}
	return nil
}

func runSkillsUpdate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}

	svc := newSkillsService(cfg)
	force, _ := cmd.Flags().GetBool("force")

	checkResult, err := svc.CheckForUpdates()
	if err != nil {
		return err
	}

	if checkResult.UpToDate {
		fmt.Printf("Skills are up to date (%s)\n", checkResult.CurrentVersion)
		return nil
	}

	fmt.Printf("New version available: %s → %s\n", checkResult.CurrentVersion, checkResult.NewVersion)

	// Prompt for modified skills confirmation (CLI concern).
	if !force && len(checkResult.Modified) > 0 {
		fmt.Println("\nThe following skills have local modifications that will be overwritten:")
		for _, m := range checkResult.Modified {
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

	fmt.Printf("Installing skills from %s...\n", cfg.SkillsRepo)
	installResult, err := svc.InstallUpdate(checkResult.Release)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		if installResult == nil {
			return err
		}
	}

	fmt.Printf("Installed %d skills (version %s):\n", len(installResult.Installed), installResult.Version)
	for _, s := range installResult.Installed {
		fmt.Printf("  • %s\n", s.Name)
	}
	return nil
}

// readLineCtx reads a line from stdin, respecting context cancellation.
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

// installLocalSkills installs all embedded skills into {projectDir}/.claude/skills/.
// Returns the names of skills that were installed or updated.
func installLocalSkills(projectDir string) ([]string, error) {
	destDir := filepath.Join(projectDir, ".claude", "skills")
	return skills.InstallAllEmbedded(destDir)
}

// installEmbeddedSkills auto-installs all embedded skills without prompting.
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
		return fmt.Errorf("%s", notInitializedError())
	}

	svc := newSkillsService(cfg)
	installed, version, dir := svc.ListInstalledSkills()

	if len(installed) == 0 {
		fmt.Println("No skills installed.")
		if cfg.SkillsRepo != "" {
			fmt.Printf("Run 'kf skills update' to install from %s\n", cfg.SkillsRepo)
		} else {
			fmt.Println("Run 'kf skills --repo owner/repo' to configure a source.")
		}
		return nil
	}

	fmt.Printf("Skills (version: %s, dir: %s):\n", version, dir)
	for _, s := range installed {
		status := "✓"
		if s.Modified {
			status = "✎ modified"
		}
		fmt.Printf("  %s  %s\n", status, s.Name)
	}
	return nil
}
