// Package ui holds terminal formatting: colours, footer, prompts.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// IsTTY reports whether f is attached to a terminal.
func IsTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func colors() bool {
	switch os.Getenv("GG_COLOR") {
	case "always":
		return true
	case "never":
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return IsTTY(os.Stdout)
}

const (
	ansiReset = "\x1b[0m"
	ansiDim   = "\x1b[90m"
	ansiCyan  = "\x1b[36m"
)

// Dim wraps s in the dim colour when colours are enabled.
func Dim(s string) string {
	if !colors() {
		return s
	}
	return ansiDim + s + ansiReset
}

// Cyan wraps s in cyan when colours are enabled.
func Cyan(s string) string {
	if !colors() {
		return s
	}
	return ansiCyan + s + ansiReset
}

// Errf writes a message to stderr.
func Errf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

// Prompt asks a question on stderr and reads one line from stdin.
// The second result is false when stdin is not a terminal.
func Prompt(question string) (string, bool) {
	if !IsTTY(os.Stdin) {
		return "", false
	}
	fmt.Fprint(os.Stderr, question)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", false
	}
	return strings.ToLower(strings.TrimSpace(line)), true
}

// Enabled reports whether colours may be emitted on stdout.
func Enabled() bool { return colors() }

const (
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiBlue   = "\x1b[34m"
	ansiMag    = "\x1b[35m"
)

// Paint wraps s in an ANSI colour when colours are enabled.
func Paint(colour, s string) string {
	if !colors() {
		return s
	}
	return colour + s + ansiReset
}

// Green, Yellow, Blue and Magenta are the palette helpers.
func Green(s string) string   { return Paint(ansiGreen, s) }
func Yellow(s string) string  { return Paint(ansiYellow, s) }
func Blue(s string) string    { return Paint(ansiBlue, s) }
func Magenta(s string) string { return Paint(ansiMag, s) }

const (
	ansiRed    = "\x1b[31m"
	ansiBGreen = "\x1b[92m"
	ansiBMag   = "\x1b[95m"
)

// Red is used for the short commit changeIDs, so they stand out.
func Red(s string) string { return Paint(ansiRed, s) }

// BrightGreen marks a commit that lives on a remote.
func BrightGreen(s string) string { return Paint(ansiBGreen, s) }

// BrightMagenta marks a commit that only exists locally.
func BrightMagenta(s string) string { return Paint(ansiBMag, s) }
