package action

import (
	"path/filepath"
	"strings"
)

func ShellBasename(shellBin string) string {
	return strings.TrimSuffix(strings.ToLower(filepath.Base(shellBin)), ".exe")
}

func ScriptArgv(shell []string, scriptPath string, stayOpen bool) []string {
	switch ShellBasename(shell[0]) {
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

// Language names the syntax a command or script file is written in, for
// the editors that highlight it: the script's own extension when there is
// one, otherwise whatever the configured shell speaks.
func Language(shellBin, scriptPath string) string {
	switch strings.ToLower(filepath.Ext(scriptPath)) {
	case ".ps1", ".psm1", ".psd1":
		return "powershell"
	case ".sh", ".bash", ".zsh":
		return "shell"
	case ".yaml", ".yml":
		return "yaml"
	case "":
	default:
		return "plain"
	}
	switch ShellBasename(shellBin) {
	case "pwsh", "powershell":
		return "powershell"
	case "cmd":
		return "plain"
	default:
		return "shell"
	}
}
