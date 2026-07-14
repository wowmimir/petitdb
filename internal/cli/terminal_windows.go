//go:build windows

package cli

import (
	"syscall"
	"unsafe"
)

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode = kernel32.NewProc("GetConsoleMode")
	setConsoleMode = kernel32.NewProc("SetConsoleMode")
)

// enableRawMode sets console to raw mode (disable echo, etc.)
// Returns the original mode for restoration.
func enableRawMode(fd int) (interface{}, error) {
	var mode uint32
	r1, _, err := getConsoleMode.Call(uintptr(fd), uintptr(unsafe.Pointer(&mode)))
	if r1 == 0 {
		return nil, err
	}
	orig := mode

	// Disable ENABLE_LINE_INPUT, ENABLE_ECHO_INPUT, ENABLE_PROCESSED_INPUT
	newMode := mode &^ (uint32(0x0002) | uint32(0x0004) | uint32(0x0001))
	// Enable ENABLE_VIRTUAL_TERMINAL_INPUT? Not needed for simple raw.
	r1, _, err = setConsoleMode.Call(uintptr(fd), uintptr(newMode))
	if r1 == 0 {
		return nil, err
	}
	return orig, nil
}

// disableRawMode restores console mode.
func disableRawMode(fd int, state interface{}) {
	mode := state.(uint32)
	setConsoleMode.Call(uintptr(fd), uintptr(mode))
}
