//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package main

import (
	"syscall"
	"unsafe"
)

// ttyWidth asks the terminal for its real column count. It returns 0 when
// stdout is not a terminal or the ioctl fails.
func ttyWidth() int {
	var ws struct{ Row, Col, Xpixel, Ypixel uint16 }
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 || ws.Col == 0 {
		return 0
	}
	return int(ws.Col)
}
