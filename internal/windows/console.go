//go:build windows

// Package windows provides Windows console helpers: ANSI/VT100 enable,
// console title management, and console-vs-pipe detection (spec §F-17).
package windows

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procGetConsTitle = kernel32.NewProc("GetConsoleTitleW")
	procSetConsTitle = kernel32.NewProc("SetConsoleTitleW")
)

var originalTitle string

// EnableANSI enables VT100/ANSI virtual terminal processing on the Windows
// console. No-op when stdout is not a real console.
func EnableANSI() {
	h := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return
	}
	_ = windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}

// SetTitle sets the console window title and saves the original for restore.
func SetTitle(title string) {
	buf := make([]uint16, 256)
	procGetConsTitle.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	originalTitle = syscall.UTF16ToString(buf)

	ptr, _ := syscall.UTF16PtrFromString(title)
	procSetConsTitle.Call(uintptr(unsafe.Pointer(ptr)))
}

// RestoreTitle restores the console title saved by SetTitle.
func RestoreTitle() {
	if originalTitle == "" {
		return
	}
	ptr, _ := syscall.UTF16PtrFromString(originalTitle)
	procSetConsTitle.Call(uintptr(unsafe.Pointer(ptr)))
}

// IsConsole reports whether f is attached to a real Windows console.
func IsConsole(f *os.File) bool {
	h := windows.Handle(f.Fd())
	var mode uint32
	return windows.GetConsoleMode(h, &mode) == nil
}
