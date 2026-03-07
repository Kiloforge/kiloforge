package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"conductor-relay/internal/config"
	"conductor-relay/internal/gitea"
	"conductor-relay/internal/relay"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Gitea and start the relay server",
	Long: `Starts a local Gitea instance via Docker, configures it with an admin user
and API token, creates a repo mirroring the current project, registers webhooks,
and starts the relay server to manage Claude agents.

This is the one command to get everything running.`,
	RunE: runInit,
}

var (
	flagGiteaPort int
	flagRelayPort int
	flagRepoName  string
	flagDataDir   string
)

func init() {
	initCmd.Flags().IntVar(&flagGiteaPort, "gitea-port", 3000, "Port for Gitea web UI")
	initCmd.Flags().IntVar(&flagRelayPort, "relay-port", 3001, "Port for the relay webhook server")
	initCmd.Flags().StringVar(&flagRepoName, "repo", "", "Repository name (defaults to current directory name)")
	initCmd.Flags().StringVar(&flagDataDir, "data-dir", "", "Persistent data directory (defaults to ~/.conductor-relay)")
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Resolve working directory (the project we're setting up for).
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if flagRepoName == "" {
		flagRepoName = filepath.Base(projectDir)
	}

	if flagDataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home directory: %w", err)
		}
		flagDataDir = filepath.Join(home, ".conductor-relay")
	}

	cfg := &config.Config{
		GiteaPort:  flagGiteaPort,
		RelayPort:  flagRelayPort,
		RepoName:   flagRepoName,
		ProjectDir: projectDir,
		DataDir:    flagDataDir,
	}

	// Ensure data directory exists.
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	// Step 1: Start Gitea.
	fmt.Println("==> Starting Gitea...")
	giteaManager := gitea.NewManager(cfg)
	if err := giteaManager.Start(ctx); err != nil {
		return fmt.Errorf("start gitea: %w", err)
	}
	fmt.Printf("    Gitea running at http://localhost:%d\n", cfg.GiteaPort)

	// Step 2: Configure Gitea (admin user, token, repo).
	fmt.Println("==> Configuring Gitea...")
	giteaClient, err := giteaManager.Configure(ctx)
	if err != nil {
		return fmt.Errorf("configure gitea: %w", err)
	}
	fmt.Printf("    Admin user: %s\n", config.GiteaAdminUser)
	fmt.Printf("    Repository: %s/%s\n", config.GiteaAdminUser, cfg.RepoName)

	// Step 3: Add git remote and push.
	fmt.Println("==> Configuring git remote...")
	if err := giteaManager.SetupGitRemote(ctx, cfg); err != nil {
		return fmt.Errorf("setup git remote: %w", err)
	}
	fmt.Printf("    Remote 'gitea' added\n")

	// Step 4: Register webhooks.
	fmt.Println("==> Registering webhooks...")
	if err := giteaClient.CreateWebhook(ctx, cfg.RepoName, cfg.RelayPort); err != nil {
		return fmt.Errorf("create webhook: %w", err)
	}
	fmt.Printf("    Webhook → http://host.docker.internal:%d/webhook\n", cfg.RelayPort)

	// Step 5: Save config for other commands.
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Step 6: Start relay server (blocking).
	fmt.Println("==> Starting relay server...")
	fmt.Printf("    Listening on http://localhost:%d\n", cfg.RelayPort)
	fmt.Println()
	fmt.Println("Ready. Gitea webhooks will spawn Claude agents automatically.")
	fmt.Println("Press Ctrl+C to stop the relay (Gitea will keep running).")
	fmt.Println()

	srv := relay.NewServer(cfg, giteaClient)
	return srv.Run(ctx)
}
