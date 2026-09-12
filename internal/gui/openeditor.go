package gui

import (
	"fmt"
	"path/filepath"

	"script-manager/internal/openfile"
)

// OpenScriptInEditor opens a script-mode action's file in whatever the OS
// edits it with. The path comes from the action the user has selected, so
// it is already one this config would run.
func (a *App) OpenScriptInEditor(path string) error {
	if path == "" {
		return fmt.Errorf("this action has no script file")
	}
	return openfile.Open(filepath.Clean(path))
}
