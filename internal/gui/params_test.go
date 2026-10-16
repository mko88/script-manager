package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"script-manager/internal/config"
	"script-manager/internal/presets"
)

const documentedScript = `#!/bin/bash
# Usage: deploy.sh [options]
#   -f, --force          Skip the confirmation
#   -e, --env ENV        Environment {dev|prod} (required)
`

func paramsTestApp(t *testing.T) (*App, string) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "deploy.sh")
	if err := os.WriteFile(script, []byte(documentedScript), 0o755); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(func() (*config.Config, error) {
		return &config.Config{
			Shell:      []string{"bash", "-c"},
			SourcePath: filepath.Join(dir, "config.yaml"),
			Env:        map[string]any{"region": "eu"},
			Items: []config.Item{{
				Name:          "web",
				Env:           map[string]any{"host": "h1"},
				Actions:       []string{"deploy"},
				CustomActions: []config.Action{{ID: "deploy", Title: "Local echo", Cmd: "echo {{.host}}"}},
			}},
			Actions: []config.Action{{ID: "deploy", Title: "Deploy", Script: script}},
		}, nil
	})
	return a, script
}

func TestGetActionDetailParams(t *testing.T) {
	a, _ := paramsTestApp(t)
	d := a.GetActionDetail(0, 0)
	if !d.Params.Documented || len(d.Params.Params) != 2 {
		t.Fatalf("params = %+v", d.Params)
	}
	if d := a.GetActionDetail(0, 1); d.Params.Documented {
		t.Errorf("cmd action documented: %+v", d.Params)
	}
}

func TestPreviewCommand(t *testing.T) {
	a, script := paramsTestApp(t)
	got := a.PreviewCommand(0, 0, RunInputDTO{
		Values: map[string]any{"force": true, "env": "prod"},
		Args:   "--region {{.region}}",
	})
	want := "'" + script + "' --force --env 'prod' --region eu"
	if got.Error != "" || got.Line != want {
		t.Errorf("preview = %+v, want %q", got, want)
	}

	missing := a.PreviewCommand(0, 0, RunInputDTO{})
	if !strings.Contains(missing.Error, "env") {
		t.Errorf("missing required: %+v", missing)
	}

	bad := a.PreviewCommand(0, 0, RunInputDTO{Values: map[string]any{"env": "qa"}})
	if bad.Error == "" {
		t.Errorf("invalid choice accepted: %+v", bad)
	}

	cmd := a.PreviewCommand(0, 1, RunInputDTO{Args: "-n"})
	if cmd.Error != "" || cmd.Line != "echo h1 -n" {
		t.Errorf("cmd preview = %+v", cmd)
	}
}

func TestRunActionInlineRejectsInvalidValues(t *testing.T) {
	a, _ := paramsTestApp(t)
	if err := a.RunActionInline(0, 0, RunInputDTO{}); err == nil || !strings.Contains(err.Error(), "env") {
		t.Errorf("err = %v, want a missing-required error", err)
	}
}

func TestPresetsSeparateGlobalAndCustom(t *testing.T) {
	a, _ := paramsTestApp(t)
	if _, err := a.SavePreset(0, 0, presets.Preset{Name: "prod", Values: map[string]any{"env": "prod"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.SavePreset(0, 1, presets.Preset{Name: "loud", Args: "-n"}); err != nil {
		t.Fatal(err)
	}
	global := a.GetPresets(0, 0)
	custom := a.GetPresets(0, 1)
	if len(global.Presets) != 1 || global.Presets[0].Name != "prod" {
		t.Errorf("global = %+v", global)
	}
	if len(custom.Presets) != 1 || custom.Presets[0].Name != "loud" {
		t.Errorf("custom = %+v", custom)
	}

	dto, err := a.DeletePreset(0, 0, "prod")
	if err != nil || len(dto.Presets) != 0 {
		t.Errorf("delete = %+v, %v", dto, err)
	}
	if _, err := os.Stat(a.cfg.SourcePath); !os.IsNotExist(err) {
		t.Errorf("the config file was written: %v", err)
	}
}

func TestSavePresetRefusesBrokenFile(t *testing.T) {
	a, _ := paramsTestApp(t)
	path := presets.PathFor(a.cfg.SourcePath)
	if err := os.WriteFile(path, []byte("actions: [broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := a.GetPresets(0, 0); got.Warning == "" {
		t.Errorf("no warning for a broken file: %+v", got)
	}
	if _, err := a.SavePreset(0, 0, presets.Preset{Name: "x"}); err == nil {
		t.Error("saved over a file that doesn't parse")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "actions: [broken" {
		t.Errorf("file changed: %q", data)
	}
}

func TestValuesForUnknownParametersAreRefused(t *testing.T) {
	a, script := paramsTestApp(t)
	got := a.PreviewCommand(0, 0, RunInputDTO{Values: map[string]any{"env": "prod", "gone": "x", "off": false}})
	if !strings.Contains(got.Error, "gone") || strings.Contains(got.Error, "off") {
		t.Errorf("unknown key: %+v", got)
	}
	if err := a.RunActionInline(0, 1, RunInputDTO{Values: map[string]any{"env": "prod"}}); err == nil || !strings.Contains(err.Error(), "env") {
		t.Errorf("cmd action with values: err = %v", err)
	}

	if err := os.WriteFile(script, []byte("#!/bin/bash\necho undocumented now\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(script, future, future); err != nil {
		t.Fatal(err)
	}
	got = a.PreviewCommand(0, 0, RunInputDTO{Values: map[string]any{"env": "prod"}})
	if !strings.Contains(got.Error, "no documented parameters") {
		t.Errorf("undocumented script with values: %+v", got)
	}
	if got := a.PreviewCommand(0, 0, RunInputDTO{Args: "--x"}); got.Error != "" {
		t.Errorf("free text refused: %+v", got)
	}
}

func TestScriptSpecCacheFollowsTheFile(t *testing.T) {
	a, script := paramsTestApp(t)
	first := a.scriptSpec(script)
	if len(first.Params) != 2 {
		t.Fatalf("params = %+v", first.Params)
	}
	if again := a.scriptSpec(script); &again.Params[0] != &first.Params[0] {
		t.Error("an unchanged file was parsed again")
	}
	if err := os.WriteFile(script, []byte(documentedScript+"#   -v, --verbose        Talk more\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := a.scriptSpec(script); len(got.Params) != 3 {
		t.Errorf("edited file not re-read: %+v", got.Params)
	}
}

func TestPreviewRefusesValuesForUnknownShell(t *testing.T) {
	a, _ := paramsTestApp(t)
	a.cfg.Shell = []string{"fish", "-c"}
	got := a.PreviewCommand(0, 0, RunInputDTO{Values: map[string]any{"env": "prod"}})
	if !strings.Contains(got.Error, "Arguments box") {
		t.Errorf("fish: %+v", got)
	}
}

func TestPresetKeysSeparateIdsFromTitles(t *testing.T) {
	a, _ := paramsTestApp(t)
	a.cfg.Actions = append(a.cfg.Actions, config.Action{Title: "deploy", Cmd: "echo"})
	a.cfg.Items[0].Actions = nil
	a.cfg.Items[0].CustomActions = nil
	if _, err := a.SavePreset(0, 1, presets.Preset{Name: "by-title"}); err != nil {
		t.Fatal(err)
	}
	if got := a.GetPresets(0, 0); len(got.Presets) != 0 {
		t.Errorf("action with id deploy sees the title-keyed preset: %+v", got)
	}
	data, _ := os.ReadFile(presets.PathFor(a.cfg.SourcePath))
	if !strings.Contains(string(data), "title:deploy") {
		t.Errorf("file:\n%s", data)
	}
}

func TestScriptSpecCacheSeesSameSizeEdits(t *testing.T) {
	a, script := paramsTestApp(t)
	info, err := os.Stat(script)
	if err != nil {
		t.Fatal(err)
	}
	if got := a.scriptSpec(script); got.Params[0].Name != "force" {
		t.Fatalf("params = %+v", got.Params)
	}
	edited := strings.Replace(documentedScript, "--force", "--forcE", 1)
	if len(edited) != len(documentedScript) {
		t.Fatal("edit changed the size")
	}
	if err := os.WriteFile(script, []byte(edited), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(script, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if got := a.scriptSpec(script); got.Params[0].Name != "forcE" {
		t.Errorf("stale spec served: %+v", got.Params)
	}
}

func TestStalePresetValuesError(t *testing.T) {
	a, _ := paramsTestApp(t)
	got := a.PreviewCommand(0, 0, RunInputDTO{Values: map[string]any{"env": "prod", "gone": "x", "old": true}})
	if got.Error != "This preset sets gone, old, which the script no longer documents. Update the preset to drop them." {
		t.Errorf("error = %q", got.Error)
	}
}
