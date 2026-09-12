package openfile

import (
	"fmt"
	"os/exec"
	stdruntime "runtime"
)

// Open hands path to whatever the OS opens it with — the default editor for
// a file, the file manager for a directory.
func Open(path string) error {
	if path == "" {
		return nil
	}
	if err := command(path).Start(); err != nil {
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	return nil
}

func command(path string) *exec.Cmd {
	if stdruntime.GOOS == "windows" {
		// The empty argument is start's title parameter: without it a quoted
		// path is taken as the window title and nothing opens.
		return exec.Command("cmd", "/c", "start", "", path)
	}
	return exec.Command("xdg-open", path)
}
