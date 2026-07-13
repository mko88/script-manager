package gui

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"script-manager/internal/theme"
)

// ThemeChangedEvent is the Wails event name the frontend listens for (see
// frontend-shared/theme.ts's watchTheme) — emitted whenever sm-theme.json
// changes on disk, e.g. because sm-config-edit saved a new custom palette
// or switched the active theme, so this app picks it up live instead of
// only at its next launch.
const ThemeChangedEvent = "theme:changed"

// themeWatchDebounce coalesces the short burst of filesystem events a
// single save tends to produce (create/write/chmod are often separate
// events for one logical write) into one reload.
const themeWatchDebounce = 150 * time.Millisecond

// watchTheme watches the app-data directory for changes to theme.Filename
// and emits ThemeChangedEvent with the reloaded state on every change.
// Watches the directory rather than the file itself so it
// still notices the file's first-ever creation — fsnotify can't watch a
// path that doesn't exist yet, which matters here since sm-theme.json
// doesn't exist until a theme has actually been switched or saved at
// least once. Best-effort: any setup failure just means no live
// reload, exactly like without this feature at all — startup must not
// depend on it, and a.ctx must already be set (call from Startup, not
// NewApp).
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
