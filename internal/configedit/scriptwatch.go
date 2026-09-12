package configedit

import (
	"script-manager/internal/filewatch"
	"script-manager/internal/openfile"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const ScriptChangedEvent = "script:changed"

// WatchScript follows the script file whose preview is on screen, so an edit
// made outside the app refreshes it. "" stops watching.
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

// OpenScriptInEditor opens an action's script file in the default editor.
func (a *App) OpenScriptInEditor(path string) error {
	return openfile.Open(path)
}
