package config_test

import (
	"testing"

	"script-manager/internal/config"
	"script-manager/internal/configmigrate"
)

// Guards the shipped example configs: they must load under the strict loader
// and must not ask the user to convert anything.
func TestExampleConfigsAreCurrent(t *testing.T) {
	for _, path := range []string{
		"../../examples/pin-secrets.yaml",
		"../../examples/pin-secrets-win.yaml",
	} {
		needed, err := configmigrate.Needed(path)
		if err != nil {
			t.Fatalf("%s: Needed() error = %v", path, err)
		}
		if needed {
			t.Errorf("%s: ships in the old item format", path)
		}
		cfg, err := config.LoadFromWithError(path)
		if err != nil {
			t.Fatalf("%s: load error = %v", path, err)
		}
		if len(cfg.Items) == 0 {
			t.Fatalf("%s: no items loaded", path)
		}
		if len(cfg.Items[0].Env) == 0 {
			t.Errorf("%s: first item has no env values", path)
		}
	}
}
