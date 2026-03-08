package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectSSHKey_FindsEd25519First(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ed25519 := filepath.Join(dir, "id_ed25519.pub")
	rsa := filepath.Join(dir, "id_rsa.pub")

	os.WriteFile(ed25519, []byte("ssh-ed25519 AAAA testkey"), 0o644)
	os.WriteFile(rsa, []byte("ssh-rsa BBBB testkey"), 0o644)

	path, content, err := DetectSSHKey(dir)
	if err != nil {
		t.Fatalf("DetectSSHKey: %v", err)
	}
	if path != ed25519 {
		t.Errorf("path: want %q, got %q", ed25519, path)
	}
	if content != "ssh-ed25519 AAAA testkey" {
		t.Errorf("content: want %q, got %q", "ssh-ed25519 AAAA testkey", content)
	}
}

func TestDetectSSHKey_FallsBackToRSA(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rsa := filepath.Join(dir, "id_rsa.pub")
	os.WriteFile(rsa, []byte("ssh-rsa BBBB testkey"), 0o644)

	path, _, err := DetectSSHKey(dir)
	if err != nil {
		t.Fatalf("DetectSSHKey: %v", err)
	}
	if path != rsa {
		t.Errorf("path: want %q, got %q", rsa, path)
	}
}

func TestDetectSSHKey_FallsBackToECDSA(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ecdsa := filepath.Join(dir, "id_ecdsa.pub")
	os.WriteFile(ecdsa, []byte("ecdsa-sha2-nistp256 CCCC testkey"), 0o644)

	path, _, err := DetectSSHKey(dir)
	if err != nil {
		t.Fatalf("DetectSSHKey: %v", err)
	}
	if path != ecdsa {
		t.Errorf("path: want %q, got %q", ecdsa, path)
	}
}

func TestDetectSSHKey_NoKeyFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, _, err := DetectSSHKey(dir)
	if err == nil {
		t.Fatal("DetectSSHKey: expected error for missing keys")
	}
}

func TestDetectSSHKey_TrimsWhitespace(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ed25519 := filepath.Join(dir, "id_ed25519.pub")
	os.WriteFile(ed25519, []byte("ssh-ed25519 AAAA testkey\n"), 0o644)

	_, content, err := DetectSSHKey(dir)
	if err != nil {
		t.Fatalf("DetectSSHKey: %v", err)
	}
	if content != "ssh-ed25519 AAAA testkey" {
		t.Errorf("content should be trimmed, got %q", content)
	}
}

func TestDiscoverSSHKeys_MultipleKeys(t *testing.T) {
	dir := t.TempDir()

	// Create fake key pairs (private + .pub).
	keys := []struct {
		name    string
		content string
		pub     string
	}{
		{"id_ed25519", "fake-ed25519-private", "ssh-ed25519 AAAA... user@host"},
		{"id_rsa", "fake-rsa-private", "ssh-rsa AAAA... user@host"},
	}
	for _, k := range keys {
		os.WriteFile(filepath.Join(dir, k.name), []byte(k.content), 0o600)
		os.WriteFile(filepath.Join(dir, k.name+".pub"), []byte(k.pub), 0o644)
	}

	got := DiscoverSSHKeys(dir)
	if len(got) != 2 {
		t.Fatalf("want 2 keys, got %d", len(got))
	}
	if got[0].Name != "id_ed25519" {
		t.Errorf("first key name = %q, want id_ed25519", got[0].Name)
	}
	if got[0].Type != "ed25519" {
		t.Errorf("first key type = %q, want ed25519", got[0].Type)
	}
	if got[1].Name != "id_rsa" {
		t.Errorf("second key name = %q, want id_rsa", got[1].Name)
	}
}

func TestDiscoverSSHKeys_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	got := DiscoverSSHKeys(dir)
	if len(got) != 0 {
		t.Fatalf("want 0 keys, got %d", len(got))
	}
}

func TestDiscoverSSHKeys_OnlyPrivateNoPublic(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "id_ed25519"), []byte("private"), 0o600)

	got := DiscoverSSHKeys(dir)
	if len(got) != 1 {
		t.Fatalf("want 1 key, got %d", len(got))
	}
	if got[0].PubContent != "" {
		t.Errorf("pub content should be empty when .pub file missing, got %q", got[0].PubContent)
	}
}

func TestDiscoverSSHKeys_IncludesNonStandardKeys(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "deploy_key"), []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nfake"), 0o600)
	os.WriteFile(filepath.Join(dir, "deploy_key.pub"), []byte("ssh-ed25519 BBBB... deploy"), 0o644)

	got := DiscoverSSHKeys(dir)
	if len(got) != 1 {
		t.Fatalf("want 1 key, got %d", len(got))
	}
	if got[0].Name != "deploy_key" {
		t.Errorf("name = %q, want deploy_key", got[0].Name)
	}
}

func TestDiscoverSSHKeys_SkipsNonKeyFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "known_hosts"), []byte("github.com ..."), 0o644)
	os.WriteFile(filepath.Join(dir, "config"), []byte("Host *"), 0o644)
	os.WriteFile(filepath.Join(dir, "authorized_keys"), []byte("ssh-rsa ..."), 0o644)
	// .pub only without private key — should also be skipped.
	os.WriteFile(filepath.Join(dir, "id_ed25519.pub"), []byte("ssh-ed25519 AAAA..."), 0o644)

	got := DiscoverSSHKeys(dir)
	if len(got) != 0 {
		t.Fatalf("want 0 keys (non-key files), got %d", len(got))
	}
}
