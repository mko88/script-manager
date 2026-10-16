package update

import "os/exec"

// Start launches exe with args as a new, independent process. The GUI calls
// it after Install and then quits.
func Start(exe string, args []string) error {
	return exec.Command(exe, args...).Start()
}
