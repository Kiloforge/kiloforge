package service_test

import (
	"strings"
	"testing"

	"crelay/internal/core/service"
)

func TestParseTracks_ValidMarkdown(t *testing.T) {
	t.Parallel()

	input := `# Tracks Registry

| Status | Track ID | Title | Created | Updated |
| ------ | -------- | ----- | ------- | ------- |
| [x] | track-1 | First Track | 2026-03-07 | 2026-03-07 |
| [ ] | track-2 | Second Track | 2026-03-07 | 2026-03-07 |
| [~] | track-3 | Third Track | 2026-03-07 | 2026-03-07 |
`

	tracks, err := service.ParseTracks(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 3 {
		t.Fatalf("expected 3 tracks, got %d", len(tracks))
	}

	tests := []struct {
		idx    int
		id     string
		title  string
		status string
	}{
		{0, "track-1", "First Track", service.StatusComplete},
		{1, "track-2", "Second Track", service.StatusPending},
		{2, "track-3", "Third Track", service.StatusInProgress},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := tracks[tt.idx]
			if got.ID != tt.id {
				t.Errorf("ID = %q, want %q", got.ID, tt.id)
			}
			if got.Title != tt.title {
				t.Errorf("Title = %q, want %q", got.Title, tt.title)
			}
			if got.Status != tt.status {
				t.Errorf("Status = %q, want %q", got.Status, tt.status)
			}
		})
	}
}

func TestParseTracks_EmptyInput(t *testing.T) {
	t.Parallel()

	tracks, err := service.ParseTracks(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 0 {
		t.Errorf("expected 0 tracks, got %d", len(tracks))
	}
}

func TestParseTracks_SkipsHeaderAndSeparator(t *testing.T) {
	t.Parallel()

	input := `| Status | Track ID | Title | Created | Updated |
| ------ | -------- | ----- | ------- | ------- |
| [x] | valid-track | Valid Title | 2026-01-01 | 2026-01-01 |`

	tracks, err := service.ParseTracks(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(tracks))
	}
	if tracks[0].ID != "valid-track" {
		t.Errorf("ID = %q, want %q", tracks[0].ID, "valid-track")
	}
}

func TestParseTracks_MalformedLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		count int
	}{
		{"no pipes", "not a table line", 0},
		{"too few columns", "| a | b |", 0},
		{"invalid status", "| invalid | id | title | 2026 | 2026 |", 0},
		{"mixed valid and invalid", "| [x] | good | Title | 2026 | 2026 |\nnot a line\n| [ ] | good2 | Title2 | 2026 | 2026 |", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tracks, err := service.ParseTracks(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tracks) != tt.count {
				t.Errorf("got %d tracks, want %d", len(tracks), tt.count)
			}
		})
	}
}

func TestParseTracks_NewStatuses(t *testing.T) {
	t.Parallel()

	input := `| [!] | track-approved | Approved Track | 2026-03-08 | 2026-03-08 |
| [r] | track-review | In Review Track | 2026-03-08 | 2026-03-08 |`

	tracks, err := service.ParseTracks(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(tracks))
	}
	if tracks[0].Status != service.StatusApproved {
		t.Errorf("status: want %q, got %q", service.StatusApproved, tracks[0].Status)
	}
	if tracks[1].Status != service.StatusInReview {
		t.Errorf("status: want %q, got %q", service.StatusInReview, tracks[1].Status)
	}
}

func TestFilterByStatus(t *testing.T) {
	t.Parallel()

	tracks := []service.TrackEntry{
		{ID: "t1", Status: service.StatusComplete},
		{ID: "t2", Status: service.StatusPending},
		{ID: "t3", Status: service.StatusPending},
		{ID: "t4", Status: service.StatusInProgress},
	}

	pending := service.FilterByStatus(tracks, service.StatusPending)
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(pending))
	}
	if pending[0].ID != "t2" || pending[1].ID != "t3" {
		t.Errorf("unexpected IDs: %v, %v", pending[0].ID, pending[1].ID)
	}

	complete := service.FilterByStatus(tracks, service.StatusComplete)
	if len(complete) != 1 {
		t.Fatalf("expected 1 complete, got %d", len(complete))
	}

	empty := service.FilterByStatus(tracks, "nonexistent")
	if len(empty) != 0 {
		t.Errorf("expected 0 results for nonexistent status, got %d", len(empty))
	}
}

func TestFilterByStatus_NilSlice(t *testing.T) {
	t.Parallel()

	result := service.FilterByStatus(nil, service.StatusPending)
	if len(result) != 0 {
		t.Errorf("expected 0, got %d", len(result))
	}
}
