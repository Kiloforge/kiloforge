package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/compose"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/gitea"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/pidfile"
	"kiloforge/internal/adapter/rest"
	"kiloforge/internal/adapter/skills"
	"kiloforge/internal/adapter/tracing"
	"kiloforge/internal/core/service"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:    "serve",
	Short:  "Run the orchestrator in the foreground (internal)",
	Hidden: true,
	RunE:   runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}

	// Open log file.
	logPath := filepath.Join(cfg.DataDir, "orchestrator.log")
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

	// Open SQLite database (creates if needed, runs migrations).
	db, err := sqlite.Open(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Create stores backed by SQLite.
	reg := sqlite.NewProjectStore(db)
	agentStore := sqlite.NewAgentStore(db)
	prTracker := sqlite.NewPRTrackingStore(db)
	traceStore := sqlite.NewTraceStore(db)
	boardStore := sqlite.NewBoardStore(db)

	// Initialize tracing (always on).
	result, tracingErr := tracing.Init(ctx, "", tracing.WithSpanRecorder(traceStore))
	if tracingErr != nil {
		log.Printf("Warning: tracing init failed: %v", tracingErr)
	} else {
		defer result.Shutdown(context.Background())
		log.Printf("OpenTelemetry tracing enabled (OTLP → localhost:4318)")
	}

	// Create in-memory quota tracker (receives live cost/token data from SDK).
	quotaTracker := agent.NewQuotaTracker(cfg.DataDir)
	_ = quotaTracker.Load()

	// Build server options.
	opts := []rest.ServerOption{
		rest.WithGiteaProxy(cfg.GiteaURL(), cfg.GiteaAdminUser),
		rest.WithTracing(traceStore),
		rest.WithTracer(tracing.NewOTelTracer()),
	}
	if cfg.IsDashboardEnabled() {
		opts = append(opts, rest.WithDashboard(agentStore, quotaTracker, "/", reg))
	}

	// Enable native board service.
	boardSvc := service.NewNativeBoardService(boardStore)
	opts = append(opts, rest.WithBoardService(boardSvc))

	// Wire consent store for agent-permissions consent API.
	consentStore := sqlite.NewConsentStore(db)
	opts = append(opts, rest.WithConsent(consentStore))

	// Wire tour store for guided tour API.
	tourStore := sqlite.NewTourStore(db)
	opts = append(opts, rest.WithTourStore(tourStore))

	// Wire interactive agent spawner for WebSocket-based agent sessions.
	spawner := agent.NewSpawner(cfg, agentStore, quotaTracker)
	opts = append(opts, rest.WithInteractiveSpawner(spawner))

	// Start auto-update checker if enabled.
	if cfg.SkillsRepo != "" && cfg.AutoUpdateSkills != nil && *cfg.AutoUpdateSkills {
		updater := skills.NewAutoUpdater(cfg.SkillsRepo, cfg.GetSkillsDir())
		updater.Start(ctx)
		log.Printf("[skills] Auto-update enabled for %s", cfg.SkillsRepo)
	}

	log.Printf("Orchestrator starting on :%d (PID %d)", cfg.OrchestratorPort, os.Getpid())

	srv := rest.NewServer(cfg, reg, agentStore, prTracker, cfg.OrchestratorPort, opts...)
	if err := srv.Run(ctx); err != nil {
		log.Printf("Server error: %v", err)
		return err
	}

	log.Printf("Orchestrator stopped")
	return nil
}
