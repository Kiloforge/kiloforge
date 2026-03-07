package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"crelay/internal/adapter/config"
	"crelay/internal/adapter/gitea"
	"crelay/internal/adapter/persistence/jsonfile"
	"crelay/internal/core/service"

	"github.com/spf13/cobra"
)

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Show or set up the track board for a project",
	Long: `Show the track board status for a project. Use --setup to create the board
if it doesn't exist yet.

Examples:
  crelay board --project myapp
  crelay board --project myapp --setup`,
	RunE: runBoard,
}

var (
	flagBoardProject string
	flagBoardSetup   bool
)

func init() {
	boardCmd.Flags().StringVar(&flagBoardProject, "project", "", "Project slug (required)")
	boardCmd.Flags().BoolVar(&flagBoardSetup, "setup", false, "Create the board if it doesn't exist")
}

func runBoard(cmd *cobra.Command, args []string) error {
	if flagBoardProject == "" {
		return fmt.Errorf("--project is required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'crelay init' first")
	}

	reg, err := jsonfile.LoadProjectStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}
	project, ok := reg.Get(flagBoardProject)
	if !ok {
		return fmt.Errorf("project %q not found", flagBoardProject)
	}

	boardStore := jsonfile.NewBoardStore(cfg.DataDir)

	if flagBoardSetup {
		client := gitea.NewClient(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.GiteaAdminPass)
		if cfg.APIToken != "" {
			client.SetToken(cfg.APIToken)
		}
		boardSvc := service.NewBoardService(client, boardStore)
		boardCfg, err := boardSvc.SetupBoard(ctx, project)
		if err != nil {
			return fmt.Errorf("setup board: %w", err)
		}
		fmt.Printf("Board set up for project %q\n", project.Slug)
		fmt.Printf("  Board ID:  %d\n", boardCfg.ProjectBoardID)
		fmt.Printf("  Columns:   %d\n", len(boardCfg.Columns))
		fmt.Printf("  Labels:    %d\n", len(boardCfg.Labels))
		fmt.Printf("  URL:       %s/%s/%s/projects\n", cfg.GiteaURL(), cfg.GiteaAdminUser, project.RepoName)
		return nil
	}

	boardCfg, err := boardStore.GetBoardConfig(project.Slug)
	if err != nil {
		return fmt.Errorf("read board config: %w", err)
	}
	if boardCfg == nil {
		fmt.Printf("No board configured for project %q.\n", project.Slug)
		fmt.Println("Run with --setup to create one.")
		return nil
	}

	fmt.Printf("Board for project %q\n", project.Slug)
	fmt.Printf("  Board ID:  %d\n", boardCfg.ProjectBoardID)
	fmt.Printf("  URL:       %s/%s/%s/projects\n", cfg.GiteaURL(), cfg.GiteaAdminUser, project.RepoName)
	fmt.Println("  Columns:")
	for name, id := range boardCfg.Columns {
		fmt.Printf("    %-15s (ID: %d)\n", name, id)
	}

	trackIssues, err := boardStore.ListTrackIssues(project.Slug)
	if err != nil {
		return fmt.Errorf("list track issues: %w", err)
	}
	if len(trackIssues) > 0 {
		fmt.Printf("  Tracks:    %d synced\n", len(trackIssues))
	}

	return nil
}
