//go:build !windows

package gui

// setWindowOpacity is a no-op outside Windows: whole-window alpha blending
// goes through a native layered-window call with no portable equivalent, so
// SetWindowTransparent still records the toggle but has no visible effect
// on Linux/macOS builds.
func setWindowOpacity(alpha uint8) {}
