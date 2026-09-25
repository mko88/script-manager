package gui

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"script-manager/internal/configmigrate"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// OpenedConfigDTO is the config a file dialog opened: Path is empty when
// nothing changed, and Warning is set when it opened but not cleanly.
type OpenedConfigDTO struct {
	Path    string `json:"path"`
	Warning string `json:"warning"`
}

func (a *App) BrowseConfig() (OpenedConfigDTO, error) {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:   "Open config file",
		Filters: []wailsruntime.FileFilter{{DisplayName: "YAML config (*.yaml, *.yml)", Pattern: "*.yaml;*.yml"}},
	})
	if err != nil || path == "" {
		return OpenedConfigDTO{}, err
	}

	// Declining leaves the config already open in place, so this reads to the
	// frontend like a cancelled dialog.
	if err := a.ensureFormat(path); errors.Is(err, configmigrate.ErrDeclined) {
		return OpenedConfigDTO{}, nil
	}

	warning, err := a.openConfigFile(path)
	if err != nil {
		return OpenedConfigDTO{}, err
	}
	return OpenedConfigDTO{Path: path, Warning: warning}, nil
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
