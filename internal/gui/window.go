package gui

import (
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const WindowClassName = "ScriptManagerGUIWindow"

func (a *App) SetAlwaysOnTop(enabled bool) {
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, enabled)
}

func (a *App) SetWindowOpacity(percent int) {
	switch {
	case percent < 0:
		percent = 0
	case percent > 100:
		percent = 100
	}
	setWindowOpacity(uint8(percent * 255 / 100))
}
