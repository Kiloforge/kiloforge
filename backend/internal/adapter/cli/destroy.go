package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"crelay/internal/adapter/compose"
	"crelay/internal/adapter/config"

	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Permanently destroy all crelay data",
	Long: `Removes the Gitea Docker Compose stack, volumes, and the entire data directory.

This action cannot be undone. You will be prompted to confirm unless --force is used.`,
	RunE: runDestroy,
}

var flagDestroyForce bool

func init() {
	destroyCmd.Flags().BoolVar(&flagDestroyForce, "force", false, "Skip confirmation prompt")
}

func runDestroy(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		fmt.Println("Nothing to destroy — crelay is not initialized.")
		return nil
	}

	if !flagDestroyForce {
		fmt.Println()
		fmt.Println("  WARNING: This will permanently delete:")
		fmt.Println("    - Gitea server and all repositories")
		fmt.Println("    - All project registrations")
		fmt.Println("    - All agent state and logs")
		fmt.Printf("    - Data directory: %s\n", cfg.DataDir)
		fmt.Println()
		fmt.Println("  This action cannot be undone.")
		fmt.Println()
		fmt.Print("  Type \"yes\" to confirm: ")

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Stop relay daemon first.
	fmt.Println("==> Stopping relay daemon...")
	if err := stopDaemon(cfg.DataDir); err != nil {
		fmt.Printf("    Warning: stop relay: %v\n", err)
	}

	// Try to bring down compose stack with volumes.
	runner, err := compose.Detect()
	if err == nil {
		fmt.Println("==> Removing Gitea containers and volumes...")
		if err := runner.Down(ctx, cfg.DataDir, true); err != nil {
			fmt.Printf("    Warning: compose down failed: %v\n", err)
		}
	}

	fmt.Printf("==> Removing data directory %s...\n", cfg.DataDir)
	if err := os.RemoveAll(cfg.DataDir); err != nil {
		return fmt.Errorf("remove data dir: %w", err)
	}

	fmt.Println("Done. All crelay data has been destroyed.")
	return nil
}
