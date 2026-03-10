package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"kiloforge/internal/adapter/auth"
	"kiloforge/internal/adapter/browser"
	"kiloforge/internal/adapter/compose"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/gitea"
	"kiloforge/internal/adapter/pidfile"
	"kiloforge/internal/adapter/prereq"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the global Gitea server via Docker Compose",
	Long: `Starts a local Gitea instance via Docker Compose, configures it with an admin
user and API token, and saves the global configuration.

This sets up the shared Gitea server. Use 'kf down' to stop and 'kf up'
to restart. Project registration is handled by 'kf add'.`,
	RunE: runInit,
}

var (
	flagGiteaPort int
	flagDataDir   string
	flagAdminPass string
	flagSSHKey    string
)

func init() {
	initCmd.Flags().IntVar(&flagGiteaPort, "gitea-port", 4000, "Port for Gitea web UI")
	initCmd.Flags().StringVar(&flagDataDir, "data-dir", "", "Persistent data directory (defaults to ~/.kiloforge)")
	initCmd.Flags().StringVar(&flagAdminPass, "admin-pass", "", "Admin password (default: generated random)")
	initCmd.Flags().StringVar(&flagSSHKey, "ssh-key", "", "Path to SSH public key (default: auto-detect)")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check prerequisites before anything else.
	if errs := prereq.Check(); len(errs) > 0 {
		return fmt.Errorf("%s", prereq.FormatErrors(errs))
	}

	// Warn (non-blocking) if Claude CLI is not authenticated.
	if err := prereq.CheckClaudeAuth(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

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

	// Password lifecycle: generate → use in Configure() → display to user → discard.
	// json_adapter.Save() strips the password from config.json using a copy.
	// The password lives only in cfg.GiteaAdminPass for the duration of this function.
	if cfg.GiteaAdminPass == "" {
		cfg.GiteaAdminPass = auth.GeneratePassword(20)
		fmt.Printf("==> Generated admin password\n")
	}

	// Check idempotency: if Gitea is already running, report and exit.
	client := gitea.NewClient(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.GiteaAdminPass)
	if _, err := client.CheckVersion(ctx); err == nil {
		fmt.Println("Kiloforge is already initialized, Kiloforger.")
		fmt.Printf("  Dashboard:  http://localhost:%d/\n", cfg.OrchestratorPort)
		fmt.Printf("  Gitea:      http://localhost:%d/gitea/ (auto-authenticated)\n", cfg.OrchestratorPort)
		fmt.Printf("  Data:       %s\n", cfg.DataDir)
		if !flagNoBrowser {
			dashURL := fmt.Sprintf("http://localhost:%d/", cfg.OrchestratorPort)
			if err := browser.Open(dashURL); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not open browser: %v\n", err)
			}
		}
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
		GiteaPort:        cfg.GiteaPort,
		OrchestratorPort: cfg.OrchestratorPort,
		DataDir:          cfg.DataDir,
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

	// Step 5c: Analytics opt-out prompt.
	if cfg.AnalyticsEnabled == nil {
		fmt.Print("==> Help improve kiloforge by sending anonymous usage data? (Y/n) ")
		scanner := bufio.NewScanner(os.Stdin)
		enabled := true
		if scanner.Scan() {
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer == "n" || answer == "no" {
				enabled = false
				fmt.Println("    Analytics disabled.")
			} else {
				fmt.Println("    Analytics enabled — thank you!")
			}
		}
		cfg.AnalyticsEnabled = &enabled
	}

	// Step 6: Save config.
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println()
	fmt.Println("You're ready, Kiloforger!")
	fmt.Printf("  Dashboard:  http://localhost:%d/\n", cfg.OrchestratorPort)
	fmt.Printf("  Gitea:      http://localhost:%d/gitea/ (auto-authenticated)\n", cfg.OrchestratorPort)
	fmt.Printf("  Data:       %s\n", cfg.DataDir)
	fmt.Println()
	fmt.Println("Register your first project with 'kf add <path>'.")
	fmt.Println()

	// Start orchestrator daemon.
	pidMgr := pidfile.New(cfg.DataDir)
	if running, pid, _ := pidMgr.IsRunning(); running {
		fmt.Printf("Orchestrator already running (PID %d)\n", pid)
	} else {
		fmt.Println("==> Starting orchestrator...")
		pid, err := startDaemon(cfg.DataDir)
		if err != nil {
			fmt.Printf("    Warning: start orchestrator: %v\n", err)
		} else {
			fmt.Printf("    Orchestrator started (PID %d)\n", pid)
		}
	}

	fmt.Println()
	fmt.Println("Use 'kf down' to stop.")

	// Auto-install embedded skills (no prompt, no repo needed).
	fmt.Println("==> Installing skills...")
	installEmbeddedSkills(cfg)

	if !flagNoBrowser {
		dashURL := fmt.Sprintf("http://localhost:%d/", cfg.OrchestratorPort)
		if err := browser.Open(dashURL); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open browser: %v\n", err)
		}
	}

	return nil
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
	if err := client.AddSSHKey(ctx, "kf-auto", keyContent); err != nil {
		fmt.Printf("    Warning: SSH key registration failed: %v\n", err)
		return
	}
	fmt.Printf("    SSH key registered\n")
}
