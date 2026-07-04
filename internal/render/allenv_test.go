package render

import (
	"strings"
	"testing"

	"script-manager/internal/config"
)

func TestShouldAutoMask(t *testing.T) {
	tests := map[string]bool{
		"password":    true,
		"PASSWORD":    true,
		"dbPassword":  true,
		"passwd":      true,
		"pwd":         true,
		"apiSecret":   true,
		"secret":      true,
		"apiKey":      true,
		"sshKey":      true,
		"authToken":   true,
		"token":       true,
		"credential":  true,
		"credentials": true,
		"basicAuth":   true,
		"clusterIp":   false,
		"region":      false,
		"name":        false,
		"description": false,
		"keyboard":    false, // contains "key" but doesn't end with it
	}
	for key, want := range tests {
		if got := ShouldAutoMask(key); got != want {
			t.Errorf("ShouldAutoMask(%q) = %v, want %v", key, got, want)
		}
	}
}

func TestExpandAllEnvNoPlaceholder(t *testing.T) {
	src := "plain markdown, nothing to expand"
	if got := ExpandAllEnv(src, map[string]any{"a": 1}); got != src {
		t.Errorf("ExpandAllEnv with no placeholder = %q, want unchanged %q", got, src)
	}
}

func TestExpandAllEnvList(t *testing.T) {
	item := map[string]any{
		"region":   "eu-west-1",
		"password": "s3cret",
	}
	out := ExpandAllEnv(AllEnvListPlaceholder, item)

	if !strings.Contains(out, "**REGION:** `eu-west-1`") {
		t.Errorf("list missing plain entry, got %q", out)
	}
	if strings.Contains(out, "s3cret") {
		t.Errorf("secret leaked into list output: %q", out)
	}
	if !strings.Contains(out, "**PASSWORD:** `"+maskPrefix) {
		t.Errorf("password entry not auto-masked, got %q", out)
	}
}

func TestExpandAllEnvTable(t *testing.T) {
	item := map[string]any{"region": "eu-west-1"}
	out := ExpandAllEnv(AllEnvTablePlaceholder, item)

	if !strings.Contains(out, "| Variable | Value |") {
		t.Errorf("table missing header, got %q", out)
	}
	if !strings.Contains(out, "| REGION | `eu-west-1` |") {
		t.Errorf("table missing row, got %q", out)
	}
}

func TestExpandAllEnvBothPlaceholders(t *testing.T) {
	src := AllEnvListPlaceholder + "\n\n" + AllEnvTablePlaceholder
	out := ExpandAllEnv(src, map[string]any{"region": "eu"})
	if strings.Contains(out, AllEnvListPlaceholder) || strings.Contains(out, AllEnvTablePlaceholder) {
		t.Errorf("placeholder left unexpanded: %q", out)
	}
	if !strings.Contains(out, "- **REGION:**") || !strings.Contains(out, "| REGION |") {
		t.Errorf("expected both list and table forms, got %q", out)
	}
}

func TestExpandAllEnvExcludesReservedKeys(t *testing.T) {
	item := map[string]any{
		"region":                "eu",
		config.KeyDisplay:       "compact",
		config.KeyActions:       []string{"ssh"},
		config.KeyActionGroups:  []string{"safe"},
		config.KeyCustomActions: []map[string]any{{"title": "x"}},
	}
	out := ExpandAllEnv(AllEnvTablePlaceholder, item)

	if !strings.Contains(out, "REGION") {
		t.Errorf("expected regular key to appear, got %q", out)
	}
	for _, reserved := range []string{"DISPLAY", "ACTIONS", "ACTIONGROUPS", "CUSTOMACTIONS"} {
		if strings.Contains(out, reserved) {
			t.Errorf("reserved key %q leaked into all-env output: %q", reserved, out)
		}
	}
}

func TestExpandAllEnvCaseCollision(t *testing.T) {
	// Two keys that only differ by case collide once uppercased for export;
	// exactly one row should be present, not two.
	item := map[string]any{"Region": "eu", "region": "us"}
	out := ExpandAllEnv(AllEnvListPlaceholder, item)
	if n := strings.Count(out, "REGION:"); n != 1 {
		t.Errorf("expected exactly one REGION entry, got %d in %q", n, out)
	}
}

func TestExpandAllEnvEmptyItem(t *testing.T) {
	out := ExpandAllEnv(AllEnvListPlaceholder, map[string]any{})
	if out != noEnvVars {
		t.Errorf("empty item list = %q, want %q", out, noEnvVars)
	}
}

func TestExpandAllEnvSorted(t *testing.T) {
	item := map[string]any{"zebra": 1, "alpha": 2, "mango": 3}
	out := ExpandAllEnv(AllEnvListPlaceholder, item)

	alphaIdx := strings.Index(out, "ALPHA")
	mangoIdx := strings.Index(out, "MANGO")
	zebraIdx := strings.Index(out, "ZEBRA")
	if !(alphaIdx < mangoIdx && mangoIdx < zebraIdx) {
		t.Errorf("entries not sorted alphabetically: %q", out)
	}
}
