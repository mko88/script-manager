//go:build windows

package gui

import (
	"os/exec"
	"strconv"
	"syscall"
)

// setProcessGroup hides the console window Windows would otherwise flash
// briefly for a console-subsystem child (pwsh/cmd) launched from a GUI app
// with no console of its own — os/exec doesn't suppress this on its own, and
// an inline run's whole point is not needing a visible window for it. There's
// no POSIX-style process-group setup needed here; killProcessTree below uses
// taskkill /T instead.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

// killProcessTree kills cmd's whole process tree via taskkill /T (terminate
// the named process and everything it spawned) /F (force), since a plain
// cmd.Process.Kill only signals cmd itself, silently orphaning whatever
// foreground command it was running.
func killProcessTree(cmd *exec.Cmd) error {
	return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}
