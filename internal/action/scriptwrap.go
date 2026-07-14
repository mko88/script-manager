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
// for the given shell. It always routes the target through the shell's own
// native invocation syntax (& for pwsh, call for cmd, a bare command line
// for POSIX shells) rather than treating scriptPath as source to interpret
// directly or passing it as a raw argument — that's what lets one mechanism
// correctly run both a script that needs an interpreter (a bare .ps1 has no
// other way to run at all on Windows) and an already-native executable
// (.exe/.bat/.cmd, or a POSIX binary/shebang script for any interpreter,
// not just the one configured as shell:) without needing to know in advance
// which kind scriptPath is.
//
// Self-delete placement:
//   - pwsh/powershell: first line — PowerShell parses the whole file before
//     executing any of it.
//   - POSIX shells (bash, sh, zsh, dash, ksh): first line too — unlinking a
//     file another process still has open is always safe on POSIX. When
//     stayOpen is set, an epilogue at the end waits for Enter before an
//     interactive terminal window closes.
//   - cmd: last line — deleting a batch file as its very first line is a
//     well-known source of quirky behavior in cmd.exe.
//
// stayOpen's pwsh/cmd behavior comes entirely from ScriptArgv's -NoExit/-k
// flags; only the POSIX default branch needs its own pause epilogue baked in
// here too, since bash has no -NoExit equivalent for running a file
// directly. Pass stayOpen=false when there's no separate terminal window to
// keep open at all (an inline/captured run, or the TUI, which hands its own
// terminal to the subprocess and already prompts for a keypress itself).
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
// target shell recognizes, and returns its path. Running the script from a
// file — rather than inlining it as a single -Command/-c argument — avoids
// depending on the launcher's reconstruction of the argv surviving embedded
// newlines and quotes, which is unreliable for anything beyond a trivial
// one-liner. script is expected to already be wrapped (by WrapScriptFile or
// a caller's own equivalent for a plain command string), so it deletes this
// very file once the shell starts executing it; note the expanded content
// (including any masked values) is on disk in plain text until then.
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
