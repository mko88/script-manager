package configedit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"script-manager/internal/appdata"
	"script-manager/internal/config"
	"script-manager/internal/terminal"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound backend for sm-config-edit.
type App struct {
	ctx context.Context
	// cfgPath is an explicit -config path, "" meaning auto-detect at Startup —
	// mirrors cmd/script-manager-gui's -config flag / gui.NewApp shape.
	cfgPath string
	cfg     *config.Config
	// path is the file InitialState/BrowseOpen loaded from or Save last wrote
	// to; "" means an unsaved new file.
	path string
	// appDataDir is the app-data directory (see internal/appdata), used to
	// resolve both this app's own and its sibling script-manager-gui's
	// runtime theme/messages files.
	appDataDir string
}

// NewApp builds the backend around an optional explicit config path (from
// -config); "" means auto-detect the same way script-manager-gui does.
func NewApp(cfgPath string) *App {
	return &App{cfgPath: cfgPath, appDataDir: appdata.Dir()}
}

// Startup is wired as the Wails OnStartup callback.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) stateFor(cfg *config.Config) StateDTO {
	a.cfg = cfg
	a.path = cfg.SourcePath
	return StateDTO{Config: ToConfigDTO(cfg), Path: cfg.SourcePath}
}

// InitialState loads the config the same way script-manager-gui would: an
// explicit -config path, or auto-detect (config-win.yaml/config.yaml, exe dir
// then cwd) otherwise. Unlike script-manager-gui, "nothing found" during
// auto-detect is not an error here — a first-time user of this editor
// plausibly has no config yet, so it just starts blank. A load error against
// an explicit -config path, or a fallback-with-warning during auto-detect
// (SourcePath set but err non-nil, same signal gui.App.ReloadConfig uses), is
// still surfaced as a non-fatal warning.
func (a *App) InitialState() StateDTO {
	var cfg *config.Config
	var err error
	if a.cfgPath != "" {
		cfg, err = config.LoadFromWithError(a.cfgPath)
	} else {
		cfg, err = config.LoadWithError()
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	state := a.stateFor(cfg)
	if err != nil && (cfg.SourcePath != "" || a.cfgPath != "") {
		state.Warning = err.Error()
	}
	return state
}

// NewBlank discards the current in-memory config in favor of an empty one
// with a single starter display, so the form isn't completely blank. It does
// not touch a.path's file on disk.
func (a *App) NewBlank() StateDTO {
	cfg := &config.Config{
		Display: config.DisplayList{{Name: "default", List: "{{.name}}", Details: "**{{.name}}**"}},
	}
	a.cfg = cfg
	a.path = ""
	return StateDTO{Config: ToConfigDTO(cfg)}
}

// BrowseOpen prompts for a YAML file and loads it. Cancelling the dialog
// returns the unchanged current state with no error. A file the user
// explicitly picked that fails to load is a real error — unlike auto-detect,
// this must not silently fall back to blank.
func (a *App) BrowseOpen() (StateDTO, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Open config file",
		Filters: []runtime.FileFilter{{DisplayName: "YAML config (*.yaml, *.yml)", Pattern: "*.yaml;*.yml"}},
	})
	if err != nil {
		return StateDTO{}, err
	}
	if path == "" {
		cfg := a.cfg
		if cfg == nil {
			cfg = &config.Config{}
		}
		return StateDTO{Config: ToConfigDTO(cfg), Path: a.path}, nil
	}
	cfg, err := config.LoadFromWithError(path)
	if err != nil {
		return StateDTO{}, err
	}
	return a.stateFor(cfg), nil
}

// BrowseSaveAs prompts for a destination path; it does not write anything.
// An empty return means the dialog was cancelled.
func (a *App) BrowseSaveAs() (string, error) {
	suggested := "config.yaml"
	if a.path != "" {
		suggested = filepath.Base(a.path)
	}
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save config as",
		DefaultFilename: suggested,
		Filters:         []runtime.FileFilter{{DisplayName: "YAML config (*.yaml, *.yml)", Pattern: "*.yaml;*.yml"}},
	})
}

// Save writes state to path (or, if path is empty, to the file last
// loaded/saved) and updates the in-memory config to match. The frontend
// gates its Save button on already having a path, calling BrowseSaveAs first
// for a never-saved file.
func (a *App) Save(state ConfigDTO, path string) (SaveResultDTO, error) {
	if path == "" {
		path = a.path
	}
	if path == "" {
		return SaveResultDTO{}, fmt.Errorf("no file path to save to")
	}
	cfg, err := FromConfigDTO(state)
	if err != nil {
		return SaveResultDTO{}, err
	}
	out, err := cfg.Marshal()
	if err != nil {
		return SaveResultDTO{}, err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return SaveResultDTO{}, err
	}
	cfg.SourcePath = path
	a.cfg = cfg
	a.path = path
	return SaveResultDTO{Path: path}, nil
}

// PreviewItem and PreviewAction are thin bindings over the package-level
// preview functions (see preview.go) so the frontend can call them.
func (a *App) PreviewItem(item ItemDTO, envFields []FieldDTO, displays []DisplayDTO, displayName string) PreviewDTO {
	return PreviewItem(item, envFields, displays, displayName, a.path)
}

func (a *App) PreviewAction(item ItemDTO, envFields []FieldDTO, act ActionDTO) ActionPreviewDTO {
	return PreviewAction(item, envFields, act)
}

// ValidateConfig is a thin binding over the package-level ValidateConfig.
func (a *App) ValidateConfig(state ConfigDTO) []ValidationIssueDTO {
	return ValidateConfig(state)
}

// ValidateField reuses Save's exact decode logic so a raw-YAML field's
// textarea can show live feedback without a separate JS YAML parser.
func (a *App) ValidateField(kind, value string) string {
	if _, err := decodeValue(kind, value); err != nil {
		return err.Error()
	}
	return ""
}

// KnownTerminals lists the built-in terminal names for the Terminal
// section's "named" mode, reusing internal/terminal's table rather than
// duplicating it.
func (a *App) KnownTerminals() []string {
	return terminal.Names()
}
