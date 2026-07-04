package render

import (
	"fmt"
	"sort"
	"strings"

	"script-manager/internal/config"
)

// Literal placeholders a details template can include verbatim (not inside
// {{ }}) to have every value the item exports to the action's subprocess
// environment rendered as Markdown — a bullet list or a table, respectively.
// ExpandAllEnv replaces these after the Go template runs and before
// mask-span processing, so auto-masked entries flow through the same
// `GLMASK__...` pipeline as an explicit {{mask ...}} call.
const (
	AllEnvListPlaceholder  = "#ALL_ENV_LIST#"
	AllEnvTablePlaceholder = "#ALL_ENV_TABLE#"
)

// autoMaskSuffixes are case-insensitive endings of an exported env var name
// that get masked automatically in the #ALL_ENV_LIST#/#ALL_ENV_TABLE#
// output, without needing an explicit {{mask ...}} call.
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

// reservedKeys are excluded from the all-env listing: they configure action
// filtering rather than holding data worth displaying, and customActions in
// particular is a slice of maps that would render as unreadable Go syntax.
var reservedKeys = map[string]bool{
	config.KeyDisplay:       true,
	config.KeyActions:       true,
	config.KeyActionGroups:  true,
	config.KeyCustomActions: true,
}

// ShouldAutoMask reports whether a value exported under envKey (the
// uppercased name a script would see, e.g. via $CLUSTERIP) should be hidden
// by default based on its name alone.
func ShouldAutoMask(envKey string) bool {
	upper := strings.ToUpper(envKey)
	for _, suf := range autoMaskSuffixes {
		if strings.HasSuffix(upper, suf) {
			return true
		}
	}
	return false
}

// ExpandAllEnv replaces #ALL_ENV_LIST#/#ALL_ENV_TABLE# in md, if present,
// with a rendered Markdown list/table of every value in item — i.e. every
// variable the action's subprocess would see in its environment.
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

// envEntry is one row of the all-env listing: the exported (uppercased) env
// var name and the backtick-span content — either the raw value or a
// MaskFunc marker for auto-masked entries.
type envEntry struct {
	key  string
	code string
}

// allEnvEntries builds the sorted, auto-masked entry list for item. Keys
// that collide once uppercased (e.g. "Region" and "region") keep only the
// first one seen, matching how action.Env exports a single env var per name.
func allEnvEntries(item map[string]any) []envEntry {
	seen := make(map[string]bool, len(item))
	entries := make([]envEntry, 0, len(item))
	for k, v := range item {
		if reservedKeys[k] {
			continue
		}
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
