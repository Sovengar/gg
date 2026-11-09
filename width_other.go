//go:build !(linux || darwin || freebsd || netbsd || openbsd || dragonfly)

package main

// ttyWidth has no ioctl on this platform, so the width falls back to COLUMNS.
func ttyWidth() int { return 0 }
