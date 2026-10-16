package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		Values: map[string]any{"force": true, "env": "prod", "unknown": "x"},
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

	cmd := a.PreviewCommand(0, 1, RunInputDTO{Values: map[string]any{"env": "prod"}, Args: "-n"})
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
