package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/template"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"script-manager/internal/config"
	"script-manager/internal/ui"
)

func main() {
	cfgPath := flag.String("config", "", "path to config file (default: auto-detect)")
	flag.Parse()

	var cfg *config.Config
	if *cfgPath != "" {
		cfg = config.LoadFrom(*cfgPath)
	} else {
		cfg = config.Load()
	}

	fd := int(os.Stdin.Fd())
	savedState, _ := term.GetState(fd)

	var state ui.State

	for {
		a := ui.NewApp(cfg)
		a.RestoreState(state)

		p := tea.NewProgram(a, tea.WithAltScreen())
		m, err := p.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}

		result := m.(*ui.App)
		state = result.SaveState()

		if result.PendingAction() == nil {
			break
		}

		runAction(cfg, *result.PendingAction(), result.MergedItem(), fd, savedState)
	}
}

func runAction(cfg *config.Config, action config.Action, item map[string]any, fd int, savedState *term.State) {
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

	if !action.NoWait {
		waitForKey(fd)
	}
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
