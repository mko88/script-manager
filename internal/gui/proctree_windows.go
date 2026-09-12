//go:build windows

package gui

import (
	"os/exec"
	"strconv"
	"syscall"
)

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func killProcessTree(cmd *exec.Cmd) error {
	return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}
