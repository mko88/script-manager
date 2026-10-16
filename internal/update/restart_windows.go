//go:build windows

package update

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
)

// RestartInPlace runs exe in the current console and exits with its status
// once it ends, so the shell keeps waiting on this process meanwhile. It
// only returns on failure to start.
func RestartInPlace(exe string, args []string) error {
	// Ctrl+C / Ctrl+Break reach every process on the console; this one must
	// survive them, or the child is left fighting the shell for input.
	// signal.Ignore does not stop the console handler from exiting us.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	cmd := exec.Command(exe, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		signal.Stop(sig)
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
