package filewatch

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const debounce = 200 * time.Millisecond

// Watcher follows one file at a time and calls onChange when it is written.
//
// It watches the file's directory rather than the file: an editor that saves
// by writing a temp file and renaming it over the original replaces the
// inode, and a watch on the file itself would follow the replaced one into
// oblivion. It also means a file that doesn't exist yet is still noticed
// when it appears.
type Watcher struct {
	mu       sync.Mutex
	watcher  *fsnotify.Watcher
	dir      string
	file     string
	debounce *time.Timer
	onChange func(path string)
}

func New(onChange func(path string)) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	fw := &Watcher{watcher: w, onChange: onChange}
	go fw.run()
	return fw, nil
}

// Follow switches to watching path. An empty path stops watching.
func (w *Watcher) Follow(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if path == "" {
		if w.dir != "" {
			_ = w.watcher.Remove(w.dir)
		}
		w.dir, w.file = "", ""
		return
	}

	path = filepath.Clean(path)
	if path == w.file {
		return
	}
	dir := filepath.Dir(path)
	if dir != w.dir {
		if w.dir != "" {
			_ = w.watcher.Remove(w.dir)
		}
		if err := w.watcher.Add(dir); err != nil {
			w.dir, w.file = "", ""
			return
		}
		w.dir = dir
	}
	w.file = path
}

func (w *Watcher) Close() {
	_ = w.watcher.Close()
}

func (w *Watcher) run() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.mu.Lock()
			watched := w.file
			matches := watched != "" && filepath.Clean(event.Name) == watched
			if matches {
				if w.debounce != nil {
					w.debounce.Stop()
				}
				w.debounce = time.AfterFunc(debounce, func() { w.onChange(watched) })
			}
			w.mu.Unlock()
		case _, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}
