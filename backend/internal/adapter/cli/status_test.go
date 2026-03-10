package cli

import (
	"testing"

	"kiloforge/pkg/kf"
)

func TestFormatTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1,000"},
		{12345, "12,345"},
		{1234567, "1,234,567"},
		{100000, "100,000"},
	}

	for _, tt := range tests {
		got := formatTokens(tt.input)
		if got != tt.want {
			t.Errorf("formatTokens(%d): want %q, got %q", tt.input, tt.want, got)
		}
	}
}

func TestFormatTrackSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		summary *kf.TrackSummary
		want    string
	}{
		{
			name: "mixed completed and archived",
			summary: &kf.TrackSummary{
				Total:      20,
				Pending:    1,
				InProgress: 2,
				Completed:  5,
				Archived:   12,
			},
			want: "Tracks:      17/20 done (1 pending, 2 in-progress)",
		},
		{
			name: "all completed no archived",
			summary: &kf.TrackSummary{
				Total:     5,
				Completed: 5,
			},
			want: "Tracks:      5/5 done",
		},
		{
			name: "all archived",
			summary: &kf.TrackSummary{
				Total:    10,
				Archived: 10,
			},
			want: "Tracks:      10/10 done",
		},
		{
			name: "nothing done yet",
			summary: &kf.TrackSummary{
				Total:      3,
				Pending:    2,
				InProgress: 1,
			},
			want: "Tracks:      0/3 done (2 pending, 1 in-progress)",
		},
		{
			name: "only pending",
			summary: &kf.TrackSummary{
				Total:   4,
				Pending: 4,
			},
			want: "Tracks:      0/4 done (4 pending)",
		},
		{
			name: "only in-progress",
			summary: &kf.TrackSummary{
				Total:      2,
				InProgress: 2,
			},
			want: "Tracks:      0/2 done (2 in-progress)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTrackSummary(tt.summary)
			if got != tt.want {
				t.Errorf("formatTrackSummary():\n  want: %q\n  got:  %q", tt.want, got)
			}
		})
	}
}
