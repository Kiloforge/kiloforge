package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"text/tabwriter"

	"crelay/internal/agent"
	"crelay/internal/config"
	"crelay/internal/core/domain"
	"crelay/internal/orchestration"
	"crelay/internal/pool"
	"crelay/internal/adapter/persistence/jsonfile"
	"crelay/internal/state"

	"github.com/spf13/cobra"
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
		return fmt.Errorf("not initialized — run 'crelay init' first")
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
		return fmt.Errorf("track ID required\n\nUsage: crelay implement <track-id>\n\nUse --list to see available tracks.")
	}

	trackID := args[0]

	// Validate track is pending.
	tracks, err := orchestration.DiscoverTracks(proj.ProjectDir)
	if err != nil {
		return fmt.Errorf("discover tracks: %w", err)
	}

	var found *orchestration.TrackEntry
	for i := range tracks {
		if tracks[i].ID == trackID {
			found = &tracks[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("track %q not found in project %q", trackID, proj.Slug)
	}
	if found.Status == orchestration.StatusComplete {
		return fmt.Errorf("track %q is already complete", trackID)
	}
	if found.Status == orchestration.StatusInProgress {
		return fmt.Errorf("track %q is already in progress", trackID)
	}

	// Acquire worktree.
	p, err := pool.Load(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load pool: %w", err)
	}
	p.ProjectRoot = proj.ProjectDir

	wt, err := p.Acquire()
	if err != nil {
		return fmt.Errorf("acquire worktree: %w", err)
	}

	// Prepare worktree for track.
	if err := p.Prepare(wt, trackID); err != nil {
		return fmt.Errorf("prepare worktree: %w", err)
	}

	// Spawn developer agent.
	store, err := state.Load(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	logDir := filepath.Join(cfg.DataDir, "projects", proj.Slug, "logs")
	spawner := agent.NewSpawner(cfg, store)
	info, err := spawner.SpawnDeveloper(ctx, agent.SpawnDeveloperOpts{
		TrackID:     trackID,
		Flags:       "--auto-merge",
		WorktreeDir: wt.Path,
		LogDir:      logDir,
	})
	if err != nil {
		return fmt.Errorf("spawn developer: %w", err)
	}

	// Record worktree-agent link.
	wt.AgentID = info.ID

	// Save pool state.
	if err := p.Save(cfg.DataDir); err != nil {
		return fmt.Errorf("save pool: %w", err)
	}

	fmt.Println()
	fmt.Printf("Developer agent spawned for track %q\n", trackID)
	fmt.Printf("  Agent:     %s\n", info.ID[:8])
	fmt.Printf("  Session:   %s\n", info.SessionID[:8])
	fmt.Printf("  Worktree:  %s\n", wt.Path)
	fmt.Printf("  Log:       %s\n", info.LogFile)
	fmt.Println()
	fmt.Printf("View logs:     crelay logs %s\n", info.ID[:8])
	fmt.Printf("Stop agent:    crelay stop %s\n", info.ID[:8])
	fmt.Printf("Resume agent:  crelay attach %s\n", info.ID[:8])
	fmt.Println()

	return nil
}

func resolveProject(reg *jsonfile.ProjectStore, slug string) (domain.Project, error) {
	if slug != "" {
		proj, ok := reg.Get(slug)
		if !ok {
			return domain.Project{}, fmt.Errorf("project %q not found — use 'crelay add' to register", slug)
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
		return domain.Project{}, fmt.Errorf("no project registered for %s — use 'crelay add' or --project flag", cwd)
	}
	return proj, nil
}

func listTracks(proj domain.Project) error {
	tracks, err := orchestration.DiscoverTracks(proj.ProjectDir)
	if err != nil {
		return fmt.Errorf("discover tracks: %w", err)
	}

	pending := orchestration.FilterByStatus(tracks, orchestration.StatusPending)
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
