package compose

import (
	"strings"
	"testing"
)

func TestGenerateComposeFile_DefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := ComposeConfig{
		GiteaPort: 3000,
		RelayPort: 3001,
		DataDir:   "/home/user/.crelay",
	}

	data, err := GenerateComposeFile(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(data)

	// Verify key fields are present.
	checks := []string{
		"gitea/gitea:latest",
		"conductor-gitea",
		"3000:3000",
		"gitea-data:/data",
		"GITEA__security__INSTALL_LOCK=true",
		"GITEA__server__ROOT_URL=http://localhost:3001/gitea/",
		"GITEA__database__DB_TYPE=sqlite3",
		"GITEA__service__DISABLE_REGISTRATION=true",
		"GITEA__webhook__ALLOWED_HOST_LIST=*",
		"host.docker.internal:host-gateway",
		"healthcheck:",
		"volumes:",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("expected compose file to contain %q", check)
		}
	}
}

func TestGenerateComposeFile_CustomPort(t *testing.T) {
	t.Parallel()

	cfg := ComposeConfig{
		GiteaPort: 4000,
		RelayPort: 5000,
		DataDir:   "/tmp/crelay",
	}

	data, err := GenerateComposeFile(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "4000:3000") {
		t.Error("expected port mapping 4000:3000")
	}
	if !strings.Contains(content, "ROOT_URL=http://localhost:5000/gitea/") {
		t.Error("expected ROOT_URL with relay port 5000 and /gitea/ sub-path")
	}
}
