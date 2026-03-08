package cli

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"kiloforge/internal/adapter/auth"
)

// PromptSSHKeySelection displays discovered SSH keys and prompts the user to
// select one. Returns the selected key's private key path, or "" if skipped.
//
// If r is nil (non-interactive), auto-selects the first key without prompting.
// If keys is empty, returns "" immediately.
func PromptSSHKeySelection(keys []auth.SSHKeyInfo, r io.Reader, w io.Writer) (string, error) {
	if len(keys) == 0 {
		return "", nil
	}

	// Non-interactive: auto-select first key.
	if r == nil {
		fmt.Fprintf(w, "    Using SSH key: %s (%s)\n", keys[0].Path, keys[0].Type)
		return keys[0].Path, nil
	}

	// Multiple keys: show selection prompt.
	fmt.Fprintln(w, "SSH remote detected. Select an SSH key for git operations:")
	for i, k := range keys {
		label := k.Type
		if label == "" {
			label = "unknown"
		}
		fmt.Fprintf(w, "  %d) %s (%s)\n", i+1, k.Path, strings.ToUpper(label))
	}
	skipNum := len(keys) + 1
	fmt.Fprintf(w, "  %d) Skip — use default SSH agent\n", skipNum)
	fmt.Fprintf(w, "Select [1]: ")

	scanner := bufio.NewScanner(r)
	for {
		if !scanner.Scan() {
			// EOF — default to first key.
			return keys[0].Path, nil
		}
		line := strings.TrimSpace(scanner.Text())

		// Empty input = default (first key).
		if line == "" {
			return keys[0].Path, nil
		}

		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > skipNum {
			fmt.Fprintf(w, "Invalid selection. Enter 1-%d: ", skipNum)
			continue
		}

		if n == skipNum {
			return "", nil
		}
		return keys[n-1].Path, nil
	}
}
