package configedit

import "testing"

func TestPreviewItem(t *testing.T) {
	displays := []DisplayDTO{{Name: "default", List: "{{.name}} ({{.region}})", Details: "**{{.name}}**"}}
	item := ItemDTO{Name: "srv1", Fields: []FieldDTO{{Key: "region", Kind: "string", Value: "eu"}}}

	got := PreviewItem(item, nil, displays, "default", "")
	if got.Error != "" {
		t.Fatalf("unexpected error: %s", got.Error)
	}
	if got.ListLabel != "srv1 (eu)" {
		t.Errorf("ListLabel = %q, want %q", got.ListLabel, "srv1 (eu)")
	}
	if got.DetailsHTML == "" {
		t.Error("expected non-empty DetailsHTML")
	}
}

func TestPreviewItemMissingField(t *testing.T) {
	displays := []DisplayDTO{{Name: "default", List: "{{.name}}", Details: "{{.missing}}"}}
	item := ItemDTO{Name: "srv1"}

	got := PreviewItem(item, nil, displays, "default", "")
	if len(got.MissingFields) != 1 || got.MissingFields[0] != "missing" {
		t.Errorf("MissingFields = %v, want [missing]", got.MissingFields)
	}
}

func TestPreviewAction(t *testing.T) {
	item := ItemDTO{Name: "srv1"}
	act := ActionDTO{Cmd: "ssh {{.name}}", Description: "Connect to {{.name}}"}

	got := PreviewAction(item, nil, act)
	if got.Cmd != "ssh srv1" {
		t.Errorf("Cmd = %q, want %q", got.Cmd, "ssh srv1")
	}
	if got.Description != "Connect to srv1" {
		t.Errorf("Description = %q, want %q", got.Description, "Connect to srv1")
	}
}
