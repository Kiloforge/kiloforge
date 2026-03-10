package cli

import "testing"

func TestSetVersionInfo(t *testing.T) {
	t.Parallel()

	// Save originals.
	origVersion, origCommit, origDate := appVersion, appCommit, appDate
	t.Cleanup(func() {
		appVersion = origVersion
		appCommit = origCommit
		appDate = origDate
	})

	SetVersionInfo("1.2.3", "abc1234", "2025-01-15")

	if appVersion != "1.2.3" {
		t.Errorf("appVersion = %q, want %q", appVersion, "1.2.3")
	}
	if appCommit != "abc1234" {
		t.Errorf("appCommit = %q, want %q", appCommit, "abc1234")
	}
	if appDate != "2025-01-15" {
		t.Errorf("appDate = %q, want %q", appDate, "2025-01-15")
	}
}

func TestVersionCmd_Registered(t *testing.T) {
	t.Parallel()
	if versionCmd == nil {
		t.Fatal("versionCmd is nil")
	}
	if versionCmd.Use != "version" {
		t.Errorf("Use = %q, want %q", versionCmd.Use, "version")
	}
}
