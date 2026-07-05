package configedit

import (
	"os"
	"path/filepath"
	"testing"

	"script-manager/internal/config"
)

func TestInitialState(t *testing.T) {
	t.Run("explicit path loads cleanly", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(path, []byte("shell: [bash]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		a := NewApp(path)
		state := a.InitialState()
		if state.Warning != "" {
			t.Errorf("warning = %q, want empty", state.Warning)
		}
		if state.Path != path {
			t.Errorf("path = %q, want %q", state.Path, path)
		}
		if len(state.Config.Shell) != 1 || state.Config.Shell[0] != "bash" {
			t.Errorf("shell = %v", state.Config.Shell)
		}
	})

	t.Run("explicit path that fails to load surfaces a warning", func(t *testing.T) {
		a := NewApp(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
		state := a.InitialState()
		if state.Warning == "" {
			t.Error("expected a warning for a missing explicit -config path")
		}
	})

	t.Run("auto-detect finding nothing starts blank with no warning", func(t *testing.T) {
		dir := t.TempDir()
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chdir(cwd) })

		a := NewApp("")
		state := a.InitialState()
		if state.Warning != "" {
			t.Errorf("warning = %q, want empty when auto-detect simply finds nothing", state.Warning)
		}
	})
}

func TestNewBlank(t *testing.T) {
	a := NewApp("")
	state := a.NewBlank()
	if state.Path != "" {
		t.Errorf("path = %q, want empty", state.Path)
	}
	if len(state.Config.Display) != 1 || state.Config.Display[0].Name != "default" {
		t.Errorf("expected one starter display, got %+v", state.Config.Display)
	}
	if a.path != "" {
		t.Errorf("internal path = %q, want empty", a.path)
	}
}

func TestSave(t *testing.T) {
	t.Run("writes valid, loadable YAML", func(t *testing.T) {
		a := NewApp("")
		a.NewBlank()
		dto := ConfigDTO{Shell: []string{"bash"}, Items: []ItemDTO{{Name: "srv1"}}}

		path := filepath.Join(t.TempDir(), "config.yaml")
		result, err := a.Save(dto, path)
		if err != nil {
			t.Fatal(err)
		}
		if result.Path != path {
			t.Errorf("result.Path = %q, want %q", result.Path, path)
		}

		reloaded, err := config.LoadFromWithError(path)
		if err != nil {
			t.Fatalf("saved file did not load back: %v", err)
		}
		if len(reloaded.Shell) != 1 || reloaded.Shell[0] != "bash" {
			t.Errorf("reloaded shell = %v", reloaded.Shell)
		}
		if a.path != path {
			t.Errorf("a.path = %q, want %q", a.path, path)
		}
	})

	t.Run("no path anywhere is an error", func(t *testing.T) {
		a := NewApp("")
		if _, err := a.Save(ConfigDTO{}, ""); err == nil {
			t.Error("expected an error when neither an explicit path nor a.path is set")
		}
	})

	t.Run("empty path reuses the previously saved path", func(t *testing.T) {
		a := NewApp("")
		path := filepath.Join(t.TempDir(), "config.yaml")
		if _, err := a.Save(ConfigDTO{Shell: []string{"bash"}}, path); err != nil {
			t.Fatal(err)
		}
		if _, err := a.Save(ConfigDTO{Shell: []string{"zsh"}}, ""); err != nil {
			t.Fatal(err)
		}
		reloaded, err := config.LoadFromWithError(path)
		if err != nil {
			t.Fatal(err)
		}
		if reloaded.Shell[0] != "zsh" {
			t.Errorf("expected the second save to overwrite the same path, got shell %v", reloaded.Shell)
		}
	})
}

func TestValidateFieldMethod(t *testing.T) {
	a := NewApp("")
	if got := a.ValidateField("number", "42"); got != "" {
		t.Errorf("ValidateField(number, 42) = %q, want empty", got)
	}
	if got := a.ValidateField("number", "not-a-number"); got == "" {
		t.Error("expected a non-empty error message")
	}
}

func TestKnownTerminals(t *testing.T) {
	a := NewApp("")
	names := a.KnownTerminals()
	if len(names) == 0 {
		t.Error("expected at least one known terminal name")
	}
}
