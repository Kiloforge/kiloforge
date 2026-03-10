package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"kiloforge/internal/adapter/config"

	"github.com/spf13/cobra"
)

// writeTestConfig creates a minimal config.json in dir.
func writeTestConfig(t *testing.T, dir string, analyticsEnabled *bool) {
	t.Helper()
	cfg := &config.Config{
		OrchestratorPort: 4001,
		OrchestratorHost: "127.0.0.1",
		DataDir:          dir,
		AnalyticsEnabled: analyticsEnabled,
	}
	adapter := config.NewJSONAdapter(dir)
	if err := adapter.Save(cfg); err != nil {
		t.Fatalf("write test config: %v", err)
	}
}

// testCmd creates a throwaway cobra.Command with the given RunE and captures output.
func testCmd(runE func(cmd *cobra.Command, args []string) error) (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cmd := &cobra.Command{RunE: runE}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd, buf
}

func TestAnalyticsCmd_Registered(t *testing.T) {
	t.Parallel()
	var found bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "analytics" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("analytics command not registered on rootCmd")
	}
}

func TestAnalyticsCmd_HasSubcommands(t *testing.T) {
	t.Parallel()
	want := map[string]bool{"enable": false, "disable": false, "status": false}
	for _, sub := range analyticsCmd.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not registered on analyticsCmd", name)
		}
	}
}

func TestAnalyticsEnable(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, nil)
	t.Setenv("KF_DATA_DIR", dir)

	cmd, buf := testCmd(runAnalyticsEnable)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("analytics enable: %v", err)
	}

	cfg, err := config.LoadFrom(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AnalyticsEnabled == nil || !*cfg.AnalyticsEnabled {
		t.Error("expected AnalyticsEnabled=true after enable")
	}

	output := buf.String()
	if !containsStr(output, "enabled") {
		t.Errorf("output should confirm enabled, got: %q", output)
	}
}

func TestAnalyticsDisable(t *testing.T) {
	dir := t.TempDir()
	boolTrue := true
	writeTestConfig(t, dir, &boolTrue)
	t.Setenv("KF_DATA_DIR", dir)

	cmd, buf := testCmd(runAnalyticsDisable)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("analytics disable: %v", err)
	}

	cfg, err := config.LoadFrom(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AnalyticsEnabled == nil || *cfg.AnalyticsEnabled {
		t.Error("expected AnalyticsEnabled=false after disable")
	}

	output := buf.String()
	if !containsStr(output, "disabled") {
		t.Errorf("output should confirm disabled, got: %q", output)
	}
}

func TestAnalyticsStatus_Default(t *testing.T) {
	dir := t.TempDir()
	writeTestConfig(t, dir, nil)
	t.Setenv("KF_DATA_DIR", dir)
	os.Unsetenv("KF_ANALYTICS_ENABLED")

	cmd, buf := testCmd(runAnalyticsStatus)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("analytics status: %v", err)
	}

	output := buf.String()
	if !containsStr(output, "enabled") {
		t.Errorf("expected 'enabled' in output, got: %q", output)
	}
	if !containsStr(output, "default") {
		t.Errorf("expected 'default' source in output, got: %q", output)
	}
}

func TestAnalyticsStatus_ConfigSet(t *testing.T) {
	dir := t.TempDir()
	boolFalse := false
	writeTestConfig(t, dir, &boolFalse)
	t.Setenv("KF_DATA_DIR", dir)
	os.Unsetenv("KF_ANALYTICS_ENABLED")

	cmd, buf := testCmd(runAnalyticsStatus)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("analytics status: %v", err)
	}

	output := buf.String()
	if !containsStr(output, "disabled") {
		t.Errorf("expected 'disabled' in output, got: %q", output)
	}
	if !containsStr(output, "config") {
		t.Errorf("expected 'config' source in output, got: %q", output)
	}
}

func TestAnalyticsStatus_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	boolTrue := true
	writeTestConfig(t, dir, &boolTrue)
	t.Setenv("KF_DATA_DIR", dir)
	t.Setenv("KF_ANALYTICS_ENABLED", "false")

	cmd, buf := testCmd(runAnalyticsStatus)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("analytics status: %v", err)
	}

	output := buf.String()
	if !containsStr(output, "disabled") {
		t.Errorf("expected 'disabled' (env override), got: %q", output)
	}
	if !containsStr(output, "env") {
		t.Errorf("expected 'env' source in output, got: %q", output)
	}
}

func TestAnalyticsEnable_NotInitialized(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	t.Setenv("KF_DATA_DIR", dir)

	cmd, _ := testCmd(runAnalyticsEnable)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for uninitialized config")
	}
}

// containsStr checks substring presence.
func containsStr(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
