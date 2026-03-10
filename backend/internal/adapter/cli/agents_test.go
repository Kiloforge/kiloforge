package cli

import "testing"

func TestAgentStatusInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    string
		resumeErr string
		want      string
	}{
		{"resume-failed with error", "resume-failed", "timeout exceeded", "timeout exceeded"},
		{"resume-failed no error", "resume-failed", "", "resume failed"},
		{"suspended", "suspended", "", "will auto-resume on startup"},
		{"force-killed", "force-killed", "", "session may be corrupt"},
		{"running", "running", "", ""},
		{"completed", "completed", "", ""},
		{"empty status", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agentStatusInfo(tt.status, tt.resumeErr)
			if got != tt.want {
				t.Errorf("agentStatusInfo(%q, %q) = %q, want %q", tt.status, tt.resumeErr, got, tt.want)
			}
		})
	}
}
