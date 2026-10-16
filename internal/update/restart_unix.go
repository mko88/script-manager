//go:build !windows

package update

import (
	"os"
	"syscall"
)

// RestartInPlace replaces the current process with exe, keeping the
// terminal. It only returns on failure.
func RestartInPlace(exe string, args []string) error {
	return syscall.Exec(exe, append([]string{exe}, args...), os.Environ())
}
