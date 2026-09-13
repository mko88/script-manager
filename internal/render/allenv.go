package render

import (
	"fmt"
	"sort"
	"strings"
)

const (
	AllEnvListPlaceholder  = "#ALL_ENV_LIST#"
	AllEnvTablePlaceholder = "#ALL_ENV_TABLE#"
)

var autoMaskSuffixes = []string{
	"PASSWORD",
	"PASSWD",
	"PWD",
	"SECRET",
	"KEY",
	"TOKEN",
	"CREDENTIAL",
	"CREDENTIALS",
	"AUTH",
}

func ShouldAutoMask(envKey string) bool {
	upper := strings.ToUpper(envKey)
	for _, suf := range autoMaskSuffixes {
		if strings.HasSuffix(upper, suf) {
			return true
		}
	}
	return false
}

func ExpandAllEnv(md string, item map[string]any) string {
	hasList := strings.Contains(md, AllEnvListPlaceholder)
	hasTable := strings.Contains(md, AllEnvTablePlaceholder)
	if !hasList && !hasTable {
		return md
	}

	entries := allEnvEntries(item)
	if hasList {
		md = strings.ReplaceAll(md, AllEnvListPlaceholder, renderAllEnvList(entries))
	}
	if hasTable {
		md = strings.ReplaceAll(md, AllEnvTablePlaceholder, renderAllEnvTable(entries))
	}
	return md
}

type envEntry struct {
	key  string
	code string
}

func allEnvEntries(item map[string]any) []envEntry {
	seen := make(map[string]bool, len(item))
	entries := make([]envEntry, 0, len(item))
	for k, v := range item {
		envKey := strings.ToUpper(k)
		if seen[envKey] {
			continue
		}
		seen[envKey] = true

		value := fmt.Sprint(v)
		if ShouldAutoMask(envKey) {
			value = MaskFunc(value)
		}
		entries = append(entries, envEntry{key: envKey, code: value})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].key < entries[j].key })
	return entries
}

const noEnvVars = "_No environment variables._"

func renderAllEnvList(entries []envEntry) string {
	if len(entries) == 0 {
		return noEnvVars
	}
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "- **%s:** `%s`\n", e.key, e.code)
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderAllEnvTable(entries []envEntry) string {
	if len(entries) == 0 {
		return noEnvVars
	}
	var b strings.Builder
	b.WriteString("| Variable | Value |\n|---|---|\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "| %s | `%s` |\n", e.key, e.code)
	}
	return strings.TrimRight(b.String(), "\n")
}
