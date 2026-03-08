package rest

import "testing"

func TestExtractTrackIDFromPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload map[string]any
		want    string
	}{
		{
			name:    "no pull_request",
			payload: map[string]any{"action": "opened"},
			want:    "",
		},
		{
			name: "no head ref",
			payload: map[string]any{
				"pull_request": map[string]any{
					"head": map[string]any{},
				},
			},
			want: "",
		},
		{
			name: "feature branch",
			payload: map[string]any{
				"pull_request": map[string]any{
					"head": map[string]any{
						"ref": "feature/auth_20250115100000Z",
					},
				},
			},
			want: "auth_20250115100000Z",
		},
		{
			name: "chore branch",
			payload: map[string]any{
				"pull_request": map[string]any{
					"head": map[string]any{
						"ref": "chore/rebrand-kiloforge_20260309055250Z",
					},
				},
			},
			want: "rebrand-kiloforge_20260309055250Z",
		},
		{
			name: "bare branch name",
			payload: map[string]any{
				"pull_request": map[string]any{
					"head": map[string]any{
						"ref": "my-branch",
					},
				},
			},
			want: "my-branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractTrackIDFromPayload(tt.payload)
			if got != tt.want {
				t.Errorf("extractTrackIDFromPayload() = %q, want %q", got, tt.want)
			}
		})
	}
}
