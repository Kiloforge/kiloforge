package compose

import (
	"strings"
	"testing"
)

func TestGenerateComposeFile_EmptyServices(t *testing.T) {
	t.Parallel()

	cfg := ComposeConfig{
		OrchestratorPort: 3001,
		DataDir:          "/home/user/.kiloforge",
	}

	data, err := GenerateComposeFile(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "services:") {
		t.Error("expected compose file to contain services key")
	}
}
