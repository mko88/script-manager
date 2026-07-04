package render

import (
	"bytes"
	"reflect"
	"testing"
	"text/template"
)

func parseDetail(t *testing.T, src string) *template.Template {
	t.Helper()
	tmpl, err := template.New("detail").Funcs(template.FuncMap{"mask": MaskFunc}).Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return tmpl
}

func TestFillMissingFields(t *testing.T) {
	tmpl := parseDetail(t, "name: {{.name}}\nip: `{{.clusterIp}}`\npass: `{{mask .password}}`\n")
	item := map[string]any{"name": "prod"}

	data, missing := FillMissingFields(tmpl, item)

	if want := []string{"clusterIp", "password"}; !reflect.DeepEqual(missing, want) {
		t.Errorf("missing = %v, want %v", missing, want)
	}
	if data["clusterIp"] != "<nil>" || data["password"] != "<nil>" {
		t.Errorf("missing fields not filled with <nil> placeholder: %v", data)
	}
	if data["name"] != "prod" {
		t.Errorf("existing field altered: %v", data["name"])
	}
	if _, ok := item["clusterIp"]; ok {
		t.Error("original item mutated")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute with filled data: %v", err)
	}
}

func TestFillMissingFieldsNilValue(t *testing.T) {
	tmpl := parseDetail(t, "{{.region}}")
	data, missing := FillMissingFields(tmpl, map[string]any{"region": nil})
	if want := []string{"region"}; !reflect.DeepEqual(missing, want) {
		t.Errorf("missing = %v, want %v", missing, want)
	}
	if data["region"] != "<nil>" {
		t.Errorf("nil value not replaced: %v", data["region"])
	}
}

func TestFillMissingFieldsNothingMissing(t *testing.T) {
	tmpl := parseDetail(t, "{{.name}} {{if .name}}yes{{end}}")
	item := map[string]any{"name": "x"}
	data, missing := FillMissingFields(tmpl, item)
	if missing != nil {
		t.Errorf("missing = %v, want nil", missing)
	}
	if !reflect.DeepEqual(data, item) {
		t.Errorf("data = %v, want original item", data)
	}
}

func TestFillMissingFieldsInsideIf(t *testing.T) {
	tmpl := parseDetail(t, "{{if .flag}}{{.inner}}{{end}}")
	_, missing := FillMissingFields(tmpl, map[string]any{})
	if want := []string{"flag", "inner"}; !reflect.DeepEqual(missing, want) {
		t.Errorf("missing = %v, want %v", missing, want)
	}
}

func TestMissingFieldsWarning(t *testing.T) {
	if got := MissingFieldsWarning(nil); got != "" {
		t.Errorf("warning for no missing fields = %q, want empty", got)
	}
	got := MissingFieldsWarning([]string{"a", "b"})
	if got != "⚠️ *Missing values (shown as &lt;nil&gt;): a, b*\n\n" {
		t.Errorf("warning = %q", got)
	}
}
