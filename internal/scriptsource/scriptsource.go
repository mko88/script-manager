package scriptsource

import (
	"fmt"
	"os"
	"unicode/utf8"
)

const MaxBytes = 256 * 1024

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
