package gui

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"script-manager/internal/theme"
)

const ThemeChangedEvent = "theme:changed"

const themeWatchDebounce = 150 * time.Millisecond

func (a *App) watchTheme() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	if err := watcher.Add(a.appDataDir); err != nil {
		watcher.Close()
		return
	}

	go func() {
		defer watcher.Close()
		var debounce *time.Timer
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if filepath.Base(event.Name) != theme.Filename {
					continue
				}
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(themeWatchDebounce, func() {
					wailsruntime.EventsEmit(a.ctx, ThemeChangedEvent, theme.Load(a.appDataDir))
				})
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()
}
