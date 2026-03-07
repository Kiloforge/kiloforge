package cli

import "testing"

func TestFormatTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1,000"},
		{12345, "12,345"},
		{1234567, "1,234,567"},
		{100000, "100,000"},
	}

	for _, tt := range tests {
		got := formatTokens(tt.input)
		if got != tt.want {
			t.Errorf("formatTokens(%d): want %q, got %q", tt.input, tt.want, got)
		}
	}
}
