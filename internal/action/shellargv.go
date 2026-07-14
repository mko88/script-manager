package action

import (
	"path/filepath"
	"strings"
)

// ShellBasename returns shellBin's filename without its directory or a
// trailing ".exe", the form ScriptArgv and callers elsewhere switch on to
// tell one configured shell from another (e.g. "pwsh" vs "cmd" vs "bash").
func ShellBasename(shellBin string) string {
	return strings.TrimSuffix(strings.ToLower(filepath.Base(shellBin)), ".exe")
}

// ScriptArgv returns the full argv (shell binary + args) that runs
// scriptPath as a script file through the given shell — not as an inline
// -c/-Command string, which would try to execute the path itself as a
// program and fail on anything that isn't natively runnable (e.g. a bare
// .ps1 with no .exe/.bat/.cmd wrapper). When stayOpen is true it adds a
// shell-specific flag so an interactive terminal window stays open (and the
// output visible) after the script finishes, rather than closing
// immediately; callers with no terminal to keep open (an inline/captured
// run, or the TUI which hands its own terminal to the subprocess and
// resumes afterwards regardless) should pass false.
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
