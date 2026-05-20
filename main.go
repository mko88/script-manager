package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"
	tl "github.com/mko88/bubbletea-tilelayout"
	"golang.org/x/term"
)

type app struct {
	layout        *tl.TileLayout
	list          *ListTile
	description   *DescriptionTile
	actionsPanel  *ActionsTile
	status        *StatusBarTile
	pendingAction *Action
	pendingItem   map[string]any
}

func newApp(cfg *Config) *app {
	list := newListTile(cfg.Items, cfg.Display.List)
	description := newDescriptionTile(list.Selected(), cfg.Display.Details)
	actionsPanel := newActionsTile(cfg.Actions)
	status := newStatusBarTile()
	list.SetFocused(true)

	leftSide := tl.NewTileLayout("left", tl.Vertical, tl.Size{Weight: 1})
	leftSide.Add(list)
	leftSide.Add(actionsPanel)

	content := tl.NewTileLayout("content", tl.Horizontal, tl.Size{Weight: 1})
	content.Add(leftSide)
	content.Add(description)

	root := tl.NewRoot(tl.Vertical)
	root.Add(content)
	root.Add(status)

	return &app{
		layout:       root,
		list:         list,
		description:  description,
		actionsPanel: actionsPanel,
		status:       status,
	}
}

func (a *app) Init() tea.Cmd {
	return a.layout.Init()
}

func (a *app) expandCmd() string {
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

func (a *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			switch {
			case a.list.IsFocused():
				a.list.SetFocused(false)
				a.actionsPanel.SetFocused(true)
			case a.actionsPanel.IsFocused():
				a.actionsPanel.SetFocused(false)
				a.description.SetFocused(true)
			default:
				a.description.SetFocused(false)
				a.list.SetFocused(true)
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
			case a.description.IsFocused():
				a.description.ScrollUp()
			case a.actionsPanel.IsFocused():
				a.actionsPanel.MoveUp()
			}
			a.description.SetCmd(a.expandCmd())
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
			case a.description.IsFocused():
				a.description.ScrollDown()
			case a.actionsPanel.IsFocused():
				a.actionsPanel.MoveDown()
			}
			a.description.SetCmd(a.expandCmd())
			a.status.ClearMessage()
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

func (a *app) View() string {
	return a.layout.View()
}

func main() {
	cfg := loadConfig()

	// Snapshot the terminal state before the TUI takes over so we can
	// restore it reliably on every platform after Bubble Tea exits.
	fd := int(os.Stdin.Fd())
	savedState, _ := term.GetState(fd)

	// State preserved across action runs.
	listSel, listOff := 0, 0
	actSel, actOff := 0, 0
	descScroll := 0
	focusedPane := 0 // 0=list 1=actions 2=description

	for {
		a := newApp(cfg)

		// Restore cursor / scroll / focus from previous run.
		a.list.selected = listSel
		a.list.offset = listOff
		a.actionsPanel.selected = actSel
		a.actionsPanel.offset = actOff
		a.description.scrollOffset = descScroll
		a.description.SetItem(a.list.Selected())
		a.description.SetCmd(a.expandCmd())
		if focusedPane != 0 {
			a.list.SetFocused(false)
			a.actionsPanel.SetFocused(focusedPane == 1)
			a.description.SetFocused(focusedPane == 2)
		}

		p := tea.NewProgram(a, tea.WithAltScreen())
		m, err := p.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}

		result := m.(*app)

		// Save state before leaving the TUI.
		listSel = result.list.selected
		listOff = result.list.offset
		actSel = result.actionsPanel.selected
		actOff = result.actionsPanel.offset
		descScroll = result.description.scrollOffset
		switch {
		case result.actionsPanel.IsFocused():
			focusedPane = 1
		case result.description.IsFocused():
			focusedPane = 2
		default:
			focusedPane = 0
		}

		if result.pendingAction == nil {
			break
		}

		runAction(cfg, *result.pendingAction, result.pendingItem, fd, savedState)
	}
}

func runAction(cfg *Config, action Action, item map[string]any, fd int, savedState *term.State) {
	if len(cfg.Shell) == 0 {
		fmt.Fprintln(os.Stderr, "no shell configured")
		return
	}

	// Restore terminal from Bubble Tea's raw mode before the subprocess runs.
	// term.Restore handles Windows; stty sane forces ONLCR output on Unix.
	if savedState != nil {
		term.Restore(fd, savedState)
	}
	if runtime.GOOS != "windows" {
		stty := exec.Command("stty", "sane")
		stty.Stdin = os.Stdin
		stty.Run()
	}

	tmpl, err := template.New("cmd").Parse(action.Cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid cmd template: %v\n", err)
		return
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, item); err != nil {
		fmt.Fprintf(os.Stderr, "cmd template error: %v\n", err)
		return
	}
	expandedCmd := buf.String()

	sep := strings.Repeat("─", 60)
	fmt.Printf("\n%s\n  %s  ›  %s\n%s\n\n", sep, action.Title, fmt.Sprint(item["name"]), sep)

	env := os.Environ()
	for k, v := range item {
		env = append(env, strings.ToUpper(k)+"="+fmt.Sprint(v))
	}

	args := make([]string, 0, len(cfg.Shell))
	args = append(args, cfg.Shell[1:]...)
	args = append(args, expandedCmd)

	cmd := exec.Command(cfg.Shell[0], args...)
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "action exited: %v\n", err)
	}

	waitForKey(fd)
}

func waitForKey(fd int) {
	fmt.Print("\nPress any key to return...")

	// Switch to raw mode so any single keypress is read without waiting for Enter.
	oldState, err := term.MakeRaw(fd)
	b := make([]byte, 1)
	os.Stdin.Read(b)
	if err == nil {
		term.Restore(fd, oldState)
	}

	fmt.Println()
}
