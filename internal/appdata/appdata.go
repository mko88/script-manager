package appdata

import (
	"os"
	"path/filepath"
)

const dirName = "script-manager"

func Dir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(base, dirName)
	os.MkdirAll(dir, 0o755)
	return dir
}
