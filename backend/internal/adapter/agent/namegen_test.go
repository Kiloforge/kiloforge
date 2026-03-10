package agent

import (
	"strings"
	"testing"
)

func TestGenerateName_Format(t *testing.T) {
	t.Parallel()
	name := GenerateName()
	parts := strings.SplitN(name, "-", 3)
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d: %q", len(parts), name)
	}
	for i, p := range parts {
		if p == "" {
			t.Errorf("part %d is empty in %q", i, name)
		}
	}
}

func TestGenerateName_Randomness(t *testing.T) {
	t.Parallel()
	seen := make(map[string]bool)
	for range 20 {
		seen[GenerateName()] = true
	}
	if len(seen) < 5 {
		t.Errorf("expected variety in 20 names, got %d unique", len(seen))
	}
}
