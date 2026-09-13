package action

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"strings"
	"text/template"

	"script-manager/internal/config"
)

func Merge(env map[string]any, item *config.Item) map[string]any {
	merged := make(map[string]any, len(env)+1)
	maps.Copy(merged, env)
	if item != nil {
		maps.Copy(merged, item.Values())
	}
	return merged
}

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

func Preview(src string, data map[string]any) string {
	out, err := Expand(src, data)
	if err != nil {
		return src
	}
	return out
}

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
