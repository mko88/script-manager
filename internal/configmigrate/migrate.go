// Package configmigrate converts configs written before items had their own
// env section, where any key that wasn't structural was an environment
// variable.
package configmigrate

import (
	"errors"
	"fmt"
	"os"
	"time"

	"script-manager/internal/config"

	"gopkg.in/yaml.v3"
)

// ErrDeclined reports that the user was asked to convert and said no.
var ErrDeclined = errors.New("config conversion declined")

// Needed reports whether any item in the file carries a key outside the six.
func Needed(path string) (bool, error) {
	doc, err := read(path)
	if err != nil {
		return false, err
	}
	for _, item := range items(doc) {
		for key := range item {
			if !config.ItemKey(key) {
				return true, nil
			}
		}
	}
	return false, nil
}

// PromptMessage is what every app asks before converting, so the three read
// alike. It ends in a question because the dialogs answer Yes/No.
func PromptMessage(path, backupPath string) string {
	return fmt.Sprintf(
		"%s stores item environment variables in the old format and has to be "+
			"converted before it can be opened.\n\n"+
			"The original is saved as:\n%s\n\n"+
			"Comments in the file will be lost.\n\nConvert it now?",
		path, backupPath)
}

// BackupPath is where Convert saves the original, timestamped so it never
// overwrites a backup taken by hand.
func BackupPath(path string, now time.Time) string {
	return fmt.Sprintf("%s.%s.bak", path, now.Format("20060102-1504"))
}

// Convert saves the file to backupPath, then rewrites each item's non-
// structural keys under env. Comments do not survive the rewrite.
func Convert(path, backupPath string) error {
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(original, &doc); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	for _, item := range items(doc) {
		env, _ := item[config.KeyEnv].(map[string]any)
		for key, value := range item {
			if config.ItemKey(key) {
				continue
			}
			if env == nil {
				env = make(map[string]any)
			}
			env[key] = value
			delete(item, key)
		}
		if len(env) > 0 {
			item[config.KeyEnv] = env
		}
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	if err := os.WriteFile(backupPath, original, 0o644); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// Ensure converts path if it needs it, asking approve first. It returns the
// backup path when a conversion happened, "" when none was needed, and
// ErrDeclined when approve said no.
func Ensure(path string, approve func(path, backupPath string) bool) (string, error) {
	needed, err := Needed(path)
	if err != nil || !needed {
		return "", err
	}
	backup := BackupPath(path, time.Now())
	if !approve(path, backup) {
		return "", ErrDeclined
	}
	if err := Convert(path, backup); err != nil {
		return "", err
	}
	return backup, nil
}

func read(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return doc, nil
}

func items(doc map[string]any) []map[string]any {
	raw, _ := doc["items"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, elem := range raw {
		if item, ok := elem.(map[string]any); ok {
			out = append(out, item)
		}
	}
	return out
}
