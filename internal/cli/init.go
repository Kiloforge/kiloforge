package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"crelay/internal/compose"
	"crelay/internal/config"
	"crelay/internal/gitea"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the global Gitea server via Docker Compose",
	Long: `Starts a local Gitea instance via Docker Compose, configures it with an admin
user and API token, and saves the global configuration.

This sets up the shared Gitea server. Project registration will be handled
separately by 'crelay add' (coming soon).`,
	RunE: runInit,
}

var (
	flagGiteaPort int
	flagDataDir   string
)

func init() {
	initCmd.Flags().IntVar(&flagGiteaPort, "gitea-port", 3000, "Port for Gitea web UI")
	initCmd.Flags().StringVar(&flagDataDir, "data-dir", "", "Persistent data directory (defaults to ~/.crelay)")
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if flagDataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home directory: %w", err)
		}
		flagDataDir = filepath.Join(home, ".crelay")
	}

	cfg := &config.Config{
		GiteaPort: flagGiteaPort,
		DataDir:   flagDataDir,
	}

	// Check idempotency: if Gitea is already running, report and exit.
	if existingCfg, err := config.LoadFrom(flagDataDir); err == nil {
		client := gitea.NewClient(existingCfg.GiteaURL(), config.GiteaAdminUser, config.GiteaAdminPass)
		if _, err := client.CheckVersion(ctx); err == nil {
			fmt.Println("Gitea is already running.")
			fmt.Printf("  URL:  %s\n", existingCfg.GiteaURL())
			fmt.Printf("  Data: %s\n", existingCfg.DataDir)
			return nil
		}
	}

	// Step 1: Detect docker compose CLI.
	fmt.Println("==> Detecting Docker Compose...")
	runner, err := compose.Detect()
	if err != nil {
		return err
	}
	fmt.Printf("    Found: %s\n", runner.Version())

	// Step 2: Create data directory and subdirectories.
	for _, dir := range []string{cfg.DataDir, filepath.Join(cfg.DataDir, "gitea-data")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	// Step 3: Generate docker-compose.yml.
	fmt.Println("==> Generating docker-compose.yml...")
	composeData, err := compose.GenerateComposeFile(compose.ComposeConfig{
		GiteaPort: cfg.GiteaPort,
		DataDir:   cfg.DataDir,
	})
	if err != nil {
		return fmt.Errorf("generate compose file: %w", err)
	}
	composeFilePath := filepath.Join(cfg.DataDir, compose.ComposeFileName)
	if err := os.WriteFile(composeFilePath, composeData, 0o644); err != nil {
		return fmt.Errorf("write compose file: %w", err)
	}
	cfg.ComposeFile = composeFilePath

	// Step 4: Start Gitea via compose.
	fmt.Println("==> Starting Gitea...")
	manager := gitea.NewManager(cfg, runner)
	if err := manager.Start(ctx); err != nil {
		return fmt.Errorf("start gitea: %w", err)
	}
	fmt.Printf("    Gitea running at %s\n", cfg.GiteaURL())

	// Step 5: Configure admin user and token.
	fmt.Println("==> Configuring Gitea...")
	if _, err := manager.Configure(ctx); err != nil {
		return fmt.Errorf("configure gitea: %w", err)
	}
	fmt.Printf("    Admin user: %s\n", config.GiteaAdminUser)

	// Step 6: Save config.
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println()
	fmt.Println("Gitea is ready!")
	fmt.Printf("  Web UI:     %s\n", cfg.GiteaURL())
	fmt.Printf("  Admin:      %s / %s\n", config.GiteaAdminUser, config.GiteaAdminPass)
	fmt.Printf("  Data:       %s\n", cfg.DataDir)
	fmt.Printf("  Compose:    %s\n", cfg.ComposeFile)
	fmt.Println()
	fmt.Println("Next: use 'crelay add' to register a project (coming soon).")

	return nil
}
