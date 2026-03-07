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
