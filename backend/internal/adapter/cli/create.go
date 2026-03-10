package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/core/service"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new project from scratch",
	Long: `Initializes a fresh git repository and registers it as a kiloforge project.

The name is used as the project slug and repo name.

Examples:
  kf create my-project
  kf create my-api`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func runCreate(cmd *cobra.Command, args []string) error {
	_, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	name := args[0]

	// Load global config.
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}

	// Open database and wire up project service.
	rt, err := NewCLIRuntimeFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = rt.Close() }()

	projectSvc := service.NewProjectService(
		rt.Projects.Store(),
		service.ProjectServiceConfig{
			DataDir:          cfg.DataDir,
			OrchestratorPort: cfg.OrchestratorPort,
		},
	)

	if p, err := rt.Projects.GetProject(name); err == nil {
		fmt.Printf("Project %q is already registered.\n", name)
		fmt.Printf("  Path:   %s\n", p.ProjectDir)
		return nil
	}

	fmt.Printf("==> Creating project %q...\n", name)
	result, err := projectSvc.CreateProject(context.Background(), name)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	p := result.Project
	fmt.Println()
	fmt.Printf("Project '%s' created!\n", p.Slug)
	fmt.Printf("  Path:  %s\n", p.ProjectDir)
	fmt.Println()

	// Install embedded skills locally into the project.
	fmt.Println("==> Transforming your agent into a high-productivity track-slinging machine...")
	installed, installErr := installLocalSkills(p.ProjectDir)
	if installErr != nil {
		fmt.Printf("    Warning: local skills installation failed: %v\n", installErr)
	} else if len(installed) == 0 {
		fmt.Println("    Skills already up to date")
	} else {
		fmt.Printf("    Installed %d skill(s) to %s/.claude/skills/\n", len(installed), p.ProjectDir)
	}

	fmt.Println()
	fmt.Println("The repository is empty — add files and commit to get started.")
	fmt.Println("View registered projects with 'kf projects'.")

	return nil
}
