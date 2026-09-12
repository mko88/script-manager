package configedit

import (
	"fmt"
	"os/exec"
	stdruntime "runtime"
)

func openFileCmd(path string) *exec.Cmd {
	if stdruntime.GOOS == "windows" {
		return exec.Command("cmd", "/c", "start", "", path)
	}
	return exec.Command("xdg-open", path)
}

func (a *App) OpenInEditor() error {
	if a.path == "" {
		return nil
	}
	if err := openFileCmd(a.path).Start(); err != nil {
		return fmt.Errorf("failed to open %s: %w", a.path, err)
	}
	return nil
}

func (a *App) OpenDataFolder() error {
	if a.appDataDir == "" {
		return nil
	}
	if err := openFileCmd(a.appDataDir).Start(); err != nil {
		return fmt.Errorf("failed to open %s: %w", a.appDataDir, err)
	}
	return nil
}
