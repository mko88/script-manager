package configedit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"script-manager/internal/appdata"
	"script-manager/internal/config"
	"script-manager/internal/configmigrate"
	"script-manager/internal/filewatch"
	"script-manager/internal/recent"
	"script-manager/internal/secret"
	"script-manager/internal/terminal"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx          context.Context
	cfgPath      string
	cfg          *config.Config
	path         string
	appDataDir   string
	secretKey    []byte
	secretParams *secret.Params
	scriptWatch  *filewatch.Watcher

	conversionMu     sync.Mutex
	conversionAnswer chan bool
}

func NewApp(cfgPath string) *App {
	return &App{cfgPath: cfgPath, appDataDir: appdata.Dir()}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// ensureStartupFormat gates the first load. Nothing is loaded yet, so the
// path has to be resolved the same way LoadWithError would.
func (a *App) ensureStartupFormat() error {
	path := a.cfgPath
	if path == "" {
		resolved, err := config.ResolvePath()
		if err != nil {
			return nil
		}
		path = resolved
	}
	return a.ensureFormat(path)
}

func (a *App) stateFor(cfg *config.Config) StateDTO {
	if !secret.SameParams(a.secretParams, cfg.Secrets) {
		a.secretKey = nil
		a.secretParams = nil
	}
	a.cfg = cfg
	a.path = cfg.SourcePath
	recent.Add(a.appDataDir, cfg.SourcePath)
	return StateDTO{Config: ToConfigDTO(cfg), Path: cfg.SourcePath}
}

func (a *App) InitialState() StateDTO {
	if err := a.ensureStartupFormat(); errors.Is(err, configmigrate.ErrDeclined) {
		runtime.Quit(a.ctx)
		return StateDTO{Config: ToConfigDTO(&config.Config{})}
	}

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

func (a *App) NewBlank() StateDTO {
	cfg := &config.Config{
		Display: config.DisplayList{{Name: "default", List: "{{.name}}", Details: "**{{.name}}**"}},
	}
	a.cfg = cfg
	a.path = ""
	a.secretKey = nil
	a.secretParams = nil
	return StateDTO{Config: ToConfigDTO(cfg)}
}

func (a *App) BrowseOpen() (StateDTO, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Open config file",
		Filters: []runtime.FileFilter{{DisplayName: "YAML config (*.yaml, *.yml)", Pattern: "*.yaml;*.yml"}},
	})
	if err != nil {
		return StateDTO{}, err
	}
	if path == "" {
		return a.currentState(), nil
	}
	// Declining leaves the config already open in place, so this reads to the
	// frontend like a cancelled dialog.
	if err := a.ensureFormat(path); errors.Is(err, configmigrate.ErrDeclined) {
		return a.currentState(), nil
	}

	cfg, err := config.LoadFromWithError(path)
	if err != nil {
		return StateDTO{}, err
	}
	return a.stateFor(cfg), nil
}

// currentState is what the frontend already has: nothing changed.
func (a *App) currentState() StateDTO {
	cfg := a.cfg
	if cfg == nil {
		cfg = &config.Config{}
	}
	return StateDTO{Config: ToConfigDTO(cfg), Path: a.path}
}

func (a *App) BrowseScriptFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Select script file"})
}

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
	recent.Add(a.appDataDir, path)
	return SaveResultDTO{Path: path}, nil
}

func (a *App) PreviewItem(item ItemDTO, envFields []FieldDTO, displays []DisplayDTO, displayName string) PreviewDTO {
	return PreviewItem(item, envFields, displays, displayName, a.path)
}

func (a *App) PreviewAction(item ItemDTO, envFields []FieldDTO, act ActionDTO) ActionPreviewDTO {
	return PreviewAction(item, envFields, act)
}

func (a *App) ValidateConfig(state ConfigDTO) []ValidationIssueDTO {
	return ValidateConfig(state)
}

func (a *App) ValidateField(kind, value string) string {
	if _, err := decodeValue(kind, value); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) KnownTerminals() []string {
	return terminal.Names()
}

func (a *App) DataFolderPath() string {
	return a.appDataDir
}
