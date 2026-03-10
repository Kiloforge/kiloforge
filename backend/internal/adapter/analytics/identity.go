package analytics

import (
	"crypto/sha256"
	"fmt"
	"os"
)

// AnonymousID returns a stable, non-reversible identifier for this machine.
// It is computed as SHA-256(hostname + dataDir) — no PII is stored or transmitted.
func AnonymousID(dataDir string) string {
	hostname, _ := os.Hostname()
	hash := sha256.Sum256([]byte(hostname + dataDir))
	return fmt.Sprintf("%x", hash[:16]) // 32-char hex string
}
