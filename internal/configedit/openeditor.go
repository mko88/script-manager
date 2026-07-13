package configedit

import (
	"fmt"
	"os/exec"
	stdruntime "runtime"
)

// openFileCmd builds the OS-appropriate command to open path with its
// default application — split out from OpenInEditor so the argv is
// unit-testable without actually spawning a process.
func openFileCmd(path string) *exec.Cmd {
	if stdruntime.GOOS == "windows" {
		// cmd's "start" builtin launches path with its default file
		// association; the empty argument after "start" is a required
		// placeholder window title, without which a path containing spaces
		// would be misread as the title instead.
		return exec.Command("cmd", "/c", "start", "", path)
	}
	return exec.Command("xdg-open", path)
}

// OpenInEditor opens the currently loaded/saved config file with the
// operating system's default handler for its file type. A no-op if nothing
// is loaded yet (a.path == "").
//
// This deliberately doesn't go through Wails' runtime.BrowserOpenURL: that
// call validates its argument as an http(s) URL and rejects the "file"
// scheme and backslashes outright, so it can never open a local Windows
// path.
func (a *App) OpenInEditor() error {
	if a.path == "" {
		return nil
	}
	if err := openFileCmd(a.path).Start(); err != nil {
		return fmt.Errorf("failed to open %s: %w", a.path, err)
	}
	return nil
}
