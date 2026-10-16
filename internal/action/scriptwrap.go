package action

import (
	"os"
	"strings"

	"script-manager/internal/shellquote"
)

const TempScriptPattern = "script-manager-action-*"

func WrapScriptFile(shellBase, scriptPath string, stayOpen bool) string {
	return WrapScriptFileArgs(shellBase, scriptPath, "", stayOpen)
}

// WrapScriptFileArgs is WrapScriptFile with args, already quoted for
// shellBase, after the script path.
func WrapScriptFileArgs(shellBase, scriptPath, args string, stayOpen bool) string {
	line := ScriptInvocation(shellBase, scriptPath, args)
	switch shellBase {
	case "pwsh", "powershell":
		return "Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue\n" + line + "\n"
	case "cmd":
		return line + "\r\n" + "del \"%~f0\"\r\n"
	default:
		var b strings.Builder
		b.WriteString("rm -f -- \"$0\"\n")
		b.WriteString(line)
		b.WriteString("\n")
		if stayOpen {
			b.WriteString("__status=$?\n")
			b.WriteString("printf '\\n[exit status %s] Press Enter to close...' \"$__status\"\n")
			b.WriteString("read -r __line\n")
		}
		return b.String()
	}
}

// ScriptInvocation is the line of the stub that calls the script.
func ScriptInvocation(shellBase, scriptPath, args string) string {
	var line string
	switch shellBase {
	case "pwsh", "powershell":
		line = "& " + psQuote(scriptPath)
	case "cmd":
		line = "call \"" + scriptPath + "\""
	default:
		line = shQuote(scriptPath)
	}
	if args != "" {
		line += " " + args
	}
	return line
}

// AppendArgs adds args to the end of an inline command.
func AppendArgs(cmd, args string) string {
	if args == "" {
		return cmd
	}
	return strings.TrimRight(cmd, " \t\r\n") + " " + args
}

func psQuote(s string) string { return shellquote.PowerShell(s) }
func shQuote(s string) string { return shellquote.POSIX(s) }

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
