package exepath

import (
	"os"
	"path/filepath"
)

// Path is the running executable, or "" if it can't be determined. Read it
// before an update swaps the file: on Linux it follows the renamed inode.
func Path() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return exe
}

func Dir() string {
	exe := Path()
	if exe == "" {
		return ""
	}
	return filepath.Dir(exe)
}
