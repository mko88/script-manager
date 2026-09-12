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

type actionFinishedMsg struct{ err error }

func (a *App) execAction(act config.Action) tea.Cmd {
	merged := a.MergedItem()

	if len(a.cfg.Shell) == 0 {
		return a.flashMessage("No shell configured", 3*time.Second)
	}

	var cmd *exec.Cmd
	var cleanupPath string
	if act.Script != "" {
		expandedScript, err := action.Expand(act.Script, merged)
		if err != nil {
			return a.flashMessage("Script path template error: "+err.Error(), 3*time.Second)
		}
		wrapped := action.WrapScriptFile(action.ShellBasename(a.cfg.Shell[0]), expandedScript, false)
		scriptPath, err := action.WriteTempScript(a.cfg.Shell[0], wrapped)
		if err != nil {
			return a.flashMessage("Failed to write temp script: "+err.Error(), 3*time.Second)
		}
		cleanupPath = scriptPath
		argv := action.ScriptArgv(a.cfg.Shell, scriptPath, false)
		cmd = exec.Command(argv[0], argv[1:]...)
	} else {
		expanded, err := action.Expand(act.Cmd, merged)
		if err != nil {
			return a.flashMessage("Command template error: "+err.Error(), 3*time.Second)
		}
		args := append(append([]string{}, a.cfg.Shell[1:]...), expanded)
		cmd = exec.Command(a.cfg.Shell[0], args...)
	}
	cmd.Env = action.Env(merged)

	proc := &actionProcess{
		cmd:         cmd,
		title:       act.Title,
		itemName:    fmt.Sprint(merged[config.KeyName]),
		wait:        !act.NoWait,
		cleanupPath: cleanupPath,
	}
	return tea.Exec(proc, func(err error) tea.Msg { return actionFinishedMsg{err} })
}

type actionProcess struct {
	cmd         *exec.Cmd
	title       string
	itemName    string
	wait        bool
	cleanupPath string

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
	if p.cleanupPath != "" {
		os.Remove(p.cleanupPath)
	}
	if err != nil {
		fmt.Fprintf(p.stderr, "action exited: %v\n", err)
	}

	if p.wait {
		waitForKey(p.stdin, p.stdout)
	}
	return err
}

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
