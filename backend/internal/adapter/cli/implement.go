package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"text/tabwriter"

	"kiloforge/internal/adapter/agent"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/pool"
	"kiloforge/internal/adapter/prereq"
	"kiloforge/internal/adapter/skills"
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
	flagImplementDryRun  bool
)

func init() {
	implementCmd.Flags().BoolVar(&flagImplementList, "list", false, "List available tracks")
	implementCmd.Flags().StringVar(&flagImplementProject, "project", "", "Project slug (auto-detect from cwd if not set)")
	implementCmd.Flags().BoolVar(&flagImplementDryRun, "dry-run", false, "Skip agent spawn; move board card to Done and mark track complete")
}

func runImplement(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}

	// Open SQLite database.
	db, err := sqlite.Open(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Resolve project.
	reg := sqlite.NewProjectStore(db)

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

	// Dry-run mode: skip agent spawn, move board card to Done.
	if flagImplementDryRun {
		return runDryRun(db, proj, trackID)
	}

	// Check agent permissions consent.
	consentStore := sqlite.NewConsentStore(db)
	if !consentStore.HasAgentPermissionsConsent() {
		fmt.Println()
		fmt.Println("WARNING: Kiloforge agents run with --dangerously-skip-permissions.")
		fmt.Println("This grants agents unrestricted access to tools (file read/write,")
		fmt.Println("shell commands, etc.) within their worktree directory.")
		fmt.Println()
		fmt.Println("This is required for non-interactive agent operation.")
		fmt.Print("\nDo you accept? [y/N] ")

		answer, ok := readLineCtx(ctx)
		if !ok {
			return fmt.Errorf("aborted")
		}
		if answer != "y" && answer != "Y" && answer != "yes" {
			return fmt.Errorf("agent spawning aborted — permissions not accepted")
		}
		if err := consentStore.RecordAgentPermissionsConsent(); err != nil {
			return fmt.Errorf("save consent: %w", err)
		}
		fmt.Println("Consent recorded.")
		fmt.Println()
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
	store := sqlite.NewAgentStore(db)
	tracker := agent.NewQuotaTracker(cfg.DataDir)
	_ = tracker.Load()

	// Create board service before spawner so the completion callback can reference it.
	boardStore := sqlite.NewBoardStore(db)
	nativeBoardSvc := service.NewNativeBoardService(boardStore)

	logDir := filepath.Join(cfg.DataDir, "projects", proj.Slug, "logs")
	spawner := agent.NewSpawner(cfg, store, tracker)
	spawner.SetTracer(tracer)

	// Pre-flight: verify Claude CLI authentication.
	if err := prereq.CheckClaudeAuthCached(ctx); err != nil {
		return fmt.Errorf("claude auth check failed: %w\n\nRun 'claude' in a terminal to authenticate, then retry.", err)
	}

	// Pre-flight skill validation: ensure required skills are installed.
	if err := spawner.ValidateSkills("developer", proj.ProjectDir); err != nil {
		var errMissing *agent.ErrSkillsMissing
		if errors.As(err, &errMissing) {
			if installErr := promptSkillInstall(ctx, cfg, errMissing.Missing, proj.ProjectDir); installErr != nil {
				return installErr
			}
			// Re-validate after install.
			if err := spawner.ValidateSkills("developer", proj.ProjectDir); err != nil {
				return fmt.Errorf("skills still missing after install: %w", err)
			}
		} else {
			return fmt.Errorf("validate skills: %w", err)
		}
	}

	// Wire completion callback: move board card and return worktree on agent exit.
	spawner.SetCompletionCallback(func(agentID, ref, status string) {
		if status == "completed" {
			if _, err := nativeBoardSvc.MoveCard(proj.Slug, ref, domain.ColumnDone); err != nil {
				fmt.Fprintf(os.Stderr, "warning: board move to done: %v\n", err)
			}
		}
		if err := p.ReturnByTrackID(ref); err != nil {
			fmt.Fprintf(os.Stderr, "warning: return worktree: %v\n", err)
		}
		if err := p.Save(cfg.DataDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: save pool: %v\n", err)
		}
	})

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
// Returns NoopTracer if OTel initialization fails.
func initTracing(ctx context.Context, _ *config.Config) (port.Tracer, func(context.Context) error) {
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

func resolveProject(reg port.ProjectStore, slug string) (domain.Project, error) {
	if slug != "" {
		proj, err := reg.Get(slug)
		if err != nil {
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

func runDryRun(db *sql.DB, proj domain.Project, trackID string) error {
	fmt.Printf("Dry run: skipping agent spawn for track %q\n\n", trackID)

	// Move board card to Done.
	boardStore := sqlite.NewBoardStore(db)
	nativeBoardSvc := service.NewNativeBoardService(boardStore)
	if result, err := nativeBoardSvc.MoveCard(proj.Slug, trackID, domain.ColumnDone); err == nil {
		fmt.Printf("  Board:     %s → %s\n", result.FromColumn, result.ToColumn)
	} else {
		fmt.Printf("  Board:     (not on board: %v)\n", err)
	}

	fmt.Printf("  Worktree:  not acquired (dry run)\n")
	fmt.Printf("  Agent:     not spawned (dry run)\n\n")
	fmt.Printf("Done. Track %q marked complete via dry-run.\n", trackID)
	return nil
}

// promptSkillInstall offers the user a choice to install missing skills
// globally, locally, or deny. Skills are extracted from the embedded assets
// bundled in the binary.
func promptSkillInstall(ctx context.Context, cfg *config.Config, missing []skills.RequiredSkill, projectDir string) error {
	fmt.Println("\nRequired skills are not installed:")
	for _, s := range missing {
		fmt.Printf("  • %s — %s\n", s.Name, s.Reason)
	}

	globalDir := cfg.GetSkillsDir()
	localDir := filepath.Join(projectDir, ".claude", "skills")

	fmt.Println("\nInstall options:")
	fmt.Printf("  1. Install globally (%s) — available to all repos\n", globalDir)
	fmt.Printf("  2. Install locally (%s) — scoped to this repo\n", localDir)
	fmt.Println("  3. Deny — agents are not compatible without these skills")
	fmt.Print("\nChoice [1/2/3]: ")

	answer, ok := readLineCtx(ctx)
	if !ok {
		return fmt.Errorf("aborted")
	}

	var destDir string
	switch answer {
	case "1":
		destDir = globalDir
	case "2":
		destDir = localDir
	default:
		return fmt.Errorf("agent spawning aborted — required skills not installed")
	}

	fmt.Println("Installing skills from embedded assets...")
	for _, s := range missing {
		path, err := skills.InstallEmbedded(s.Name, destDir)
		if err != nil {
			return fmt.Errorf("install skill %s: %w", s.Name, err)
		}
		fmt.Printf("  • %s → %s\n", s.Name, path)
	}
	fmt.Println()
	return nil
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
