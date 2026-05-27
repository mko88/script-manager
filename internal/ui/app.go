package ui

import (
	"bytes"
	"text/template"

	"script-manager/internal/config"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	tl "github.com/mko88/bubbletea-tilelayout"
)

// App is the root Bubble Tea model.
type App struct {
	layout        *tl.TileLayout
	list          *ListTile
	description   *DescriptionTile
	actionsPanel  *ActionsTile
	cmdBar        *CmdBarTile
	status        *StatusBarTile
	pendingAction *config.Action
	pendingItem   map[string]any
}

// State captures all cursor/scroll/focus positions so they can be restored
// after an action subprocess returns.
type State struct {
	ListSel, ListOff int
	ActSel, ActOff   int
	DescScroll       int
	CmdScroll        int
	FocusedPane      int // 0=list, 1=actions, 2=description, 3=cmdBar
}

func NewApp(cfg *config.Config) *App {
	list := newListTile(cfg.Items, cfg.Display.List)
	description := newDescriptionTile(list.Selected(), cfg.Display.Details)
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

	return &App{
		layout:       root,
		list:         list,
		description:  description,
		actionsPanel: actionsPanel,
		cmdBar:       cmdBar,
		status:       status,
	}
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
	a.actionsPanel.selected = s.ActSel
	a.actionsPanel.offset = s.ActOff
	a.description.scrollOffset = s.DescScroll
	a.cmdBar.scrollOffset = s.CmdScroll
	a.description.SetItem(a.list.Selected())
	a.cmdBar.SetCmd(a.expandCmd())
	if s.FocusedPane != 0 {
		a.list.SetFocused(false)
		a.actionsPanel.SetFocused(s.FocusedPane == 1)
		a.description.SetFocused(s.FocusedPane == 2)
		a.cmdBar.SetFocused(s.FocusedPane == 3)
	}
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
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "right":
			switch {
			case a.list.IsFocused():
				a.list.SetFocused(false)
				a.actionsPanel.SetFocused(true)
			case a.actionsPanel.IsFocused():
				a.actionsPanel.SetFocused(false)
				a.description.SetFocused(true)
			case a.description.IsFocused():
				a.description.SetFocused(false)
				a.cmdBar.SetFocused(true)
			default: // cmdBar
				a.cmdBar.SetFocused(false)
				a.list.SetFocused(true)
			}
			return a, nil

		case "shift+tab", "left":
			switch {
			case a.list.IsFocused():
				a.list.SetFocused(false)
				a.cmdBar.SetFocused(true)
			case a.actionsPanel.IsFocused():
				a.actionsPanel.SetFocused(false)
				a.list.SetFocused(true)
			case a.description.IsFocused():
				a.description.SetFocused(false)
				a.actionsPanel.SetFocused(true)
			default: // cmdBar
				a.cmdBar.SetFocused(false)
				a.description.SetFocused(true)
			}
			return a, nil

		case "up", "k":
			switch {
			case a.list.IsFocused():
				a.list.MoveUp()
				a.description.SetItem(a.list.Selected())
				a.description.ResetScroll()
				a.actionsPanel.selected = 0
				a.actionsPanel.offset = 0
				a.cmdBar.SetCmd(a.expandCmd())
				a.cmdBar.ResetScroll()
			case a.description.IsFocused():
				a.description.ScrollUp()
			case a.actionsPanel.IsFocused():
				a.actionsPanel.MoveUp()
				a.cmdBar.SetCmd(a.expandCmd())
				a.cmdBar.ResetScroll()
			case a.cmdBar.IsFocused():
				a.cmdBar.ScrollUp()
			}
			a.status.ClearMessage()
			return a, nil

		case "down", "j":
			switch {
			case a.list.IsFocused():
				a.list.MoveDown()
				a.description.SetItem(a.list.Selected())
				a.description.ResetScroll()
				a.actionsPanel.selected = 0
				a.actionsPanel.offset = 0
				a.cmdBar.SetCmd(a.expandCmd())
				a.cmdBar.ResetScroll()
			case a.description.IsFocused():
				a.description.ScrollDown()
			case a.actionsPanel.IsFocused():
				a.actionsPanel.MoveDown()
				a.cmdBar.SetCmd(a.expandCmd())
				a.cmdBar.ResetScroll()
			case a.cmdBar.IsFocused():
				a.cmdBar.ScrollDown()
			}
			a.status.ClearMessage()
			return a, nil

		case "y":
			cmd := a.cmdBar.Cmd()
			if cmd == "" {
				return a, nil
			}
			if err := clipboard.WriteAll(cmd); err != nil {
				a.status.SetMessage("clipboard unavailable: " + err.Error())
			} else {
				a.status.SetMessage("Command copied to clipboard")
			}
			return a, nil

		case "enter":
			if action := a.actionsPanel.Selected(); action != nil {
				a.pendingAction = action
				a.pendingItem = a.list.Selected()
				return a, tea.Quit
			}
			return a, nil

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			n := int(msg.String()[0]-'0') - 1
			if n < len(a.actionsPanel.actions) {
				a.pendingAction = &a.actionsPanel.actions[n]
				a.pendingItem = a.list.Selected()
				return a, tea.Quit
			}

		case "q", "Q", "ctrl+c", "esc":
			return a, tea.Quit
		}
	}

	_, cmd := a.layout.Update(msg)
	return a, cmd
}

func (a *App) View() string {
	return a.layout.View()
}

