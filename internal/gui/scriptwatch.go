package gui

import (
	"script-manager/internal/filewatch"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const ScriptChangedEvent = "script:changed"

// WatchScript follows the script file the Command pane is showing, so an
// edit made outside the app shows up without reselecting the action. The
// frontend calls it with whatever it is displaying, and with "" when that
// is a command rather than a file.
func (a *App) WatchScript(path string) {
	if a.scriptWatch == nil {
		w, err := filewatch.New(func(changed string) {
			wailsruntime.EventsEmit(a.ctx, ScriptChangedEvent, changed)
		})
		if err != nil {
			return
		}
		a.scriptWatch = w
	}
	a.scriptWatch.Follow(path)
}
