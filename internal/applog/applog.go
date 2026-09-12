package applog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const Filename = "sm-debug.log"

var (
	mu   sync.Mutex
	file *os.File
)

// Init starts a fresh log in dir. Each run truncates the previous one, so
// the file always describes the session being debugged rather than growing
// without bound.
func Init(dir string) {
	if dir == "" {
		return
	}
	f, err := os.Create(filepath.Join(dir, Filename))
	if err != nil {
		return
	}
	mu.Lock()
	file = f
	mu.Unlock()
	Printf("log started")
}

func Printf(format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()
	if file == nil {
		return
	}
	fmt.Fprintf(file, "%s  %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
	file.Sync()
}

func Path(dir string) string {
	return filepath.Join(dir, Filename)
}
