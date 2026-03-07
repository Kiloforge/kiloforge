package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"crelay/internal/compose"
	"crelay/internal/config"
	"crelay/internal/gitea"

	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the Gitea server",
	Long:  `Starts the Gitea Docker Compose stack. Requires 'crelay init' to have been run first.`,
	RunE:  runUp,
}

func runUp(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'crelay init' first")
	}

	// Check if already running.
	client := gitea.NewClient(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.GiteaAdminPass)
	if _, err := client.CheckVersion(ctx); err == nil {
		fmt.Println("Gitea is already running.")
		fmt.Printf("  URL: %s\n", cfg.GiteaURL())
		return nil
	}

	runner, err := compose.Detect()
	if err != nil {
		return err
	}

	fmt.Println("==> Starting Gitea...")
	manager := gitea.NewManager(cfg, runner)
	if err := manager.Start(ctx); err != nil {
		return fmt.Errorf("start gitea: %w", err)
	}
	fmt.Printf("    Gitea running at %s\n", cfg.GiteaURL())

	return nil
}
