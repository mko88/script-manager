// Package appdata resolves the per-user, per-OS application-data directory
// the GUI backends (internal/gui, internal/configedit) use for anything that
// needs a writable, stable home shared across processes and app upgrades:
// the theme/messages override files (see internal/theme, internal/messages)
// and the working directory actions run in. Unlike internal/exepath's
// executable directory, this location is guaranteed user-writable even when
// the binaries themselves are installed somewhere read-only (e.g. Program
// Files) — the reason this package exists rather than everything staying
// anchored to exepath.Dir().
package appdata

import (
	"os"
	"path/filepath"
)

// dirName is the subdirectory created under the OS's standard per-user
// config location.
const dirName = "script-manager"

// Dir returns the script-manager app-data directory, creating it if it
// doesn't exist yet: %AppData%\script-manager on Windows, $XDG_CONFIG_HOME/
// script-manager (or ~/.config/script-manager) on Linux, per os.UserConfigDir.
// Returns "" if the OS-standard location can't be determined, mirroring
// exepath.Dir()'s failure mode; callers already treat "" as "skip this."
func Dir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(base, dirName)
	os.MkdirAll(dir, 0o755)
	return dir
}
