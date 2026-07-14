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

// cleanupTempScriptMinAge is how old a matched file must be before
// cleanupTempScripts will remove it. NewApp spawns the sweep in the
// background (not to slow down startup waiting on a directory scan) — with
// no minimum age at all, that sweep can race a script this same instance is
// about to write moments later (e.g. the window opens and the user
// immediately clicks Run/Run here) and delete it out from under the shell
// that's about to read it. Something orphaned by a previous, crashed run is
// at least this old by the time a new instance starts; anything newer is
// left for a later sweep rather than risked.
const cleanupTempScriptMinAge = 2 * time.Second

// cleanupTempScripts removes every action script left behind by previous
// runs, regardless of how old that makes it — as long as it's older than
// cleanupTempScriptMinAge (see there for why that floor exists). Every
// script wrapScript produces deletes itself once the launched shell actually
// starts executing it (see wrapScript); this is only the fallback for
// whatever that missed — e.g. the terminal or shell never started at all —
// so nothing lingers across restarts, however old it is.
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

// RunAction launches the item/action pair in a terminal window. Which
// terminal is used is resolved by terminal.Resolve: an explicit config.
// Terminal override takes precedence, otherwise it auto-detects the most
// common terminal for the current OS (see internal/terminal). macOS and
// other platforms get a clear error instead of a silent no-op.
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
	merged := a.mergedItem(item)

	title := act.Title
	if name, ok := item[config.KeyName].(string); ok && name != "" {
		title = act.Title + " · " + name
	}

	// Resolve the terminal before writing anything to disk, so a missing or
	// misconfigured terminal fails fast without leaving a temp script behind.
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

// wrapScript wraps the expanded command with a self-delete of its own temp
// file, so cleanup is synchronized to actual execution instead of guessed by
// an external timer: whichever line runs the delete, the interpreter must
// already have opened (and read up to) that point in the file, so it can
// never race a terminal/shell that is merely slow to start — the previous
// approach (an external goroutine deleting the file on a timer) could win
// that race on a slow wt.exe/pwsh cold start, deleting the script before
// PowerShell ever opened it and making -File fail with "term ... is not
// recognized".
//
//   - pwsh/powershell: self-delete is the first line. PowerShell parses the
//     whole file before executing any of it, so this is also the fastest
//     point to get a secret-bearing script off disk.
//   - POSIX shells (bash, sh, zsh, dash, ksh): self-delete is the first line
//     too, for the same reason (unlinking a file another process still has
//     open is always safe on POSIX). When stayOpen is set, an epilogue at the
//     end waits for Enter before the terminal window closes.
//   - cmd: self-delete is the *last* line. Deleting a batch file as its very
//     first line is a well-known source of quirky behavior in cmd.exe (its
//     line-by-line reads can get confused); appending it after the real
//     command, once cmd.exe has already consumed everything before it, is
//     the safe, commonly-recommended placement.
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

// wrapScriptFile, writeTempScript, psQuote, and shQuote moved to
// script-manager/internal/action (as WrapScriptFile/WriteTempScript) so
// internal/ui's TUI could share the same script-mode wrapping the GUI
// already used — see action.WrapScriptFile's doc comment for the full
// reasoning.
