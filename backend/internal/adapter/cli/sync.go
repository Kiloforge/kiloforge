package cli

import (
	"fmt"

	"kiloforge/internal/core/service"
	"kiloforge/pkg/kf"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync kf tracks to the native board",
	Long: `Discover tracks from a project's .agent/kf/tracks.yaml and sync them
to the native track board. Creates cards for new tracks and updates columns
for tracks that changed status.

Examples:
  kf sync --project myapp`,
	RunE: runSync,
}

var flagSyncProject string

func init() {
	syncCmd.Flags().StringVar(&flagSyncProject, "project", "", "Project slug (required)")
}

func runSync(cmd *cobra.Command, args []string) error {
	if flagSyncProject == "" {
		return fmt.Errorf("--project is required")
	}

	rt, err := NewCLIRuntime()
	if err != nil {
		return fmt.Errorf("%s", notInitializedError())
	}
	defer func() { _ = rt.Close() }()

	project, err := rt.Projects.GetProject(flagSyncProject)
	if err != nil {
		return fmt.Errorf("project %q not found", flagSyncProject)
	}

	// Discover tracks via kf SDK.
	reader := service.NewTrackReader()
	tracks, err := reader.DiscoverTracks(project.ProjectDir)
	if err != nil {
		return fmt.Errorf("discover tracks: %w", err)
	}

	if len(tracks) == 0 {
		fmt.Println(emptyState("tracks found", "Create tracks in .agent/kf/tracks.yaml or use the architect workflow."))
		return nil
	}

	// Build track types map from kf SDK.
	client := kf.NewClientFromProject(project.ProjectDir)
	trackTypes := make(map[string]string)
	for _, t := range tracks {
		if entry, err := client.GetTrackEntry(t.ID); err == nil {
			trackTypes[t.ID] = entry.Type
		}
	}

	// Sync.
	result, err := rt.Board.SyncFromTracks(project.Slug, tracks, trackTypes)
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	fmt.Printf("Sync complete for project %q\n", project.Slug)
	fmt.Printf("  Created:   %d\n", result.Created)
	fmt.Printf("  Updated:   %d\n", result.Updated)
	fmt.Printf("  Unchanged: %d\n", result.Unchanged)

	return nil
}
