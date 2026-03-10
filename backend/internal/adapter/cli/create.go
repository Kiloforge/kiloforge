package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/gitea"
	"kiloforge/internal/core/service"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new project from scratch and register it with the Gitea server",
	Long: `Initializes a fresh git repository, creates a matching Gitea repo, adds a
'gitea' remote, and sets up a webhook. No remote URL is needed.

The name is used as both the project slug and the Gitea repo name.

Examples:
  kf create my-project
  kf create my-api`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func runCreate(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	name := args[0]

	// Load global config, verify Gitea is initialized and running.
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}

	client := gitea.NewClientWithToken(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.APIToken)
	if _, err := client.CheckVersion(ctx); err != nil {
		return fmt.Errorf("Gitea is not running — run 'kf init' or 'kf up' first")
	}

	// Open database and wire up project service.
	rt, err := NewCLIRuntimeFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = rt.Close() }()

	// Build a project service with the real Gitea client.
	projectSvc := service.NewProjectService(
		rt.Projects.Store(),
		client,
		service.ProjectServiceConfig{
			DataDir:          cfg.DataDir,
			OrchestratorPort: cfg.OrchestratorPort,
			GiteaAdminUser:   cfg.GiteaAdminUser,
			APIToken:         cfg.APIToken,
		},
	)

	if p, err := rt.Projects.GetProject(name); err == nil {
		fmt.Printf("Project %q is already registered.\n", name)
		fmt.Printf("  Path:   %s\n", p.ProjectDir)
		fmt.Printf("  Gitea:  %s/%s/%s\n", cfg.GiteaURL(), cfg.GiteaAdminUser, p.RepoName)
		return nil
	}

	fmt.Printf("==> Creating project %q...\n", name)
	result, err := projectSvc.CreateProject(ctx, name)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	p := result.Project
	fmt.Println()
	fmt.Printf("Project '%s' created!\n", p.Slug)
	fmt.Printf("  Path:  %s\n", p.ProjectDir)
	fmt.Printf("  Gitea: %s/%s/%s\n", cfg.GiteaURL(), cfg.GiteaAdminUser, p.RepoName)
	fmt.Println()
	fmt.Println("The repository is empty — add files and commit to get started.")
	fmt.Println("View registered projects with 'kf projects'.")

	return nil
}
