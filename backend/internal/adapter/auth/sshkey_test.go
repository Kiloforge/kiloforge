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
