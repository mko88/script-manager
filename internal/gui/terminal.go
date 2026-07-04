package gui

import (
	"fmt"
	"os/exec"
	"strings"
)

// linuxTerminal is one terminal emulator RunAction can launch on Linux: its
// binary name, the PATH-resolved location (filled by findLinuxTerminal), and
// how to assemble its argv — flag syntax for title, working directory, and
// the command differs per emulator.
type linuxTerminal struct {
	bin  string
	path string
	args func(title, dir string, shellArgv []string) []string
}

// linuxTerminals is the detection order. x-terminal-emulator goes first: on
// Debian-family systems it's the alternatives-managed symlink to whatever
// the user picked as their default, and Debian policy guarantees -T/-e.
var linuxTerminals = []linuxTerminal{
	{bin: "x-terminal-emulator", args: func(title, _ string, shellArgv []string) []string {
		return append([]string{"-T", title, "-e"}, shellArgv...)
	}},
	{bin: "gnome-terminal", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"--title", title}
		if dir != "" {
			args = append(args, "--working-directory", dir)
		}
		return append(append(args, "--"), shellArgv...)
	}},
	{bin: "konsole", args: func(_, dir string, shellArgv []string) []string {
		var args []string
		if dir != "" {
			args = append(args, "--workdir", dir)
		}
		return append(append(args, "-e"), shellArgv...)
	}},
	{bin: "xfce4-terminal", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"-T", title}
		if dir != "" {
			args = append(args, "--working-directory", dir)
		}
		return append(append(args, "-x"), shellArgv...)
	}},
	{bin: "kitty", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"--title", title}
		if dir != "" {
			args = append(args, "--directory", dir)
		}
		return append(args, shellArgv...)
	}},
	{bin: "alacritty", args: func(title, dir string, shellArgv []string) []string {
		args := []string{"-T", title}
		if dir != "" {
			args = append(args, "--working-directory", dir)
		}
		return append(append(args, "-e"), shellArgv...)
	}},
	{bin: "xterm", args: func(title, _ string, shellArgv []string) []string {
		return append([]string{"-T", title, "-e"}, shellArgv...)
	}},
}

// findLinuxTerminal returns the first emulator from linuxTerminals present
// on PATH, with its path field resolved.
func findLinuxTerminal() (linuxTerminal, error) {
	for _, t := range linuxTerminals {
		if path, err := exec.LookPath(t.bin); err == nil {
			t.path = path
			return t, nil
		}
	}
	names := make([]string, len(linuxTerminals))
	for i, t := range linuxTerminals {
		names[i] = t.bin
	}
	return linuxTerminal{}, fmt.Errorf("no terminal emulator found on PATH (tried %s)", strings.Join(names, ", "))
}
