package cli

import (
	"testing"
)

func TestRepoNameFromURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		want    string
		wantErr bool
	}{
		{
			name:   "SSH URL with .git suffix",
			rawURL: "git@github.com:user/my-project.git",
			want:   "my-project",
		},
		{
			name:   "SSH URL without .git suffix",
			rawURL: "git@github.com:user/my-project",
			want:   "my-project",
		},
		{
			name:   "HTTPS URL with .git suffix",
			rawURL: "https://github.com/user/my-project.git",
			want:   "my-project",
		},
		{
			name:   "HTTPS URL without .git suffix",
			rawURL: "https://github.com/user/my-project",
			want:   "my-project",
		},
		{
			name:   "HTTPS URL with trailing slash",
			rawURL: "https://github.com/user/my-project/",
			want:   "my-project",
		},
		{
			name:   "SSH URL with nested path",
			rawURL: "git@gitlab.com:group/subgroup/repo.git",
			want:   "repo",
		},
		{
			name:   "HTTPS URL with nested path",
			rawURL: "https://gitlab.com/group/subgroup/repo.git",
			want:   "repo",
		},
		{
			name:    "empty string",
			rawURL:  "",
			wantErr: true,
		},
		{
			name:    "just a hostname",
			rawURL:  "github.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repoNameFromURL(tt.rawURL)
			if tt.wantErr {
				if err == nil {
					t.Errorf("repoNameFromURL(%q) = %q, want error", tt.rawURL, got)
				}
				return
			}
			if err != nil {
				t.Errorf("repoNameFromURL(%q) error = %v", tt.rawURL, err)
				return
			}
			if got != tt.want {
				t.Errorf("repoNameFromURL(%q) = %q, want %q", tt.rawURL, got, tt.want)
			}
		})
	}
}

func TestIsRemoteURL(t *testing.T) {
	tests := []struct {
		arg  string
		want bool
	}{
		{"git@github.com:user/repo.git", true},
		{"https://github.com/user/repo.git", true},
		{"http://github.com/user/repo.git", true},
		{"ssh://git@github.com/user/repo.git", true},
		{".", false},
		{"/some/local/path", false},
		{"relative/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			got := isRemoteURL(tt.arg)
			if got != tt.want {
				t.Errorf("isRemoteURL(%q) = %v, want %v", tt.arg, got, tt.want)
			}
		})
	}
}
