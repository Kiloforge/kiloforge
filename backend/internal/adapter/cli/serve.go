package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"crelay/internal/adapter/agent"
	"crelay/internal/adapter/compose"
	"crelay/internal/adapter/config"
	"crelay/internal/adapter/gitea"
	"crelay/internal/adapter/persistence/jsonfile"
	"crelay/internal/adapter/pidfile"
	"crelay/internal/adapter/rest"
	"crelay/internal/core/service"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:    "serve",
	Short:  "Run the relay server in the foreground (internal)",
	Hidden: true,
	RunE:   runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'crelay init' first")
	}

	// Open log file.
	logPath := filepath.Join(cfg.DataDir, "relay.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags)

	// Write PID file.
	pidMgr := pidfile.New(cfg.DataDir)
	if err := pidMgr.Write(os.Getpid()); err != nil {
		return fmt.Errorf("write pid file: %w", err)
	}

	// Set up signal handling for graceful shutdown.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// Ensure PID file is removed on exit.
	defer pidMgr.Remove()

	// Start Gitea if not running.
	client := gitea.NewClientWithToken(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.APIToken)
	if _, err := client.CheckVersion(ctx); err != nil {
		runner, detectErr := compose.Detect()
		if detectErr == nil {
			manager := gitea.NewManager(cfg, runner)
			_ = manager.Start(ctx)
		}
	}

	// Load project registry.
	reg, err := jsonfile.LoadProjectStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load project registry: %w", err)
	}

	// Build server options.
	opts := []rest.ServerOption{
		rest.WithGiteaProxy(cfg.GiteaURL()),
	}
	if cfg.IsDashboardEnabled() {
		store, storeErr := jsonfile.LoadAgentStore(cfg.DataDir)
		if storeErr == nil {
			tracker := agent.NewQuotaTracker(cfg.DataDir)
			_ = tracker.Load()
			opts = append(opts, rest.WithDashboard(store, tracker, "/", reg))
		}
	}

	// Enable board sync.
	boardStore := jsonfile.NewBoardStore(cfg.DataDir)
	boardClient := gitea.NewClientWithToken(cfg.GiteaURL(), cfg.GiteaAdminUser, cfg.APIToken)
	boardSvc := service.NewBoardService(boardClient, boardStore)
	opts = append(opts, rest.WithBoardSync(boardSvc, boardStore))

	log.Printf("Relay server starting on :%d (PID %d)", cfg.RelayPort, os.Getpid())

	srv := rest.NewServer(cfg, reg, cfg.RelayPort, opts...)
	if err := srv.Run(ctx); err != nil {
		log.Printf("Server error: %v", err)
		return err
	}

	log.Printf("Relay server stopped")
	return nil
}
