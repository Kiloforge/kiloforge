package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/persistence/jsonfile"
	"kiloforge/internal/core/service"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync conductor tracks to the native board",
	Long: `Discover tracks from a project's .agent/conductor/tracks.md and sync them
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

	cfg, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("not initialized — run 'kf init' first")
	}

	reg, err := jsonfile.LoadProjectStore(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}
	project, ok := reg.Get(flagSyncProject)
	if !ok {
		return fmt.Errorf("project %q not found", flagSyncProject)
	}

	boardStore := jsonfile.NewBoardStore(cfg.DataDir)
	boardSvc := service.NewNativeBoardService(boardStore)

	// Discover tracks.
	tracks, err := service.DiscoverTracks(project.ProjectDir)
	if err != nil {
		return fmt.Errorf("discover tracks: %w", err)
	}

	if len(tracks) == 0 {
		fmt.Println("No tracks found.")
		return nil
	}

	// Build track types map from metadata.json files.
	trackTypes := make(map[string]string)
	for _, t := range tracks {
		trackTypes[t.ID] = readTrackType(project.ProjectDir, t.ID)
	}

	// Sync.
	result, err := boardSvc.SyncFromTracks(project.Slug, tracks, trackTypes)
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	fmt.Printf("Sync complete for project %q\n", project.Slug)
	fmt.Printf("  Created:   %d\n", result.Created)
	fmt.Printf("  Updated:   %d\n", result.Updated)
	fmt.Printf("  Unchanged: %d\n", result.Unchanged)

	return nil
}

// readTrackType reads the track type from metadata.json.
func readTrackType(projectDir, trackID string) string {
	path := filepath.Join(projectDir, ".agent", "conductor", "tracks", trackID, "metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var meta struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return ""
	}
	return meta.Type
}
