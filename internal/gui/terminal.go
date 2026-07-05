package gui

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"script-manager/internal/config"
)

// terminalLauncher is one terminal emulator RunAction can launch: its PATH
// name, the resolved location (filled once found), and how to assemble its
// argv — flag syntax for title, working directory, and the command differs
// per emulator. exec.LookPath resolves bare names like "wt" or "cmd" through
// PATHEXT on Windows, so entries don't need an explicit ".exe" suffix.
type terminalLauncher struct {
	bin  string
	path string
	args func(title, dir string, shellArgv []string) []string
}

// knownTerminals is the built-in table, keyed by the name a `terminal:`
// config value can reference. Cross-platform apps (kitty, alacritty,
// wezterm) get one shared entry since their CLI flags don't vary by OS;
// platform-exclusive launchers (wt, cmd's start, the X11/Wayland terminals)
// each get their own.
var knownTerminals = map[string]terminalLauncher{
	// Windows: reuses the same dedicated window across every run via
	// `-w <name> new-tab`, giving the app one persistent WT instance instead
	// of a new window per action.
	"wt": {bin: "wt", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"-w", wtWindowName, "new-tab", "--title", title}
		if dir != "" {
			args = append(args, "-d", dir)
		}
		args = append(args, "--")
		return append(args, shellArgv...)
	}},
	// Windows universal fallback: cmd.exe's `start` builtin opens a plain
	// conhost console with no extra dependency beyond cmd.exe itself, which
	// every Windows install has. Last resort when nothing nicer is found.
	"cmd": {bin: "cmd", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"/c", "start", title}
		if dir != "" {
			args = append(args, "/D", dir)
		}
		return append(args, shellArgv...)
	}},

	// Linux: x-terminal-emulator is the Debian-alternatives symlink to
	// whatever the user picked as their default, so it's tried first there.
	"x-terminal-emulator": {bin: "x-terminal-emulator", args: func(title, _ string, shellArgv []string) []string {
		return append([]string{"-T", title, "-e"}, shellArgv...)
	}},
	"gnome-terminal": {bin: "gnome-terminal", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"--title", title}
		if dir != "" {
			args = append(args, "--working-directory", dir)
		}
		return append(append(args, "--"), shellArgv...)
	}},
	"konsole": {bin: "konsole", args: func(_, dir string, shellArgv []string) []string {
		var args []string
		if dir != "" {
			args = append(args, "--workdir", dir)
		}
		return append(append(args, "-e"), shellArgv...)
	}},
	"xfce4-terminal": {bin: "xfce4-terminal", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"-T", title}
		if dir != "" {
			args = append(args, "--working-directory", dir)
		}
		return append(append(args, "-x"), shellArgv...)
	}},
	"terminator": {bin: "terminator", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"-T", title}
		if dir != "" {
			args = append(args, "--working-directory", dir)
		}
		return append(append(args, "-x"), shellArgv...)
	}},
	"foot": {bin: "foot", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"-T", title}
		if dir != "" {
			args = append(args, "-D", dir)
		}
		return append(args, shellArgv...)
	}},
	"xterm": {bin: "xterm", args: func(title, _ string, shellArgv []string) []string {
		return append([]string{"-T", title, "-e"}, shellArgv...)
	}},

	// Cross-platform (Windows, Linux, macOS): identical CLI on every OS.
	"kitty": {bin: "kitty", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"--title", title}
		if dir != "" {
			args = append(args, "--directory", dir)
		}
		return append(args, shellArgv...)
	}},
	"alacritty": {bin: "alacritty", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"-T", title}
		if dir != "" {
			args = append(args, "--working-directory", dir)
		}
		return append(append(args, "-e"), shellArgv...)
	}},
	"wezterm": {bin: "wezterm", args: func(_, dir string, shellArgv []string) []string {
		// `wezterm start` has no direct --title flag; the tab/window title
		// comes from the running program instead, so title is unused here.
		args := []string{"start"}
		if dir != "" {
			args = append(args, "--cwd", dir)
		}
		args = append(args, "--")
		return append(args, shellArgv...)
	}},
}

// windowsAutoDetect and linuxAutoDetect are the auto-detection orders used
// when config.TerminalConfig is unset, trying the most common/nicest option
// first. windowsAutoDetect always succeeds in practice: "cmd" (plain
// conhost) is present on every Windows install.
var windowsAutoDetect = []string{"wt", "wezterm", "alacritty", "cmd"}
var linuxAutoDetect = []string{
	"x-terminal-emulator", "gnome-terminal", "konsole", "xfce4-terminal",
	"terminator", "kitty", "alacritty", "wezterm", "foot", "xterm",
}

// findTerminal returns the first entry from names present on PATH, tried in
// order, with its path field resolved.
func findTerminal(names []string) (terminalLauncher, error) {
	for _, name := range names {
		lt, ok := knownTerminals[name]
		if !ok {
			continue
		}
		if path, err := exec.LookPath(lt.bin); err == nil {
			lt.path = path
			return lt, nil
		}
	}
	return terminalLauncher{}, fmt.Errorf("no terminal emulator found on PATH (tried %s)", strings.Join(names, ", "))
}

// KnownTerminalNames returns every built-in terminal name, alphabetically —
// the valid values for a `terminal: <name>` config entry. Exported so
// config-editing tools (internal/configedit) can hint at valid names without
// duplicating this table.
func KnownTerminalNames() []string {
	names := make([]string, 0, len(knownTerminals))
	for n := range knownTerminals {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// namedTerminal resolves an explicit `terminal: <name>` config value against
// the built-in table, skipping auto-detection entirely.
func namedTerminal(name string) (terminalLauncher, error) {
	lt, ok := knownTerminals[name]
	if !ok {
		return terminalLauncher{}, fmt.Errorf("unknown terminal %q (known: %s)", name, strings.Join(KnownTerminalNames(), ", "))
	}
	path, err := exec.LookPath(lt.bin)
	if err != nil {
		return terminalLauncher{}, fmt.Errorf("configured terminal %q (%s) not found on PATH", name, lt.bin)
	}
	lt.path = path
	return lt, nil
}

// customTerminal builds a terminalLauncher from a `terminal: [argv, ...]`
// config value: the first element is the binary, the rest are its flags with
// "{{title}}"/"{{dir}}" substituted. The resolved shell command is always
// appended as the final arguments, the same convention every built-in entry
// follows.
func customTerminal(argvTemplate []string, title, dir string) (terminalLauncher, error) {
	if len(argvTemplate) == 0 {
		return terminalLauncher{}, fmt.Errorf("terminal: custom argv template is empty")
	}
	bin := argvTemplate[0]
	path, err := exec.LookPath(bin)
	if err != nil {
		return terminalLauncher{}, fmt.Errorf("configured terminal %q not found on PATH", bin)
	}
	flags := make([]string, len(argvTemplate)-1)
	for i, tok := range argvTemplate[1:] {
		tok = strings.ReplaceAll(tok, "{{title}}", title)
		tok = strings.ReplaceAll(tok, "{{dir}}", dir)
		flags[i] = tok
	}
	return terminalLauncher{
		bin:  bin,
		path: path,
		args: func(_, _ string, shellArgv []string) []string {
			return append(append([]string{}, flags...), shellArgv...)
		},
	}, nil
}

// resolveTerminal decides which terminal RunAction launches the script in:
// an explicit custom argv template or named override from config.Terminal
// takes precedence; otherwise it auto-detects the most common terminal for
// the current OS.
func resolveTerminal(cfg config.TerminalConfig, goos, title, dir string) (terminalLauncher, error) {
	if len(cfg.Argv) > 0 {
		return customTerminal(cfg.Argv, title, dir)
	}
	if cfg.Name != "" {
		return namedTerminal(cfg.Name)
	}
	if goos == "windows" {
		return findTerminal(windowsAutoDetect)
	}
	return findTerminal(linuxAutoDetect)
}
