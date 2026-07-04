package action

import (
	"slices"
	"strings"
	"testing"
)

func TestMerge(t *testing.T) {
	env := map[string]any{"region": "eu", "user": "admin"}
	item := map[string]any{"user": "bob", "host": "srv1"}

	merged := Merge(env, item)

	want := map[string]any{"region": "eu", "user": "bob", "host": "srv1"}
	for k, v := range want {
		if merged[k] != v {
			t.Errorf("merged[%q] = %v, want %v", k, merged[k], v)
		}
	}
	if len(merged) != len(want) {
		t.Errorf("merged has %d keys, want %d", len(merged), len(want))
	}

	// The inputs must not be mutated.
	if env["user"] != "admin" || item["region"] != nil {
		t.Error("Merge mutated an input map")
	}
}

func TestMergeNilInputs(t *testing.T) {
	if got := Merge(nil, nil); len(got) != 0 {
		t.Errorf("Merge(nil, nil) = %v, want empty", got)
	}
	if got := Merge(map[string]any{"a": 1}, nil); got["a"] != 1 {
		t.Errorf("Merge(env, nil) lost env key: %v", got)
	}
}

func TestExpand(t *testing.T) {
	got, err := Expand("ssh {{.user}}@{{.host}}", map[string]any{"user": "bob", "host": "srv1"})
	if err != nil {
		t.Fatalf("Expand returned error: %v", err)
	}
	if want := "ssh bob@srv1"; got != want {
		t.Errorf("Expand = %q, want %q", got, want)
	}
}

func TestExpandParseError(t *testing.T) {
	if _, err := Expand("{{.user", nil); err == nil {
		t.Error("Expand with unclosed action should return an error")
	}
}

func TestExpandExecError(t *testing.T) {
	if _, err := Expand("{{call .missing}}", map[string]any{}); err == nil {
		t.Error("Expand with failing execution should return an error")
	}
}

func TestPreviewFallsBackToSource(t *testing.T) {
	src := "{{.user"
	if got := Preview(src, nil); got != src {
		t.Errorf("Preview on parse error = %q, want raw source %q", got, src)
	}
	if got := Preview("hi {{.name}}", map[string]any{"name": "x"}); got != "hi x" {
		t.Errorf("Preview = %q, want %q", got, "hi x")
	}
}

func TestEnv(t *testing.T) {
	env := Env(map[string]any{
		"host":  "srv1",
		"Port":  22,
		"":      "skipped",
		"a=b":   "skipped",
		"x\x00": "skipped",
	})

	if !slices.Contains(env, "HOST=srv1") {
		t.Error("Env missing HOST=srv1")
	}
	if !slices.Contains(env, "PORT=22") {
		t.Error("Env missing PORT=22 (non-string value)")
	}
	for _, e := range env {
		if strings.Contains(e, "skipped") {
			t.Errorf("Env contains entry from invalid key: %q", e)
		}
	}
}
