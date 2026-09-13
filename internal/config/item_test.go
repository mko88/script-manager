package config

import (
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestItemUnmarshalNewShape(t *testing.T) {
	src := `
name: srv1
display: prod
actions: [ssh, logs]
actionGroups: [remote]
customActions:
  - title: Tail
    cmd: tail -f /var/log/app.log
    requiresPin: true
env:
  sshUser: root
  dbHost: db1.internal
`
	var item Item
	if err := yaml.Unmarshal([]byte(src), &item); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if item.Name != "srv1" || item.Display != "prod" {
		t.Errorf("Name/Display = %q/%q", item.Name, item.Display)
	}
	if !reflect.DeepEqual(item.Actions, []string{"ssh", "logs"}) {
		t.Errorf("Actions = %v", item.Actions)
	}
	if !reflect.DeepEqual(item.ActionGroups, []string{"remote"}) {
		t.Errorf("ActionGroups = %v", item.ActionGroups)
	}
	if len(item.CustomActions) != 1 || !item.CustomActions[0].RequiresPIN {
		t.Errorf("CustomActions = %+v", item.CustomActions)
	}
	if item.Env["sshUser"] != "root" || item.Env["dbHost"] != "db1.internal" {
		t.Errorf("Env = %v", item.Env)
	}
}

func TestItemUnmarshalRejectsUnknownKey(t *testing.T) {
	var item Item
	err := yaml.Unmarshal([]byte("name: srv1\nsshUser: root\n"), &item)
	if err == nil {
		t.Fatal("Unmarshal() error = nil, want an error naming the unknown key")
	}
	for _, want := range []string{"srv1", "sshUser", "env"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestItemEnvMayShadowStructuralNames(t *testing.T) {
	src := "name: srv1\nenv:\n  display: screen-0\n  actions: two\n"
	var item Item
	if err := yaml.Unmarshal([]byte(src), &item); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if item.Display != "" || item.Actions != nil {
		t.Errorf("env keys leaked into the struct: Display=%q Actions=%v", item.Display, item.Actions)
	}
	values := item.Values()
	if values["display"] != "screen-0" || values["actions"] != "two" {
		t.Errorf("Values() = %v, want the env values intact", values)
	}
}

func TestItemValues(t *testing.T) {
	item := Item{
		Name:    "srv1",
		Display: "prod",
		Actions: []string{"ssh"},
		Env:     map[string]any{"sshUser": "root"},
	}
	got := item.Values()
	want := map[string]any{"name": "srv1", "sshUser": "root"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Values() = %v, want %v", got, want)
	}
}

func TestItemValuesNameWinsOverEnv(t *testing.T) {
	item := Item{Name: "srv1", Env: map[string]any{"name": "shadowed"}}
	if got := item.Values()["name"]; got != "srv1" {
		t.Errorf("Values()[name] = %v, want srv1", got)
	}
}

func TestItemValuesEmptyNameNotSet(t *testing.T) {
	item := Item{Env: map[string]any{"sshUser": "root"}}
	if _, ok := item.Values()["name"]; ok {
		t.Error("Values() set name for an item with no name")
	}
}

func TestItemKey(t *testing.T) {
	for _, k := range []string{"name", "display", "actions", "actionGroups", "customActions", "env"} {
		if !ItemKey(k) {
			t.Errorf("ItemKey(%q) = false, want true", k)
		}
	}
	if ItemKey("sshUser") {
		t.Error("ItemKey(sshUser) = true, want false")
	}
}
