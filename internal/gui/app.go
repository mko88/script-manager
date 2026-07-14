// Package gui is the Wails-bound backend for the desktop frontend. Every
// exported method on App becomes a callable binding in the frontend, under
// the "gui" namespace (window.go.gui.App).
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
	"script-manager/internal/scriptsource"

	"github.com/atotto/clipboard"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// App is the Wails-bound backend.
type App struct {
	ctx context.Context
	cfg *config.Config
	load func() (*config.Config, error)
	md   goldmark.Markdown
	// exeDir anchors things that must sit next to this binary specifically:
	// finding the sibling sm-config-edit executable (see browse.go). Theme,
	// messages, and the working directory actions run in all use appDataDir
	// instead, since exeDir isn't reliably writable (e.g. Program Files).
	exeDir     string
	appDataDir string
	loadErr    error // from the initial load; the frontend fetches it once via LoadError

	// inlineMu guards inlineRuns. Different item/action pairs may run
	// concurrently — switching to another action in the UI doesn't stop
	// one already running — but the same pair can't be started twice at
	// once; see RunActionInline.
	inlineMu   sync.Mutex
	inlineRuns map[inlineKey]*inlineRun

	// configEditorMu guards configEditorCmd; see LaunchConfigEditor.
	configEditorMu  sync.Mutex
	configEditorCmd *exec.Cmd
}

// NewApp builds the backend around a config loader, so an explicit -config
// path and F5 reloads go through the same resolution.
func NewApp(load func() (*config.Config, error)) *App {
	cfg, err := load()
	go cleanupTempScripts()
	return &App{
		cfg:        cfg,
		load:       load,
		exeDir:     exepath.Dir(),
		appDataDir: appdata.Dir(),
		loadErr:    err,
		inlineRuns: make(map[inlineKey]*inlineRun),
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	}
}

// LoadError returns the error from the initial config load, if any, so the
// frontend can surface a startup failure the same way ReloadConfig errors are
// surfaced. Returns "" when the initial load succeeded.
func (a *App) LoadError() string {
	if a.loadErr == nil {
		return ""
	}
	return a.loadErr.Error()
}

// Startup is wired as the Wails OnStartup callback.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.watchTheme()
}

// ReloadConfig re-reads the config from disk. On total failure — nothing at
// all could be loaded, e.g. a missing file — the previously loaded config is
// kept and an error is returned so the frontend can surface it without losing
// the current view. A preferred file (e.g. config-win.yaml) failing to parse
// while a fallback (config.yaml) still loads is not total failure: the
// fallback is applied and its parse error comes back as a non-fatal warning
// string instead, since a Go error return would otherwise reject the whole
// call on the frontend regardless of the fallback having succeeded.
func (a *App) ReloadConfig() (string, error) {
	cfg, err := a.load()
	if cfg.SourcePath == "" {
		return "", err
	}
	a.cfg = cfg
	if err != nil {
		return err.Error(), nil
	}
	return "", nil
}

// ActionGroupDTO is one entry of the config's optional actionGroups catalog
// — the frontend uses Color to paint group chips instead of showing every
// group with the same flat color; a group with no catalog entry (or no
// Color set) just falls back to the default chip styling.
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

// ItemDTO is a row in the item list.
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

// renderListLabel expands the list template for the item, falling back to
// the item's name when the template failed to parse or execute.
func (a *App) renderListLabel(item map[string]any) string {
	d := config.FindDisplay(a.cfg.Display, item)
	out, err := action.Expand(d.List, item)
	if err != nil {
		return fmt.Sprint(item[config.KeyName])
	}
	return out
}

func (a *App) mergedItem(item map[string]any) map[string]any {
	return action.Merge(a.cfg.Env, item)
}

func (a *App) itemAt(index int) map[string]any {
	if index < 0 || index >= len(a.cfg.Items) {
		return nil
	}
	return a.cfg.Items[index]
}

// ActionDTO is a row in the actions list for the selected item.
type ActionDTO struct {
	Index  int      `json:"index"`
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Groups []string `json:"groups"`
}

// GetActions returns the actions available for the item, in the same order
// GetActionDetail expects to receive back via ActionDTO.Index. Action IDs are
// optional (customActions rarely set one) so the index, not the ID, is the
// only reliable way to address a specific action.
func (a *App) GetActions(itemIndex int) []ActionDTO {
	item := a.itemAt(itemIndex)
	actions := config.ActionsForItem(a.cfg.Actions, item)
	out := make([]ActionDTO, len(actions))
	for i, act := range actions {
		out[i] = ActionDTO{Index: i, ID: act.ID, Title: act.Title, Groups: act.Groups}
	}
	return out
}

// ActionDetailDTO carries the expanded (but not yet run) command preview for
// the selected item/action pair.
type ActionDetailDTO struct {
	Description string `json:"description"`
	Cmd         string `json:"cmd"`
	Script      string `json:"script"`
	// ScriptContent is the Script file's own text, read fresh on every call
	// (scriptsource.Read, shared with sm-config-edit's Action editor
	// preview) — empty with ScriptError set if it couldn't be read (e.g. a
	// path that doesn't resolve, or an oversized/binary file).
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

// CopyToClipboard writes value to the system clipboard.
func (a *App) CopyToClipboard(value string) error {
	return clipboard.WriteAll(value)
}
