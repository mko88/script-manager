package gui

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"script-manager/internal/action"
	"script-manager/internal/appdata"
	"script-manager/internal/config"
	"script-manager/internal/exepath"
	"script-manager/internal/recent"
	"script-manager/internal/scriptsource"
	"script-manager/internal/secret"
	"script-manager/internal/version"

	"github.com/atotto/clipboard"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

type App struct {
	ctx        context.Context
	cfg        *config.Config
	load       func() (*config.Config, error)
	md         goldmark.Markdown
	exeDir     string
	appDataDir string
	loadErr    error

	secretMu     sync.RWMutex
	secretKey    []byte
	secretParams *secret.Params

	inlineMu   sync.Mutex
	inlineRuns map[inlineKey]*inlineRun

	configEditorMu  sync.Mutex
	configEditorCmd *exec.Cmd
}

func NewApp(load func() (*config.Config, error)) *App {
	cfg, err := load()
	go cleanupTempScripts()
	appDataDir := appdata.Dir()
	if cfg != nil && cfg.SourcePath != "" {
		recent.Add(appDataDir, cfg.SourcePath)
	}
	return &App{
		cfg:        cfg,
		load:       load,
		exeDir:     exepath.Dir(),
		appDataDir: appDataDir,
		loadErr:    err,
		inlineRuns: make(map[inlineKey]*inlineRun),
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	}
}

func (a *App) LoadError() string {
	if a.loadErr == nil {
		return ""
	}
	return a.loadErr.Error()
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.watchTheme()
}

func (a *App) ReloadConfig() (string, error) {
	cfg, err := a.load()
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
	for i, item := range a.cfg.Items {
		items[i] = ItemDTO{Index: i, Label: a.renderListLabel(item)}
	}
	return items
}

func (a *App) renderListLabel(item map[string]any) string {
	d := config.FindDisplay(a.cfg.Display, item)
	out, err := action.Expand(d.List, item)
	if err != nil {
		return fmt.Sprint(item[config.KeyName])
	}
	return out
}

func (a *App) mergedItem(item map[string]any) map[string]any {
	return secret.Display(action.Merge(a.cfg.Env, item), a.sessionKey())
}

func (a *App) mergedItemForRun(item map[string]any, act config.Action) (map[string]any, error) {
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

func (a *App) itemAt(index int) map[string]any {
	if index < 0 || index >= len(a.cfg.Items) {
		return nil
	}
	return a.cfg.Items[index]
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
