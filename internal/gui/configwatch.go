package gui

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const ConfigChangedEvent = "config:changed"

const configWatchDebounce = 250 * time.Millisecond

type configWatcher struct {
	mu      sync.Mutex
	watcher *fsnotify.Watcher
	dir     string
	file    string
}

func (a *App) watchConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	a.configWatch = &configWatcher{watcher: watcher}
	a.watchConfigPath(a.configSourcePath())

	go func() {
		defer watcher.Close()
		var debounce *time.Timer
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !a.configWatch.matches(event.Name) {
					continue
				}
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(configWatchDebounce, func() {
					wailsruntime.EventsEmit(a.ctx, ConfigChangedEvent)
				})
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()
}

func (a *App) configSourcePath() string {
	if a.cfg == nil {
		return ""
	}
	return a.cfg.SourcePath
}

func (a *App) watchConfigPath(path string) {
	if a.configWatch == nil || path == "" {
		return
	}
	a.configWatch.follow(path)
}

func (w *configWatcher) follow(path string) {
	dir := filepath.Dir(path)

	w.mu.Lock()
	defer w.mu.Unlock()
	if dir == w.dir {
		w.file = path
		return
	}
	if w.dir != "" {
		_ = w.watcher.Remove(w.dir)
	}
	if err := w.watcher.Add(dir); err != nil {
		w.dir = ""
		w.file = ""
		return
	}
	w.dir = dir
	w.file = path
}

func (w *configWatcher) matches(name string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == "" {
		return false
	}
	return filepath.Clean(name) == filepath.Clean(w.file)
}
