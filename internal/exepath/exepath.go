// Package exepath resolves the directory the running executable lives in —
// the anchor both GUI backends use for everything that sits next to the
// binaries (config auto-detection, sm-theme.json, runtime message overrides,
// the sibling app's executable). Its own package so internal/configedit
// doesn't have to depend on internal/gui just for this.
package exepath

import (
	"os"
	"path/filepath"
)

// Dir returns the directory containing the running executable, or "" if it
// can't be determined.
func Dir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}
