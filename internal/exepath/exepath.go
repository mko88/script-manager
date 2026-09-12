package exepath

import (
	"os"
	"path/filepath"
)

func Dir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}
