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

var gwlExStyle int32 = -20

const (
	wsExLayered = 0x00080000
	lwaAlpha    = 0x2
)

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
