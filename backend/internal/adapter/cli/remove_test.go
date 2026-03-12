package cli

import "testing"

func TestRemoveCommand_Registered(t *testing.T) {
	var found bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "remove" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("remove command not registered on root")
	}
}

func TestRemoveCommand_Usage(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "remove" {
			if cmd.Use != "remove <slug>" {
				t.Errorf("Use = %q, want %q", cmd.Use, "remove <slug>")
			}
			return
		}
	}
	t.Fatal("remove command not registered")
}

func TestRemoveCommand_Args(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "remove" {
			if cmd.Args == nil {
				t.Error("remove command should have an Args validator")
			}
			return
		}
	}
	t.Fatal("remove command not registered")
}

func TestRemoveCommand_Flags(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "remove" {
			flags := cmd.Flags()
			if flags.Lookup("cleanup") == nil {
				t.Error("flag --cleanup not registered on remove")
			}
			if flags.Lookup("force") == nil {
				t.Error("flag --force not registered on remove")
			}
			return
		}
	}
	t.Fatal("remove command not registered")
}
