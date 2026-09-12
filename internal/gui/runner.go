package gui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"script-manager/internal/action"
	"script-manager/internal/config"
	"script-manager/internal/terminal"
)

const cleanupTempScriptMinAge = 2 * time.Second

func cleanupTempScripts() {
	cutoff := time.Now().Add(-cleanupTempScriptMinAge)
	for _, pattern := range []string{action.TempScriptPattern, inlineOutPattern} {
		matches, err := filepath.Glob(filepath.Join(os.TempDir(), pattern))
		if err != nil {
			continue
		}
		for _, path := range matches {
			if info, err := os.Stat(path); err == nil && info.ModTime().After(cutoff) {
				continue
			}
			os.Remove(path)
		}
	}
}

func (a *App) RunAction(itemIndex, actionIndex int) error {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return fmt.Errorf("running actions is not supported on %s", runtime.GOOS)
	}
	item := a.itemAt(itemIndex)
	if item == nil {
		return fmt.Errorf("invalid item")
	}
	actions := config.ActionsForItem(a.cfg.Actions, item)
	if actionIndex < 0 || actionIndex >= len(actions) {
		return fmt.Errorf("invalid action")
	}
	if len(a.cfg.Shell) == 0 {
		return fmt.Errorf("no shell configured")
	}

	act := actions[actionIndex]
	merged, err := a.mergedItemForRun(item, act)
	if err != nil {
		return err
	}

	title := act.Title
	if item.Name != "" {
		title = act.Title + " · " + item.Name
	}

	term, err := terminal.Resolve(a.cfg.Terminal, runtime.GOOS, title, a.appDataDir)
	if err != nil {
		return err
	}

	var scriptPath string
	if act.Script != "" {
		expandedScript, err := action.Expand(act.Script, merged)
		if err != nil {
			return fmt.Errorf("script path template error: %w", err)
		}
		wrapped := action.WrapScriptFile(action.ShellBasename(a.cfg.Shell[0]), expandedScript, !act.NoWait)
		scriptPath, err = action.WriteTempScript(a.cfg.Shell[0], wrapped)
		if err != nil {
			return fmt.Errorf("failed to write temp script: %w", err)
		}
	} else {
		expandedCmd, err := action.Expand(act.Cmd, merged)
		if err != nil {
			return fmt.Errorf("cmd template error: %w", err)
		}
		wrapped := wrapScript(action.ShellBasename(a.cfg.Shell[0]), expandedCmd, !act.NoWait)
		scriptPath, err = action.WriteTempScript(a.cfg.Shell[0], wrapped)
		if err != nil {
			return fmt.Errorf("failed to write temp script: %w", err)
		}
	}
	shellArgv := action.ScriptArgv(a.cfg.Shell, scriptPath, !act.NoWait)

	cmd := exec.Command(term.Path(), term.Args(title, a.appDataDir, shellArgv)...)
	if a.appDataDir != "" {
		cmd.Dir = a.appDataDir
	}
	cmd.Env = action.Env(merged)
	if err := cmd.Start(); err != nil {
		os.Remove(scriptPath)
		return err
	}
	return nil
}

func wrapScript(shellBase, script string, stayOpen bool) string {
	switch shellBase {
	case "pwsh", "powershell":
		return "Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue\n" + script + "\n"
	case "cmd":
		return script + "\r\ndel \"%~f0\"\r\n"
	default:
		var b strings.Builder
		b.WriteString("rm -f -- \"$0\"\n")
		b.WriteString(script)
		b.WriteString("\n")
		if stayOpen {
			b.WriteString("__status=$?\n")
			b.WriteString("printf '\\n[exit status %s] Press Enter to close...' \"$__status\"\n")
			b.WriteString("read -r __line\n")
		}
		return b.String()
	}
}
