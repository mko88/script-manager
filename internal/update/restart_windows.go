//go:build windows

package update

import (
	"errors"
	"os"
	"os/exec"
)

// RestartInPlace runs exe in the current console and exits with its status
// once it ends, so the shell keeps waiting on this process meanwhile. It
// only returns on failure to start.
func RestartInPlace(exe string, args []string) error {
	cmd := exec.Command(exe, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	err := cmd.Wait()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		os.Exit(exit.ExitCode())
	}
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
	return nil
}
