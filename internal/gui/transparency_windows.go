//go:build windows

package gui

import (
	"syscall"
	"unsafe"
)

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	procFindWindowW           = user32.NewProc("FindWindowW")
	procGetWindowLongPtrW     = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW     = user32.NewProc("SetWindowLongPtrW")
	procSetLayeredWindowAttrs = user32.NewProc("SetLayeredWindowAttributes")
)

// gwlExStyle is a var, not a const: converting a negative constant directly
// to uintptr is a compile-time overflow error in Go, but converting a
// negative *variable* is well-defined (two's-complement bit pattern) and
// matches what GetWindowLongPtrW/SetWindowLongPtrW expect for their signed
// nIndex parameter.
var gwlExStyle int32 = -20

const (
	wsExLayered = 0x00080000
	lwaAlpha    = 0x2
)

// setWindowOpacity applies a whole-window alpha blend via a layered window
// — the same mechanism a title-bar "opacity" slider uses — so the entire
// rendered window fades uniformly into the desktop behind it, regardless of
// what's drawn inside it. Best-effort: the window is looked up fresh on
// every call (by WindowClassName, set explicitly in main.go) rather than
// cached, since nothing here runs often enough for that to matter, and a
// lookup failure just leaves opacity unchanged instead of erroring the
// whole call.
func setWindowOpacity(alpha uint8) {
	classPtr, err := syscall.UTF16PtrFromString(WindowClassName)
	if err != nil {
		return
	}
	hwnd, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(classPtr)), 0)
	if hwnd == 0 {
		return
	}

	exStyle, _, _ := procGetWindowLongPtrW.Call(hwnd, uintptr(gwlExStyle))
	if exStyle&wsExLayered == 0 {
		procSetWindowLongPtrW.Call(hwnd, uintptr(gwlExStyle), exStyle|wsExLayered)
	}
	procSetLayeredWindowAttrs.Call(hwnd, 0, uintptr(alpha), uintptr(lwaAlpha))
}
