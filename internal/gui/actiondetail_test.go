package gui

import (
	"os"
	"path/filepath"
	"testing"

	"script-manager/internal/config"
)

func TestGetActionDetailReadsScriptContent(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "deploy.sh")
	content := "#!/bin/bash\necho hi\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	a := NewApp(func() (*config.Config, error) {
		return &config.Config{
			Items:   []map[string]any{{"name": "item1"}},
			Actions: []config.Action{{Title: "Deploy", Script: scriptPath}},
		}, nil
	})

	got := a.GetActionDetail(0, 0)
	if got.ScriptError != "" {
		t.Fatalf("unexpected ScriptError: %s", got.ScriptError)
	}
	if got.ScriptContent != content {
		t.Errorf("ScriptContent = %q, want %q", got.ScriptContent, content)
	}
	if got.Script != scriptPath {
		t.Errorf("Script = %q, want %q", got.Script, scriptPath)
	}
}

func TestGetActionDetailScriptReadError(t *testing.T) {
	a := NewApp(func() (*config.Config, error) {
		return &config.Config{
			Items:   []map[string]any{{"name": "item1"}},
			Actions: []config.Action{{Title: "Deploy", Script: "/does/not/exist.sh"}},
		}, nil
	})

	got := a.GetActionDetail(0, 0)
	if got.ScriptError == "" {
		t.Error("expected a ScriptError for a missing file")
	}
	if got.ScriptContent != "" {
		t.Errorf("ScriptContent = %q, want empty when the read failed", got.ScriptContent)
	}
}

func TestGetActionDetailCmdModeLeavesScriptContentEmpty(t *testing.T) {
	a := NewApp(func() (*config.Config, error) {
		return &config.Config{
			Items:   []map[string]any{{"name": "item1"}},
			Actions: []config.Action{{Title: "Deploy", Cmd: "echo hi"}},
		}, nil
	})

	got := a.GetActionDetail(0, 0)
	if got.ScriptError != "" || got.ScriptContent != "" {
		t.Errorf("cmd-mode action should leave ScriptContent/ScriptError empty, got %+v", got)
	}
}
