//go:build darwin

package cli

import (
	"syscall"
	"unsafe"
)

func enableRawMode(fd int) (interface{}, error) {
	var termios syscall.Termios
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCGETA, uintptr(unsafe.Pointer(&termios)), 0, 0, 0); err != 0 {
		return nil, err
	}
	orig := termios

	termios.Lflag &^= syscall.ICANON | syscall.ECHO
	termios.Cc[syscall.VMIN] = 1
	termios.Cc[syscall.VTIME] = 0

	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCSETA, uintptr(unsafe.Pointer(&termios)), 0, 0, 0); err != 0 {
		return nil, err
	}
	return orig, nil
}

func disableRawMode(fd int, state interface{}) {
	orig := state.(syscall.Termios)
	syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCSETA, uintptr(unsafe.Pointer(&orig)), 0, 0, 0)
}
