package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"kiloforge/internal/adapter/browser"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/pidfile"
	"kiloforge/internal/adapter/prereq"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the kiloforge orchestrator",
	Long: `Sets up the kiloforge data directory, saves the global configuration,
and starts the orchestrator daemon.

Use 'kf down' to stop and 'kf up' to restart.
Project registration is handled by 'kf add'.`,
	RunE: runInit,
}

var (
	flagDataDir string
)

func init() {
	initCmd.Flags().StringVar(&flagDataDir, "data-dir", "", "Persistent data directory (defaults to ~/.kiloforge)")
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

	_, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var flagOpts []config.FlagOption
	if cmd.Flags().Changed("data-dir") {
		flagOpts = append(flagOpts, config.WithDataDir(flagDataDir))
	}

	cfg, err := config.Resolve(config.NewFlagsAdapter(flagOpts...))
	if err != nil {
		return fmt.Errorf("resolve config: %w", err)
	}

	// Check idempotency: if orchestrator is already running, report and exit.
	pidMgr := pidfile.New(cfg.DataDir)
	if running, _, _ := pidMgr.IsRunning(); running {
		fmt.Println("Kiloforge is already initialized, Kiloforger.")
		fmt.Printf("  Dashboard:  http://localhost:%d/\n", cfg.OrchestratorPort)
		fmt.Printf("  Data:       %s\n", cfg.DataDir)
		if !flagNoBrowser {
			dashURL := fmt.Sprintf("http://localhost:%d/", cfg.OrchestratorPort)
			if err := browser.Open(dashURL); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not open browser: %v\n", err)
			}
		}
		return nil
	}

	// Create data directory.
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", cfg.DataDir, err)
	}

	// Analytics opt-out prompt.
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

	// Save config.
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println()
	fmt.Println("You're ready, Kiloforger!")
	fmt.Printf("  Dashboard:  http://localhost:%d/\n", cfg.OrchestratorPort)
	fmt.Printf("  Data:       %s\n", cfg.DataDir)
	fmt.Println()
	fmt.Println("Register your first project with 'kf add <path>'.")
	fmt.Println()

	// Start orchestrator daemon.
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
