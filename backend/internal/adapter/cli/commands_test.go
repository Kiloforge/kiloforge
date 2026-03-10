package cli

import "testing"

// TestCommandFlags verifies that important command flags are registered.
func TestCommandFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cmdName  string
		flagName string
	}{
		{"agents", "json"},
		{"cost", "json"},
		{"logs", "follow"},
		{"destroy", "force"},
		{"push", "branch"},
		{"push", "all"},
		{"sync", "project"},
		{"add", "name"},
		{"add", "ssh-key"},
	}

	cmds := make(map[string]*struct {
		flags func(string) bool
	})
	// Build lookup from root commands.
	for _, cmd := range rootCmd.Commands() {
		name := cmd.Name()
		f := cmd.Flags()
		cmds[name] = &struct {
			flags func(string) bool
		}{flags: func(n string) bool { return f.Lookup(n) != nil }}
	}

	for _, tt := range tests {
		t.Run(tt.cmdName+"/"+tt.flagName, func(t *testing.T) {
			entry, ok := cmds[tt.cmdName]
			if !ok {
				t.Fatalf("command %q not registered", tt.cmdName)
			}
			if !entry.flags(tt.flagName) {
				t.Errorf("flag --%s not registered on %q", tt.flagName, tt.cmdName)
			}
		})
	}
}

// TestCommandUsage verifies command Use strings are set.
func TestCommandUsage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
	}{
		{"stop", "stop <agent-id>"},
		{"logs", "logs <agent-id>"},
		{"pool", "pool"},
		{"escalated", "escalated"},
	}

	cmds := make(map[string]string)
	for _, cmd := range rootCmd.Commands() {
		cmds[cmd.Name()] = cmd.Use
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			use, ok := cmds[tt.name]
			if !ok {
				t.Fatalf("command %q not registered", tt.name)
			}
			if use != tt.want {
				t.Errorf("Use = %q, want %q", use, tt.want)
			}
		})
	}
}

// TestCommandArgs verifies that commands requiring args have validators set.
func TestCommandArgs(t *testing.T) {
	t.Parallel()

	// These commands should reject being called with no args.
	exactArgsCmds := []string{"stop", "logs", "add"}

	for _, name := range exactArgsCmds {
		t.Run(name, func(t *testing.T) {
			var found bool
			for _, cmd := range rootCmd.Commands() {
				if cmd.Name() == name {
					found = true
					if cmd.Args == nil {
						t.Errorf("command %q should have an Args validator", name)
					}
					break
				}
			}
			if !found {
				t.Fatalf("command %q not registered", name)
			}
		})
	}
}
