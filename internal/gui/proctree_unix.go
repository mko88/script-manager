//go:build !windows

package gui

import (
	"os/exec"
	"syscall"
)

// setProcessGroup puts cmd in its own process group so killProcessTree can
// terminate it and everything it spawned together.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessTree kills cmd's whole process group — the negative pid form of
// kill(2) targets every process in that group, not just cmd's own.
func killProcessTree(cmd *exec.Cmd) error {
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
