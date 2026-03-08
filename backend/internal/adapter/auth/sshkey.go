package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// sshKeyNames lists public key filenames in preference order.
var sshKeyNames = []string{
	"id_ed25519.pub",
	"id_rsa.pub",
	"id_ecdsa.pub",
}

// knownNonKeyFiles are filenames in ~/.ssh/ that are not SSH keys.
var knownNonKeyFiles = map[string]bool{
	"known_hosts":     true,
	"known_hosts.old": true,
	"config":          true,
	"authorized_keys": true,
	"environment":     true,
	"rc":              true,
}

// SSHKeyInfo describes a discovered SSH key pair.
type SSHKeyInfo struct {
	Name       string // e.g., "id_ed25519"
	Path       string // full path to private key
	Type       string // e.g., "ed25519", "rsa", "ecdsa"
	PubContent string // trimmed content of .pub file (empty if .pub missing)
}

// DetectSSHKey searches sshDir for a public key file.
// Returns the path and trimmed content of the first key found.
func DetectSSHKey(sshDir string) (path, content string, err error) {
	for _, name := range sshKeyNames {
		p := filepath.Join(sshDir, name)
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		return p, strings.TrimSpace(string(data)), nil
	}
	return "", "", fmt.Errorf("no SSH public key found in %s", sshDir)
}

// DiscoverSSHKeys scans sshDir for SSH private keys and returns info about each.
// Standard keys (id_ed25519, id_rsa, id_ecdsa) are listed first in preference order,
// followed by any non-standard keys found by scanning all files.
func DiscoverSSHKeys(sshDir string) []SSHKeyInfo {
	var keys []SSHKeyInfo
	seen := make(map[string]bool)

	// First pass: standard key names in preference order.
	standardPrivateKeys := []string{"id_ed25519", "id_rsa", "id_ecdsa"}
	for _, name := range standardPrivateKeys {
		privPath := filepath.Join(sshDir, name)
		if _, err := os.Stat(privPath); err != nil {
			continue
		}
		info := buildKeyInfo(sshDir, name)
		keys = append(keys, info)
		seen[name] = true
	}

	// Second pass: discover non-standard keys.
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return keys
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if seen[name] {
			continue
		}
		// Skip .pub files, known non-key files, and dotfiles.
		if strings.HasSuffix(name, ".pub") {
			continue
		}
		if knownNonKeyFiles[name] {
			continue
		}
		if strings.HasPrefix(name, ".") {
			continue
		}
		// Check if it looks like a private key (has content, not too large).
		privPath := filepath.Join(sshDir, name)
		fi, err := os.Stat(privPath)
		if err != nil || fi.Size() == 0 || fi.Size() > 100*1024 {
			continue
		}
		// Read first bytes to check for key header.
		data, err := os.ReadFile(privPath)
		if err != nil {
			continue
		}
		content := string(data)
		if !strings.Contains(content, "PRIVATE KEY") && !isLikelySSHKey(content) {
			continue
		}
		info := buildKeyInfo(sshDir, name)
		keys = append(keys, info)
		seen[name] = true
	}

	return keys
}

// buildKeyInfo creates an SSHKeyInfo for a private key file.
func buildKeyInfo(sshDir, name string) SSHKeyInfo {
	info := SSHKeyInfo{
		Name: name,
		Path: filepath.Join(sshDir, name),
		Type: keyTypeFromName(name),
	}
	// Try to read the matching .pub file.
	pubPath := info.Path + ".pub"
	if pubData, err := os.ReadFile(pubPath); err == nil {
		info.PubContent = strings.TrimSpace(string(pubData))
	}
	return info
}

// keyTypeFromName infers the key type from the filename.
func keyTypeFromName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "ed25519"):
		return "ed25519"
	case strings.Contains(lower, "ecdsa"):
		return "ecdsa"
	case strings.Contains(lower, "rsa"):
		return "rsa"
	case strings.Contains(lower, "dsa"):
		return "dsa"
	default:
		return "unknown"
	}
}

// isLikelySSHKey checks if content looks like an SSH private key
// without the standard PEM header (e.g., OpenSSH format).
func isLikelySSHKey(content string) bool {
	return strings.HasPrefix(content, "-----BEGIN OPENSSH PRIVATE KEY-----")
}
