package auth

import (
	"testing"
)

func TestGeneratePassword_Length(t *testing.T) {
	t.Parallel()

	for _, length := range []int{8, 16, 20, 32} {
		pw := GeneratePassword(length)
		if len(pw) != length {
			t.Errorf("GeneratePassword(%d): got length %d", length, len(pw))
		}
	}
}

func TestGeneratePassword_CharacterSet(t *testing.T) {
	t.Parallel()

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	pw := GeneratePassword(100)
	for i, c := range pw {
		found := false
		for _, valid := range charset {
			if c == valid {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GeneratePassword: char at index %d (%c) not in charset", i, c)
		}
	}
}

func TestGeneratePassword_Uniqueness(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)
	for range 50 {
		pw := GeneratePassword(20)
		if seen[pw] {
			t.Errorf("GeneratePassword: duplicate password %q", pw)
		}
		seen[pw] = true
	}
}
