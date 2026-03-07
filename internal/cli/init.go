package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"crelay/internal/auth"
	"crelay/internal/compose"
	"crelay/internal/config"
	"crelay/internal/gitea"
	"crelay/internal/project"
	"crelay/internal/relay"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the global Gitea server via Docker Compose",
	Long: `Starts a local Gitea instance via Docker Compose, configures it with an admin
user and API token, and saves the global configuration.

This sets up the shared Gitea server. Use 'crelay down' to stop and 'crelay up'
to restart. Project registration is handled by 'crelay add'.`,
	RunE: runInit,
}

var (
	flagGiteaPort int
	flagDataDir   string
	flagAdminPass string
	flagSSHKey    string
)

func init() {
	initCmd.Flags().IntVar(&flagGiteaPort, "gitea-port", 3000, "Port for Gitea web UI")
	initCmd.Flags().StringVar(&flagDataDir, "data-dir", "", "Persistent data directory (defaults to ~/.crelay)")
	initCmd.Flags().StringVar(&flagAdminPass, "admin-pass", "", "Admin password (default: generated random)")
	initCmd.Flags().StringVar(&flagSSHKey, "ssh-key", "", "Path to SSH public key (default: auto-detect)")
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var flagOpts []config.FlagOption
	if cmd.Flags().Changed("data-dir") {
		flagOpts = append(flagOpts, config.WithDataDir(flagDataDir))
	}
	if cmd.Flags().Changed("gitea-port") {
		flagOpts = append(flagOpts, config.WithGiteaPort(flagGiteaPort))
	}
	if cmd.Flags().Changed("admin-pass") {
		flagOpts = append(flagOpts, config.WithGiteaAdminPass(flagAdminPass))
	}

	cfg, err := config.Resolve(config.NewFlagsAdapter(flagOpts...))
	if err != nil {
		return fmt.Errorf("resolve config: %w", err)
	}

	// Resolve admin password: flag > saved config > generate random.
	if cfg.GiteaAdminPass == "" {
		cfg.GiteaAdminPass = auth.GeneratePassword(20)
		fmt.Printf("==> Generated admin password\n")
	}

	// Check idempotency: if Gitea is already running, report and exit.
	client := gitea.NewClient(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.GiteaAdminPass)
	if _, err := client.CheckVersion(ctx); err == nil {
		fmt.Println("Gitea is already running.")
		fmt.Printf("  URL:  %s\n", cfg.GiteaURL())
		fmt.Printf("  Data: %s\n", cfg.DataDir)
		return nil
	}

	// Step 1: Detect docker compose CLI.
	fmt.Println("==> Detecting Docker Compose...")
	runner, err := compose.Detect()
	if err != nil {
		return err
	}
	fmt.Printf("    Found: %s\n", runner.Version())

	// Step 2: Create data directory.
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", cfg.DataDir, err)
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
	client, err = manager.Configure(ctx)
	if err != nil {
		return fmt.Errorf("configure gitea: %w", err)
	}
	fmt.Printf("    Admin user: %s\n", cfg.GiteaAdminUser)

	// Step 5b: Register SSH key.
	registerSSHKey(ctx, client, flagSSHKey)

	// Step 6: Save config.
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println()
	fmt.Println("Gitea is ready!")
	fmt.Printf("  Web UI:     %s\n", cfg.GiteaURL())
	fmt.Printf("  Admin:      %s / %s\n", cfg.GiteaAdminUser, cfg.GiteaAdminPass)
	fmt.Printf("  Data:       %s\n", cfg.DataDir)
	fmt.Printf("  Compose:    %s\n", cfg.ComposeFile)
	fmt.Println()
	fmt.Println("Register a project with 'crelay add <path>'.")
	fmt.Println()

	// Start relay server (blocking).
	reg, err := project.LoadRegistry(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load project registry: %w", err)
	}

	fmt.Printf("==> Starting relay on :%d...\n", cfg.RelayPort)
	fmt.Println("    Press Ctrl+C to stop the relay.")
	fmt.Println()

	srv := relay.NewServer(cfg, reg, cfg.RelayPort)
	return srv.Run(ctx)
}

func registerSSHKey(ctx context.Context, client *gitea.Client, customPath string) {
	var keyPath, keyContent string
	var err error

	if customPath != "" {
		data, readErr := os.ReadFile(customPath)
		if readErr != nil {
			fmt.Printf("    Warning: cannot read SSH key %s: %v\n", customPath, readErr)
			return
		}
		keyPath = customPath
		keyContent = string(data)
	} else {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			fmt.Printf("    Warning: cannot detect home directory: %v\n", homeErr)
			return
		}
		keyPath, keyContent, err = auth.DetectSSHKey(filepath.Join(home, ".ssh"))
		if err != nil {
			fmt.Printf("    Warning: no SSH key found — git-over-SSH won't be available\n")
			return
		}
	}

	fmt.Printf("==> Registering SSH key (%s)...\n", keyPath)
	if err := client.AddSSHKey(ctx, "crelay-auto", keyContent); err != nil {
		fmt.Printf("    Warning: SSH key registration failed: %v\n", err)
		return
	}
	fmt.Printf("    SSH key registered\n")
}
