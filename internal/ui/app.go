package ui

import (
	"bytes"
	"text/template"

	"script-manager/internal/config"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	tl "github.com/mko88/bubbletea-tilelayout"
)

type appMode int

const (
	modeSelectItem   appMode = 0
	modeSelectAction appMode = 1
)

// App is the root Bubble Tea model.
type App struct {
	layout        *tl.TileLayout
	list          *ListTile
	description   *DescriptionTile
	actionsPanel  *ActionsTile
	cmdBar        *CmdBarTile
	status        *StatusBarTile
	mode          appMode
	windowSize    tea.WindowSizeMsg
	pendingAction *config.Action
	pendingItem   map[string]any
}

// State captures all cursor/scroll/focus/mode positions so they can be
// restored after an action subprocess returns.
type State struct {
	ListSel, ListOff int
	ActSel, ActOff   int
	DescScroll       int
	CmdScroll        int
	FocusedPane      int     // 1=actions, 2=description, 3=cmdBar (mode 1 only)
	Mode             appMode // 0=selectItem, 1=selectAction
}

func orTitle(configured, def string) string {
	if configured != "" {
		return configured
	}
	return def
}

func NewApp(cfg *config.Config) *App {
	list := newListTile(cfg.Items, cfg.Display.List)
	list.title = orTitle(cfg.Titles.Items, list.title)

	description := newDescriptionTile(list.Selected(), cfg.Display.Details)
	description.title = orTitle(cfg.Titles.Details, description.title)

	actionsPanel := newActionsTile(cfg.Actions)
	actionsPanel.title = orTitle(cfg.Titles.Actions, actionsPanel.title)

	cmdBar := newCmdBarTile()
	cmdBar.title = orTitle(cfg.Titles.Command, cmdBar.title)
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
		layout:       root,
		list:         list,
		description:  description,
		actionsPanel: actionsPanel,
		cmdBar:       cmdBar,
		status:       status,
	}
	a.actionsPanel.selected = -1
	return a
}

func (a *App) PendingAction() *config.Action { return a.pendingAction }
func (a *App) PendingItem() map[string]any   { return a.pendingItem }

func (a *App) SaveState() State {
	s := State{
		ListSel:    a.list.selected,
		ListOff:    a.list.offset,
		ActSel:     a.actionsPanel.selected,
		ActOff:     a.actionsPanel.offset,
		DescScroll: a.description.scrollOffset,
		CmdScroll:  a.cmdBar.scrollOffset,
		Mode:       a.mode,
	}
	switch {
	case a.actionsPanel.IsFocused():
		s.FocusedPane = 1
	case a.description.IsFocused():
		s.FocusedPane = 2
	case a.cmdBar.IsFocused():
		s.FocusedPane = 3
	}
	return s
}

func (a *App) RestoreState(s State) {
	a.list.selected = s.ListSel
	a.list.offset = s.ListOff
	a.actionsPanel.offset = s.ActOff
	a.description.scrollOffset = s.DescScroll
	a.cmdBar.scrollOffset = s.CmdScroll
	a.description.SetItem(a.list.Selected())

	if s.Mode == modeSelectAction {
		a.actionsPanel.selected = s.ActSel
		a.cmdBar.SetCmd(a.expandCmd())
		a.mode = modeSelectAction
		a.list.Size = tl.Size{FixedHeight: 3}
		a.list.SetFocused(false)
		a.actionsPanel.SetFocused(s.FocusedPane == 1 || s.FocusedPane == 0)
		a.description.SetFocused(s.FocusedPane == 2)
		a.cmdBar.SetFocused(s.FocusedPane == 3)
		a.status.SetMode(modeSelectAction)
	}
}

func (a *App) enterActionMode() {
	a.mode = modeSelectAction
	a.list.Size = tl.Size{FixedHeight: 3}
	a.layout.Update(a.windowSize)
	a.list.SetFocused(false)
	a.actionsPanel.selected = 0
	a.actionsPanel.offset = 0
	a.actionsPanel.SetFocused(true)
	a.cmdBar.SetCmd(a.expandCmd())
	a.cmdBar.ResetScroll()
	a.status.SetMode(modeSelectAction)
}

func (a *App) enterItemMode() {
	a.mode = modeSelectItem
	a.list.Size = tl.Size{Weight: 1}
	a.layout.Update(a.windowSize)
	a.actionsPanel.selected = -1
	a.actionsPanel.offset = 0
	a.actionsPanel.SetFocused(false)
	a.description.SetFocused(false)
	a.cmdBar.SetFocused(false)
	a.cmdBar.SetCmd("")
	a.list.SetFocused(true)
	a.status.SetMode(modeSelectItem)
	a.status.ClearMessage()
}

func (a *App) expandCmd() string {
	action := a.actionsPanel.Selected()
	item := a.list.Selected()
	if action == nil || item == nil {
		return ""
	}
	tmpl, err := template.New("cmd").Parse(action.Cmd)
	if err != nil {
		return action.Cmd
	}
	var buf bytes.Buffer
	tmpl.Execute(&buf, item)
	return buf.String()
}

func (a *App) Init() tea.Cmd {
	return a.layout.Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.windowSize = msg
		_, cmd := a.layout.Update(msg)
		return a, cmd

	case tea.KeyMsg:
		// Global exits.
		switch msg.String() {
		case "q", "Q", "ctrl+c":
			return a, tea.Quit
		case "esc":
			if a.mode == modeSelectAction {
				a.enterItemMode()
				return a, nil
			}
			return a, tea.Quit
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
		a.description.SetItem(a.list.Selected())
		a.description.ResetScroll()
		a.cmdBar.SetCmd(a.expandCmd())
		a.cmdBar.ResetScroll()
		a.status.ClearMessage()
	case "down", "j":
		a.list.MoveDown()
		a.description.SetItem(a.list.Selected())
		a.description.ResetScroll()
		a.cmdBar.SetCmd(a.expandCmd())
		a.cmdBar.ResetScroll()
		a.status.ClearMessage()
	case "enter":
		a.enterActionMode()
	}
	return a, nil
}

func (a *App) updateActionMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "right":
		switch {
		case a.actionsPanel.IsFocused():
			a.actionsPanel.SetFocused(false)
			a.description.SetFocused(true)
		case a.description.IsFocused():
			a.description.SetFocused(false)
			a.cmdBar.SetFocused(true)
		default: // cmdBar
			a.cmdBar.SetFocused(false)
			a.actionsPanel.SetFocused(true)
		}

	case "shift+tab", "left":
		switch {
		case a.actionsPanel.IsFocused():
			a.actionsPanel.SetFocused(false)
			a.cmdBar.SetFocused(true)
		case a.description.IsFocused():
			a.description.SetFocused(false)
			a.actionsPanel.SetFocused(true)
		default: // cmdBar
			a.cmdBar.SetFocused(false)
			a.description.SetFocused(true)
		}

	case "up", "k":
		switch {
		case a.actionsPanel.IsFocused():
			a.actionsPanel.MoveUp()
			a.cmdBar.SetCmd(a.expandCmd())
			a.cmdBar.ResetScroll()
		case a.description.IsFocused():
			a.description.ScrollUp()
		case a.cmdBar.IsFocused():
			a.cmdBar.ScrollUp()
		}
		a.status.ClearMessage()

	case "down", "j":
		switch {
		case a.actionsPanel.IsFocused():
			a.actionsPanel.MoveDown()
			a.cmdBar.SetCmd(a.expandCmd())
			a.cmdBar.ResetScroll()
		case a.description.IsFocused():
			a.description.ScrollDown()
		case a.cmdBar.IsFocused():
			a.cmdBar.ScrollDown()
		}
		a.status.ClearMessage()

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
		if action := a.actionsPanel.Selected(); action != nil {
			a.pendingAction = action
			a.pendingItem = a.list.Selected()
			return a, tea.Quit
		}

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		n := int(msg.String()[0]-'0') - 1
		if n < len(a.actionsPanel.actions) {
			a.pendingAction = &a.actionsPanel.actions[n]
			a.pendingItem = a.list.Selected()
			return a, tea.Quit
		}
	}

	return a, nil
}

func (a *App) View() string {
	return a.layout.View()
}
