package gui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"script-manager/internal/config"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) BrowseConfig() (string, error) {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:   "Open config file",
		Filters: []wailsruntime.FileFilter{{DisplayName: "YAML config (*.yaml, *.yml)", Pattern: "*.yaml;*.yml"}},
	})
	if err != nil || path == "" {
		return "", err
	}

	cfg, err := config.LoadFromWithError(path)
	if err != nil {
		return "", err
	}
	a.cfg = cfg
	a.load = func() (*config.Config, error) { return config.LoadFromWithError(path) }
	return path, nil
}

func siblingBinaryName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func (a *App) configEditorArgv() (bin string, args []string) {
	bin = filepath.Join(a.exeDir, siblingBinaryName("sm-config-edit"))
	if a.cfg != nil && a.cfg.SourcePath != "" {
		args = []string{"-config", a.cfg.SourcePath}
	}
	return bin, args
}

func (a *App) LaunchConfigEditor() (alreadyRunning bool, err error) {
	a.configEditorMu.Lock()
	defer a.configEditorMu.Unlock()

	if a.configEditorCmd != nil {
		return true, nil
	}

	bin, args := a.configEditorArgv()
	cmd := exec.Command(bin, args...)
	if a.exeDir != "" {
		cmd.Dir = a.exeDir
	}
	if err := cmd.Start(); err != nil {
		return false, fmt.Errorf("failed to launch config editor: %w", err)
	}

	a.configEditorCmd = cmd
	go func() {
		cmd.Wait()
		a.configEditorMu.Lock()
		a.configEditorCmd = nil
		a.configEditorMu.Unlock()
	}()
	return false, nil
}
