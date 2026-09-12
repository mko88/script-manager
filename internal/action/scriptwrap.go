package action

import (
	"os"
	"strings"
)

const TempScriptPattern = "script-manager-action-*"

func WrapScriptFile(shellBase, scriptPath string, stayOpen bool) string {
	switch shellBase {
	case "pwsh", "powershell":
		return "Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue\n" +
			"& " + psQuote(scriptPath) + "\n"
	case "cmd":
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

func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

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
