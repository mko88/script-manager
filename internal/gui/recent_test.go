package gui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"script-manager/internal/config"
	"script-manager/internal/recent"
)

const brokenSecretsID = "0123456789abcdef0123456789abcdef"

// configWithBrokenSecrets writes a config whose local secrets file is not
// valid YAML, so it loads with an error but is still usable.
func configWithBrokenSecrets(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AppData", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	local := config.LocalSecretsPath(brokenSecretsID)
	if err := os.MkdirAll(filepath.Dir(local), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(local, []byte("env: [unclosed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	body := "secretsId: " + brokenSecretsID + "\nitems:\n  - name: one\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadRecentConfigOpensAConfigWithABrokenSecretsFile(t *testing.T) {
	path := configWithBrokenSecrets(t)
	a := NewApp("")
	recent.Add(a.appDataDir, path)

	warning, err := a.LoadRecentConfig(path)
	if err != nil {
		t.Fatalf("LoadRecentConfig() error = %v, want the config opened with a warning", err)
	}
	if warning == "" {
		t.Error("LoadRecentConfig() returned no warning for a broken secrets file")
	}
	if len(a.cfg.Items) != 1 {
		t.Errorf("items = %d, want the recent config loaded", len(a.cfg.Items))
	}
	if !slices.Contains(recent.Load(a.appDataDir), a.cfg.SourcePath) {
		t.Error("the config was dropped from the recent list")
	}
}

func TestOpenConfigFileWarnsAboutABrokenSecretsFile(t *testing.T) {
	path := configWithBrokenSecrets(t)
	a := NewApp("")

	warning, err := a.openConfigFile(path)
	if err != nil {
		t.Fatalf("openConfigFile() error = %v, want the config opened with a warning", err)
	}
	if warning == "" {
		t.Error("openConfigFile() returned no warning for a broken secrets file")
	}
	if len(a.cfg.Items) != 1 {
		t.Errorf("items = %d, want the config loaded", len(a.cfg.Items))
	}
}
