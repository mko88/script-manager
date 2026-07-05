package configedit

import "testing"

func hasIssue(issues []ValidationIssueDTO, severity string) bool {
	for _, i := range issues {
		if i.Severity == severity {
			return true
		}
	}
	return false
}

func TestValidateConfig(t *testing.T) {
	t.Run("clean config has no issues", func(t *testing.T) {
		dto := ConfigDTO{
			Display: []DisplayDTO{{Name: "default"}},
			Actions: []ActionDTO{{ID: "ssh", Groups: []string{"remote"}}},
			Items:   []ItemDTO{{Name: "srv1", Display: "default", Actions: []string{"ssh"}, ActionGroups: []string{"remote"}}},
		}
		if issues := ValidateConfig(dto); len(issues) != 0 {
			t.Errorf("expected no issues, got %+v", issues)
		}
	})

	t.Run("duplicate action id is an error", func(t *testing.T) {
		dto := ConfigDTO{Actions: []ActionDTO{{ID: "ssh"}, {ID: "ssh"}}}
		issues := ValidateConfig(dto)
		if !hasIssue(issues, "error") {
			t.Errorf("expected an error-severity issue, got %+v", issues)
		}
	})

	t.Run("duplicate item name is a warning", func(t *testing.T) {
		dto := ConfigDTO{Items: []ItemDTO{{Name: "srv1"}, {Name: "srv1"}}}
		issues := ValidateConfig(dto)
		if !hasIssue(issues, "warning") {
			t.Errorf("expected a warning-severity issue, got %+v", issues)
		}
		if hasIssue(issues, "error") {
			t.Errorf("duplicate item names should not be blocking: %+v", issues)
		}
	})

	t.Run("dangling display reference is a warning", func(t *testing.T) {
		dto := ConfigDTO{Items: []ItemDTO{{Name: "srv1", Display: "missing"}}}
		issues := ValidateConfig(dto)
		if !hasIssue(issues, "warning") {
			t.Errorf("expected a warning, got %+v", issues)
		}
	})

	t.Run("dangling action id reference is a warning", func(t *testing.T) {
		dto := ConfigDTO{Items: []ItemDTO{{Name: "srv1", Actions: []string{"missing"}}}}
		issues := ValidateConfig(dto)
		if !hasIssue(issues, "warning") {
			t.Errorf("expected a warning, got %+v", issues)
		}
	})

	t.Run("dangling action group reference is a warning", func(t *testing.T) {
		dto := ConfigDTO{Items: []ItemDTO{{Name: "srv1", ActionGroups: []string{"missing"}}}}
		issues := ValidateConfig(dto)
		if !hasIssue(issues, "warning") {
			t.Errorf("expected a warning, got %+v", issues)
		}
	})
}
