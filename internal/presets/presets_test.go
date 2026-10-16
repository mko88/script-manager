package presets

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestLoadMissingIsEmpty(t *testing.T) {
	f, err := Load(filepath.Join(t.TempDir(), FileName))
	if err != nil || len(f.Actions) != 0 {
		t.Fatalf("Load = %+v, %v", f, err)
	}
}

func TestLoadTolerant(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	data := `
unknownTopLevel: 1
actions:
  deploy:
    - name: prod
      values:
        Env: prod
        Replicas: 3
        DryRun: true
        Tags: [a, 2]
        Nested: {x: 1}
      args: --verbose
      extra: ignored
    - name: ""
    - values: {Env: dev}
customActions:
  web-1:
    deploy:
      - name: local
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	list := f.List(Key{Action: "deploy"})
	if len(list) != 1 {
		t.Fatalf("list = %+v", list)
	}
	want := map[string]any{"Env": "prod", "Replicas": "3", "DryRun": true, "Tags": []string{"a", "2"}}
	if !reflect.DeepEqual(list[0].Values, want) || list[0].Args != "--verbose" {
		t.Errorf("preset = %+v", list[0])
	}
	if got := f.List(Key{Item: "web-1", Action: "deploy", Custom: true}); len(got) != 1 || got[0].Name != "local" {
		t.Errorf("custom = %+v", got)
	}
}

func TestLoadBrokenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, []byte("actions: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err == nil {
		t.Error("expected a parse error")
	}
	if f == nil || len(f.Actions) != 0 {
		t.Errorf("want an empty file, got %+v", f)
	}
}

func TestPutDeleteSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	f := &File{}
	global := Key{Action: "deploy"}
	custom := Key{Item: "web-1", Action: "deploy", Custom: true}

	if err := f.Put(global, Preset{Name: " prod ", Values: map[string]any{"Env": "prod"}}); err != nil {
		t.Fatal(err)
	}
	if err := f.Put(global, Preset{Name: "dev", Args: "-x"}); err != nil {
		t.Fatal(err)
	}
	if err := f.Put(global, Preset{Name: "prod", Values: map[string]any{"Env": "prod2"}}); err != nil {
		t.Fatal(err)
	}
	if err := f.Put(custom, Preset{Name: "local"}); err != nil {
		t.Fatal(err)
	}
	if err := f.Put(global, Preset{Name: "  "}); err == nil {
		t.Error("accepted an empty name")
	}
	if err := Save(path, f); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	list := got.List(global)
	if len(list) != 2 || list[0].Name != "prod" || list[0].Values["Env"] != "prod2" || list[1].Args != "-x" {
		t.Errorf("global = %+v", list)
	}
	if len(got.List(custom)) != 1 || len(got.List(Key{Action: "web-1"})) != 0 {
		t.Errorf("custom presets leaked: %+v", got)
	}

	got.Delete(custom, "local")
	got.Delete(global, "dev")
	if err := Save(path, got); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "customActions") || strings.Contains(string(data), "dev") {
		t.Errorf("deleted presets left behind:\n%s", data)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("temporary files left behind: %v", entries)
	}
}

func TestPathFor(t *testing.T) {
	got := PathFor(filepath.Join("a", "b", "config.yaml"))
	if got != filepath.Join("a", "b", FileName) {
		t.Errorf("PathFor = %q", got)
	}
}

func TestActionKeySeparatesIdsFromTitles(t *testing.T) {
	if ActionKey("deploy", "Deploy") != "id:deploy" || ActionKey("", "deploy") != "title:deploy" {
		t.Errorf("keys = %q, %q", ActionKey("deploy", "Deploy"), ActionKey("", "deploy"))
	}
}

func TestSaveKeepsFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not kept on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := Save(path, &File{}); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o644 {
		t.Errorf("new file mode = %v, want 0644", info.Mode().Perm())
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, &File{}); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Errorf("mode after save = %v, want 0600", info.Mode().Perm())
	}
}

func TestLoadUnquotedPrefixedKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	data := "actions:\n  id:build:\n    - name: release\ncustomActions:\n  web-1:\n    title:Restart now:\n      - name: quick\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.List(Key{Action: ActionKey("build", "")})) != 1 {
		t.Errorf("id key not read: %+v", f.Actions)
	}
	if len(f.List(Key{Item: "web-1", Action: ActionKey("", "Restart now"), Custom: true})) != 1 {
		t.Errorf("title key not read: %+v", f.CustomActions)
	}
}
