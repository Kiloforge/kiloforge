package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"crelay/internal/compose"
	"crelay/internal/config"
	"crelay/internal/gitea"
	"crelay/internal/adapter/persistence/jsonfile"
	"crelay/internal/relay"

	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the Gitea server and relay",
	Long: `Starts the Gitea Docker Compose stack and runs the webhook relay server
in the foreground. Requires 'crelay init' to have been run first.

Press Ctrl+C to stop the relay. Gitea stays running via Docker Compose.
Use 'crelay down' to stop Gitea.`,
	RunE: runUp,
}

func runUp(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'crelay init' first")
	}

	client := gitea.NewClient(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.GiteaAdminPass)

	// Start Gitea if not running.
	if _, err := client.CheckVersion(ctx); err != nil {
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
	} else {
		fmt.Printf("Gitea already running at %s\n", cfg.GiteaURL())
	}

	// Load project registry.
	reg, err := jsonfile.LoadProjectStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load project registry: %w", err)
	}

	projects := reg.List()
	if len(projects) == 0 {
		fmt.Println()
		fmt.Println("No projects registered. Use 'crelay add <path>' to register a project.")
		fmt.Println()
	}

	// Start relay server (blocking).
	fmt.Printf("==> Starting relay on :%d (%d project(s))...\n", cfg.RelayPort, len(projects))
	fmt.Println("    Press Ctrl+C to stop the relay.")
	fmt.Println()

	srv := relay.NewServer(cfg, reg, cfg.RelayPort)
	return srv.Run(ctx)
}
