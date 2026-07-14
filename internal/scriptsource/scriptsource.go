// Package scriptsource reads a script-mode action's target file for
// preview, shared by sm-config-edit's Action editor and script-manager-gui's
// Command pane so both show the same file the same way.
package scriptsource

import (
	"fmt"
	"os"
	"unicode/utf8"
)

// MaxBytes caps how much of a script file Read will return — script files
// are expected to be small; this just guards against a large or binary
// file being pointed to by mistake.
const MaxBytes = 256 * 1024

// Read reads path's content for a script-mode action's source preview.
// Returns an error meant to be shown directly to the user (e.g. "no such
// file or directory") for any problem: missing file, directory, oversized,
// or not valid UTF-8 text.
func Read(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("that's a directory, not a file")
	}
	if info.Size() > MaxBytes {
		return "", fmt.Errorf("too large to preview (%d KB)", info.Size()/1024)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("doesn't look like a text file")
	}
	return string(data), nil
}
