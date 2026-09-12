package terminal

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"script-manager/internal/config"
)

const wtWindowName = "script-manager"

type Launcher struct {
	bin  string
	path string
	args func(title, dir string, shellArgv []string) []string
}

func (l Launcher) Path() string { return l.path }

func (l Launcher) Args(title, dir string, shellArgv []string) []string {
	return l.args(title, dir, shellArgv)
}

var knownTerminals = map[string]Launcher{
	"wt": {bin: "wt", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"-w", wtWindowName, "new-tab", "--title", title}
		if dir != "" {
			args = append(args, "-d", dir)
		}
		args = append(args, "--")
		return append(args, shellArgv...)
	}},
	"cmd": {bin: "cmd", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"/c", "start", title}
		if dir != "" {
			args = append(args, "/D", dir)
		}
		return append(args, shellArgv...)
	}},

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
		args := []string{"start"}
		if dir != "" {
			args = append(args, "--cwd", dir)
		}
		args = append(args, "--")
		return append(args, shellArgv...)
	}},
}

var windowsAutoDetect = []string{"wt", "wezterm", "alacritty", "cmd"}
var linuxAutoDetect = []string{
	"x-terminal-emulator", "gnome-terminal", "konsole", "xfce4-terminal",
	"terminator", "kitty", "alacritty", "wezterm", "foot", "xterm",
}

func findTerminal(names []string) (Launcher, error) {
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
	return Launcher{}, fmt.Errorf("no terminal emulator found on PATH (tried %s)", strings.Join(names, ", "))
}

func Names() []string {
	names := make([]string, 0, len(knownTerminals))
	for n := range knownTerminals {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func namedTerminal(name string) (Launcher, error) {
	lt, ok := knownTerminals[name]
	if !ok {
		return Launcher{}, fmt.Errorf("unknown terminal %q (known: %s)", name, strings.Join(Names(), ", "))
	}
	path, err := exec.LookPath(lt.bin)
	if err != nil {
		return Launcher{}, fmt.Errorf("configured terminal %q (%s) not found on PATH", name, lt.bin)
	}
	lt.path = path
	return lt, nil
}

func customTerminal(argvTemplate []string, title, dir string) (Launcher, error) {
	if len(argvTemplate) == 0 {
		return Launcher{}, fmt.Errorf("terminal: custom argv template is empty")
	}
	bin := argvTemplate[0]
	path, err := exec.LookPath(bin)
	if err != nil {
		return Launcher{}, fmt.Errorf("configured terminal %q not found on PATH", bin)
	}
	flags := make([]string, len(argvTemplate)-1)
	for i, tok := range argvTemplate[1:] {
		tok = strings.ReplaceAll(tok, "{{title}}", title)
		tok = strings.ReplaceAll(tok, "{{dir}}", dir)
		flags[i] = tok
	}
	return Launcher{
		bin:  bin,
		path: path,
		args: func(_, _ string, shellArgv []string) []string {
			return append(append([]string{}, flags...), shellArgv...)
		},
	}, nil
}

func Resolve(cfg config.TerminalConfig, goos, title, dir string) (Launcher, error) {
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
