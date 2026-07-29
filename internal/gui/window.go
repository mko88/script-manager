package gui

import (
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// WindowClassName is this app's native window class, set once on the
// window in main.go's Windows options and reused here (see
// transparency_windows.go) to look up the window handle unambiguously —
// sm-config-edit is a second Wails app that often runs alongside this one
// and would otherwise share the default "wailsWindow" class name.
const WindowClassName = "ScriptManagerGUIWindow"

// SetAlwaysOnTop pins or unpins the window above every other window.
func (a *App) SetAlwaysOnTop(enabled bool) {
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, enabled)
}

// SetWindowOpacity fades the whole window — including its normally opaque
// content — to percent (0-100) via a native layered-window call
// (setWindowOpacity), rather than Wails' WebView2-level transparency
// options: those only let individual page elements with alpha in their own
// CSS show through, which wouldn't affect this app's opaque theme panels at
// all, whereas a whole-window blend needs no frontend changes. percent is
// clamped to [0,100] here rather than trusting the frontend slider's own
// range, since this is a public binding any frontend call can reach
// directly. Windows-only for now (see transparency_windows.go /
// transparency_other.go); a no-op elsewhere.
func (a *App) SetWindowOpacity(percent int) {
	switch {
	case percent < 0:
		percent = 0
	case percent > 100:
		percent = 100
	}
	setWindowOpacity(uint8(percent * 255 / 100))
}
