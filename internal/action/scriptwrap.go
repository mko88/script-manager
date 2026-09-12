package action

import (
	"os"
	"strings"
)

// TempScriptPattern matches the temp files WriteTempScript creates for a
// script-mode or cmd-mode action run — shared across the TUI and GUI so a
// cleanup sweep started by one (see internal/gui's cleanupTempScripts) also
// catches anything orphaned by the other.
const TempScriptPattern = "script-manager-action-*"

// WrapScriptFile wraps a direct invocation of scriptPath (a script-mode
// action's target file) with a self-delete of the wrapper's own temp file,
// for the given shell. The target always goes through the shell's native
// invocation syntax (& for pwsh, call for cmd, a bare command line for
// POSIX shells), never read as source or passed as a raw argument: that is
// what lets one mechanism run both a script needing an interpreter (a bare
// .ps1 has no other way to run at all on Windows) and an already-native
// executable, without knowing in advance which scriptPath is.
//
// Self-delete placement:
//   - pwsh/powershell: first line — PowerShell parses the whole file before
//     executing any of it.
//   - POSIX shells (bash, sh, zsh, dash, ksh): first line too — unlinking a
//     file another process still has open is always safe there.
//   - cmd: last line — deleting a batch file from its own first line makes
//     cmd.exe behave erratically.
//
// stayOpen adds a pause epilogue for POSIX shells only; pwsh and cmd get the
// same effect from ScriptArgv's -NoExit/-k. Pass false when no separate
// terminal window needs keeping open — an inline/captured run, or the TUI,
// which prompts for a keypress itself.
func WrapScriptFile(shellBase, scriptPath string, stayOpen bool) string {
	switch shellBase {
	case "pwsh", "powershell":
		return "Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue\n" +
			"& " + psQuote(scriptPath) + "\n"
	case "cmd":
		// call, not a bare invocation: if scriptPath is itself a .bat/.cmd, a
		// bare call would transfer control away and never run the
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

// WriteTempScript writes script to a new temp file with an extension the
// target shell recognizes, and returns its path. Running from a file, rather
// than inlining as one -Command/-c argument, avoids depending on the
// launcher reconstructing an argv with embedded newlines and quotes intact.
// script is expected to already carry its own self-delete (WrapScriptFile,
// or a caller's equivalent for a plain command string), so the file removes
// itself once the shell starts it — until then the expanded content,
// including any masked values, sits on disk in plain text.
func WriteTempScript(shellBin, script string) (string, error) {
	ext := ".txt"
	switch ShellBasename(shellBin) {
	case "pwsh", "powershell":
		ext = ".ps1"
	case "cmd":
		ext = ".bat"
	case "bash", "sh", "zsh", "dash", "ksh":
		ext = ".sh"
	}
	f, err := os.CreateTemp("", TempScriptPattern+ext)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(script); err != nil {
		return "", err
	}
	return f.Name(), nil
}
