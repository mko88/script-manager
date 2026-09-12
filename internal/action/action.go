package action

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"strings"
	"text/template"
)

func Merge(env, item map[string]any) map[string]any {
	merged := make(map[string]any, len(env)+len(item))
	maps.Copy(merged, env)
	maps.Copy(merged, item)
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
