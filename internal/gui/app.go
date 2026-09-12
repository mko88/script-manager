package gui

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"time"

	"script-manager/internal/action"
	"script-manager/internal/appdata"
	"script-manager/internal/applog"
	"script-manager/internal/config"
	"script-manager/internal/configmigrate"
	"script-manager/internal/exepath"
	"script-manager/internal/filewatch"
	"script-manager/internal/recent"
	"script-manager/internal/scriptsource"
	"script-manager/internal/secret"
	"script-manager/internal/version"

	"github.com/atotto/clipboard"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

type App struct {
	ctx        context.Context
	cfg        *config.Config
	configPath string
	loadConfig func(path string) (*config.Config, error)
	md         goldmark.Markdown
	exeDir     string
	appDataDir string
	loadErr    error

	declinedMu sync.Mutex
	declinedAt time.Time // mtime of the config the user declined to convert

	secretMu     sync.RWMutex
	secretKey    []byte
	secretParams *secret.Params

	configWatch *configWatcher
	scriptWatch *filewatch.Watcher

	inlineMu   sync.Mutex
	inlineRuns map[inlineKey]*inlineRun

	configEditorMu  sync.Mutex
	configEditorCmd *exec.Cmd
}

// NewApp defers loading to Startup: an old-format config has to be approved
// for conversion first, and that dialog needs a context.
func NewApp(cfgPath string) *App {
	go cleanupTempScripts()
	return &App{
		configPath: cfgPath,
		loadConfig: loadConfigFile,
		exeDir:     exepath.Dir(),
		appDataDir: appdata.Dir(),
		cfg:        &config.Config{},
		inlineRuns: make(map[inlineKey]*inlineRun),
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	}
}

func loadConfigFile(path string) (*config.Config, error) {
	if path != "" {
		return config.LoadFromWithError(path)
	}
	return config.LoadWithError()
}

func (a *App) loadFrom(path string) (*config.Config, error) {
	return a.loadConfig(path)
}

// Log lets the frontend put its own errors in the same file, so a JS
// exception and the Go call that preceded it sit next to each other.
func (a *App) Log(message string) {
	applog.Printf("ui: %s", message)
}

func (a *App) LogPath() string {
	return applog.Path(a.appDataDir)
}

// ConfigPath is the config file in use.
func (a *App) ConfigPath() string {
	return a.configSourcePath()
}

func (a *App) LoadError() string {
	if a.loadErr == nil {
		return ""
	}
	return a.loadErr.Error()
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	applog.Init(a.appDataDir)

	if err := a.ensureStartupFormat(); errors.Is(err, configmigrate.ErrDeclined) {
		applog.Printf("startup: %v", err)
		wailsruntime.Quit(ctx)
		return
	}

	a.cfg, a.loadErr = a.loadFrom(a.configPath)
	if a.cfg.SourcePath != "" {
		recent.Add(a.appDataDir, a.cfg.SourcePath)
	}

	applog.Printf("startup version=%s config=%s", version.Version, a.configSourcePath())
	a.watchTheme()
	a.watchConfig()
}

// ensureStartupFormat gates the first load. Nothing is loaded yet, so the
// path has to be resolved the same way LoadWithError would.
func (a *App) ensureStartupFormat() error {
	path := a.configPath
	if path == "" {
		resolved, err := config.ResolvePath()
		if err != nil {
			return nil
		}
		path = resolved
	}
	return a.ensureFormat(path)
}

// ReloadConfig is the F5 reload, so it asks again even about a config the
// user has already declined to convert.
func (a *App) ReloadConfig() (string, error) {
	return a.reload(a.ensureFormat)
}

// ReloadConfigOnChange is the reload the file watcher triggers.
func (a *App) ReloadConfigOnChange() (string, error) {
	return a.reload(a.ensureFormatOnChange)
}

// reload only refuses on a decline. Any other problem with the file — gone,
// unparseable — is left to the load, whose error says more.
func (a *App) reload(ensure func(string) error) (string, error) {
	if path := a.configSourcePath(); path != "" {
		if err := ensure(path); errors.Is(err, configmigrate.ErrDeclined) {
			return "", nil
		}
	}
	cfg, err := a.loadFrom(a.configPath)
	if cfg.SourcePath == "" {
		return "", err
	}
	a.setConfig(cfg)
	if err != nil {
		return err.Error(), nil
	}
	return "", nil
}

type ActionGroupDTO struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Color string `json:"color"`
}

func (a *App) GetActionGroups() []ActionGroupDTO {
	out := make([]ActionGroupDTO, len(a.cfg.ActionGroups))
	for i, g := range a.cfg.ActionGroups {
		out[i] = ActionGroupDTO{ID: g.ID, Title: g.Title, Color: g.Color}
	}
	return out
}

type ItemDTO struct {
	Index int    `json:"index"`
	Label string `json:"label"`
}

func (a *App) GetItems() []ItemDTO {
	items := make([]ItemDTO, len(a.cfg.Items))
	for i := range a.cfg.Items {
		items[i] = ItemDTO{Index: i, Label: a.renderListLabel(&a.cfg.Items[i])}
	}
	return items
}

func (a *App) renderListLabel(item *config.Item) string {
	d := config.FindDisplay(a.cfg.Display, item)
	out, err := action.Expand(d.List, item.Values())
	if err != nil {
		return item.Name
	}
	return out
}

func (a *App) scriptLanguage(scriptPath string) string {
	shell := ""
	if len(a.cfg.Shell) > 0 {
		shell = a.cfg.Shell[0]
	}
	return action.Language(shell, scriptPath)
}

func (a *App) mergedItem(item *config.Item) map[string]any {
	return secret.Redact(action.Merge(a.cfg.Env, item))
}

func (a *App) mergedItemForRun(item *config.Item, act config.Action) (map[string]any, error) {
	merged := action.Merge(a.cfg.Env, item)
	if !act.RequiresPIN {
		return secret.Strip(merged), nil
	}
	revealed, err := secret.Reveal(merged, a.sessionKey())
	if err == secret.ErrWrongPIN {
		a.forgetSessionKey()
	}
	return revealed, err
}

func (a *App) itemAt(index int) *config.Item {
	if index < 0 || index >= len(a.cfg.Items) {
		return nil
	}
	return &a.cfg.Items[index]
}

type ActionDTO struct {
	Index  int      `json:"index"`
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Groups []string `json:"groups"`
}

func (a *App) GetActions(itemIndex int) []ActionDTO {
	item := a.itemAt(itemIndex)
	actions := config.ActionsForItem(a.cfg.Actions, item)
	out := make([]ActionDTO, len(actions))
	for i, act := range actions {
		out[i] = ActionDTO{Index: i, ID: act.ID, Title: act.Title, Groups: act.Groups}
	}
	return out
}

type ActionDetailDTO struct {
	Description   string `json:"description"`
	Cmd           string `json:"cmd"`
	Script        string `json:"script"`
	ScriptContent string `json:"scriptContent"`
	ScriptError   string `json:"scriptError"`
	NoWait        bool   `json:"noWait"`
	Interactive   bool   `json:"interactive"`
	Language      string `json:"language"`
}

func (a *App) GetActionDetail(itemIndex, actionIndex int) ActionDetailDTO {
	item := a.itemAt(itemIndex)
	if item == nil {
		return ActionDetailDTO{}
	}
	actions := config.ActionsForItem(a.cfg.Actions, item)
	if actionIndex < 0 || actionIndex >= len(actions) {
		return ActionDetailDTO{}
	}
	act := actions[actionIndex]
	merged := a.mergedItem(item)
	script := action.Preview(act.Script, merged)
	var scriptContent, scriptErr string
	if script != "" {
		if content, err := scriptsource.Read(script); err != nil {
			scriptErr = err.Error()
		} else {
			scriptContent = content
		}
	}
	return ActionDetailDTO{
		Description:   action.Preview(act.Description, merged),
		Cmd:           action.Preview(act.Cmd, merged),
		Script:        script,
		ScriptContent: scriptContent,
		ScriptError:   scriptErr,
		Language:      a.scriptLanguage(script),
		NoWait:        act.NoWait,
		Interactive:   act.Interactive,
	}
}

func (a *App) CopyToClipboard(value string) error {
	return clipboard.WriteAll(value)
}

func (a *App) GetVersion() map[string]string {
	info := version.Get()
	return map[string]string{
		"version": info.Version,
		"commit":  info.Commit,
		"date":    info.Date,
	}
}
