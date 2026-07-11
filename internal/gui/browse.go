package gui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"script-manager/internal/config"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// BrowseConfig prompts for a YAML config file and, if one is picked, loads
// and switches to it — including making future ReloadConfig/F5 calls reload
// this same file (by replacing a.load with a closure over the picked path)
// instead of whatever was auto-detected or passed via -config at startup.
// Returns the picked path on success; "" with a nil error means the dialog
// was cancelled. A file the user explicitly picked that fails to load is a
// real error — unlike auto-detect, this must not silently fall back.
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

// siblingBinaryName returns the OS-appropriate executable name for a binary
// expected to live alongside this one — the inverse of shellBasename's .exe
// stripping.
func siblingBinaryName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

// configEditorArgv resolves the sibling sm-config-edit binary's path and
// argv — split out from LaunchConfigEditor so the argument-building logic
// (particularly whether -config is included) is unit-testable without
// actually spawning a process.
func (a *App) configEditorArgv() (bin string, args []string) {
	bin = filepath.Join(a.exeDir, siblingBinaryName("sm-config-edit"))
	if a.cfg != nil && a.cfg.SourcePath != "" {
		args = []string{"-config", a.cfg.SourcePath}
	}
	return bin, args
}

// LaunchConfigEditor starts sm-config-edit as a sibling process (same
// directory as this executable), pointed at whichever config file is
// currently loaded if its on-disk path is known, so the editor opens the
// same file instead of re-auto-detecting. Returns true if an instance this
// method previously launched is still running, in which case it's left
// alone rather than starting a second one — the frontend flashes an
// informational toast for that case rather than treating it as an error.
// Otherwise fire-and-forget, mirroring RunAction's exec.Command pattern —
// no synchronous Wait(), cmd.Dir set the same way.
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
