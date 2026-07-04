// Package action holds the logic shared by the TUI and GUI frontends for
// preparing an action to run: merging global env defaults into an item,
// expanding cmd/description templates, and building the subprocess
// environment.
package action

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"strings"
	"text/template"
)

// Merge returns a copy of item with the global env values as defaults.
// Item-level keys always win over globals.
func Merge(env, item map[string]any) map[string]any {
	merged := make(map[string]any, len(env)+len(item))
	maps.Copy(merged, env)
	maps.Copy(merged, item)
	return merged
}

// Expand renders a text/template source against data.
func Expand(src string, data map[string]any) (string, error) {
	tmpl, err := template.New("t").Parse(src)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Preview renders src against data like Expand, but falls back to the raw
// source on any error. Meant for read-only previews (the command pane), where
// showing the unexpanded template beats showing nothing; run paths must use
// Expand so a broken template is an error, not a command.
func Preview(src string, data map[string]any) string {
	out, err := Expand(src, data)
	if err != nil {
		return src
	}
	return out
}

// Env returns the current process environment plus every item field exported
// as an uppercase variable. Keys that cannot form a valid variable name
// (empty, or containing '=' or NUL) are skipped.
func Env(item map[string]any) []string {
	env := os.Environ()
	for k, v := range item {
		if k == "" || strings.ContainsAny(k, "=\x00") {
			continue
		}
		env = append(env, strings.ToUpper(k)+"="+fmt.Sprint(v))
	}
	return env
}
