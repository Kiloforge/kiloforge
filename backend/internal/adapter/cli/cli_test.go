package cli

import (
	"testing"
)

// TestAllCommandsRegistered verifies every expected subcommand is registered
// on the root command. If a command is accidentally removed or not added,
// this test catches it.
func TestAllCommandsRegistered(t *testing.T) {
	t.Parallel()

	expected := []string{
		"init", "up", "down", "status", "add", "projects",
		"destroy", "pool", "implement", "agents", "logs",
		"stop", "attach", "escalated", "cost", "dashboard",
		"board", "sync", "serve",
	}

	registered := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		registered[cmd.Name()] = true
	}

	for _, name := range expected {
		if !registered[name] {
			t.Errorf("expected command %q to be registered on root", name)
		}
	}
}

// TestCommandHelp verifies that --help doesn't panic for each command.
// Not parallel because Cobra's global rootCmd is not thread-safe.
func TestCommandHelp(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		t.Run(cmd.Name(), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("command %q panicked on --help: %v", cmd.Name(), r)
				}
			}()
			cmd.SetArgs([]string{"--help"})
			cmd.Execute()
		})
	}
}
