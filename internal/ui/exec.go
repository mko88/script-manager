package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"script-manager/internal/action"
	"script-manager/internal/config"
)

// actionFinishedMsg is delivered when an action subprocess completes.
type actionFinishedMsg struct{ err error }

// execAction builds the tea.Cmd that runs the action in the configured
// shell. Bubble Tea suspends the TUI, hands the terminal to the subprocess,
// and resumes the same model afterwards — no state is lost.
func (a *App) execAction(act config.Action) tea.Cmd {
	merged := a.MergedItem()

	var cmd *exec.Cmd
	if act.Script != "" {
		expandedScript, err := action.Expand(act.Script, merged)
		if err != nil {
			return a.flashMessage("Script path template error: "+err.Error(), 3*time.Second)
		}
		cmd = exec.Command(expandedScript)
	} else {
		if len(a.cfg.Shell) == 0 {
			return a.flashMessage("No shell configured", 3*time.Second)
		}
		expanded, err := action.Expand(act.Cmd, merged)
		if err != nil {
			return a.flashMessage("Command template error: "+err.Error(), 3*time.Second)
		}
		args := append(append([]string{}, a.cfg.Shell[1:]...), expanded)
		cmd = exec.Command(a.cfg.Shell[0], args...)
	}
	cmd.Env = action.Env(merged)

	proc := &actionProcess{
		cmd:      cmd,
		title:    act.Title,
		itemName: fmt.Sprint(merged[config.KeyName]),
		wait:     !act.NoWait,
	}
	return tea.Exec(proc, func(err error) tea.Msg { return actionFinishedMsg{err} })
}

// actionProcess adapts the shell subprocess to tea.ExecCommand so a header is
// printed before it runs and, unless the action sets noWait, a keypress is
// awaited after it exits — all while Bubble Tea has released the terminal.
type actionProcess struct {
	cmd      *exec.Cmd
	title    string
	itemName string
	wait     bool

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func (p *actionProcess) SetStdin(r io.Reader)  { p.stdin = r }
func (p *actionProcess) SetStdout(w io.Writer) { p.stdout = w }
func (p *actionProcess) SetStderr(w io.Writer) { p.stderr = w }

func (p *actionProcess) Run() error {
	p.cmd.Stdin, p.cmd.Stdout, p.cmd.Stderr = p.stdin, p.stdout, p.stderr

	sep := strings.Repeat("─", 60)
	fmt.Fprintf(p.stdout, "\n%s\n  %s  ›  %s\n%s\n\n", sep, p.title, p.itemName, sep)

	err := p.cmd.Run()
	if err != nil {
		// Report here so the user sees it before the keypress prompt; the
		// error also travels back via actionFinishedMsg for the status bar.
		fmt.Fprintf(p.stderr, "action exited: %v\n", err)
	}

	if p.wait {
		waitForKey(p.stdin, p.stdout)
	}
	return err
}

// waitForKey blocks until a single keypress. The terminal is in cooked mode
// while Bubble Tea has released it, so switch to raw mode for the read —
// otherwise the user would have to press Enter.
func waitForKey(in io.Reader, out io.Writer) {
	fmt.Fprint(out, "\nPress any key to return...")

	if f, ok := in.(*os.File); ok {
		fd := int(f.Fd())
		if oldState, err := term.MakeRaw(fd); err == nil {
			defer term.Restore(fd, oldState)
		}
	}
	b := make([]byte, 1)
	in.Read(b)

	fmt.Fprintln(out)
}
