package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"text/tabwriter"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/persistence/jsonfile"
	"kiloforge/internal/adapter/pool"
	"kiloforge/internal/adapter/tracing"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
	"kiloforge/internal/core/service"

	"github.com/spf13/cobra"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var implementCmd = &cobra.Command{
	Use:   "implement <track-id>",
	Short: "Approve a track and spawn a developer agent",
	Long: `Acquires a worktree from the pool, spawns a Claude Code developer agent
to implement the given conductor track, and records the agent state.

Use --list to see available tracks for the resolved project.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runImplement,
}

var (
	flagImplementList    bool
	flagImplementProject string
)

func init() {
	implementCmd.Flags().BoolVar(&flagImplementList, "list", false, "List available tracks")
	implementCmd.Flags().StringVar(&flagImplementProject, "project", "", "Project slug (auto-detect from cwd if not set)")
}

func runImplement(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}

	// Resolve project.
	reg, err := jsonfile.LoadProjectStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	proj, err := resolveProject(reg, flagImplementProject)
	if err != nil {
		return err
	}

	// --list mode: show available tracks.
	if flagImplementList {
		return listTracks(proj)
	}

	if len(args) == 0 {
		return fmt.Errorf("track ID required\n\nUsage: kf implement <track-id>\n\nUse --list to see available tracks.")
	}

	trackID := args[0]

	// Validate track is pending.
	tracks, err := service.DiscoverTracks(proj.ProjectDir)
	if err != nil {
		return fmt.Errorf("discover tracks: %w", err)
	}

	var found *service.TrackEntry
	for i := range tracks {
		if tracks[i].ID == trackID {
			found = &tracks[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("track %q not found in project %q", trackID, proj.Slug)
	}
	if found.Status == service.StatusComplete {
		return fmt.Errorf("track %q is already complete", trackID)
	}
	if found.Status == service.StatusInProgress {
		return fmt.Errorf("track %q is already in progress", trackID)
	}

	// Initialize tracing for track lifecycle.
	tracer, tracingShutdown := initTracing(ctx, cfg)
	if tracingShutdown != nil {
		defer tracingShutdown(context.Background())
	}

	// Start root trace span for the track lifecycle.
	ctx, trackSpan := tracer.StartSpan(ctx, "track/"+trackID,
		port.StringAttr("track.id", trackID),
		port.StringAttr("track.title", found.Title),
		port.StringAttr("project.slug", proj.Slug),
	)
	defer trackSpan.End()

	// Extract trace ID from span context for cross-process propagation.
	traceID := extractTraceID(ctx)

	// Acquire worktree.
	ctx, acquireSpan := tracer.StartSpan(ctx, "worktree.acquire",
		port.StringAttr("track.id", trackID),
	)
	p, err := pool.Load(cfg.DataDir)
	if err != nil {
		acquireSpan.SetError(err)
		acquireSpan.End()
		return fmt.Errorf("load pool: %w", err)
	}
	p.ProjectRoot = proj.ProjectDir

	wt, err := p.Acquire()
	if err != nil {
		acquireSpan.SetError(err)
		acquireSpan.End()
		return fmt.Errorf("acquire worktree: %w", err)
	}
	acquireSpan.SetAttributes(port.StringAttr("worktree.path", wt.Path))
	acquireSpan.End()

	// Prepare worktree for track.
	_, prepareSpan := tracer.StartSpan(ctx, "worktree.prepare",
		port.StringAttr("track.id", trackID),
		port.StringAttr("worktree.path", wt.Path),
	)
	if err := p.Prepare(wt, trackID); err != nil {
		prepareSpan.SetError(err)
		prepareSpan.End()
		return fmt.Errorf("prepare worktree: %w", err)
	}
	prepareSpan.End()

	// Spawn developer agent.
	store, err := jsonfile.LoadAgentStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	tracker := agent.NewQuotaTracker(cfg.DataDir)
	_ = tracker.Load()

	logDir := filepath.Join(cfg.DataDir, "projects", proj.Slug, "logs")
	spawner := agent.NewSpawner(cfg, store, tracker)
	spawner.SetTracer(tracer)
	info, err := spawner.SpawnDeveloper(ctx, agent.SpawnDeveloperOpts{
		TrackID:     trackID,
		Flags:       "--auto-merge",
		WorktreeDir: wt.Path,
		LogDir:      logDir,
		Model:       cfg.Model,
	})
	if err != nil {
		trackSpan.SetError(err)
		return fmt.Errorf("spawn developer: %w", err)
	}

	// Record worktree-agent link.
	wt.AgentID = info.ID

	// Save pool state.
	if err := p.Save(cfg.DataDir); err != nil {
		return fmt.Errorf("save pool: %w", err)
	}

	// Move track card to In Progress on the native board and store trace ID.
	boardStore := jsonfile.NewBoardStore(cfg.DataDir)
	nativeBoardSvc := service.NewNativeBoardService(boardStore)
	if moveResult, err := nativeBoardSvc.MoveCard(proj.Slug, trackID, domain.ColumnInProgress); err == nil {
		if moveResult.FromColumn != moveResult.ToColumn {
			fmt.Println("  Board:     → In Progress")
		}
	}

	// Store trace ID in board card for cross-process propagation.
	if traceID != "" {
		_ = nativeBoardSvc.StoreTraceID(proj.Slug, trackID, traceID)
	}

	fmt.Println()
	fmt.Printf("Developer agent spawned for track %q\n", trackID)
	fmt.Printf("  Agent:     %s\n", info.ID[:8])
	fmt.Printf("  Session:   %s\n", info.SessionID[:8])
	fmt.Printf("  Worktree:  %s\n", wt.Path)
	fmt.Printf("  Log:       %s\n", info.LogFile)
	if traceID != "" {
		fmt.Printf("  Trace:     %s\n", traceID)
	}
	fmt.Println()
	fmt.Printf("View logs:     kf logs %s\n", info.ID[:8])
	fmt.Printf("Stop agent:    kf stop %s\n", info.ID[:8])
	fmt.Printf("Resume agent:  kf attach %s\n", info.ID[:8])
	fmt.Println()

	return nil
}

// initTracing sets up a tracer for the implement command.
// Returns NoopTracer if tracing is not enabled.
func initTracing(ctx context.Context, cfg *config.Config) (port.Tracer, func(context.Context) error) {
	if !cfg.IsTracingEnabled() {
		return port.NoopTracer{}, nil
	}
	result, err := tracing.Init(ctx, "")
	if err != nil {
		return port.NoopTracer{}, nil
	}
	return tracing.NewOTelTracer(), result.Shutdown
}

// extractTraceID gets the hex trace ID from the current span context.
func extractTraceID(ctx context.Context) string {
	sc := oteltrace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}

func resolveProject(reg *jsonfile.ProjectStore, slug string) (domain.Project, error) {
	if slug != "" {
		proj, ok := reg.Get(slug)
		if !ok {
			return domain.Project{}, fmt.Errorf("project %q not found — use 'kf add' to register", slug)
		}
		return proj, nil
	}

	// Auto-detect from cwd.
	cwd, err := os.Getwd()
	if err != nil {
		return domain.Project{}, fmt.Errorf("get cwd: %w", err)
	}

	proj, ok := reg.FindByDir(cwd)
	if !ok {
		return domain.Project{}, fmt.Errorf("no project registered for %s — use 'kf add' or --project flag", cwd)
	}
	return proj, nil
}

func listTracks(proj domain.Project) error {
	tracks, err := service.DiscoverTracks(proj.ProjectDir)
	if err != nil {
		return fmt.Errorf("discover tracks: %w", err)
	}

	pending := service.FilterByStatus(tracks, service.StatusPending)
	if len(pending) == 0 {
		fmt.Printf("No pending tracks for project %q.\n", proj.Slug)
		return nil
	}

	fmt.Printf("Available tracks for %q:\n\n", proj.Slug)
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "TRACK ID\tTITLE")
	for _, t := range pending {
		fmt.Fprintf(w, "%s\t%s\n", t.ID, t.Title)
	}
	return w.Flush()
}
