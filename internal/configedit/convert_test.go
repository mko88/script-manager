package configedit

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"script-manager/internal/config"
)

// TestEmptyConfigDTOHasNoNullSlices guards against a real incident: Go's JSON
// encoder marshals a nil slice as `null`, and the frontend treats every DTO
// slice field as always-iterable ({#each}, .map, .some, .includes). A null
// reaching a Svelte reactive statement throws, which silently breaks all
// further reactivity for the rest of the session — not a visible crash, just
// every click quietly doing nothing from that point on. Confirmed
// end-to-end (via the WebKit inspector) that ValidateConfig returning a nil
// []ValidationIssueDTO for a clean config was exactly this bug. This test
// marshals the DTOs an empty/near-empty config produces and asserts the raw
// JSON never contains "null" for a slice field.
func TestEmptyConfigDTOHasNoNullSlices(t *testing.T) {
	dto := ToConfigDTO(&config.Config{})
	assertNoNullSlices(t, "ToConfigDTO(empty)", dto)

	itemDTO := ToItemDTO(map[string]any{})
	assertNoNullSlices(t, "ToItemDTO(empty)", itemDTO)

	issues := ValidateConfig(ConfigDTO{})
	assertNoNullSlices(t, "ValidateConfig(empty)", issues)

	preview := PreviewItem(ItemDTO{}, nil, nil, "", "")
	assertNoNullSlices(t, "PreviewItem(empty)", preview)

	termAuto := terminalToDTO(config.TerminalConfig{})
	assertNoNullSlices(t, "terminalToDTO(auto)", termAuto)
}

func assertNoNullSlices(t *testing.T, label string, v any) {
	t.Helper()
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("%s: marshal: %v", label, err)
	}
	// A conservative check: any bare `null` value in the JSON is suspect for
	// these DTOs, since every slice field should default to `[]`.
	if strings.Contains(string(out), ":null") || string(out) == "null" {
		t.Errorf("%s: JSON contains a null field, want every slice defaulted to []:\n%s", label, out)
	}
}

func TestClassifyAndDecodeValue(t *testing.T) {
	// Numbers intentionally don't round-trip to the exact same Go type
	// (int vs int64): decodeValue always produces int64 for whole numbers.
	// What matters is that re-classifying the decoded value reproduces the
	// same (kind, value) pair — i.e. the YAML text it marshals to is
	// unchanged — not Go type identity.
	tests := []struct {
		name string
		in   any
	}{
		{"string", "hello"},
		{"bool true", true},
		{"bool false", false},
		{"int", 42},
		{"negative int", -7},
		{"float", 3.5},
		{"nested list", []interface{}{"a", "b"}},
		{"nested map", map[string]interface{}{"x": "y"}},
		{"nil", nil},
		{"multiline string", "line1\nline2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// "field" deliberately doesn't match looksLikeSecretKey, so these
			// cases exercise classification purely by value shape.
			kind, value, _ := classifyValue("field", tt.in)
			decoded, err := decodeValue(kind, value)
			if err != nil {
				t.Fatalf("decodeValue(%q, %q): %v", kind, value, err)
			}
			gotKind, gotValue, _ := classifyValue("field", decoded)
			if gotKind != kind || gotValue != value {
				t.Errorf("round trip = (%q, %q), want (%q, %q)", gotKind, gotValue, kind, value)
			}
		})
	}
}

func TestClassifyValueIntStaysUndotted(t *testing.T) {
	kind, value, _ := classifyValue("field", 42)
	if kind != "number" || value != "42" {
		t.Errorf("classifyValue(42) = (%q, %q), want (number, 42) — not 42.0 or exponential form", kind, value)
	}
}

func TestClassifyValueSecretKey(t *testing.T) {
	// Secret is independent of kind — a secret-looking key still classifies
	// by its value's shape as usual; only the secret flag differs.
	tests := []struct {
		key  string
		want bool
	}{
		{"apiSecret", true},
		{"DB_PASSWORD", true},
		{"ApiKey", true},
		{"secretkey", true},
		{"username", false},
		{"keyboard", false}, // contains "key" but doesn't end with it
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			kind, _, secret := classifyValue(tt.key, "some-value")
			if kind != "string" {
				t.Errorf("classifyValue(%q, ...) kind = %q, want %q regardless of secret", tt.key, kind, "string")
			}
			if secret != tt.want {
				t.Errorf("classifyValue(%q, ...) secret = %v, want %v", tt.key, secret, tt.want)
			}
		})
	}
}

func TestDecodeValueErrors(t *testing.T) {
	tests := []struct {
		kind, value string
	}{
		{"bool", "not-a-bool"},
		{"number", "not-a-number"},
		{"yaml", "["}, // invalid YAML
		{"bogus-kind", "x"},
	}
	for _, tt := range tests {
		if _, err := decodeValue(tt.kind, tt.value); err == nil {
			t.Errorf("decodeValue(%q, %q): expected an error", tt.kind, tt.value)
		}
	}
}

func TestFieldsFromMapSortedAndExcludes(t *testing.T) {
	m := map[string]any{"z": "1", "a": "2", "name": "srv"}
	fields := FieldsFromMap(m, map[string]bool{"name": true})
	if len(fields) != 2 || fields[0].Key != "a" || fields[1].Key != "z" {
		t.Errorf("got %+v, want sorted [a z] with name excluded", fields)
	}
}

func TestFieldsToMapSkipsEmptyKey(t *testing.T) {
	out, err := FieldsToMap([]FieldDTO{{Key: "", Kind: "string", Value: "x"}, {Key: "ok", Kind: "string", Value: "y"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out["ok"] != "y" {
		t.Errorf("got %+v", out)
	}
}

func TestFieldsToMapPropagatesError(t *testing.T) {
	_, err := FieldsToMap([]FieldDTO{{Key: "bad", Kind: "number", Value: "nope"}})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestItemDTORoundTrip(t *testing.T) {
	item := map[string]any{
		config.KeyName:    "srv1",
		config.KeyDisplay: "prod",
		config.KeyActions: []interface{}{"deploy", "logs"},
		config.KeyCustomActions: []interface{}{
			map[string]interface{}{"id": "c1", "title": "Custom", "cmd": "echo hi", "groups": []interface{}{"g1"}, "noWait": true, "interactive": true},
		},
		"sshUser":     "root",
		"port":        22,
		"verbose":     true,
		"extraConfig": map[string]interface{}{"k": "v"},
	}

	dto := ToItemDTO(item)
	if dto.Name != "srv1" || dto.Display != "prod" {
		t.Fatalf("got %+v", dto)
	}
	if !reflect.DeepEqual(dto.Actions, []string{"deploy", "logs"}) {
		t.Errorf("actions = %v", dto.Actions)
	}
	if len(dto.CustomActions) != 1 || dto.CustomActions[0].Title != "Custom" || !dto.CustomActions[0].NoWait || !dto.CustomActions[0].Interactive {
		t.Errorf("customActions = %+v", dto.CustomActions)
	}
	if len(dto.Fields) != 4 {
		t.Fatalf("fields = %+v, want 4 (sshUser, port, verbose, extraConfig)", dto.Fields)
	}

	back, err := FromItemDTO(dto)
	if err != nil {
		t.Fatal(err)
	}
	roundTripped := ToItemDTO(back)
	if !reflect.DeepEqual(dto, roundTripped) {
		t.Errorf("round trip mismatch:\nfirst:  %+v\nsecond: %+v", dto, roundTripped)
	}
}

func TestFromItemDTOOmitsEmptyReservedKeys(t *testing.T) {
	item, err := FromItemDTO(ItemDTO{Name: "srv1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{config.KeyDisplay, config.KeyActions, config.KeyActionGroups, config.KeyCustomActions} {
		if _, ok := item[key]; ok {
			t.Errorf("expected %q to be omitted, got %+v", key, item)
		}
	}
}

func TestConfigDTORoundTrip(t *testing.T) {
	cfg := &config.Config{
		Shell:  []string{"pwsh", "-NoLogo"},
		Titles: config.TitlesConfig{Items: "Servers"},
		Env:    map[string]any{"region": "eu", "retries": 3},
		Display: config.DisplayList{
			{Name: "default", List: "{{.name}}", Details: "**{{.name}}**"},
		},
		ActionGroups: []config.ActionGroup{
			{ID: "remote", Title: "Remote access", Color: "#7fd4ff"},
		},
		Actions: []config.Action{
			{ID: "ssh", Title: "SSH", Cmd: "ssh {{.host}}", Groups: []string{"remote"}},
		},
		Items: []map[string]any{
			{config.KeyName: "srv1", "sshUser": "root"},
		},
	}

	dto := ToConfigDTO(cfg)
	back, err := FromConfigDTO(dto)
	if err != nil {
		t.Fatal(err)
	}
	roundTripped := ToConfigDTO(back)
	if !reflect.DeepEqual(dto, roundTripped) {
		t.Errorf("round trip mismatch:\nfirst:  %+v\nsecond: %+v", dto, roundTripped)
	}

	firstOut, err := cfg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	secondOut, err := back.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if string(firstOut) != string(secondOut) {
		t.Errorf("marshal output mismatch:\nfirst:\n%s\nsecond:\n%s", firstOut, secondOut)
	}
}

func TestTerminalDTORoundTrip(t *testing.T) {
	tests := []config.TerminalConfig{
		{},
		{Name: "wt"},
		{Argv: []string{"my-term", "--title", "{{title}}"}},
	}
	for _, tc := range tests {
		dto := terminalToDTO(tc)
		back := terminalFromDTO(dto)
		if !reflect.DeepEqual(tc, back) {
			t.Errorf("terminal round trip: %+v -> %+v -> %+v", tc, dto, back)
		}
	}
}
