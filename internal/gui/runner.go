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

// tempScriptPattern matches the temp files RunAction writes; see
// cleanupTempScripts for their lifecycle.
const tempScriptPattern = "script-manager-action-*"

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
	for _, pattern := range []string{tempScriptPattern, inlineOutPattern} {
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
		wrapped := wrapScriptFile(shellBasename(a.cfg.Shell[0]), expandedScript, !act.NoWait)
		scriptPath, err = writeTempScript(a.cfg.Shell[0], wrapped)
		if err != nil {
			return fmt.Errorf("failed to write temp script: %w", err)
		}
	} else {
		expandedCmd, err := action.Expand(act.Cmd, merged)
		if err != nil {
			return fmt.Errorf("cmd template error: %w", err)
		}
		wrapped := wrapScript(shellBasename(a.cfg.Shell[0]), expandedCmd, !act.NoWait)
		scriptPath, err = writeTempScript(a.cfg.Shell[0], wrapped)
		if err != nil {
			return fmt.Errorf("failed to write temp script: %w", err)
		}
	}
	shellArgv := buildShellArgv(a.cfg.Shell, scriptPath, !act.NoWait)

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

// wrapScriptFile is wrapScript's counterpart for script-mode actions: it
// wraps a direct invocation of scriptPath instead of an expanded command
// string — no interpreter, no extra flags, the file is expected to already
// be natively runnable. Self-delete placement follows the exact same
// reasoning as wrapScript. pwsh/cmd's stay-open behavior still comes
// entirely from buildShellArgv's -NoExit/-k flags, unchanged; only the
// POSIX default branch needs its own pause epilogue baked in here too,
// since bash has no -NoExit equivalent for running a script file directly.
func wrapScriptFile(shellBase, scriptPath string, stayOpen bool) string {
	switch shellBase {
	case "pwsh", "powershell":
		return "Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue\n" +
			"& " + psQuote(scriptPath) + "\n"
	case "cmd":
		// call, not a bare invocation: if scriptPath is itself a .bat/.cmd,
		// a bare call would transfer control away and never run the
		// self-delete line after it.
		return "call \"" + scriptPath + "\"\r\n" + "del \"%~f0\"\r\n"
	default:
		var b strings.Builder
		b.WriteString("rm -f -- \"$0\"\n")
		b.WriteString(shQuote(scriptPath))
		b.WriteString("\n")
		if stayOpen {
			b.WriteString("__status=$?\n")
			b.WriteString("printf '\\n[exit status %s] Press Enter to close...' \"$__status\"\n")
			b.WriteString("read -r __line\n")
		}
		return b.String()
	}
}

// psQuote/shQuote wrap a path in single quotes for their respective shells,
// escaping any embedded single quote (doubled for PowerShell, '\'' for
// POSIX shells) — paths containing spaces are the common case this guards
// against.
func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// writeTempScript writes script to a new temp file with an extension the
// target shell recognizes, and returns its path. Running the script from a
// file — rather than inlining it as a single -Command/-c argument — avoids
// depending on the terminal launcher's reconstruction of the argv surviving
// embedded newlines and quotes, which is unreliable for anything beyond a
// trivial one-liner. script is expected to already be wrapped by wrapScript,
// so it deletes this very file once the shell starts executing it; note the
// expanded command (including any masked values) is on disk in plain text
// until then.
func writeTempScript(shellBin, script string) (string, error) {
	ext := ".txt"
	switch shellBasename(shellBin) {
	case "pwsh", "powershell":
		ext = ".ps1"
	case "cmd":
		ext = ".bat"
	case "bash", "sh", "zsh", "dash", "ksh":
		ext = ".sh"
	}
	f, err := os.CreateTemp("", tempScriptPattern+ext)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(script); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func shellBasename(shellBin string) string {
	return strings.TrimSuffix(strings.ToLower(filepath.Base(shellBin)), ".exe")
}

// buildShellArgv returns the full argv (shell binary + args) that runs
// scriptPath, for the given shell. When stayOpen is true it uses a
// shell-specific flag so the tab remains open (and the output visible) after
// the script finishes, rather than closing immediately.
func buildShellArgv(shell []string, scriptPath string, stayOpen bool) []string {
	switch shellBasename(shell[0]) {
	case "pwsh", "powershell":
		argv := []string{shell[0]}
		for _, a := range shell[1:] {
			if strings.EqualFold(a, "-command") {
				continue
			}
			argv = append(argv, a)
		}
		if stayOpen {
			argv = append(argv, "-NoExit")
		}
		return append(argv, "-File", scriptPath)
	case "cmd":
		flag := "/c"
		if stayOpen {
			flag = "/k"
		}
		return []string{shell[0], flag, scriptPath}
	default:
		// POSIX shells: -c makes the next argument a command *string*, which
		// would try to execute the script path as a program (and fail without
		// an exec bit). Strip it so the path is read as a script file, the
		// same way the -Command strip works for pwsh above.
		argv := []string{shell[0]}
		for _, a := range shell[1:] {
			if a == "-c" {
				continue
			}
			argv = append(argv, a)
		}
		return append(argv, scriptPath)
	}
}
