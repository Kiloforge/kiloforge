package cli

import (
	"bytes"
	"strings"
	"testing"

	"kiloforge/internal/adapter/auth"
)

func TestPromptSSHKeySelection_SelectsFirst(t *testing.T) {
	keys := []auth.SSHKeyInfo{
		{Name: "id_ed25519", Path: "/home/user/.ssh/id_ed25519", Type: "ed25519", PubContent: "ssh-ed25519 AAAA..."},
		{Name: "id_rsa", Path: "/home/user/.ssh/id_rsa", Type: "rsa", PubContent: "ssh-rsa BBBB..."},
	}

	input := strings.NewReader("1\n")
	var output bytes.Buffer

	path, err := PromptSSHKeySelection(keys, input, &output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/home/user/.ssh/id_ed25519" {
		t.Errorf("path = %q, want /home/user/.ssh/id_ed25519", path)
	}
}

func TestPromptSSHKeySelection_SelectsSecond(t *testing.T) {
	keys := []auth.SSHKeyInfo{
		{Name: "id_ed25519", Path: "/home/user/.ssh/id_ed25519", Type: "ed25519"},
		{Name: "id_rsa", Path: "/home/user/.ssh/id_rsa", Type: "rsa"},
	}

	input := strings.NewReader("2\n")
	var output bytes.Buffer

	path, err := PromptSSHKeySelection(keys, input, &output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/home/user/.ssh/id_rsa" {
		t.Errorf("path = %q, want /home/user/.ssh/id_rsa", path)
	}
}

func TestPromptSSHKeySelection_SkipOption(t *testing.T) {
	keys := []auth.SSHKeyInfo{
		{Name: "id_ed25519", Path: "/home/user/.ssh/id_ed25519", Type: "ed25519"},
	}

	// Select the skip option (last option = len(keys) + 1).
	input := strings.NewReader("2\n")
	var output bytes.Buffer

	path, err := PromptSSHKeySelection(keys, input, &output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Errorf("path = %q, want empty (skip)", path)
	}
}

func TestPromptSSHKeySelection_DefaultOnEmpty(t *testing.T) {
	keys := []auth.SSHKeyInfo{
		{Name: "id_ed25519", Path: "/home/user/.ssh/id_ed25519", Type: "ed25519"},
		{Name: "id_rsa", Path: "/home/user/.ssh/id_rsa", Type: "rsa"},
	}

	// Empty input should default to first key.
	input := strings.NewReader("\n")
	var output bytes.Buffer

	path, err := PromptSSHKeySelection(keys, input, &output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/home/user/.ssh/id_ed25519" {
		t.Errorf("path = %q, want /home/user/.ssh/id_ed25519 (default)", path)
	}
}

func TestPromptSSHKeySelection_InvalidThenValid(t *testing.T) {
	keys := []auth.SSHKeyInfo{
		{Name: "id_ed25519", Path: "/home/user/.ssh/id_ed25519", Type: "ed25519"},
	}

	// First input invalid, second valid.
	input := strings.NewReader("99\n1\n")
	var output bytes.Buffer

	path, err := PromptSSHKeySelection(keys, input, &output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/home/user/.ssh/id_ed25519" {
		t.Errorf("path = %q, want /home/user/.ssh/id_ed25519", path)
	}
	if !strings.Contains(output.String(), "Invalid") {
		t.Error("expected 'Invalid' in output for bad input")
	}
}

func TestPromptSSHKeySelection_SingleKeyAutoSelect(t *testing.T) {
	keys := []auth.SSHKeyInfo{
		{Name: "id_ed25519", Path: "/home/user/.ssh/id_ed25519", Type: "ed25519"},
	}

	// When autoSelect is true (nil reader), single key is auto-selected.
	var output bytes.Buffer

	path, err := PromptSSHKeySelection(keys, nil, &output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/home/user/.ssh/id_ed25519" {
		t.Errorf("path = %q, want /home/user/.ssh/id_ed25519 (auto)", path)
	}
	if !strings.Contains(output.String(), "Using SSH key") {
		t.Error("expected auto-select message")
	}
}

func TestPromptSSHKeySelection_NoKeys(t *testing.T) {
	var output bytes.Buffer

	path, err := PromptSSHKeySelection(nil, nil, &output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Errorf("path = %q, want empty for no keys", path)
	}
}

func TestPromptSSHKeySelection_NonInteractiveFallback(t *testing.T) {
	keys := []auth.SSHKeyInfo{
		{Name: "id_ed25519", Path: "/home/user/.ssh/id_ed25519", Type: "ed25519"},
		{Name: "id_rsa", Path: "/home/user/.ssh/id_rsa", Type: "rsa"},
	}

	// nil reader = non-interactive; multiple keys should auto-select first.
	var output bytes.Buffer

	path, err := PromptSSHKeySelection(keys, nil, &output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/home/user/.ssh/id_ed25519" {
		t.Errorf("path = %q, want /home/user/.ssh/id_ed25519 (auto-detect fallback)", path)
	}
}
