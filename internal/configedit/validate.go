package configedit

import "fmt"

// ValidateConfig checks draft form state for problems ActionsForItem would
// otherwise resolve silently (an unknown display/action/group reference just
// falls back or drops the entry at runtime) or that would make IDs
// ambiguous. Duplicate action IDs are the only blocking error — everything
// else is a warning.
func ValidateConfig(dto ConfigDTO) []ValidationIssueDTO {
	issues := []ValidationIssueDTO{}

	actionIDs := map[string]bool{}
	for _, act := range dto.Actions {
		if act.ID == "" {
			continue
		}
		if actionIDs[act.ID] {
			issues = append(issues, ValidationIssueDTO{
				Severity: "error",
				Message:  fmt.Sprintf("duplicate action id %q", act.ID),
			})
		}
		actionIDs[act.ID] = true
	}

	itemNames := map[string]bool{}
	for _, item := range dto.Items {
		if item.Name == "" {
			continue
		}
		if itemNames[item.Name] {
			issues = append(issues, ValidationIssueDTO{
				Severity: "warning",
				Message:  fmt.Sprintf("duplicate item name %q", item.Name),
			})
		}
		itemNames[item.Name] = true
	}

	displayNames := map[string]bool{}
	for _, d := range dto.Display {
		displayNames[d.Name] = true
	}
	actionGroups := map[string]bool{}
	for _, act := range dto.Actions {
		for _, g := range act.Groups {
			actionGroups[g] = true
		}
	}

	for _, item := range dto.Items {
		label := item.Name
		if label == "" {
			label = "(unnamed item)"
		}
		if item.Display != "" && !displayNames[item.Display] {
			issues = append(issues, ValidationIssueDTO{
				Severity: "warning",
				Message:  fmt.Sprintf("item %q references unknown display %q", label, item.Display),
			})
		}
		for _, id := range item.Actions {
			if !actionIDs[id] {
				issues = append(issues, ValidationIssueDTO{
					Severity: "warning",
					Message:  fmt.Sprintf("item %q references unknown action id %q", label, id),
				})
			}
		}
		for _, g := range item.ActionGroups {
			if !actionGroups[g] {
				issues = append(issues, ValidationIssueDTO{
					Severity: "warning",
					Message:  fmt.Sprintf("item %q references unknown action group %q", label, g),
				})
			}
		}
	}

	return issues
}
