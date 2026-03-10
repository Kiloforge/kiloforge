package cli

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"kiloforge/internal/adapter/browser"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/pidfile"
	"kiloforge/internal/adapter/prereq"

	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the Cortex (auto-initializes on first run)",
	Long: `Starts the Cortex as a background daemon. On first run, performs
full initialization: creates the data directory, saves configuration,
installs skills, and starts the Cortex.

Use 'kf down' to stop the Cortex.`,
	RunE: runUp,
}

func init() {
	upCmd.Flags().StringVar(&flagHost, "host", "", "Host address to bind (default 127.0.0.1)")
	upCmd.Flags().IntVar(&flagPort, "port", 0, "Port to listen on (default 39517)")
}

// isFirstRun returns true if no config file exists in the data directory.
func isFirstRun(dataDir string) bool {
	_, err := os.Stat(filepath.Join(dataDir, config.ConfigFileName))
	return os.IsNotExist(err)
}

func runUp(cmd *cobra.Command, args []string) error {
	// Build config with optional CLI flags.
	var flagOpts []config.FlagOption
	if cmd.Flags().Changed("data-dir") {
		flagOpts = append(flagOpts, config.WithDataDir(flagDataDir))
	}
	if cmd.Flags().Changed("host") {
		flagOpts = append(flagOpts, config.WithOrchestratorHost(flagHost))
	}
	if cmd.Flags().Changed("port") {
		flagOpts = append(flagOpts, config.WithOrchestratorPort(flagPort))
	}

	cfg, err := config.Resolve(config.NewFlagsAdapter(flagOpts...))
	if err != nil {
		return fmt.Errorf("resolve config: %w", err)
	}

	firstRun := isFirstRun(cfg.DataDir)

	if firstRun {
		// --- First-run setup ---

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
		fmt.Printf("  Command Deck:  %s\n", dashboardURL(cfg))
		fmt.Printf("  Data:       %s\n", cfg.DataDir)
		fmt.Println()
		fmt.Println("Register your first project with 'kf add <path>'.")
		fmt.Println()
	}

	// Port conflict pre-check before starting the daemon.
	addr := fmt.Sprintf("%s:%d", cfg.OrchestratorHost, cfg.OrchestratorPort)
	ln, listenErr := net.Listen("tcp", addr)
	if listenErr != nil {
		return fmt.Errorf("port %d is already in use on %s\n\n"+
			"  Try: kf up --port %d\n"+
			"  Or:  KF_ORCH_PORT=%d kf up",
			cfg.OrchestratorPort, cfg.OrchestratorHost,
			cfg.OrchestratorPort+1, cfg.OrchestratorPort+1)
	}
	ln.Close()

	// Start Cortex daemon.
	pidMgr := pidfile.New(cfg.DataDir)
	if running, pid, _ := pidMgr.IsRunning(); running {
		fmt.Printf("Cortex already running (PID %d)\n", pid)
	} else {
		fmt.Println("==> Starting Cortex...")
		pid, err := startDaemon(cfg.DataDir)
		if err != nil {
			if firstRun {
				fmt.Printf("    Warning: start Cortex: %v\n", err)
			} else {
				return fmt.Errorf("start Cortex: %w", err)
			}
		} else {
			fmt.Printf("    Cortex started (PID %d)\n", pid)
		}
	}

	if !firstRun {
		fmt.Println()
		fmt.Printf("Command Deck:   %s\n", dashboardURL(cfg))
	}

	fmt.Println()
	fmt.Println("Use 'kf down' to stop.")

	if firstRun {
		// Auto-install embedded skills (no prompt, no repo needed).
		fmt.Println("==> Installing skills...")
		installEmbeddedSkills(cfg)
	}

	if !flagNoBrowser {
		dashURL := dashboardURL(cfg)
		if err := browser.Open(dashURL); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open browser: %v\n", err)
		}
	}

	return nil
}

// dashboardURL returns the dashboard URL using the configured host and port.
func dashboardURL(cfg *config.Config) string {
	host := cfg.OrchestratorHost
	if host == "0.0.0.0" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%d/", host, cfg.OrchestratorPort)
}
