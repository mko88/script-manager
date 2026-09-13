package ui

import (
	"time"

	"script-manager/internal/action"
	"script-manager/internal/config"
	"script-manager/internal/secret"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tl "github.com/mko88/bubbletea-tilelayout"
)

type clearMsgToken struct{ token int }

type appMode int

const (
	modeSelectItem   appMode = 0
	modeSelectAction appMode = 1
)

type App struct {
	layout           *tl.TileLayout
	list             *ListTile
	description      *DescriptionTile
	actionsPanel     *ActionsTile
	cmdBar           *CmdBarTile
	status           *StatusBarTile
	mode             appMode
	windowSize       tea.WindowSizeMsg
	globalEnv        map[string]any
	allActions       []config.Action
	itemActions      []config.Action
	activeGroup      string
	actionsTileTitle string
	savedListOffset  int
	msgToken         int
	reload           func() (*config.Config, error)
	cfg              *config.Config
	loadErr          error

	secretKey    []byte
	secretParams *secret.Params
	pinPrompt    bool
	pinEntry     string
	pinPending   config.Action
}

func NewApp(cfg *config.Config, reload func() (*config.Config, error), loadErr error) *App {
	list := newListTile(cfg.Items, cfg.Display)

	description := newDescriptionTile(cfg.Display)
	description.SetConfigPath(cfg.SourcePath)

	actionsPanel := newActionsTile(cfg.Actions)

	cmdBar := newCmdBarTile()
	status := newStatusBarTile()
	list.SetFocused(true)

	leftSide := tl.NewTileLayout("left", tl.Vertical, tl.Size{Weight: 1})
	leftSide.Add(list)
	leftSide.Add(actionsPanel)

	rightSide := tl.NewTileLayout("right", tl.Vertical, tl.Size{Weight: 2})
	rightSide.Add(description)
	rightSide.Add(cmdBar)

	content := tl.NewTileLayout("content", tl.Horizontal, tl.Size{Weight: 1})
	content.Add(leftSide)
	content.Add(rightSide)

	root := tl.NewRoot(tl.Vertical)
	root.Add(content)
	root.Add(status)

	a := &App{
		layout:           root,
		list:             list,
		description:      description,
		actionsPanel:     actionsPanel,
		cmdBar:           cmdBar,
		status:           status,
		globalEnv:        cfg.Env,
		allActions:       cfg.Actions,
		actionsTileTitle: actionsPanel.title,
		reload:           reload,
		cfg:              cfg,
		loadErr:          loadErr,
	}
	a.actionsPanel.selected = -1
	a.description.SetItem(list.Selected(), a.mergedItem(list.Selected()))
	a.updateActionsForItem()
	return a
}

func (a *App) applyConfig(cfg *config.Config) {
	if !secret.SameParams(a.secretParams, cfg.Secrets) {
		a.secretKey = nil
		a.secretParams = nil
	}
	a.cfg = cfg
	a.list.SetItems(cfg.Items, cfg.Display)

	a.description.SetDisplays(cfg.Display)
	a.description.SetConfigPath(cfg.SourcePath)

	a.globalEnv = cfg.Env
	a.allActions = cfg.Actions

	a.description.SetItem(a.list.Selected(), a.mergedItem(a.list.Selected()))
	a.description.ResetScroll()
	a.updateActionsForItem()

	if a.mode == modeSelectAction {
		switch {
		case len(a.actionsPanel.actions) == 0:
			a.actionsPanel.selected = -1
		case a.actionsPanel.selected >= len(a.actionsPanel.actions):
			a.actionsPanel.selected = len(a.actionsPanel.actions) - 1
		case a.actionsPanel.selected < 0:
			a.actionsPanel.selected = 0
		}
		a.refreshCmdBar()
	}
}

func (a *App) reloadConfig() tea.Cmd {
	if a.reload == nil {
		return nil
	}
	cfg, err := a.reload()
	if cfg.SourcePath == "" {
		return a.flashMessage("Reload failed: "+err.Error(), 3*time.Second)
	}
	a.applyConfig(cfg)
	if err != nil {
		return a.flashMessage("Config reloaded with a warning: "+err.Error(), 4*time.Second)
	}
	return a.flashMessage("Config reloaded", 2*time.Second)
}

func (a *App) enterActionMode() {
	a.savedListOffset = a.list.offset
	a.mode = modeSelectAction
	a.list.Size = tl.Size{FixedHeight: 3}
	a.layout.Update(a.windowSize)
	a.list.SetFocused(false)
	a.updateActionsForItem()
	a.actionsPanel.selected = 0
	a.actionsPanel.offset = 0
	a.actionsPanel.SetFocused(true)
	a.refreshCmdBar()
	a.status.SetContext(ctxActionsFocused)
}

func (a *App) enterItemMode() {
	a.description.ExitCopyMode()
	a.mode = modeSelectItem
	a.list.Size = tl.Size{Weight: 1}
	a.list.offset = a.savedListOffset
	a.layout.Update(a.windowSize)
	a.actionsPanel.selected = -1
	a.actionsPanel.offset = 0
	a.actionsPanel.SetFocused(false)
	a.description.SetFocused(false)
	a.cmdBar.SetFocused(false)
	a.cmdBar.SetCmd("")
	a.cmdBar.SetDescription("")
	a.list.SetFocused(true)
	a.status.SetContext(ctxItemSelect)
	a.status.ClearMessage()
}

func (a *App) updateActionsForItem() {
	a.itemActions = config.ActionsForItem(a.allActions, a.list.Selected())
	a.activeGroup = ""
	a.applyGroupFilter()
}

func (a *App) applyGroupFilter() {
	if a.activeGroup == "" {
		a.actionsPanel.SetActions(a.itemActions)
		a.actionsPanel.title = a.actionsTileTitle + " [all]"
		return
	}
	var filtered []config.Action
	for _, act := range a.itemActions {
		for _, g := range act.Groups {
			if g == a.activeGroup {
				filtered = append(filtered, act)
				break
			}
		}
	}
	a.actionsPanel.SetActions(filtered)
	a.actionsPanel.title = a.actionsTileTitle + " [" + a.activeGroup + "]"
}

func (a *App) cycleGroup(delta int) {
	seen := make(map[string]bool)
	groups := []string{""}
	for _, act := range a.itemActions {
		for _, g := range act.Groups {
			if !seen[g] {
				seen[g] = true
				groups = append(groups, g)
			}
		}
	}
	if len(groups) <= 1 {
		return
	}
	idx := 0
	for i, g := range groups {
		if g == a.activeGroup {
			idx = i
			break
		}
	}
	a.activeGroup = groups[(idx+delta+len(groups))%len(groups)]
	a.applyGroupFilter()
	a.actionsPanel.selected = 0
	a.actionsPanel.offset = 0
	a.refreshCmdBar()
}

func (a *App) refreshCmdBar() {
	a.cmdBar.SetCmd(a.previewCmd())
	a.cmdBar.SetDescription(a.previewDescription())
	a.cmdBar.ResetScroll()
}

func (a *App) onItemChanged() {
	a.description.SetItem(a.list.Selected(), a.mergedItem(a.list.Selected()))
	a.description.ResetScroll()
	a.updateActionsForItem()
	a.refreshCmdBar()
	a.status.ClearMessage()
}

func (a *App) mergedItem(item *config.Item) map[string]any {
	return secret.Redact(action.Merge(a.globalEnv, item))
}

func (a *App) mergedItemForRun(item *config.Item, act config.Action) (map[string]any, error) {
	merged := action.Merge(a.globalEnv, item)
	if !act.RequiresPIN {
		return secret.Strip(merged), nil
	}
	revealed, err := secret.Reveal(merged, a.secretKey)
	if err == secret.ErrWrongPIN {
		a.secretKey = nil
		a.secretParams = nil
	}
	return revealed, err
}

func (a *App) MergedItem() map[string]any {
	if item := a.list.Selected(); item != nil {
		return a.mergedItem(item)
	}
	return nil
}

func (a *App) previewDescription() string {
	act := a.actionsPanel.Selected()
	item := a.list.Selected()
	if act == nil || act.Description == "" || item == nil {
		return ""
	}
	return action.Preview(act.Description, a.mergedItem(item))
}

func (a *App) previewCmd() string {
	act := a.actionsPanel.Selected()
	item := a.list.Selected()
	if act == nil || item == nil {
		return ""
	}
	return action.Preview(act.Cmd, a.mergedItem(item))
}

func (a *App) flashMessage(text string, d time.Duration) tea.Cmd {
	a.msgToken++
	tok := a.msgToken
	a.status.SetMessage(text)
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearMsgToken{tok}
	})
}

func (a *App) Init() tea.Cmd {
	if a.loadErr != nil {
		return tea.Batch(a.layout.Init(), a.flashMessage("Config load failed: "+a.loadErr.Error(), 5*time.Second))
	}
	return a.layout.Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearMsgToken:
		if msg.token == a.msgToken {
			a.status.ClearMessage()
		}
		return a, nil

	case actionFinishedMsg:
		if msg.err != nil {
			return a, a.flashMessage("Action exited: "+msg.err.Error(), 3*time.Second)
		}
		return a, nil

	case tea.WindowSizeMsg:
		a.windowSize = msg
		_, cmd := a.layout.Update(msg)
		return a, cmd

	case tea.KeyMsg:
		if a.pinPrompt {
			return a.updatePINPrompt(msg)
		}
		switch msg.String() {
		case "q", "Q", "ctrl+c":
			return a, tea.Quit
		case "f5":
			return a, a.reloadConfig()
		case "esc":
			if a.description.IsCopyMode() {
				a.description.ExitCopyMode()
				a.status.SetContext(ctxDetailsFocused)
				return a, nil
			}
			if a.mode == modeSelectAction {
				a.enterItemMode()
				return a, nil
			}
		}

		if a.mode == modeSelectItem {
			return a.updateItemMode(msg)
		}
		return a.updateActionMode(msg)
	}

	_, cmd := a.layout.Update(msg)
	return a, cmd
}

func (a *App) updateItemMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		a.list.MoveUp()
		a.onItemChanged()
	case "down", "j":
		a.list.MoveDown()
		a.onItemChanged()
	case "enter", "tab":
		a.enterActionMode()
	}
	return a, nil
}

func (a *App) updateActionMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "right":
		if a.description.IsCopyMode() {
			a.description.CycleCopy(1)
			break
		}
		switch {
		case a.actionsPanel.IsFocused():
			a.actionsPanel.SetFocused(false)
			a.description.SetFocused(true)
			a.status.SetContext(ctxDetailsFocused)
		case a.description.IsFocused():
			a.description.SetFocused(false)
			a.cmdBar.SetFocused(true)
			a.status.SetContext(ctxCommandFocused)
		default:
			a.cmdBar.SetFocused(false)
			a.actionsPanel.SetFocused(true)
			a.status.SetContext(ctxActionsFocused)
		}

	case "shift+tab", "left":
		if a.description.IsCopyMode() {
			a.description.CycleCopy(-1)
			break
		}
		switch {
		case a.actionsPanel.IsFocused():
			a.actionsPanel.SetFocused(false)
			a.cmdBar.SetFocused(true)
			a.status.SetContext(ctxCommandFocused)
		case a.description.IsFocused():
			a.description.SetFocused(false)
			a.actionsPanel.SetFocused(true)
			a.status.SetContext(ctxActionsFocused)
		default:
			a.cmdBar.SetFocused(false)
			a.description.SetFocused(true)
			a.status.SetContext(ctxDetailsFocused)
		}

	case "up", "k":
		if a.description.IsCopyMode() {
			a.description.CycleCopy(-1)
		} else {
			switch {
			case a.actionsPanel.IsFocused():
				a.actionsPanel.MoveUp()
				a.refreshCmdBar()
			case a.description.IsFocused():
				a.description.ScrollUp()
			case a.cmdBar.IsFocused():
				a.cmdBar.ScrollUp()
			}
		}
		a.status.ClearMessage()

	case "down", "j":
		if a.description.IsCopyMode() {
			a.description.CycleCopy(1)
		} else {
			switch {
			case a.actionsPanel.IsFocused():
				a.actionsPanel.MoveDown()
				a.refreshCmdBar()
			case a.description.IsFocused():
				a.description.ScrollDown()
			case a.cmdBar.IsFocused():
				a.cmdBar.ScrollDown()
			}
		}
		a.status.ClearMessage()

	case "[":
		if a.actionsPanel.IsFocused() {
			a.cycleGroup(-1)
		}

	case "]":
		if a.actionsPanel.IsFocused() {
			a.cycleGroup(1)
		}

	case "y":
		cmd := a.cmdBar.Cmd()
		if cmd != "" {
			if err := clipboard.WriteAll(cmd); err != nil {
				a.status.SetMessage("clipboard unavailable: " + err.Error())
			} else {
				a.status.SetMessage("Command copied to clipboard")
			}
		}

	case "enter":
		if a.description.IsCopyMode() {
			return a, a.copySelectedValue()
		}
		if a.description.IsFocused() {
			if a.description.HasCopyValues() {
				a.description.EnterCopyMode()
				a.status.SetContext(ctxDetailsCopyMode)
			} else {
				a.status.SetMessage("No copyable values — wrap text in backticks in the template")
			}
		} else if a.actionsPanel.IsFocused() {
			if act := a.actionsPanel.Selected(); act != nil {
				return a, a.execAction(*act)
			}
		}
	}

	return a, nil
}

func (a *App) copySelectedValue() tea.Cmd {
	val, ok := a.description.CurrentCopyValue()
	if !ok {
		return nil
	}
	if err := clipboard.WriteAll(val); err != nil {
		a.status.SetMessage("clipboard unavailable: " + err.Error())
		return nil
	}

	field := a.description.CopyValueLabel()
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("236"))
	norm := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("236"))
	var msg string
	switch {
	case a.description.IsCurrentMasked() && field != "":
		msg = "Copied value of " + bold.Render(field) + norm.Render(" to clipboard")
	case a.description.IsCurrentMasked():
		msg = "Copied to clipboard"
	case field != "":
		msg = "Copied value of " + bold.Render(field) + norm.Render(": "+val)
	default:
		msg = "Copied: " + val
	}
	return a.flashMessage(msg, 2*time.Second)
}

func (a *App) View() string {
	return a.layout.View()
}
