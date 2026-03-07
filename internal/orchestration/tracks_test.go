package orchestration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverTracks_ParsesTracksMarkdown(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	conductorDir := filepath.Join(dir, ".agent", "conductor")
	os.MkdirAll(conductorDir, 0o755)

	tracksContent := `# Tracks Registry

| Status | Track ID | Title | Created | Updated |
| ------ | -------- | ----- | ------- | ------- |
| [x] | completed-track_20260101Z | Completed Track | 2026-01-01 | 2026-01-01 |
| [ ] | pending-track_20260102Z | Pending Track | 2026-01-02 | 2026-01-02 |
| [~] | in-progress-track_20260103Z | In-Progress Track | 2026-01-03 | 2026-01-03 |
| [ ] | another-pending_20260104Z | Another Pending | 2026-01-04 | 2026-01-04 |
`
	os.WriteFile(filepath.Join(conductorDir, "tracks.md"), []byte(tracksContent), 0o644)

	tracks, err := DiscoverTracks(dir)
	if err != nil {
		t.Fatalf("DiscoverTracks: %v", err)
	}

	if len(tracks) != 4 {
		t.Fatalf("expected 4 tracks, got %d", len(tracks))
	}
}

func TestDiscoverTracks_FilterPending(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	conductorDir := filepath.Join(dir, ".agent", "conductor")
	os.MkdirAll(conductorDir, 0o755)

	tracksContent := `# Tracks Registry

| Status | Track ID | Title | Created | Updated |
| ------ | -------- | ----- | ------- | ------- |
| [x] | done_20260101Z | Done | 2026-01-01 | 2026-01-01 |
| [ ] | pending_20260102Z | Pending | 2026-01-02 | 2026-01-02 |
| [~] | wip_20260103Z | WIP | 2026-01-03 | 2026-01-03 |
`
	os.WriteFile(filepath.Join(conductorDir, "tracks.md"), []byte(tracksContent), 0o644)

	tracks, err := DiscoverTracks(dir)
	if err != nil {
		t.Fatalf("DiscoverTracks: %v", err)
	}

	pending := FilterByStatus(tracks, StatusPending)
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
	if pending[0].ID != "pending_20260102Z" {
		t.Errorf("expected pending_20260102Z, got %s", pending[0].ID)
	}
}

func TestDiscoverTracks_NoFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := DiscoverTracks(dir)
	if err == nil {
		t.Fatal("expected error for missing tracks.md")
	}
}
